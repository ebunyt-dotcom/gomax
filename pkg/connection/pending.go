package connection

import (
	"errors"
	"sync"

	"gomax/pkg/protocol"
)

var (
	// ErrRequestCancelled is returned when a pending request is explicitly discarded or cancelled.
	ErrRequestCancelled = errors.New("connection: request cancelled")
	// ErrConnectionClosed is returned when all pending requests are cancelled due to connection termination.
	ErrConnectionClosed = errors.New("connection: connection closed")
)

// PendingTracker is a thread-safe registry that maps outbound frame sequence numbers (seq)
// to inbound response channels.
type PendingTracker struct {
	mu      sync.Mutex
	pending map[uint16]chan *protocol.InboundFrame
}

// NewPendingTracker constructs an empty PendingTracker.
func NewPendingTracker() *PendingTracker {
	return &PendingTracker{
		pending: make(map[uint16]chan *protocol.InboundFrame),
	}
}

// Create allocates a buffered response channel for the given sequence number.
// If an entry for this seq already exists (e.g. sequence collision on overflow),
// the previous channel is closed to prevent dangling waiters.
func (p *PendingTracker) Create(seq uint16) chan *protocol.InboundFrame {
	p.mu.Lock()
	defer p.mu.Unlock()

	if oldCh, exists := p.pending[seq]; exists {
		close(oldCh)
	}

	ch := make(chan *protocol.InboundFrame, 1)
	p.pending[seq] = ch
	return ch
}

// Resolve delivers an inbound frame to the pending request matching seq,
// removes the entry from the tracker, and closes the channel.
// Returns true if a matching pending request was found and resolved.
func (p *PendingTracker) Resolve(seq uint16, frame *protocol.InboundFrame) bool {
	p.mu.Lock()
	ch, ok := p.pending[seq]
	if !ok {
		p.mu.Unlock()
		return false
	}
	delete(p.pending, seq)
	p.mu.Unlock()

	ch <- frame
	close(ch)
	return true
}

// Discard removes and closes the pending request channel for seq without delivering a response.
// Typically called in defer statements when a request times out or caller cancels.
func (p *PendingTracker) Discard(seq uint16) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ch, ok := p.pending[seq]; ok {
		delete(p.pending, seq)
		close(ch)
	}
}

// CancelAll cancels and closes all in-flight pending requests.
// Used when the underlying network connection fails or is closed.
func (p *PendingTracker) CancelAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for seq, ch := range p.pending {
		delete(p.pending, seq)
		close(ch)
	}
}

// Count returns the number of currently active pending requests.
func (p *PendingTracker) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pending)
}

// Has checks if a pending request exists for the given sequence number.
func (p *PendingTracker) Has(seq uint16) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.pending[seq]
	return ok
}
