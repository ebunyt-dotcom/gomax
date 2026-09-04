package connection

import (
	"context"

	"gomax/pkg/protocol"
)

// Connection defines the public interface required by RPC services, clients, and event dispatchers.
type Connection interface {
	// Start establishes the transport connection and launches background loops.
	Start(ctx context.Context) error
	// Close gracefully terminates the connection.
	Close() error
	// WaitClosed blocks until the connection is terminated or fails.
	WaitClosed() error
	// SendRequest transmits an RPC request and waits for the correlated response matching seq.
	SendRequest(ctx context.Context, opcode protocol.Opcode, payload any) (*protocol.InboundFrame, error)
	// SendEvent sends a one-way notification/event frame to the server.
	SendEvent(ctx context.Context, opcode protocol.Opcode, payload any) error
	// Events returns the receive-only channel for server push events.
	Events() <-chan *protocol.InboundFrame
	// Handshake executes the mandatory SESSION_INIT exchange (Opcode 6).
	Handshake(ctx context.Context, payload any) (*protocol.InboundFrame, error)
	// IsOpen reports whether the connection is currently established and active.
	IsOpen() bool
}
