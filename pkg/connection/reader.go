package connection

import (
	"errors"
	"fmt"
	"io"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/transport"
)

const (
	// MaxPayloadSize defines the maximum permissible payload size in bytes (16 MB).
	// 24-bit max length is 16,777,215 bytes.
	MaxPayloadSize = 16 * 1024 * 1024
)

var (
	// ErrFrameTooLarge is returned when payload length exceeds MaxPayloadSize.
	ErrFrameTooLarge = errors.New("connection: frame payload exceeds maximum size limit (16MB)")
	// ErrIncompleteFrame is returned when transport returns fewer bytes than expected.
	ErrIncompleteFrame = errors.New("connection: incomplete frame read from transport")
)

// Reader abstracts the protocol frame extraction from the network transport.
type Reader interface {
	// ReadFrame reads and returns a complete raw binary frame (10-byte header + payload).
	ReadFrame() ([]byte, error)
}

// TCPReader implements Reader for continuous stream transports (TCP TLS).
type TCPReader struct {
	t transport.Transport
}

// NewTCPReader constructs a TCPReader over the given Transport.
func NewTCPReader(t transport.Transport) *TCPReader {
	return &TCPReader{t: t}
}

// ReadFrame reads exactly 10 bytes of header, determines payload length,
// reads the payload, and returns the full frame bytes.
func (r *TCPReader) ReadFrame() ([]byte, error) {
	// 1. Read exactly 10 bytes header
	headerBytes, err := r.t.Recv(protocol.HeaderSize)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("read header: %w", err)
	}

	if len(headerBytes) < protocol.HeaderSize {
		return nil, io.ErrUnexpectedEOF
	}

	// 2. Decode header to extract PayloadLen
	header, err := protocol.DecodeHeader(headerBytes)
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}

	// 3. Security threshold validation
	if header.PayloadLen > MaxPayloadSize {
		return nil, fmt.Errorf("%w: %d bytes", ErrFrameTooLarge, header.PayloadLen)
	}

	// 4. Handle zero-length payload
	if header.PayloadLen == 0 {
		return headerBytes, nil
	}

	// 5. Read exactly PayloadLen bytes
	payloadBytes, err := r.t.Recv(int(header.PayloadLen))
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, fmt.Errorf("read payload: %w", err)
	}

	if len(payloadBytes) < int(header.PayloadLen) {
		return nil, io.ErrUnexpectedEOF
	}

	// 6. Assemble complete frame
	frame := make([]byte, protocol.HeaderSize+int(header.PayloadLen))
	copy(frame[:protocol.HeaderSize], headerBytes)
	copy(frame[protocol.HeaderSize:], payloadBytes)
	return frame, nil
}

// WSReader implements Reader for message-oriented transports (WebSocket).
type WSReader struct {
	t transport.Transport
}

// NewWSReader constructs a WSReader over the given Transport.
func NewWSReader(t transport.Transport) *WSReader {
	return &WSReader{t: t}
}

// ReadFrame reads a single binary WebSocket message containing the complete frame.
func (r *WSReader) ReadFrame() ([]byte, error) {
	// -1 signifies full frame read from WebSocket transport
	data, err := r.t.Recv(-1)
	if err != nil {
		return nil, err
	}

	if len(data) < protocol.HeaderSize {
		return nil, ErrIncompleteFrame
	}

	return data, nil
}
