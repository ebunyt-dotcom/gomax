package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const (
	// DefaultWSURL is the production Max API WebSocket endpoint.
	DefaultWSURL = "wss://api.oneme.ru/websocket"
	// DefaultWSOrigin is the required browser Origin header for Max WebSocket.
	DefaultWSOrigin = "https://web.max.ru"
	// DefaultMaxWSMessageSize is the 10 MB maximum frame limit.
	DefaultMaxWSMessageSize = 10 * 1024 * 1024
)

// WSOptions holds configuration options for WebSocketTransport.
type WSOptions struct {
	URL              string
	Origin           string
	ProxyURL         string
	TLSConfig        *tls.Config
	HandshakeTimeout time.Duration
	CloseTimeout     time.Duration
	MaxMessageSize   int64
}

// DefaultWSOptions returns standard production options for Max WebSocket transport.
func DefaultWSOptions(wsURL string) WSOptions {
	if wsURL == "" {
		wsURL = DefaultWSURL
	}
	return WSOptions{
		URL:              wsURL,
		Origin:           DefaultWSOrigin,
		HandshakeTimeout: DefaultConnectTimeout,
		CloseTimeout:     DefaultCloseTimeout,
		MaxMessageSize:   DefaultMaxWSMessageSize,
	}
}

// WebSocketTransport implements the Transport interface over a WebSocket connection.
type WebSocketTransport struct {
	opts WSOptions

	mu        sync.RWMutex
	conn      *websocket.Conn
	connected bool

	writeMu sync.Mutex
	readMu  sync.Mutex
	readBuf []byte
}

// NewWebSocketTransport creates a new WebSocketTransport.
func NewWebSocketTransport(opts WSOptions) *WebSocketTransport {
	if opts.URL == "" {
		opts.URL = DefaultWSURL
	}
	if opts.Origin == "" {
		opts.Origin = DefaultWSOrigin
	}
	if opts.HandshakeTimeout <= 0 {
		opts.HandshakeTimeout = DefaultConnectTimeout
	}
	if opts.CloseTimeout <= 0 {
		opts.CloseTimeout = DefaultCloseTimeout
	}
	if opts.MaxMessageSize <= 0 {
		opts.MaxMessageSize = DefaultMaxWSMessageSize
	}
	return &WebSocketTransport{opts: opts}
}

// Connect dials the WebSocket endpoint with required Origin header and TLS configuration.
func (w *WebSocketTransport) Connect(ctx context.Context) error {
	w.mu.Lock()
	if w.connected {
		w.mu.Unlock()
		return ErrAlreadyConnected
	}
	w.mu.Unlock()

	parsedURL, err := url.Parse(w.opts.URL)
	if err != nil {
		return fmt.Errorf("invalid websocket url %s: %w", w.opts.URL, err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: w.opts.HandshakeTimeout,
	}

	if parsedURL.Scheme == "wss" {
		tlsConfig := w.opts.TLSConfig
		if tlsConfig == nil {
			var err error
			tlsConfig, err = NewTLSConfig(parsedURL.Hostname())
			if err != nil {
				return fmt.Errorf("prepare tls config for ws: %w", err)
			}
		} else if tlsConfig.ServerName == "" {
			tlsConfig = tlsConfig.Clone()
			tlsConfig.ServerName = parsedURL.Hostname()
		}
		dialer.TLSClientConfig = tlsConfig
	}

	if w.opts.ProxyURL != "" {
		proxyParsed, err := url.Parse(w.opts.ProxyURL)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidProxy, err)
		}
		switch proxyParsed.Scheme {
		case "http", "https":
			dialer.Proxy = http.ProxyURL(proxyParsed)
		case "socks5", "socks5h":
			var auth *proxy.Auth
			if proxyParsed.User != nil {
				auth = &proxy.Auth{User: proxyParsed.User.Username()}
				if password, ok := proxyParsed.User.Password(); ok {
					auth.Password = password
				}
			}
			socksDialer, err := proxy.SOCKS5("tcp", proxyParsed.Host, auth, proxy.Direct)
			if err != nil {
				return fmt.Errorf("socks5 proxy dialer for ws: %w", err)
			}
			if cd, ok := socksDialer.(proxy.ContextDialer); ok {
				dialer.NetDialContext = cd.DialContext
			} else {
				dialer.NetDial = socksDialer.Dial
			}
		default:
			return fmt.Errorf("%w: unsupported proxy scheme %s", ErrInvalidProxy, proxyParsed.Scheme)
		}
	}

	headers := make(http.Header)
	headers.Set("Origin", w.opts.Origin)

	conn, resp, err := dialer.DialContext(ctx, w.opts.URL, headers)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("websocket dial failed with http status %s: %w", resp.Status, err)
		}
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	conn.SetReadLimit(w.opts.MaxMessageSize)

	w.mu.Lock()
	w.conn = conn
	w.connected = true
	w.readBuf = nil
	w.mu.Unlock()

	return nil
}

// Send transmits a binary frame to the WebSocket server.
func (w *WebSocketTransport) Send(data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	w.mu.RLock()
	conn := w.conn
	connected := w.connected
	w.mu.RUnlock()

	if !connected || conn == nil {
		return ErrNotConnected
	}

	err := conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		w.markClosed()
		return fmt.Errorf("ws send: %w", err)
	}
	return nil
}

// Recv retrieves data from the WebSocket.
//
// If n <= 0, returns the complete binary message.
// If n > 0, returns exactly n bytes from buffered and newly fetched frames.
func (w *WebSocketTransport) Recv(n int) ([]byte, error) {
	w.readMu.Lock()
	defer w.readMu.Unlock()

	w.mu.RLock()
	conn := w.conn
	connected := w.connected
	w.mu.RUnlock()

	if !connected || conn == nil {
		return nil, ErrNotConnected
	}

	// Mode 1: Whole-message fetch when n <= 0
	if n <= 0 {
		if len(w.readBuf) > 0 {
			out := w.readBuf
			w.readBuf = nil
			return out, nil
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			w.markClosed()
			return nil, fmt.Errorf("ws recv: %w", err)
		}
		return msg, nil
	}

	// Mode 2: Stream-slicing when n > 0
	for len(w.readBuf) < n {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			w.markClosed()
			return nil, fmt.Errorf("ws recv: %w", err)
		}
		w.readBuf = append(w.readBuf, msg...)
	}

	out := make([]byte, n)
	copy(out, w.readBuf[:n])
	w.readBuf = w.readBuf[n:]
	return out, nil
}

// Connected reports whether the WebSocket is open.
func (w *WebSocketTransport) Connected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.connected
}

// Close cleanly terminates the WebSocket session.
func (w *WebSocketTransport) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.connected || w.conn == nil {
		return nil
	}
	w.connected = false

	closeDeadline := time.Now().Add(w.opts.CloseTimeout)
	_ = w.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "normal closure"),
		closeDeadline,
	)
	err := w.conn.Close()
	w.conn = nil
	w.readBuf = nil
	return err
}

func (w *WebSocketTransport) markClosed() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.connected = false
}
