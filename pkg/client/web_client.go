package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"gomax/pkg/api/chats"
	"gomax/pkg/api/messages"
	"gomax/pkg/api/uploads"
	"gomax/pkg/api/users"
	"gomax/pkg/auth"
	"gomax/pkg/connection"
	"gomax/pkg/dispatch"
	"gomax/pkg/protocol"
	"gomax/pkg/session"
	"gomax/pkg/transport"
	"gomax/pkg/types"
)

// WebClient implements the WebSocket-based client with QR authentication.
type WebClient struct {
	cfg    *Config
	conn   *connection.ConnectionManager
	store  session.Store
	router *dispatch.Router

	Messages *messages.MessageService
	Chats    *chats.ChatService
	Users    *users.UserService
	Uploads  *uploads.UploadService

	Me      *types.User
	mu      sync.RWMutex
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewWebClient creates a WebClient matching pymax.WebClient.
func NewWebClient(cfg *Config) *WebClient {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.URL == "" {
		cfg.URL = "wss://api.oneme.ru/websocket"
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = randomHex(8)
	}

	var store session.Store
	if cfg.Store != nil {
		store = cfg.Store
	} else if cfg.PersistSession {
		store = session.NewFileStore(cfg.WorkDir, cfg.SessionName)
	} else {
		store = session.NewInMemoryStore()
	}

	wc := &WebClient{
		cfg:    cfg,
		store:  store,
		router: dispatch.NewRouter(),
	}

	wc.Messages = messages.NewMessageService(wc)
	wc.Chats = chats.NewChatService(wc)
	wc.Users = users.NewUserService(wc)
	wc.Uploads = uploads.NewUploadService(wc)

	return wc
}

// OnMessage registers an incoming message listener.
func (wc *WebClient) OnMessage(handler func(ctx context.Context, msg *types.Message) error) {
	wc.router.OnMessage(handler)
}

// OnStart registers an on_start listener.
func (wc *WebClient) OnStart(handler func(ctx context.Context) error) {
	wc.router.OnStart(handler)
}

// Invoke implements api.Invoker.
func (wc *WebClient) Invoke(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	wc.mu.RLock()
	conn := wc.conn
	wc.mu.RUnlock()

	if conn == nil || !conn.IsOpen() {
		return nil, errors.New("web_client: connection is not open")
	}

	inbound, err := conn.SendRequest(ctx, op, payload)
	if err != nil {
		return nil, err
	}

	if inbound.Cmd == protocol.CmdError {
		return nil, fmt.Errorf("api error from server on opcode %s: %v", inbound.Opcode, inbound.Payload)
	}

	return inbound.Payload, nil
}

// Start connects via WebSocket, handles QR authentication, and runs the client loop.
// Start connects via WebSocket, handles QR authentication, and runs the client loop.
// If Reconnect is enabled, it automatically re-establishes connection on drops.
func (wc *WebClient) Start(ctx context.Context) error {
	wc.mu.Lock()
	if wc.started {
		wc.mu.Unlock()
		return errors.New("web client already started")
	}
	wc.started = true
	wc.ctx, wc.cancel = context.WithCancel(ctx)
	wc.mu.Unlock()

	defer func() {
		_ = wc.Close()
	}()

	for {
		err := wc.runSession(wc.ctx)
		if err == nil || errors.Is(err, context.Canceled) || wc.ctx.Err() != nil {
			return nil
		}

		if !wc.cfg.Reconnect {
			return err
		}

		delay := wc.cfg.ReconnectDelay
		if delay <= 0 {
			delay = 3 * time.Second
		}

		select {
		case <-wc.ctx.Done():
			return nil
		case <-time.After(delay):
			// retry reconnecting
		}
	}
}

func (wc *WebClient) runSession(ctx context.Context) error {
	wsOpts := transport.DefaultWSOptions(wc.cfg.URL)
	wsOpts.ProxyURL = wc.cfg.Proxy
	wsTransport := transport.NewWebSocketTransport(wsOpts)
	wsReader := connection.NewWSReader(wsTransport)

	wsProto, err := protocol.NewWsProtocol(true)
	if err != nil {
		return fmt.Errorf("init ws protocol failed: %w", err)
	}

	disconnectCh := make(chan error, 1)

	connManager := connection.NewConnectionManager(
		wsReader,
		wsTransport,
		wsProto,
		nil,
		func(err error) {
			select {
			case disconnectCh <- err:
			default:
			}
		},
		func(event *protocol.InboundFrame) {
			wc.handleEvent(event)
		},
	)

	wc.mu.Lock()
	wc.conn = connManager
	wc.mu.Unlock()

	if err := connManager.Start(ctx); err != nil {
		return fmt.Errorf("websocket connect failed: %w", err)
	}

	// 1. Session Init
	_, err = wc.Invoke(ctx, protocol.OpSessionInit, map[string]interface{}{
		"deviceId": wc.cfg.DeviceID,
	})
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("session init failed: %w", err)
	}

	// 2. Auth or Load token
	sessInfo, _ := wc.store.LoadSession()
	token := wc.cfg.Token
	if token == "" && sessInfo != nil {
		token = sessInfo.Token
	}

	if token == "" {
		qrFlow := auth.NewQrAuthFlow(nil, nil)
		authRes, err := qrFlow.Authenticate(ctx, wc)
		if err != nil {
			_ = connManager.Close()
			return fmt.Errorf("qr authentication failed: %w", err)
		}
		token = authRes.Token
		_ = wc.store.SaveSession(&session.SessionInfo{
			Token:    token,
			DeviceID: wc.cfg.DeviceID,
		})
	}

	// 3. Login
	loginRes, err := wc.Invoke(ctx, protocol.OpLogin, map[string]interface{}{
		"token":    token,
		"deviceId": wc.cfg.DeviceID,
	})
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("web login failed: %w", err)
	}

	wc.Me = &types.User{}
	if profileData, ok := loginRes["profile"].(map[string]interface{}); ok {
		if id, ok := profileData["id"].(int64); ok {
			wc.Me.ID = id
		}
	}

	wc.router.DispatchStart(ctx)

	select {
	case <-ctx.Done():
		_ = connManager.Close()
		return ctx.Err()
	case disErr := <-disconnectCh:
		_ = connManager.Close()
		return disErr
	}
}

func (wc *WebClient) handleEvent(frame *protocol.InboundFrame) {
	if frame.Opcode == protocol.OpMsgSend || frame.Opcode == protocol.OpChatHistory {
		mData := frame.Payload
		msg := &types.Message{
			Time: time.Now().Unix(),
		}
		if id, ok := mData["id"].(int64); ok {
			msg.ID = id
		}
		if text, ok := mData["text"].(string); ok {
			msg.Text = text
		}
		if chatID, ok := mData["chatId"].(int64); ok {
			msg.ChatID = chatID
		}
		wc.router.DispatchMessage(wc.ctx, msg)
	}
}

// Close disconnects the WebSocket client.
func (wc *WebClient) Close() error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if !wc.started {
		return nil
	}
	wc.started = false
	if wc.cancel != nil {
		wc.cancel()
	}
	if wc.conn != nil {
		return wc.conn.Close()
	}
	return nil
}
