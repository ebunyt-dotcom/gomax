package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api/chats"
	"github.com/ebunyt-dotcom/gomax/pkg/api/messages"
	"github.com/ebunyt-dotcom/gomax/pkg/api/uploads"
	"github.com/ebunyt-dotcom/gomax/pkg/api/users"
	"github.com/ebunyt-dotcom/gomax/pkg/auth"
	"github.com/ebunyt-dotcom/gomax/pkg/connection"
	"github.com/ebunyt-dotcom/gomax/pkg/dispatch"
	"github.com/ebunyt-dotcom/gomax/pkg/fingerprint"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/session"
	"github.com/ebunyt-dotcom/gomax/pkg/transport"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// Config configures client network endpoints and session behavior.
type Config struct {
	Phone          string
	WorkDir        string
	SessionName    string
	Host           string
	Port           int
	URL            string // WebSocket URL
	UseSSL         bool
	Proxy          string
	DeviceID       string
	MtInstanceID   string
	Token          string
	PersistSession bool
	Reconnect      bool
	ReconnectDelay time.Duration

	Store    session.Store
	AuthFlow auth.SmsAuthFlow
}

// DefaultConfig returns default client configuration matching PyMax.
func DefaultConfig() *Config {
	return &Config{
		Host:           "api2.oneme.ru",
		Port:           443,
		URL:            "wss://api.oneme.ru/websocket",
		UseSSL:         true,
		WorkDir:        "cache",
		SessionName:    "main.json",
		PersistSession: true,
		Reconnect:      true,
		ReconnectDelay: 3 * time.Second,
	}
}

// Client is the primary high-level TCP client for Max API.
type Client struct {
	cfg       *Config
	conn      *connection.ConnectionManager
	store     session.Store
	router    *dispatch.Router
	fpGen     *fingerprint.FingerprintGenerator
	callsSeed int64

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

// CallsSeed returns the callsSeed received from the handshake.
func (c *Client) CallsSeed() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.callsSeed
}

// GetDeviceID returns the client device ID.
func (c *Client) GetDeviceID() string {
	return c.cfg.DeviceID
}

// GetCallsSeed returns the handshake callsSeed.
func (c *Client) GetCallsSeed() int64 {
	return c.CallsSeed()
}

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "a1b2c3d4e5f60718"
	}
	return hex.EncodeToString(bytes)
}

// NewClient creates a new Max TCP client matching pymax.Client.
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.Host == "" {
		cfg.Host = "api2.oneme.ru"
	}
	if cfg.Port == 0 {
		cfg.Port = 443
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = randomHex(8)
	}
	if cfg.MtInstanceID == "" {
		cfg.MtInstanceID = randomHex(8)
	}

	var store session.Store
	if cfg.Store != nil {
		store = cfg.Store
	} else if cfg.PersistSession {
		store = session.NewFileStore(cfg.WorkDir, cfg.SessionName)
	} else {
		store = session.NewInMemoryStore()
	}

	c := &Client{
		cfg:    cfg,
		store:  store,
		router: dispatch.NewRouter(),
		fpGen:  fingerprint.NewFingerprintGenerator(fingerprint.DefaultFingerprint()),
	}

	c.Messages = messages.NewMessageService(c)
	c.Chats = chats.NewChatService(c)
	c.Users = users.NewUserService(c)
	c.Uploads = uploads.NewUploadService(c)

	return c
}

// OnMessage registers an incoming message listener.
func (c *Client) OnMessage(handler func(ctx context.Context, msg *types.Message) error) {
	c.router.OnMessage(handler)
}

// OnStart registers an on_start listener executed after successful login.
func (c *Client) OnStart(handler func(ctx context.Context) error) {
	c.router.OnStart(handler)
}

// Invoke implements the api.Invoker interface for RPC commands.
func (c *Client) Invoke(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil || !conn.IsOpen() {
		return nil, errors.New("client: connection is not open")
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

// Start connects to Max server, handles auth, and begins processing events.
// If Reconnect is enabled, it automatically re-establishes connection on drops.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return errors.New("client already started")
	}
	c.started = true
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.mu.Unlock()

	defer func() {
		_ = c.Close()
	}()

	for {
		err := c.runSession(c.ctx)
		if err == nil || errors.Is(err, context.Canceled) || c.ctx.Err() != nil {
			return nil
		}

		if !c.cfg.Reconnect {
			return err
		}

		delay := c.cfg.ReconnectDelay
		if delay <= 0 {
			delay = 3 * time.Second
		}

		select {
		case <-c.ctx.Done():
			return nil
		case <-time.After(delay):
			// retry reconnecting
		}
	}
}

func (c *Client) runSession(ctx context.Context) error {
	tcpOpts := transport.DefaultTCPOptions()
	if c.cfg.Host != "" {
		tcpOpts.Host = c.cfg.Host
	}
	if c.cfg.Port != 0 {
		tcpOpts.Port = c.cfg.Port
	}
	tcpOpts.UseSSL = c.cfg.UseSSL
	tcpOpts.ProxyURL = c.cfg.Proxy

	tcpTransport := transport.NewTCPTransport(tcpOpts)
	reader := connection.NewTCPReader(tcpTransport)

	tcpProto, err := protocol.NewTcpProtocol()
	if err != nil {
		return fmt.Errorf("init protocol failed: %w", err)
	}

	disconnectCh := make(chan error, 1)

	connManager := connection.NewConnectionManager(
		reader,
		tcpTransport,
		tcpProto,
		nil,
		func(err error) {
			select {
			case disconnectCh <- err:
			default:
			}
		},
		func(event *protocol.InboundFrame) {
			c.handleEvent(event)
		},
	)

	c.mu.Lock()
	c.conn = connManager
	c.mu.Unlock()

	log.Printf("[gomax] Connecting to Max server at %s:%d (SSL=%v)...", tcpOpts.Host, tcpOpts.Port, tcpOpts.UseSSL)
	if err := connManager.Start(ctx); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	log.Printf("[gomax] Connected to Max server successfully")

	// Load session info to reuse deviceId and mt_instanceid if previously saved
	sessInfo, err := c.store.LoadSession()
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("failed to load session: %w", err)
	}

	if sessInfo != nil {
		if sessInfo.DeviceID != "" {
			c.cfg.DeviceID = sessInfo.DeviceID
		}
		if sessInfo.MTInstanceID != "" {
			c.cfg.MtInstanceID = sessInfo.MTInstanceID
		}
	}
	if c.cfg.DeviceID == "" {
		c.cfg.DeviceID = randomHex(8)
	}
	if c.cfg.MtInstanceID == "" {
		c.cfg.MtInstanceID = randomHex(8)
	}

	// 1. Session Init Handshake
	csID, err := rand.Int(rand.Reader, big.NewInt(70))
	clientSessionID := 1
	if err == nil {
		clientSessionID = int(csID.Int64()) + 1
	}

	userAgent := map[string]interface{}{
		"deviceType":     "ANDROID",
		"appVersion":     "26.25.0",
		"osVersion":      "Android 14",
		"timezone":       "Europe/Moscow",
		"screen":         "405dpi 405dpi 1080x2400",
		"pushDeviceType": "GCM",
		"arch":           "arm64-v8a",
		"locale":         "ru",
		"deviceLocale":   "ru",
		"buildNumber":    6790,
		"deviceName":     "Samsung SM-A536B",
	}

	initPayload := map[string]interface{}{
		"mt_instanceid":   c.cfg.MtInstanceID,
		"userAgent":       userAgent,
		"clientSessionId": clientSessionID,
		"deviceId":        c.cfg.DeviceID,
	}

	log.Printf("[gomax] Sending mobile session handshake (SESSION_INIT, deviceId=%s, mt_instanceid=%s, clientSessionId=%d)...",
		c.cfg.DeviceID, c.cfg.MtInstanceID, clientSessionID)
	initRes, err := c.Invoke(ctx, protocol.OpSessionInit, initPayload)
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("session init failed: %w", err)
	}

	// Validate response does not contain an API validation error
	if errMsg, ok := initRes["error"].(string); ok && errMsg != "" {
		_ = connManager.Close()
		return fmt.Errorf("session init validation error from server: %s (message: %v)", errMsg, initRes["message"])
	}
	if errCode, ok := initRes["err"].(string); ok && errCode != "" {
		_ = connManager.Close()
		return fmt.Errorf("session init error from server: %s", errCode)
	}

	var callsSeed int64
	if cs, ok := initRes["callsSeed"].(int64); ok {
		callsSeed = cs
	} else if csF, ok := initRes["callsSeed"].(float64); ok {
		callsSeed = int64(csF)
	} else if csI, ok := initRes["callsSeed"].(int); ok {
		callsSeed = int64(csI)
	} else if csU, ok := initRes["callsSeed"].(uint64); ok {
		callsSeed = int64(csU)
	}

	c.mu.Lock()
	c.callsSeed = callsSeed
	c.mu.Unlock()

	log.Printf("[gomax] Handshake completed successfully (callsSeed=%d)", callsSeed)

	// 2. Load or execute Auth
	token := c.cfg.Token
	if token == "" && sessInfo != nil {
		token = sessInfo.Token
	}

	if token == "" {
		if c.cfg.Phone == "" {
			_ = connManager.Close()
			return errors.New("phone number required for initial authentication")
		}
		log.Printf("[gomax] No active session token found; starting SMS authentication for %s...", c.cfg.Phone)
		smsFlow := c.cfg.AuthFlow
		if smsFlow.CodeProvider == nil {
			smsFlow.CodeProvider = &auth.ConsoleCodeProvider{}
		}
		if smsFlow.PasswordProvider == nil {
			smsFlow.PasswordProvider = &auth.ConsolePasswordProvider{}
		}
		smsFlow.DeviceID = c.cfg.DeviceID
		smsFlow.CallsSeed = callsSeed
		if smsFlow.FpGen == nil {
			smsFlow.FpGen = c.fpGen
		}
		if smsFlow.Arch == "" {
			smsFlow.Arch = "arm64-v8a"
		}

		authRes, err := smsFlow.Authenticate(ctx, c, c.cfg.Phone)
		if err != nil {
			_ = connManager.Close()
			return fmt.Errorf("authentication failed: %w", err)
		}
		token = authRes.Token
		_ = c.store.SaveSession(&session.SessionInfo{
			Token:        token,
			Phone:        c.cfg.Phone,
			DeviceID:     c.cfg.DeviceID,
			MTInstanceID: c.cfg.MtInstanceID,
		})
		log.Printf("[gomax] Session credentials saved to store")
	} else {
		log.Printf("[gomax] Resuming session for device %s...", c.cfg.DeviceID)
	}

	// 3. Login
	log.Printf("[gomax] Logging in with session token...")
	loginPayload := map[string]interface{}{
		"token":         token,
		"deviceId":      c.cfg.DeviceID,
		"userAgent":     userAgent,
		"chatsSync":     -1,
		"contactsSync":  -1,
		"presenceSync":  -1,
		"draftsSync":    -1,
		"interactive":   true,
	}
	if callsSeed != 0 {
		fp, _ := c.fpGen.GenerateFingerprint(c.cfg.DeviceID, callsSeed, "arm64-v8a")
		if len(fp) > 0 {
			loginPayload["chatCacheFingerprint"] = fp
			loginPayload["fingerprint"] = fp
		}
	}

	loginRes, err := c.Invoke(ctx, protocol.OpLogin, loginPayload)
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("login failed: %w", err)
	}

	// If token refreshed
	if newToken, ok := loginRes["token"].(string); ok && newToken != "" && newToken != token {
		token = newToken
		_ = c.store.UpdateToken(c.cfg.Phone, token)
	}

	// Resolve Self User profile
	c.Me = &types.User{}
	if profileData, ok := loginRes["profile"].(map[string]interface{}); ok {
		userData := profileData
		if contactData, ok := profileData["contact"].(map[string]interface{}); ok {
			userData = contactData
		}
		if id, ok := userData["id"].(int64); ok {
			c.Me.ID = id
		} else if idF, ok := userData["id"].(float64); ok {
			c.Me.ID = int64(idF)
		} else if idI, ok := userData["id"].(int); ok {
			c.Me.ID = int64(idI)
		}
		if fn, ok := userData["firstName"].(string); ok {
			c.Me.FirstName = fn
		}
		if ln, ok := userData["lastName"].(string); ok {
			c.Me.LastName = ln
		}
		if ph, ok := userData["phone"].(string); ok {
			c.Me.Phone = ph
		}
	} else if userMap, ok := loginRes["user"].(map[string]interface{}); ok {
		if id, ok := userMap["id"].(int64); ok {
			c.Me.ID = id
		} else if idF, ok := userMap["id"].(float64); ok {
			c.Me.ID = int64(idF)
		} else if idI, ok := userMap["id"].(int); ok {
			c.Me.ID = int64(idI)
		}
		if fn, ok := userMap["firstName"].(string); ok {
			c.Me.FirstName = fn
		}
	}

	log.Printf("[gomax] Login successful! Logged in as '%s' (ID: %d)", c.Me.FirstName, c.Me.ID)

	// Dispatch OnStart hooks
	c.router.DispatchStart(ctx)

	// Await disconnect or cancellation
	select {
	case <-ctx.Done():
		_ = connManager.Close()
		return ctx.Err()
	case disErr := <-disconnectCh:
		_ = connManager.Close()
		return disErr
	}
}

func (c *Client) handleEvent(frame *protocol.InboundFrame) {
	if frame.Opcode == protocol.OpMsgSend || frame.Opcode == protocol.OpChatHistory {
		mData := frame.Payload
		msg := &types.Message{
			Time: time.Now().Unix(),
		}
		if id, ok := mData["id"].(int64); ok {
			msg.ID = id
		} else if idF, ok := mData["id"].(float64); ok {
			msg.ID = int64(idF)
		}
		if text, ok := mData["text"].(string); ok {
			msg.Text = text
		}
		if chatID, ok := mData["chatId"].(int64); ok {
			msg.ChatID = chatID
		} else if chatIDF, ok := mData["chatId"].(float64); ok {
			msg.ChatID = int64(chatIDF)
		}
		if sender, ok := mData["sender"].(int64); ok {
			msg.SenderID = sender
		} else if senderF, ok := mData["sender"].(float64); ok {
			msg.SenderID = int64(senderF)
		}

		c.router.DispatchMessage(c.ctx, msg)
	}
}

// Close disconnects and terminates client session.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}
	c.started = false
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
