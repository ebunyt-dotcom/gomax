package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api/chats"
	"github.com/ebunyt-dotcom/gomax/pkg/api/messages"
	selfapi "github.com/ebunyt-dotcom/gomax/pkg/api/selfapi"
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
	Self     *selfapi.SelfService

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

// NonRecoverableError indicates an error that should terminate client.Start immediately without reconnecting.
type NonRecoverableError struct {
	Err error
}

func (e *NonRecoverableError) Error() string {
	return e.Err.Error()
}

func (e *NonRecoverableError) Unwrap() error {
	return e.Err
}

func nonRecoverable(err error) error {
	if err == nil {
		return nil
	}
	return &NonRecoverableError{Err: err}
}

func extractInt64(val any) (int64, bool) {
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int16:
		return int64(v), true
	case int8:
		return int64(v), true
	case uint64:
		return int64(v), true
	case uint:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint8:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, true
		}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, true
		}
	}
	return 0, false
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
	c.Self = selfapi.NewSelfService(c)

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

// OnMessageEdit registers a handler for message edit events.
func (c *Client) OnMessageEdit(handler func(ctx context.Context, msg *types.Message) error) {
	c.router.OnMessageEdit(handler)
}

// OnMessageDelete registers a handler for message delete events.
func (c *Client) OnMessageDelete(handler func(ctx context.Context, chatID, msgID int64) error) {
	c.router.OnMessageDelete(handler)
}

// OnReaction registers a handler for reaction add/remove events.
func (c *Client) OnReaction(handler func(ctx context.Context, ev *types.ReactionEvent) error) {
	c.router.OnReaction(handler)
}

// OnChatUpdate registers a handler for chat metadata update events.
func (c *Client) OnChatUpdate(handler func(ctx context.Context, chat *types.Chat) error) {
	c.router.OnChatUpdate(handler)
}

// OnPresence registers a handler for user online/offline status events.
func (c *Client) OnPresence(handler func(ctx context.Context, ev *types.PresenceEvent) error) {
	c.router.OnPresence(handler)
}

// OnTyping registers a handler for user typing indicator events.
func (c *Client) OnTyping(handler func(ctx context.Context, ev *types.TypingEvent) error) {
	c.router.OnTyping(handler)
}

// OnDisconnect registers a handler called when the client disconnects.
func (c *Client) OnDisconnect(handler func(ctx context.Context, err error)) {
	c.router.OnDisconnect(handler)
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

		var nre *NonRecoverableError
		if errors.As(err, &nre) {
			return err
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

	if c.fpGen == nil {
		c.fpGen = fingerprint.NewFingerprintGenerator(fingerprint.DefaultFingerprint())
	}

	// Load session info to reuse deviceId and mt_instanceid if previously saved
	sessInfo, err := c.store.LoadSession()
	if err != nil {
		_ = connManager.Close()
		return nonRecoverable(fmt.Errorf("failed to load session: %w", err))
	}

	if sessInfo != nil {
		if c.cfg.DeviceID == "" && sessInfo.DeviceID != "" {
			c.cfg.DeviceID = sessInfo.DeviceID
		}
		if c.cfg.MtInstanceID == "" && sessInfo.MTInstanceID != "" {
			c.cfg.MtInstanceID = sessInfo.MTInstanceID
		}
		if c.cfg.Phone == "" && sessInfo.Phone != "" {
			c.cfg.Phone = sessInfo.Phone
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
	if errVal, ok := initRes["error"]; ok && errVal != nil {
		_ = connManager.Close()
		msg := initRes["message"]
		if msg == nil {
			msg = initRes["localizedMessage"]
		}
		return nonRecoverable(fmt.Errorf("session init validation error from server: %v (message: %v)", errVal, msg))
	}
	if errVal, ok := initRes["err"]; ok && errVal != nil {
		_ = connManager.Close()
		return nonRecoverable(fmt.Errorf("session init error from server: %v", errVal))
	}

	var callsSeed int64
	if cs, ok := extractInt64(initRes["callsSeed"]); ok {
		callsSeed = cs
	} else if cs, ok := extractInt64(initRes["calls_seed"]); ok {
		callsSeed = cs
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
			return nonRecoverable(errors.New("phone number required for initial authentication"))
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
			return nonRecoverable(fmt.Errorf("authentication failed: %w", err))
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

	// Validate login response for server rejection
	if errVal, ok := loginRes["error"]; ok && errVal != nil {
		_ = connManager.Close()
		msg := loginRes["message"]
		if msg == nil {
			msg = loginRes["localizedMessage"]
		}
		return nonRecoverable(fmt.Errorf("login rejected by server: %v (message: %v)", errVal, msg))
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
		if id, ok := extractInt64(userData["id"]); ok {
			c.Me.ID = id
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
		if id, ok := extractInt64(userMap["id"]); ok {
			c.Me.ID = id
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
		c.router.DispatchDisconnect(ctx, ctx.Err())
		return ctx.Err()
	case disErr := <-disconnectCh:
		_ = connManager.Close()
		c.router.DispatchDisconnect(ctx, disErr)
		return disErr
	}
}

// parseMessage extracts a full types.Message from a raw payload map.
// msgPayload may be the top-level frame payload or a nested "message" sub-object.
func (c *Client) parseMessage(payload map[string]interface{}) *types.Message {
	src := payload
	if nested, ok := payload["message"].(map[string]interface{}); ok {
		src = nested
	}

	msg := &types.Message{Time: time.Now().Unix()}

	if id, ok := extractInt64(src["id"]); ok {
		msg.ID = id
	}
	if cid, ok := extractInt64(src["cid"]); ok {
		msg.CID = cid
	}
	if chatID, ok := extractInt64(src["chatId"]); ok {
		msg.ChatID = chatID
	}
	if sender, ok := extractInt64(src["sender"]); ok {
		msg.SenderID = sender
	} else if sender, ok := extractInt64(src["senderId"]); ok {
		msg.SenderID = sender
	}
	if text, ok := src["text"].(string); ok {
		msg.Text = text
	}
	if ts, ok := src["time"].(float64); ok {
		msg.Time = int64(ts)
	} else if ts, ok := extractInt64(src["time"]); ok {
		msg.Time = ts
	}
	if replyTo, ok := extractInt64(src["replyTo"]); ok {
		msg.ReplyToMsgID = replyTo
	}
	if editedAt, ok := extractInt64(src["editedAt"]); ok {
		msg.EditedAt = editedAt
	}
	if pinned, ok := src["isPinned"].(bool); ok {
		msg.IsPinned = pinned
	}

	// Mark as outgoing if sender matches logged-in user
	c.mu.RLock()
	me := c.Me
	c.mu.RUnlock()
	if me != nil && msg.SenderID != 0 && msg.SenderID == me.ID {
		msg.IsOutgoing = true
	}

	// Parse attachments
	if rawAttaches, ok := src["attaches"].([]interface{}); ok {
		for _, item := range rawAttaches {
			if aData, ok := item.(map[string]interface{}); ok {
				attach := types.Attachment{}
				if t, ok := aData["type"].(string); ok {
					attach.Type = types.AttachmentType(t)
				}
				if url, ok := aData["url"].(string); ok {
					attach.URL = url
				}
				if token, ok := aData["token"].(string); ok {
					attach.Token = token
				}
				if id, ok := aData["id"].(string); ok {
					attach.ID = id
				}
				if fname, ok := aData["fileName"].(string); ok {
					attach.FileName = fname
				}
				if size, ok := aData["fileSize"].(float64); ok {
					attach.FileSize = int64(size)
				}
				if dur, ok := aData["duration"].(float64); ok {
					attach.Duration = int(dur)
				}
				msg.Attachments = append(msg.Attachments, attach)
			}
		}
	}

	return msg
}

// handleEvent dispatches inbound server push frames to registered handlers.
func (c *Client) handleEvent(frame *protocol.InboundFrame) {
	ctx := c.ctx
	if ctx == nil {
		return
	}

	switch frame.Opcode {
	// Incoming new message
	case protocol.OpMsgSend, protocol.OpNotifMessage:
		msg := c.parseMessage(frame.Payload)
		if msg.ChatID != 0 || msg.ID != 0 {
			c.router.DispatchMessage(ctx, msg)
		}

	// Bulk history push during sync
	case protocol.OpChatHistory:
		if msgList, ok := frame.Payload["messages"].([]interface{}); ok {
			for _, item := range msgList {
				if mData, ok := item.(map[string]interface{}); ok {
					msg := c.parseMessage(mData)
					c.router.DispatchMessage(ctx, msg)
				}
			}
		} else {
			// Single message in history event
			msg := c.parseMessage(frame.Payload)
			if msg.ID != 0 {
				c.router.DispatchMessage(ctx, msg)
			}
		}

	// Message edited
	case protocol.OpMsgEdit:
		msg := c.parseMessage(frame.Payload)
		c.router.DispatchMessageEdit(ctx, msg)

	// Single message deleted
	case protocol.OpMsgDelete, protocol.OpNotifMsgDelete:
		chatID, _ := extractInt64(frame.Payload["chatId"])
		msgID, _ := extractInt64(frame.Payload["messageId"])
		if msgID == 0 {
			msgID, _ = extractInt64(frame.Payload["id"])
		}
		c.router.DispatchMessageDelete(ctx, chatID, msgID)

	// Reaction changed on a message
	case protocol.OpMsgReaction, protocol.OpNotifMsgReactionsChanged, protocol.OpNotifMsgYouReacted:
		ev := &types.ReactionEvent{}
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.MessageID, _ = extractInt64(frame.Payload["messageId"])
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if rData, ok := frame.Payload["reaction"].(map[string]interface{}); ok {
			if id, ok := rData["id"].(string); ok {
				ev.Reaction = id
			}
		} else if r, ok := frame.Payload["reaction"].(string); ok {
			ev.Reaction = r
		}
		if removed, ok := frame.Payload["removed"].(bool); ok {
			ev.Removed = removed
		}
		c.router.DispatchReaction(ctx, ev)

	// Chat metadata updated
	case protocol.OpChatUpdate, protocol.OpNotifChat:
		chat := &types.Chat{}
		src := frame.Payload
		if cData, ok := frame.Payload["chat"].(map[string]interface{}); ok {
			src = cData
		}
		if id, ok := extractInt64(src["id"]); ok {
			chat.ID = id
		}
		if title, ok := src["title"].(string); ok {
			chat.Title = title
		}
		if isChannel, ok := src["isChannel"].(bool); ok {
			chat.IsChannel = isChannel
		}
		c.router.DispatchChatUpdate(ctx, chat)

	// User presence / online status changed
	case protocol.OpContactPresence, protocol.OpNotifPresence:
		ev := &types.PresenceEvent{}
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if online, ok := frame.Payload["online"].(bool); ok {
			ev.Online = online
		} else if status, ok := frame.Payload["status"].(string); ok {
			ev.Online = (status == "online" || status == "ONLINE")
		}
		c.router.DispatchPresence(ctx, ev)

	// User typing indicator
	case protocol.OpMsgTyping, protocol.OpNotifTyping:
		ev := &types.TypingEvent{}
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if ev.UserID == 0 {
			ev.UserID, _ = extractInt64(frame.Payload["sender"])
		}
		c.router.DispatchTyping(ctx, ev)

	// Any other server push event — route as raw
	default:
		if frame.Cmd == protocol.CmdEvent {
			c.router.DispatchEvent(ctx, &types.RawEvent{
				Opcode:  uint16(frame.Opcode),
				Payload: frame.Payload,
			})
		}
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

