package api

import (
	"context"
	"gomax/pkg/protocol"
)

// Invoker represents an RPC invoker capable of sending protocol frames and awaiting responses.
type Invoker interface {
	Invoke(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error)
}
