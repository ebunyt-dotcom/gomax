package transport_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"gomax/pkg/transport"
)

// ============================================================================
// MockTransport Harness
// ============================================================================

// MockTransport is a fully featured in-memory test double for transport.Transport.
type MockTransport struct {
	mu           sync.Mutex
	connected    bool
	connectErr   error
	closeErr     error
	sendErr      error
	recvErr      error
	sentData     [][]byte
	inQueue      [][]byte
	currentChunk []byte

	connectCalls int
	closeCalls   int
	sendCalls    int
	recvCalls    int

	onSend func(data []byte)
}

func NewMockTransport() *MockTransport {
	return &MockTransport{
		sentData: make([][]byte, 0),
		inQueue:  make([][]byte, 0),
	}
}

func (m *MockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectCalls++
	if m.connectErr != nil {
		return m.connectErr
	}
	if m.connected {
		return transport.ErrAlreadyConnected
	}
	m.connected = true
	return nil
}

func (m *MockTransport) Send(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls++
	if !m.connected {
		return transport.ErrNotConnected
	}
	if m.sendErr != nil {
		return m.sendErr
	}
	copied := make([]byte, len(data))
	copy(copied, data)
	m.sentData = append(m.sentData, copied)
	if m.onSend != nil {
		m.onSend(copied)
	}
	return nil
}

func (m *MockTransport) Recv(n int) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvCalls++
	if !m.connected {
		return nil, transport.ErrNotConnected
	}
	if m.recvErr != nil {
		return nil, m.recvErr
	}

	if n <= 0 {
		if len(m.currentChunk) > 0 {
			out := m.currentChunk
			m.currentChunk = nil
			return out, nil
		}
		if len(m.inQueue) == 0 {
			return nil, io.EOF
		}
		msg := m.inQueue[0]
		m.inQueue = m.inQueue[1:]
		return msg, nil
	}

	for len(m.currentChunk) < n {
		if len(m.inQueue) == 0 {
			return nil, io.EOF
		}
		m.currentChunk = append(m.currentChunk, m.inQueue[0]...)
		m.inQueue = m.inQueue[1:]
	}

	out := make([]byte, n)
	copy(out, m.currentChunk[:n])
	m.currentChunk = m.currentChunk[n:]
	return out, nil
}

func (m *MockTransport) Connected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalls++
	m.connected = false
	return m.closeErr
}

func (m *MockTransport) FeedIncoming(msg []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]byte, len(msg))
	copy(copied, msg)
	m.inQueue = append(m.inQueue, copied)
}

func (m *MockTransport) GetSent() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([][]byte, len(m.sentData))
	for i, b := range m.sentData {
		res[i] = append([]byte(nil), b...)
	}
	return res
}

func (m *MockTransport) SetConnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectErr = err
}

func (m *MockTransport) SetSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendErr = err
}

func (m *MockTransport) SetRecvError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvErr = err
}

// ============================================================================
// Unit Tests: Embedded Certificate
// ============================================================================

func TestEmbeddedRootCACertificate(t *testing.T) {
	pemBytes := transport.GetEmbeddedRootCAPEM()
	if len(pemBytes) == 0 {
		t.Fatal("expected embedded root CA certificate PEM to be non-empty")
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("failed to decode PEM block of type CERTIFICATE")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse x509 certificate: %v", err)
	}

	expectedIssuer := "Russian Trusted Root CA"
	if !strings.Contains(cert.Issuer.CommonName, expectedIssuer) {
		t.Errorf("expected issuer CN to contain %q, got %q", expectedIssuer, cert.Issuer.CommonName)
	}

	expectedOrg := "The Ministry of Digital Development and Communications"
	foundOrg := false
	for _, org := range cert.Issuer.Organization {
		if org == expectedOrg {
			foundOrg = true
			break
		}
	}
	if !foundOrg {
		t.Errorf("expected issuer Organization to contain %q, got %v", expectedOrg, cert.Issuer.Organization)
	}

	pool, err := transport.GetRootCACertPool()
	if err != nil {
		t.Fatalf("GetRootCACertPool failed: %v", err)
	}
	if pool == nil {
		t.Fatal("returned cert pool is nil")
	}

	tlsConfig, err := transport.NewTLSConfig("api2.oneme.ru")
	if err != nil {
		t.Fatalf("NewTLSConfig failed: %v", err)
	}
	if tlsConfig.ServerName != "api2.oneme.ru" {
		t.Errorf("expected ServerName api2.oneme.ru, got %s", tlsConfig.ServerName)
	}
	if tlsConfig.RootCAs == nil {
		t.Error("expected non-nil RootCAs")
	}
}

// ============================================================================
// Unit Tests: TCP Transport Loopback
// ============================================================================

func TestTCPTransport_Loopback(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer listener.Close()

	host, portStr, _ := net.SplitHostPort(listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)

	serverReceived := make(chan []byte, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 5)
		_, _ = io.ReadFull(conn, buf)
		serverReceived <- buf

		_, _ = conn.Write([]byte("pong-response"))
	}()

	client := transport.NewTCPTransport(transport.TCPOptions{
		Host:           host,
		Port:           port,
		UseSSL:         false,
		ConnectTimeout: 5 * time.Second,
		CloseTimeout:   1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	if !client.Connected() {
		t.Error("expected client to be connected")
	}

	if err := client.Send([]byte("hello")); err != nil {
		t.Fatalf("client send failed: %v", err)
	}

	gotServer := <-serverReceived
	if string(gotServer) != "hello" {
		t.Errorf("server received %q, expected %q", string(gotServer), "hello")
	}

	// Test readexactly semantics (4 bytes)
	part1, err := client.Recv(4)
	if err != nil {
		t.Fatalf("client recv part1 failed: %v", err)
	}
	if string(part1) != "pong" {
		t.Errorf("expected %q, got %q", "pong", string(part1))
	}

	// Test read remaining 9 bytes
	part2, err := client.Recv(9)
	if err != nil {
		t.Fatalf("client recv part2 failed: %v", err)
	}
	if string(part2) != "-response" {
		t.Errorf("expected %q, got %q", "-response", string(part2))
	}

	if err := client.Close(); err != nil {
		t.Fatalf("client close failed: %v", err)
	}
	if client.Connected() {
		t.Error("expected client to be disconnected after close")
	}

	if err := client.Send([]byte("fail")); !errors.Is(err, transport.ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

// ============================================================================
// Unit Tests: WebSocket Transport Loopback
// ============================================================================

func TestWebSocketTransport_Loopback(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Origin") == "https://web.max.ru"
		},
	}

	serverReceived := make(chan []byte, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "https://web.max.ru" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()

		msgType, data, err := ws.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.BinaryMessage {
			t.Errorf("expected binary message type, got %d", msgType)
		}
		serverReceived <- data

		_ = ws.WriteMessage(websocket.BinaryMessage, []byte("echo:"+string(data)))
	}))
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	opts := transport.DefaultWSOptions(wsURL)
	client := transport.NewWebSocketTransport(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("ws connect failed: %v", err)
	}
	if !client.Connected() {
		t.Error("expected ws to be connected")
	}

	testPayload := []byte{0x0A, 0x00, 0x01, 0x02, 0x03}
	if err := client.Send(testPayload); err != nil {
		t.Fatalf("ws send failed: %v", err)
	}

	gotServer := <-serverReceived
	if !bytes.Equal(gotServer, testPayload) {
		t.Errorf("server got %v, expected %v", gotServer, testPayload)
	}

	resp, err := client.Recv(-1)
	if err != nil {
		t.Fatalf("ws recv failed: %v", err)
	}
	expectedResp := append([]byte("echo:"), testPayload...)
	if !bytes.Equal(resp, expectedResp) {
		t.Errorf("expected response %v, got %v", expectedResp, resp)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("ws close failed: %v", err)
	}
	if client.Connected() {
		t.Error("expected client to be disconnected")
	}
}
