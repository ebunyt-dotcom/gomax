package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

const (
	// MaxDecompressedSize is the maximum permitted size of a decompressed payload (5 MB).
	MaxDecompressedSize = 5 * 1024 * 1024
)

var (
	// ErrOutputTooLarge is returned when decompressed data exceeds the 5MB ceiling.
	ErrOutputTooLarge = errors.New("protocol: decompression output too large: exceeds 5MB")

	// ErrInvalidCompressionFactor is returned when flags byte > 0x7F and != 0xFF.
	ErrInvalidCompressionFactor = errors.New("protocol: invalid TCP compression factor")

	// LZ4 specific decompression errors matching PyMax verbatim.
	ErrLZ4LiteralOutOfBounds = errors.New("LZ4: literal length out of bounds")
	ErrLZ4IncompleteOffset   = errors.New("LZ4: incomplete offset")
	ErrLZ4ZeroOffset         = errors.New("LZ4: zero offset")
	ErrLZ4MatchOutOfBounds   = errors.New("LZ4: match out of bounds")
	ErrZstdDecompress        = errors.New("Zstd: failed to decompress payload")
)

// Decompressor defines the interface for payload decompression algorithms.
type Decompressor interface {
	Decompress(src []byte, maxOutput int) ([]byte, error)
}

// LZ4BlockDecompressor implements pure Go raw LZ4 block decompression without Cgo.
type LZ4BlockDecompressor struct{}

// NewLZ4BlockDecompressor returns a new LZ4BlockDecompressor.
func NewLZ4BlockDecompressor() *LZ4BlockDecompressor {
	return &LZ4BlockDecompressor{}
}

// Decompress decodes raw LZ4 block bytes, ensuring the output does not exceed maxOutput.
func (d *LZ4BlockDecompressor) Decompress(src []byte, maxOutput int) ([]byte, error) {
	if maxOutput <= 0 {
		maxOutput = MaxDecompressedSize
	}

	dst := make([]byte, 0, len(src)*2)
	pos := 0

	for pos < len(src) {
		token := src[pos]
		pos++

		// High 4 bits: literal length
		litLen := int(token >> 4)
		if litLen == 15 {
			for pos < len(src) {
				b := int(src[pos])
				pos++
				litLen += b
				if b != 255 {
					break
				}
			}
		}

		if litLen > 0 {
			if pos+litLen > len(src) {
				return nil, ErrLZ4LiteralOutOfBounds
			}
			dst = append(dst, src[pos:pos+litLen]...)
			pos += litLen
			if len(dst) > maxOutput {
				return nil, ErrOutputTooLarge
			}
		}

		if pos >= len(src) {
			break
		}

		if pos+1 >= len(src) {
			return nil, ErrLZ4IncompleteOffset
		}

		// 2-byte little endian offset
		offset := int(src[pos]) | (int(src[pos+1]) << 8)
		pos += 2

		if offset == 0 {
			return nil, ErrLZ4ZeroOffset
		}

		// Low 4 bits: match length (+ 4)
		matchLen := int(token&0x0F) + 4
		if (token & 0x0F) == 0x0F {
			for pos < len(src) {
				b := int(src[pos])
				pos++
				matchLen += b
				if b != 255 {
					break
				}
			}
		}

		matchPos := len(dst) - offset
		if matchPos < 0 {
			return nil, ErrLZ4MatchOutOfBounds
		}

		for i := 0; i < matchLen; i++ {
			dst = append(dst, dst[matchPos+(i%offset)])
		}

		if len(dst) > maxOutput {
			return nil, ErrOutputTooLarge
		}
	}

	return dst, nil
}

// ZstdDecompressor implements Zstandard decompression using klauspost/compress/zstd.
type ZstdDecompressor struct {
	mu      sync.Mutex
	decoder *zstd.Decoder
}

// NewZstdDecompressor returns a new ZstdDecompressor instance.
func NewZstdDecompressor() (*ZstdDecompressor, error) {
	dec, err := zstd.NewReader(nil, zstd.WithDecoderMaxWindow(8<<20))
	if err != nil {
		return nil, fmt.Errorf("protocol: init zstd reader: %w", err)
	}
	return &ZstdDecompressor{decoder: dec}, nil
}

// Decompress decodes Zstandard bytes, strictly checking the maxOutput ceiling.
func (d *ZstdDecompressor) Decompress(src []byte, maxOutput int) ([]byte, error) {
	if maxOutput <= 0 {
		maxOutput = MaxDecompressedSize
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	in := bytes.NewReader(src)
	if err := d.decoder.Reset(in); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrZstdDecompress, err)
	}

	limited := io.LimitReader(d.decoder, int64(maxOutput+1))
	out, err := io.ReadAll(limited)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: %v", ErrZstdDecompress, err)
	}

	if len(out) > maxOutput {
		return nil, ErrOutputTooLarge
	}

	return out, nil
}

// PayloadDecoder coordinates decompression and MessagePack decoding of frame payloads.
type PayloadDecoder struct {
	codec *MsgpackCodec
	lz4   *LZ4BlockDecompressor
	zstd  *ZstdDecompressor
}

// NewPayloadDecoder initializes a PayloadDecoder with both LZ4 and Zstd engines.
func NewPayloadDecoder(codec *MsgpackCodec) (*PayloadDecoder, error) {
	zstdDec, err := NewZstdDecompressor()
	if err != nil {
		return nil, err
	}
	return &PayloadDecoder{
		codec: codec,
		lz4:   NewLZ4BlockDecompressor(),
		zstd:  zstdDec,
	}, nil
}

// Decode applies decompression (if indicated by flags) and deserializes the payload into a map[string]any.
func (d *PayloadDecoder) Decode(payloadBytes []byte, flags uint8) (map[string]any, error) {
	if len(payloadBytes) == 0 {
		return make(map[string]any), nil
	}

	decompressed := payloadBytes

	switch {
	case flags == CompressionZstd:
		var err error
		decompressed, err = d.zstd.Decompress(payloadBytes, MaxDecompressedSize)
		if err != nil {
			return nil, err
		}

	case flags > 0x7F:
		return nil, fmt.Errorf("%w: 0x%02X", ErrInvalidCompressionFactor, flags)

	case flags > CompressionNone:
		var err error
		decompressed, err = d.lz4.Decompress(payloadBytes, MaxDecompressedSize)
		if err != nil {
			return nil, err
		}
	}

	decoded, err := d.codec.Decode(decompressed)
	if err != nil {
		return nil, err
	}

	if asMap, ok := decoded.(map[string]any); ok {
		return asMap, nil
	}

	return map[string]any{"_value": decoded}, nil
}
