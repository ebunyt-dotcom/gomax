package connection_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gomax/pkg/connection"
	"gomax/pkg/protocol"
)

// ==========================================
// Mocks & Test Doubles
// ==========================================

type MockTransport struct {
	mu        sync.Mutex
	sent      [][]byte
	recvQueue [][]byte
	recvErr   error
	connected bool
	closed    bool
	blockCh   chan struct{}
}

func NewMockTransport() *MockTransport {
	return &MockTransport{
		connected: true,
		blockCh:   make(chan struct{}),
	}
}

func (m *MockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	m.closed = false
	m.blockCh = make(chan struct{})
	return nil
}

func (m *MockTransport) Send(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("mock transport closed")
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.sent = append(m.sent, cp)
	return nil
}

func (m *MockTransport) Recv(n int) ([]byte, error) {
	m.mu.Lock()
	if m.recvErr != nil {
		defer m.mu.Unlock()
		return nil, m.recvErr
	}
	if len(m.recvQueue) > 0 {
		chunk := m.recvQueue[0]
		m.recvQueue = m.recvQueue[1:]
		m.mu.Unlock()
		return chunk, nil
	}
	blockCh := m.blockCh
	m.mu.Unlock()

	<-blockCh
	return nil, io.EOF
}

func (m *MockTransport) Connected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		m.connected = false
		close(m.blockCh)
	}
	return nil
}

func (m *MockTransport) EnqueueRecv(chunks ...[]byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvQueue = append(m.recvQueue, chunks...)
}

func (m *MockTransport) SetRecvError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvErr = err
}

type MockProtocol struct{}

func (p *MockProtocol) Version() uint8 {
	return 10
}

func (p *MockProtocol) Encode(frame *protocol.OutboundFrame) ([]byte, error) {
	hdr := protocol.Header{
		Version:    frame.Version,
		Cmd:        frame.Cmd,
		Seq:        frame.Seq,
		Opcode:     frame.Opcode,
		Flags:      frame.Flags,
		PayloadLen: 0,
	}
	return hdr.Encode(), nil
}

func (p *MockProtocol) Decode(data []byte) (*protocol.InboundFrame, error) {
	hdr, err := protocol.DecodeHeader(data)
	if err != nil {
		return nil, err
	}
	return &protocol.InboundFrame{
		Header:  *hdr,
		Opcode:  hdr.Opcode,
		Cmd:     hdr.Cmd,
		Seq:     hdr.Seq,
		Payload: map[string]any{"ok": true},
		Raw:     data,
	}, nil
}

// ==========================================
// Unit Tests: SeqGenerator
// ==========================================

func TestSeqGenerator_Sequential(t *testing.T) {
	s := connection.NewSeqGenerator()
	for i := uint16(0); i < 100; i++ {
		got := s.Next()
		if got != i {
			t.Fatalf("expected seq %d, got %d", i, got)
		}
	}
}

func TestSeqGenerator_Wrapping(t *testing.T) {
	s := connection.NewSeqGeneratorWithStart(0xFFFE)
	if got := s.Next(); got != 0xFFFE {
		t.Fatalf("expected 0xFFFE, got %d", got)
	}
	if got := s.Next(); got != 0xFFFF {
		t.Fatalf("expected 0xFFFF, got %d", got)
	}
	if got := s.Next(); got != 0x0000 {
		t.Fatalf("expected wrap to 0, got %d", got)
	}
	if got := s.Next(); got != 0x0001 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestSeqGenerator_Concurrent(t *testing.T) {
	s := connection.NewSeqGenerator()
	var wg sync.WaitGroup
	var counts [65536]atomic.Uint32

	workers := 10
	iterations := 65536

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				seq := s.Next()
				counts[seq].Add(1)
			}
		}()
	}
	wg.Wait()

	// Ensure no race conditions and all sequence bins were incremented
	for i := 0; i < 65536; i++ {
		if counts[i].Load() == 0 {
			t.Fatalf("sequence %d was never generated", i)
		}
	}
}

// ==========================================
// Unit Tests: PendingTracker
// ==========================================

func TestPendingTracker_ResolveAndDiscard(t *testing.T) {
	tracker := connection.NewPendingTracker()

	ch1 := tracker.Create(1)
	if tracker.Count() != 1 {
		t.Fatalf("expected count 1, got %d", tracker.Count())
	}

	frame := &protocol.InboundFrame{
		Seq:     1,
		Opcode:  10,
		Cmd:     protocol.CmdResponse,
		Payload: map[string]any{"status": "success"},
	}

	resolved := tracker.Resolve(1, frame)
	if !resolved {
		t.Fatal("expected resolve to succeed")
	}

	select {
	case res := <-ch1:
		if res.Seq != 1 || res.Opcode != 10 {
			t.Fatalf("unexpected frame received: %+v", res)
		}
	default:
		t.Fatal("expected frame in channel")
	}

	// Second resolve should return false
	if tracker.Resolve(1, frame) {
		t.Fatal("expected second resolve to return false")
	}

	// Discard test
	_ = tracker.Create(2)
	tracker.Discard(2)
	if tracker.Has(2) {
		t.Fatal("expected seq 2 to be discarded")
	}
}

func TestPendingTracker_CancelAll(t *testing.T) {
	tracker := connection.NewPendingTracker()
	ch1 := tracker.Create(10)
	ch2 := tracker.Create(20)

	tracker.CancelAll()

	if tracker.Count() != 0 {
		t.Fatalf("expected 0 pending, got %d", tracker.Count())
	}

	// Channels must be closed
	if _, ok := <-ch1; ok {
		t.Fatal("expected ch1 to be closed")
	}
	if _, ok := <-ch2; ok {
		t.Fatal("expected ch2 to be closed")
	}
}

// ==========================================
// Unit Tests: TCPReader & WSReader
// ==========================================

func TestTCPReader_ReadHeaderThenPayload(t *testing.T) {
	mt := NewMockTransport()
	reader := connection.NewTCPReader(mt)

	// Create 10-byte header with payload length = 3
	hdr := protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdRequest,
		Seq:        1,
		Opcode:     protocol.OpPing,
		PayloadLen: 3,
	}
	headerBytes := hdr.Encode()
	payloadBytes := []byte("abc")

	mt.EnqueueRecv(headerBytes, payloadBytes)

	frame, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if len(frame) != 13 {
		t.Fatalf("expected 13 bytes, got %d", len(frame))
	}
}

func TestTCPReader_ZeroPayload(t *testing.T) {
	mt := NewMockTransport()
	reader := connection.NewTCPReader(mt)

	hdr := protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdResponse,
		Seq:        1,
		Opcode:     protocol.OpPing,
		PayloadLen: 0,
	}
	mt.EnqueueRecv(hdr.Encode())

	frame, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}
	if len(frame) != 10 {
		t.Fatalf("expected 10 bytes, got %d", len(frame))
	}
}

func TestTCPReader_EOFOnHeader(t *testing.T) {
	mt := NewMockTransport()
	reader := connection.NewTCPReader(mt)

	mt.SetRecvError(io.EOF)

	_, err := reader.ReadFrame()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF error, got %v", err)
	}
}

func TestWSReader_BinaryMessage(t *testing.T) {
	mt := NewMockTransport()
	reader := connection.NewWSReader(mt)

	fullMessage := append((&protocol.Header{Version: 10, Opcode: 1}).Encode(), []byte("data")...)
	mt.EnqueueRecv(fullMessage)

	got, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("WSReader ReadFrame failed: %v", err)
	}
	if len(got) != len(fullMessage) {
		t.Fatalf("expected %d bytes, got %d", len(fullMessage), len(got))
	}
}

// ==========================================
// Unit Tests: ConnectionManager Multiplexing & Keepalive
// ==========================================

func TestConnectionManager_SendRequestAndMultiplex(t *testing.T) {
	mt := NewMockTransport()
	reader := connection.NewTCPReader(mt)
	proto := &MockProtocol{}

	cfg := connection.DefaultConfig()
	cfg.PingInterval = 10 * time.Second // disable fast ping in this test

	mgr := connection.NewConnectionManager(reader, mt, proto, &cfg, nil, nil)
	ctx := context.Background()

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}
	defer mgr.Close()

	// Simulate incoming response frame for seq=0 (first generated seq)
	respHeader := protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdResponse,
		Seq:        0,
		Opcode:     protocol.OpSessionInit,
		PayloadLen: 0,
	}
	mt.EnqueueRecv(respHeader.Encode())

	resp, err := mgr.SendRequest(ctx, protocol.OpSessionInit, map[string]any{})
	if err != nil {
		t.Fatalf("SendRequest failed: %v", err)
	}

	if resp.Seq != 0 || resp.Opcode != protocol.OpSessionInit {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestConnectionManager_KeepaliveTimeout(t *testing.T) {
	mt := NewMockTransport()
	reader := connection.NewTCPReader(mt)
	proto := &MockProtocol{}

	var closedErr atomic.Pointer[error]
	onClose := func(err error) {
		closedErr.Store(&err)
	}

	cfg := connection.DefaultConfig()
	cfg.PingInterval = 50 * time.Millisecond
	cfg.PingTimeout = 50 * time.Millisecond

	mgr := connection.NewConnectionManager(reader, mt, proto, &cfg, onClose, nil)
	ctx := context.Background()

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}

	// Do not enqueue ping responses: ping should time out after 50ms
	err := mgr.WaitClosed()
	if err == nil {
		t.Fatal("expected connection to fail on ping timeout")
	}

	if !errors.Is(err, connection.ErrPingTimeout) {
		t.Fatalf("expected ErrPingTimeout, got %v", err)
	}

	if mgr.IsOpen() {
		t.Fatal("expected manager to be closed")
	}
}
