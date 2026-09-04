package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// HeaderSize is the fixed size of the binary frame header in bytes.
	HeaderSize = 10

	// VersionTcp is the protocol version byte for TCP TLS and binary WebSocket framing.
	VersionTcp uint8 = 10

	// VersionWs is the legacy protocol version byte for JSON string WebSocket framing.
	VersionWs uint8 = 11

	// MaxPayloadLen is the 24-bit maximum payload size (16,777,215 bytes ~ 16 MB).
	MaxPayloadLen = 0x00FFFFFF

	// CompressionNone indicates uncompressed MessagePack payload.
	CompressionNone uint8 = 0x00

	// CompressionZstd indicates Zstandard compressed payload.
	CompressionZstd uint8 = 0xFF
)

var (
	// ErrHeaderTooShort is returned when fewer than 10 bytes are supplied.
	ErrHeaderTooShort = errors.New("protocol: data too short for 10-byte header")

	// ErrPayloadLengthExceeded is returned when a payload exceeds the 24-bit limit.
	ErrPayloadLengthExceeded = errors.New("protocol: payload length exceeds 24-bit maximum (16MB)")
)

// Header represents the unpacked 10-byte Max protocol binary frame header.
type Header struct {
	Version    uint8   `json:"ver"`
	Cmd        Command `json:"cmd"`
	Seq        uint16  `json:"seq"`
	Opcode     Opcode  `json:"opcode"`
	Flags      uint8   `json:"flags"`
	PayloadLen uint32  `json:"payload_len"`
}

// Encode serializes the Header into a 10-byte binary slice in Network Byte Order (Big-Endian).
func (h *Header) Encode() []byte {
	buf := make([]byte, HeaderSize)
	_ = h.EncodeTo(buf)
	return buf
}

// EncodeTo encodes the Header into an existing slice of at least 10 bytes without heap allocation.
func (h *Header) EncodeTo(dst []byte) error {
	if len(dst) < HeaderSize {
		return ErrHeaderTooShort
	}
	if h.PayloadLen > MaxPayloadLen {
		return ErrPayloadLengthExceeded
	}

	dst[0] = h.Version
	dst[1] = uint8(h.Cmd)
	binary.BigEndian.PutUint16(dst[2:4], h.Seq)
	binary.BigEndian.PutUint16(dst[4:6], uint16(h.Opcode))

	// Bit 31..24 = Flags (8 bits), Bit 23..0 = PayloadLen (24 bits)
	packedLen := (uint32(h.Flags) << 24) | (h.PayloadLen & MaxPayloadLen)
	binary.BigEndian.PutUint32(dst[6:10], packedLen)
	return nil
}

// DecodeHeader deserializes a 10-byte binary slice into a Header struct.
func DecodeHeader(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("%w: got %d bytes, expected %d", ErrHeaderTooShort, len(data), HeaderSize)
	}

	packedLen := binary.BigEndian.Uint32(data[6:10])
	return &Header{
		Version:    data[0],
		Cmd:        Command(data[1]),
		Seq:        binary.BigEndian.Uint16(data[2:4]),
		Opcode:     Opcode(binary.BigEndian.Uint16(data[4:6])),
		Flags:      uint8(packedLen >> 24),
		PayloadLen: packedLen & MaxPayloadLen,
	}, nil
}

// ExtractPayloadLen fast-unpacks only the payload length from a 10-byte header buffer.
func ExtractPayloadLen(headerData []byte) (uint32, error) {
	if len(headerData) < HeaderSize {
		return 0, ErrHeaderTooShort
	}
	packedLen := binary.BigEndian.Uint32(headerData[6:10])
	return packedLen & MaxPayloadLen, nil
}
