package transport

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

const (
	// DefaultTCPHost is the production Max API host.
	DefaultTCPHost = "api2.oneme.ru"
	// DefaultTCPPort is the standard TLS port for Max API.
	DefaultTCPPort = 443
	// DefaultConnectTimeout is the connection establishment timeout.
	DefaultConnectTimeout = 30 * time.Second
	// DefaultCloseTimeout is the timeout for graceful connection teardown.
	DefaultCloseTimeout = 5 * time.Second
)

// TCPOptions holds configuration options for TCPTransport.
type TCPOptions struct {
	Host           string
	Port           int
	UseSSL         bool
	ProxyURL       string
	TLSConfig      *tls.Config
	ConnectTimeout time.Duration
	CloseTimeout   time.Duration
}

// DefaultTCPOptions returns the standard production options for Max TCP transport.
func DefaultTCPOptions() TCPOptions {
	return TCPOptions{
		Host:           DefaultTCPHost,
		Port:           DefaultTCPPort,
		UseSSL:         true,
		ProxyURL:       "",
		ConnectTimeout: DefaultConnectTimeout,
		CloseTimeout:   DefaultCloseTimeout,
	}
}

// TCPTransport implements the Transport interface over a TLS TCP socket.
type TCPTransport struct {
	opts TCPOptions

	mu        sync.RWMutex
	conn      net.Conn
	connected bool

	writeMu sync.Mutex
}

// NewTCPTransport creates a new TCPTransport with the given options.
func NewTCPTransport(opts TCPOptions) *TCPTransport {
	if opts.Host == "" {
		opts.Host = DefaultTCPHost
	}
	if opts.Port == 0 {
		opts.Port = DefaultTCPPort
	}
	if opts.ConnectTimeout <= 0 {
		opts.ConnectTimeout = DefaultConnectTimeout
	}
	if opts.CloseTimeout <= 0 {
		opts.CloseTimeout = DefaultCloseTimeout
	}
	return &TCPTransport{opts: opts}
}

// Connect dials the remote host (optionally via proxy) and performs TLS negotiation.
func (t *TCPTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	if t.connected {
		t.mu.Unlock()
		return ErrAlreadyConnected
	}
	t.mu.Unlock()

	targetAddr := net.JoinHostPort(t.opts.Host, strconv.Itoa(t.opts.Port))

	rawConn, err := t.dialRaw(ctx, targetAddr)
	if err != nil {
		return fmt.Errorf("tcp dial %s: %w", targetAddr, err)
	}

	var finalConn net.Conn = rawConn

	if t.opts.UseSSL {
		tlsConfig := t.opts.TLSConfig
		if tlsConfig == nil {
			var err error
			tlsConfig, err = NewTLSConfig(t.opts.Host)
			if err != nil {
				_ = rawConn.Close()
				return fmt.Errorf("prepare tls config: %w", err)
			}
		} else if tlsConfig.ServerName == "" {
			tlsConfig = tlsConfig.Clone()
			tlsConfig.ServerName = t.opts.Host
		}

		tlsConn := tls.Client(rawConn, tlsConfig)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = rawConn.Close()
			return fmt.Errorf("tls handshake %s: %w", targetAddr, err)
		}
		finalConn = tlsConn
	}

	t.mu.Lock()
	t.conn = finalConn
	t.connected = true
	t.mu.Unlock()

	return nil
}

func (t *TCPTransport) dialRaw(ctx context.Context, targetAddr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: t.opts.ConnectTimeout}

	if t.opts.ProxyURL == "" {
		return dialer.DialContext(ctx, "tcp", targetAddr)
	}

	proxyURL, err := url.Parse(t.opts.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProxy, err)
	}

	switch proxyURL.Scheme {
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if proxyURL.User != nil {
			auth = &proxy.Auth{User: proxyURL.User.Username()}
			if password, ok := proxyURL.User.Password(); ok {
				auth.Password = password
			}
		}
		socksDialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, dialer)
		if err != nil {
			return nil, fmt.Errorf("socks5 proxy dialer: %w", err)
		}
		if cd, ok := socksDialer.(proxy.ContextDialer); ok {
			return cd.DialContext(ctx, "tcp", targetAddr)
		}
		return socksDialer.Dial("tcp", targetAddr)

	case "http", "https":
		// HTTP CONNECT tunnel
		conn, err := dialer.DialContext(ctx, "tcp", proxyURL.Host)
		if err != nil {
			return nil, fmt.Errorf("http proxy dial %s: %w", proxyURL.Host, err)
		}

		req := &http.Request{
			Method: http.MethodConnect,
			URL:    &url.URL{Opaque: targetAddr},
			Host:   targetAddr,
			Header: make(http.Header),
		}
		if proxyURL.User != nil {
			user := proxyURL.User.Username()
			pass, _ := proxyURL.User.Password()
			auth := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
			req.Header.Set("Proxy-Authorization", "Basic "+auth)
		}

		if err := req.Write(conn); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("http proxy connect request: %w", err)
		}

		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, req)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("http proxy connect response: %w", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			_ = conn.Close()
			return nil, fmt.Errorf("http proxy connect failed with status: %s", resp.Status)
		}

		return conn, nil

	default:
		return nil, fmt.Errorf("%w: unsupported scheme %s", ErrInvalidProxy, proxyURL.Scheme)
	}
}

// Send transmits data atomically to the remote endpoint.
func (t *TCPTransport) Send(data []byte) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	t.mu.RLock()
	conn := t.conn
	connected := t.connected
	t.mu.RUnlock()

	if !connected || conn == nil {
		return ErrNotConnected
	}

	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			t.markClosed()
			return fmt.Errorf("tcp write: %w", err)
		}
		if n == 0 {
			t.markClosed()
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

// Recv reads exactly n bytes (io.ReadFull) when n > 0, or up to 4096 bytes when n <= 0.
func (t *TCPTransport) Recv(n int) ([]byte, error) {
	t.mu.RLock()
	conn := t.conn
	connected := t.connected
	t.mu.RUnlock()

	if !connected || conn == nil {
		return nil, ErrNotConnected
	}

	if n <= 0 {
		buf := make([]byte, 4096)
		nRead, err := conn.Read(buf)
		if err != nil {
			t.markClosed()
			return nil, err
		}
		return buf[:nRead], nil
	}

	buf := make([]byte, n)
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		t.markClosed()
		return nil, err
	}
	return buf, nil
}

// Connected returns whether the transport is active.
func (t *TCPTransport) Connected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// Close gracefully closes the socket within CloseTimeout.
func (t *TCPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected || t.conn == nil {
		return nil
	}
	t.connected = false

	_ = t.conn.SetDeadline(time.Now().Add(t.opts.CloseTimeout))
	err := t.conn.Close()
	t.conn = nil
	return err
}

func (t *TCPTransport) markClosed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connected = false
}
