package connection_test

import (
	"context"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/connection"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
)

// ============================================================================
// 1. Abrupt Socket Closure / EOF during Header vs Payload Read
// ============================================================================

func TestAdversarial_Reader_AbruptEOF_Header(t *testing.T) {
	// Case A: EOF immediately (0 bytes)
	mtA := NewMockTransport()
	readerA := connection.NewTCPReader(mtA)
	mtA.SetRecvError(io.EOF)

	_, errA := readerA.ReadFrame()
	if !errors.Is(errA, io.EOF) {
		t.Fatalf("expected io.EOF on 0 bytes read, got: %v", errA)
	}

	// Case B: Partial header (4 bytes then EOF)
	mtB := NewMockTransport()
	readerB := connection.NewTCPReader(mtB)
	mtB.EnqueueRecv([]byte{0x0A, 0x00, 0x01, 0x02}) // 4 bytes only
	mtB.SetRecvError(io.EOF)

	_, errB := readerB.ReadFrame()
	if errB == nil {
		t.Fatal("expected error on partial header EOF, got nil")
	}
	// TCPReader wraps with "read header: ..."
	if !errors.Is(errB, io.ErrUnexpectedEOF) && !errors.Is(errB, io.EOF) {
		t.Logf("partial header returned: %v", errB)
	}

	// Case C: 9 bytes then EOF (1 byte short)
	mtC := NewMockTransport()
	readerC := connection.NewTCPReader(mtC)
	mtC.EnqueueRecv(make([]byte, 9))
	mtC.SetRecvError(io.EOF)

	_, errC := readerC.ReadFrame()
	if errC == nil {
		t.Fatal("expected error on 9-byte header EOF, got nil")
	}
}

func TestAdversarial_Reader_AbruptEOF_Payload(t *testing.T) {
	// Case A: 10 bytes header (PayloadLen=50), then EOF immediately
	hdr := protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdRequest,
		Seq:        1,
		Opcode:     protocol.OpPing,
		PayloadLen: 50,
	}
	headerBytes := hdr.Encode()

	mtA := NewMockTransport()
	readerA := connection.NewTCPReader(mtA)
	mtA.EnqueueRecv(headerBytes)
	mtA.SetRecvError(io.EOF)

	_, errA := readerA.ReadFrame()
	if errA == nil {
		t.Fatal("expected error when EOF occurs at payload start, got nil")
	}
	if !errors.Is(errA, io.ErrUnexpectedEOF) {
		t.Fatalf("expected io.ErrUnexpectedEOF, got: %v", errA)
	}

	// Case B: 10 bytes header (PayloadLen=50), 20 bytes payload then EOF
	mtB := NewMockTransport()
	readerB := connection.NewTCPReader(mtB)
	mtB.EnqueueRecv(headerBytes, make([]byte, 20))
	mtB.SetRecvError(io.EOF)

	_, errB := readerB.ReadFrame()
	if errB == nil {
		t.Fatal("expected error when EOF occurs during payload, got nil")
	}
	if !errors.Is(errB, io.ErrUnexpectedEOF) {
		t.Fatalf("expected io.ErrUnexpectedEOF, got: %v", errB)
	}
}

func TestAdversarial_Reader_OversizedPayloadRejection(t *testing.T) {
	hdr := protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdRequest,
		Seq:        1,
		Opcode:     protocol.OpPing,
		PayloadLen: connection.MaxPayloadSize + 1, // Exceeds 16MB limit
	}

	mt := NewMockTransport()
	reader := connection.NewTCPReader(mt)
	mt.EnqueueRecv(hdr.Encode())

	_, err := reader.ReadFrame()
	if err == nil {
		t.Fatal("expected ErrFrameTooLarge for oversized payload, got nil")
	}
	if !errors.Is(err, connection.ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got: %v", err)
	}
}

// ============================================================================
// 2. ConnectionManager Lifecycle on Abrupt Transport Failure
// ============================================================================

func TestAdversarial_ConnectionManager_AbruptEOFTeardown(t *testing.T) {
	mt := NewMockTransport()
	reader := connection.NewTCPReader(mt)
	proto := &MockProtocol{}

	cfg := connection.DefaultConfig()
	cfg.PingInterval = 10 * time.Second

	var closedErr atomic.Pointer[error]
	onClose := func(err error) {
		closedErr.Store(&err)
	}

	mgr := connection.NewConnectionManager(reader, mt, proto, &cfg, onClose, nil)
	ctx := context.Background()

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}

	// Create a pending request
	reqDone := make(chan error, 1)
	go func() {
		reqCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := mgr.SendRequest(reqCtx, protocol.OpSessionInit, map[string]any{})
		reqDone <- err
	}()

	// Allow goroutine to register pending request
	time.Sleep(10 * time.Millisecond)

	// Inject abrupt EOF from server
	mt.SetRecvError(io.EOF)
	_ = mt.Close() // unblocks blockCh in MockTransport

	// Pending request must unblock with ErrConnectionClosed
	select {
	case err := <-reqDone:
		if err == nil {
			t.Fatal("expected pending request to fail on abrupt EOF, got nil")
		}
		if !errors.Is(err, connection.ErrConnectionClosed) {
			t.Logf("pending request failed with: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK: pending request did not unblock after transport abrupt EOF")
	}

	// WaitClosed must unblock
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- mgr.WaitClosed()
	}()

	select {
	case err := <-waitCh:
		if err == nil {
			t.Fatal("expected WaitClosed to return error, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK: WaitClosed did not unblock after transport failure")
	}

	if mgr.IsOpen() {
		t.Fatal("expected manager to not be open")
	}
}

// ============================================================================
// 3. Keepalive Ping Loop Failure & Pending Request Abort
// ============================================================================

func TestAdversarial_KeepalivePingTimeoutTeardown(t *testing.T) {
	mt := NewMockTransport()
	reader := connection.NewTCPReader(mt)
	proto := &MockProtocol{}

	cfg := connection.DefaultConfig()
	cfg.PingInterval = 30 * time.Millisecond
	cfg.PingTimeout = 30 * time.Millisecond

	var closedErr atomic.Pointer[error]
	onClose := func(err error) {
		closedErr.Store(&err)
	}

	mgr := connection.NewConnectionManager(reader, mt, proto, &cfg, onClose, nil)
	ctx := context.Background()

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}

	// Also launch an RPC request that will be pending when ping times out
	reqDone := make(chan error, 1)
	go func() {
		reqCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := mgr.SendRequest(reqCtx, protocol.OpProfile, map[string]any{})
		reqDone <- err
	}()

	// Wait for ping to time out (30ms interval + 30ms timeout = ~60ms)
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- mgr.WaitClosed()
	}()

	select {
	case err := <-waitDone:
		if err == nil {
			t.Fatal("expected error from WaitClosed on ping timeout")
		}
		if !errors.Is(err, connection.ErrPingTimeout) {
			t.Fatalf("expected ErrPingTimeout, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK: WaitClosed did not unblock after ping timeout")
	}

	// Pending request must also have unblocked
	select {
	case err := <-reqDone:
		if err == nil {
			t.Fatal("expected pending request to fail when connection failed on ping timeout")
		}
		if !errors.Is(err, connection.ErrConnectionClosed) {
			t.Logf("pending request failed with: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("DEADLOCK: pending request did not abort after ping timeout")
	}

	if mgr.IsOpen() {
		t.Fatal("manager must not be open after ping timeout")
	}
}

// ============================================================================
// 4. Concurrent SendRequest and Close Stress Harness
// ============================================================================

func TestAdversarial_ConcurrentSendRequestAndClose(t *testing.T) {
	for iter := 0; iter < 10; iter++ {
		mt := NewMockTransport()
		reader := connection.NewTCPReader(mt)
		proto := &MockProtocol{}

		cfg := connection.DefaultConfig()
		cfg.PingInterval = 10 * time.Second // disable fast ping

		mgr := connection.NewConnectionManager(reader, mt, proto, &cfg, nil, nil)
		ctx := context.Background()

		if err := mgr.Start(ctx); err != nil {
			t.Fatalf("iter %d Start failed: %v", iter, err)
		}

		var wg sync.WaitGroup
		workers := 20
		stopCh := make(chan struct{})

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for {
					select {
					case <-stopCh:
						return
					default:
						reqCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
						_, err := mgr.SendRequest(reqCtx, protocol.OpPing, map[string]any{"id": id})
						cancel()
						if err != nil {
							// After Close, errors are expected and should not hang
							return
						}
					}
				}
			}(w)
		}

		time.Sleep(10 * time.Millisecond)

		// Close manager
		closeErr := mgr.Close()
		if closeErr != nil {
			t.Logf("iter %d Close returned: %v", iter, closeErr)
		}
		close(stopCh)

		// Await all workers
		doneCh := make(chan struct{})
		go func() {
			wg.Wait()
			close(doneCh)
		}()

		select {
		case <-doneCh:
			// All workers exited cleanly
		case <-time.After(3 * time.Second):
			t.Fatalf("iter %d DEADLOCK: concurrent SendRequest goroutines hung after Close", iter)
		}

		// WaitClosed must return immediately
		if err := mgr.WaitClosed(); err != nil {
			t.Logf("iter %d WaitClosed returned: %v", iter, err)
		}
	}
}

// ============================================================================
// 5. Goroutine Leak Verification
// ============================================================================

func TestAdversarial_NoGoroutineLeaksOnLifecycle(t *testing.T) {
	// Give runtime time to settle
	runtime.GC()
	time.Sleep(20 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		mt := NewMockTransport()
		reader := connection.NewTCPReader(mt)
		proto := &MockProtocol{}

		cfg := connection.DefaultConfig()
		cfg.PingInterval = 20 * time.Millisecond
		cfg.PingTimeout = 20 * time.Millisecond

		mgr := connection.NewConnectionManager(reader, mt, proto, &cfg, nil, nil)
		if err := mgr.Start(context.Background()); err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		time.Sleep(10 * time.Millisecond)
		if err := mgr.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()

	// Leaked goroutines threshold check
	diff := finalGoroutines - initialGoroutines
	if diff > 2 {
		t.Fatalf("GOROUTINE LEAK: initial=%d, final=%d (diff=%d)", initialGoroutines, finalGoroutines, diff)
	}
}
