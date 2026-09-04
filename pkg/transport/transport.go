package transport

import (
	"context"
	"errors"
	"io"
)

var (
	// ErrNotConnected is returned when an operation is attempted on an unestablished transport.
	ErrNotConnected = errors.New("transport: not connected")

	// ErrAlreadyConnected is returned when Connect is called on an already established transport.
	ErrAlreadyConnected = errors.New("transport: already connected")

	// ErrClosed is returned when an operation is attempted on a closed transport.
	ErrClosed = errors.New("transport: connection closed")

	// ErrInvalidProxy is returned when proxy configuration or dialing fails.
	ErrInvalidProxy = errors.New("transport: invalid proxy configuration")

	// ErrMessageTooLarge is returned when an incoming message exceeds maximum transport size.
	ErrMessageTooLarge = errors.New("transport: message exceeds maximum size")
)

// Transport represents a full-duplex network transport.
//
// All implementations must be safe for concurrent use by multiple goroutines:
// - Send must be serialized to guarantee atomic frame writes.
// - Recv is typically invoked by a single read-loop goroutine, but must safely handle concurrent Close().
// - Close must be idempotent and interrupt any active Recv/Send calls without hanging.
type Transport interface {
	io.Closer

	// Connect establishes the underlying network connection using the provided context.
	// Returns ErrAlreadyConnected if the transport is currently connected.
	Connect(ctx context.Context) error

	// Send transmits a serialized byte frame to the remote endpoint.
	// Implementations must ensure atomic writes without byte interleaving.
	Send(data []byte) error

	// Recv receives data from the remote endpoint.
	//
	// Semantics:
	// - For stream-based transports (TCP): if n > 0, Recv blocks until exactly n bytes
	//   are read (io.ReadFull semantics) or an error occurs. If n <= 0, Recv reads available
	//   bytes into a default buffer.
	// - For frame-based transports (WebSocket): if n <= 0, Recv returns the complete
	//   next binary message/frame. If n > 0, it reads exactly n bytes from the internal
	//   message buffer, fetching subsequent frames as necessary.
	Recv(n int) ([]byte, error)

	// Connected reports whether the transport is currently connected and ready for I/O.
	Connected() bool
}
