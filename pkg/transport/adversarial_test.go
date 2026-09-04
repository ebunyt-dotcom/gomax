package transport_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"gomax/pkg/transport"
)

// ============================================================================
// 1. Abrupt Socket Closure & EOF Simulation during TCP I/O
// ============================================================================

func TestAdversarial_TCPTransport_AbruptServerClosureDuringRecv(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	serverConnCh := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			serverConnCh <- conn
		}
	}()

	host, portStr, _ := net.SplitHostPort(listener.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	client := transport.NewTCPTransport(transport.TCPOptions{
		Host:           host,
		Port:           port,
		UseSSL:         false,
		ConnectTimeout: 2 * time.Second,
		CloseTimeout:   1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}

	serverConn := <-serverConnCh

	// Send 5 bytes from server
	_, _ = serverConn.Write([]byte("12345"))
	// Abruptly close server socket
	_ = serverConn.Close()

	// Client requests 10 bytes -> should fail with EOF or ErrUnexpectedEOF, NOT hang
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Recv(10)
		errCh <- err
	}()

	select {
	case recvErr := <-errCh:
		if recvErr == nil {
			t.Fatal("expected error on abrupt server closure, got nil")
		}
		if !errors.Is(recvErr, io.EOF) && !errors.Is(recvErr, io.ErrUnexpectedEOF) {
			t.Logf("abrupt close error received: %v", recvErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK / HANG: client.Recv did not unblock after abrupt server socket closure")
	}

	_ = client.Close()
}

// ============================================================================
// 2. Concurrent Calls to Send and Recv while Close is Triggered (TCP)
// ============================================================================

func TestAdversarial_TCPTransport_ConcurrentSendRecvClose(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					// echo back
					_, _ = c.Write(buf[:n])
				}
			}(conn)
		}
	}()

	host, portStr, _ := net.SplitHostPort(listener.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	// Run stress loop 10 times
	for iter := 0; iter < 10; iter++ {
		client := transport.NewTCPTransport(transport.TCPOptions{
			Host:           host,
			Port:           port,
			UseSSL:         false,
			ConnectTimeout: 2 * time.Second,
			CloseTimeout:   500 * time.Millisecond,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := client.Connect(ctx); err != nil {
			cancel()
			t.Fatalf("iter %d connect failed: %v", iter, err)
		}
		cancel()

		var wg sync.WaitGroup
		stopSending := make(chan struct{})

		// 8 concurrent senders
		for s := 0; s < 8; s++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				data := []byte(fmt.Sprintf("msg-from-%d", id))
				for {
					select {
					case <-stopSending:
						return
					default:
						err := client.Send(data)
						if err != nil {
							// After close, errors are expected
							return
						}
					}
				}
			}(s)
		}

		// 8 concurrent receivers
		for r := 0; r < 8; r++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stopSending:
						return
					default:
						_, err := client.Recv(5)
						if err != nil {
							// After close, errors are expected
							return
						}
					}
				}
			}()
		}

		// Let them hammer for 10ms
		time.Sleep(10 * time.Millisecond)

		// Trigger Close
		closeErr := client.Close()
		if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			t.Logf("iter %d close error: %v", iter, closeErr)
		}

		// Also close stopSending to allow workers to exit if they checked channel
		close(stopSending)

		// Wait for all workers with timeout to detect deadlocks
		doneCh := make(chan struct{})
		go func() {
			wg.Wait()
			close(doneCh)
		}()

		select {
		case <-doneCh:
			// Success, no deadlocks
		case <-time.After(3 * time.Second):
			t.Fatalf("iter %d DEADLOCK: goroutines did not exit after client.Close()", iter)
		}

		// Verify client.Connected() is false
		if client.Connected() {
			t.Fatalf("iter %d expected Connected() to be false after Close()", iter)
		}

		// Subsequent Send and Recv must fail with ErrNotConnected
		if err := client.Send([]byte("test")); !errors.Is(err, transport.ErrNotConnected) {
			t.Errorf("iter %d expected ErrNotConnected on Send after Close, got: %v", iter, err)
		}
		if _, err := client.Recv(5); !errors.Is(err, transport.ErrNotConnected) {
			t.Errorf("iter %d expected ErrNotConnected on Recv after Close, got: %v", iter, err)
		}
	}
}

// ============================================================================
// 3. Concurrent Calls to Send and Recv while Close is Triggered (WebSocket)
// ============================================================================

func TestAdversarial_WebSocketTransport_ConcurrentSendRecvClose(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()

		for {
			msgType, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if err := ws.WriteMessage(msgType, data); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + stringsTrimPrefix(server.URL, "http")

	for iter := 0; iter < 10; iter++ {
		opts := transport.DefaultWSOptions(wsURL)
		opts.CloseTimeout = 500 * time.Millisecond
		client := transport.NewWebSocketTransport(opts)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := client.Connect(ctx); err != nil {
			cancel()
			t.Fatalf("iter %d ws connect failed: %v", iter, err)
		}
		cancel()

		var wg sync.WaitGroup
		stopCh := make(chan struct{})

		// Concurrent senders
		for s := 0; s < 6; s++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				data := []byte(fmt.Sprintf("ws-payload-%d", id))
				for {
					select {
					case <-stopCh:
						return
					default:
						if err := client.Send(data); err != nil {
							return
						}
					}
				}
			}(s)
		}

		// Concurrent receivers (mix of full frame n <= 0 and sliced n > 0)
		for r := 0; r < 6; r++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for {
					select {
					case <-stopCh:
						return
					default:
						var err error
						if id%2 == 0 {
							_, err = client.Recv(-1)
						} else {
							_, err = client.Recv(4)
						}
						if err != nil {
							return
						}
					}
				}
			}(r)
		}

		time.Sleep(15 * time.Millisecond)

		// Concurrent Close
		_ = client.Close()
		close(stopCh)

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(3 * time.Second):
			t.Fatalf("iter %d DEADLOCK in WebSocketTransport during concurrent Close", iter)
		}

		if client.Connected() {
			t.Fatalf("iter %d expected Connected() == false", iter)
		}

		if err := client.Send([]byte("post-close")); !errors.Is(err, transport.ErrNotConnected) {
			t.Errorf("iter %d expected ErrNotConnected, got %v", iter, err)
		}
	}
}

// Helper string trimmer
func stringsTrimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// ============================================================================
// 4. Certificate & TLS Config Adversarial Verification
// ============================================================================

func TestAdversarial_CertificatesAndTLSConfig(t *testing.T) {
	// 1. Stress test concurrent access to GetRootCACertPool
	var wg sync.WaitGroup
	errCh := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool, err := transport.GetRootCACertPool()
			if err != nil {
				errCh <- err
				return
			}
			if pool == nil {
				errCh <- errors.New("nil cert pool")
				return
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent GetRootCACertPool failed: %v", err)
	}

	// 2. Validate certificate properties directly
	rawPEM := transport.GetEmbeddedRootCAPEM()
	if len(rawPEM) == 0 {
		t.Fatal("empty embedded root CA PEM")
	}

	block, _ := pem.Decode(rawPEM)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	// Verify not expired
	now := time.Now()
	if now.Before(cert.NotBefore) {
		t.Errorf("cert is not yet valid: NotBefore=%v, now=%v", cert.NotBefore, now)
	}
	if now.After(cert.NotAfter) {
		t.Errorf("cert is EXPIRED: NotAfter=%v, now=%v", cert.NotAfter, now)
	}

	// Verify it is a CA certificate
	if !cert.IsCA {
		t.Error("expected certificate to have IsCA=true")
	}

	// 3. Validate NewTLSConfig
	tlsConfig, err := transport.NewTLSConfig("test.oneme.ru")
	if err != nil {
		t.Fatalf("NewTLSConfig failed: %v", err)
	}
	if tlsConfig.MinVersion < tls.VersionTLS12 {
		t.Errorf("expected MinVersion >= TLS1.2, got: %x", tlsConfig.MinVersion)
	}
	if tlsConfig.ServerName != "test.oneme.ru" {
		t.Errorf("expected ServerName test.oneme.ru, got: %s", tlsConfig.ServerName)
	}
}
