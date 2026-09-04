package connection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/transport"
)

var (
	// ErrConnectionNotOpen is returned when sending over a closed connection.
	ErrConnectionNotOpen = errors.New("connection: connection is not open")
	// ErrConnectionAlreadyOpen is returned when Start is called on an already active connection.
	ErrConnectionAlreadyOpen = errors.New("connection: connection is already open")
	// ErrPingTimeout is returned when a keepalive ping times out.
	ErrPingTimeout = errors.New("connection: keepalive ping timed out")
)

// Default connection settings matching PyMax
const (
	DefaultPingInterval   = 30 * time.Second
	DefaultPingTimeout    = 30 * time.Second
	DefaultRequestTimeout = 30 * time.Second
	DefaultEventsChanSize = 2048
)

// Config specifies connection manager parameters.
type Config struct {
	Interactive     bool
	PingInterval    time.Duration
	PingTimeout     time.Duration
	RequestTimeout  time.Duration
	EventsChanSize  int
	ProtocolVersion uint8
}

// DefaultConfig returns default configuration parameters.
func DefaultConfig() Config {
	return Config{
		Interactive:     true,
		PingInterval:    DefaultPingInterval,
		PingTimeout:     DefaultPingTimeout,
		RequestTimeout:  DefaultRequestTimeout,
		EventsChanSize:  DefaultEventsChanSize,
		ProtocolVersion: protocol.VersionTcp,
	}
}

// ConnectionManager manages connection lifecycle, framing, RPC correlation, and keepalive.
type ConnectionManager struct {
	cfg       Config
	reader    Reader
	transport transport.Transport
	protocol  protocol.Protocol
	seqGen    *SeqGenerator
	pending   *PendingTracker

	isOpen         atomic.Bool
	closedReported atomic.Bool
	closedErr      atomic.Pointer[error]
	closedCh       chan struct{}

	loopCtx    context.Context
	loopCancel context.CancelFunc
	wg         sync.WaitGroup

	eventsCh chan *protocol.InboundFrame

	onClose func(error)
	onEvent func(*protocol.InboundFrame)
}

// NewConnectionManager constructs a ConnectionManager.
func NewConnectionManager(
	reader Reader,
	t transport.Transport,
	p protocol.Protocol,
	cfg *Config,
	onClose func(error),
	onEvent func(*protocol.InboundFrame),
) *ConnectionManager {
	c := DefaultConfig()
	if cfg != nil {
		c = *cfg
	}

	return &ConnectionManager{
		cfg:       c,
		reader:    reader,
		transport: t,
		protocol:  p,
		seqGen:    NewSeqGenerator(),
		pending:   NewPendingTracker(),
		closedCh:  make(chan struct{}),
		eventsCh:  make(chan *protocol.InboundFrame, c.EventsChanSize),
		onClose:   onClose,
		onEvent:   onEvent,
	}
}

// Start opens the transport, launches background receive and keepalive loops.
func (m *ConnectionManager) Start(ctx context.Context) error {
	if m.isOpen.Swap(true) {
		return ErrConnectionAlreadyOpen
	}

	if err := m.transport.Connect(ctx); err != nil {
		m.isOpen.Store(false)
		return fmt.Errorf("transport connect: %w", err)
	}

	m.closedReported.Store(false)
	m.closedErr.Store(nil)
	m.closedCh = make(chan struct{})

	m.loopCtx, m.loopCancel = context.WithCancel(context.Background())

	// Launch receive loop
	m.wg.Add(1)
	go m.recvLoop(m.loopCtx)

	// Launch keepalive ping loop
	m.wg.Add(1)
	go m.keepaliveLoop(m.loopCtx)

	return nil
}

// Close gracefully shuts down the connection.
func (m *ConnectionManager) Close() error {
	if !m.isOpen.Swap(false) {
		return nil
	}

	if m.loopCancel != nil {
		m.loopCancel()
	}

	m.pending.CancelAll()
	err := m.transport.Close()

	m.wg.Wait()
	m.markClosed(nil)
	return err
}

// Fail marks the connection as failed due to network error or timeout.
func (m *ConnectionManager) Fail(err error) {
	if !m.isOpen.Swap(false) {
		return
	}

	if m.loopCancel != nil {
		m.loopCancel()
	}

	m.pending.CancelAll()
	_ = m.transport.Close()

	m.markClosed(err)
}

// WaitClosed blocks until the connection is closed or fails.
func (m *ConnectionManager) WaitClosed() error {
	<-m.closedCh
	if ptr := m.closedErr.Load(); ptr != nil {
		return *ptr
	}
	return nil
}

func (m *ConnectionManager) markClosed(err error) {
	if m.closedReported.Swap(true) {
		return
	}

	if err != nil {
		m.closedErr.Store(&err)
	}

	close(m.closedCh)

	if m.onClose != nil {
		m.onClose(err)
	}
}

// IsOpen returns whether the connection is active.
func (m *ConnectionManager) IsOpen() bool {
	return m.isOpen.Load()
}

// Events returns the receive-only channel for streaming push events.
func (m *ConnectionManager) Events() <-chan *protocol.InboundFrame {
	return m.eventsCh
}

// Send sends an outbound frame without registering for a correlated response.
func (m *ConnectionManager) Send(ctx context.Context, frame *protocol.OutboundFrame) error {
	if !m.IsOpen() {
		return ErrConnectionNotOpen
	}

	raw, err := m.protocol.Encode(frame)
	if err != nil {
		return fmt.Errorf("encode frame: %w", err)
	}

	if err := m.transport.Send(raw); err != nil {
		return fmt.Errorf("transport send: %w", err)
	}

	return nil
}

// SendEvent sends a one-way event frame (cmd=CmdEvent, seq=0).
func (m *ConnectionManager) SendEvent(ctx context.Context, opcode protocol.Opcode, payload any) error {
	frame := &protocol.OutboundFrame{
		Version: m.cfg.ProtocolVersion,
		Cmd:     protocol.CmdEvent,
		Seq:     0,
		Opcode:  opcode,
		Payload: payload,
	}
	return m.Send(ctx, frame)
}

// SendRequest sends a request frame and awaits the correlated response matching seq.
func (m *ConnectionManager) SendRequest(ctx context.Context, opcode protocol.Opcode, payload any) (*protocol.InboundFrame, error) {
	if !m.IsOpen() {
		return nil, ErrConnectionNotOpen
	}

	seq := m.seqGen.Next()
	ch := m.pending.Create(seq)
	defer m.pending.Discard(seq)

	frame := &protocol.OutboundFrame{
		Version: m.cfg.ProtocolVersion,
		Cmd:     protocol.CmdRequest,
		Seq:     seq,
		Opcode:  opcode,
		Payload: payload,
	}

	raw, err := m.protocol.Encode(frame)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	if err := m.transport.Send(raw); err != nil {
		return nil, fmt.Errorf("transport send: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-ch:
		if !ok || resp == nil {
			if !m.IsOpen() {
				if ptr := m.closedErr.Load(); ptr != nil {
					return nil, fmt.Errorf("%w: %v", ErrConnectionClosed, *ptr)
				}
				return nil, ErrConnectionClosed
			}
			return nil, ErrRequestCancelled
		}
		if resp.Cmd == protocol.CmdError {
			return nil, protocol.NewApiError(resp)
		}
		return resp, nil
	}
}

// Handshake executes the mandatory SESSION_INIT exchange (Opcode 6) upon connection.
func (m *ConnectionManager) Handshake(ctx context.Context, payload any) (*protocol.InboundFrame, error) {
	return m.SendRequest(ctx, protocol.OpSessionInit, payload)
}

// recvLoop continuously reads frames and dispatches them.
func (m *ConnectionManager) recvLoop(ctx context.Context) {
	defer m.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		raw, err := m.reader.ReadFrame()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				m.Fail(fmt.Errorf("connection closed by server: %w", err))
			} else {
				m.Fail(fmt.Errorf("read frame failed: %w", err))
			}
			return
		}

		frame, err := m.protocol.Decode(raw)
		if err != nil {
			continue // Skip malformed frames and keep reading
		}

		m.handleInbound(frame)
	}
}

// handleInbound routes incoming frames to pending requests or event listeners.
func (m *ConnectionManager) handleInbound(frame *protocol.InboundFrame) {
	// If frame is a response or error matching a pending request, resolve it
	if frame.Cmd == protocol.CmdResponse || frame.Cmd == protocol.CmdError {
		m.pending.Resolve(frame.Seq, frame)
	}

	// Deliver to event channel
	select {
	case m.eventsCh <- frame:
	default:
		// Prevent event backlog from stalling receive loop
	}

	// Fire callback if registered
	if m.onEvent != nil {
		go m.onEvent(frame)
	}
}

// keepaliveLoop sends OpcodePing (1) every 30 seconds.
func (m *ConnectionManager) keepaliveLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, m.cfg.PingTimeout)
			_, err := m.SendRequest(pingCtx, protocol.OpPing, map[string]any{
				"interactive": m.cfg.Interactive,
			})
			cancel()

			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				m.Fail(fmt.Errorf("%w: %v", ErrPingTimeout, err))
				return
			}
		}
	}
}
