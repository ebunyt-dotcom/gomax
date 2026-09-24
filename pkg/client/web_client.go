package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	authapi "github.com/ebunyt-dotcom/gomax/pkg/api/auth"
	"github.com/ebunyt-dotcom/gomax/pkg/api/bots"
	"github.com/ebunyt-dotcom/gomax/pkg/api/chats"
	"github.com/ebunyt-dotcom/gomax/pkg/api/messages"
	selfapi "github.com/ebunyt-dotcom/gomax/pkg/api/selfapi"
	sessionapi "github.com/ebunyt-dotcom/gomax/pkg/api/sessionapi"
	"github.com/ebunyt-dotcom/gomax/pkg/api/uploads"
	"github.com/ebunyt-dotcom/gomax/pkg/api/users"
	"github.com/ebunyt-dotcom/gomax/pkg/auth"
	"github.com/ebunyt-dotcom/gomax/pkg/connection"
	"github.com/ebunyt-dotcom/gomax/pkg/dispatch"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/session"
	"github.com/ebunyt-dotcom/gomax/pkg/telemetry"
	"github.com/ebunyt-dotcom/gomax/pkg/transport"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// WebClient implements the WebSocket-based client with QR authentication.
type WebClient struct {
	cfg    *Config
	conn   *connection.ConnectionManager
	store  session.Store
	router *dispatch.Router

	Messages  *messages.MessageService
	Auth      *authapi.AuthService
	Chats     *chats.ChatService
	Users     *users.UserService
	Uploads   *uploads.UploadService
	Bots      *bots.BotsService
	Self      *selfapi.SelfService
	Session   *sessionapi.Service
	Telemetry *telemetry.Service

	Me           *types.User
	ChatCache    map[int64]types.Chat
	MessageCache map[int64][]types.Message
	mu           sync.RWMutex
	started      bool
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewWebClient creates a WebClient matching pymax.WebClient.
func NewWebClient(cfg *Config) *WebClient {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.URL == "" {
		cfg.URL = "wss://api.oneme.ru/websocket"
	}
	applyDefaults(cfg, true)
	var store session.Store
	if !cfg.PersistSession {
		store = session.NewInMemoryStore()
	} else if cfg.Store != nil {
		store = cfg.Store
	} else {
		store = session.NewFileStore(cfg.WorkDir, cfg.SessionName)
	}

	wc := &WebClient{
		cfg:          cfg,
		store:        store,
		router:       dispatch.NewRouter(),
		ChatCache:    make(map[int64]types.Chat),
		MessageCache: make(map[int64][]types.Message),
	}

	wc.Messages = messages.NewMessageService(wc)
	wc.Auth = authapi.NewAuthService(wc)
	wc.Chats = chats.NewChatService(wc)
	wc.Users = users.NewUserService(wc)
	wc.Chats.SetMessageActions(wc.Messages)
	wc.Uploads = uploads.NewUploadServiceWithOptions(wc, uploads.Options{
		Timeout: cfg.UploadTimeout, ProxyURL: cfg.Proxy,
		UserAgent: fmt.Sprintf("OKMessages/%s (%s; %s; %s)", cfg.AppVersion, cfg.OSVersion, cfg.DeviceName, cfg.Screen),
	})
	wc.Messages.SetAttachmentReadyWaiter(wc.Uploads.WaitForMessageAttachments)
	wc.Bots = bots.NewBotsService(wc)
	wc.Self = selfapi.NewSelfService(wc)
	wc.Self.SetUserActions(wc.Users)
	wc.Session = sessionapi.NewService(wc)
	wc.Telemetry = telemetry.NewService(wc, wc.telemetrySnapshot)
	wc.Self.SetSessionHooks(
		func(_ string, newToken string) error { return updateStoredToken(wc.store, wc.cfg, newToken) },
		func(hash string) error { return updateStoredConfigHash(wc.store, hash) },
	)

	return wc
}

func (wc *WebClient) telemetrySnapshot() telemetry.Snapshot {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	chats := make([]types.Chat, 0, len(wc.ChatCache))
	for _, chat := range wc.ChatCache {
		chats = append(chats, chat)
	}
	userID := int64(0)
	if wc.Me != nil {
		userID = wc.Me.ID
	}
	return telemetry.Snapshot{Ready: wc.started && wc.conn != nil && wc.conn.IsOpen() && wc.Me != nil, UserID: userID, Chats: chats}
}

// OnMessage registers an incoming message listener.
func (wc *WebClient) OnMessage(handler func(ctx context.Context, msg *types.Message) error, filters ...dispatch.MessagePredicate) {
	wc.router.OnMessage(handler, filters...)
}

// OnStart registers an on_start listener.
func (wc *WebClient) OnStart(handler func(ctx context.Context) error) {
	wc.router.OnStart(handler)
}

// OnMessageEdit registers a handler for message edit events.
func (wc *WebClient) OnMessageEdit(handler func(ctx context.Context, msg *types.Message) error, filters ...dispatch.MessagePredicate) {
	wc.router.OnMessageEdit(handler, filters...)
}

// OnMessageDelete registers a handler for message delete events.
func (wc *WebClient) OnMessageDelete(handler func(ctx context.Context, chatID, msgID int64) error) {
	wc.router.OnMessageDelete(handler)
}

func (wc *WebClient) OnMessageDeleteEvent(handler func(ctx context.Context, ev *types.MessageDeleteEvent) error, filters ...dispatch.MessageDeletePredicate) {
	wc.router.OnMessageDeleteEvent(handler, filters...)
}

// OnMessageRead registers a read-marker handler.
func (wc *WebClient) OnMessageRead(handler func(ctx context.Context, ev *types.MessageReadEvent) error, filters ...dispatch.MessageReadPredicate) {
	wc.router.OnMessageRead(handler, filters...)
}

// OnUserUpdate registers a contact/profile update handler.
func (wc *WebClient) OnUserUpdate(handler func(ctx context.Context, ev *types.UserUpdateEvent) error, filters ...dispatch.UserUpdatePredicate) {
	wc.router.OnUserUpdate(handler, filters...)
}

// OnReaction registers a handler for reaction add/remove events.
func (wc *WebClient) OnReaction(handler func(ctx context.Context, ev *types.ReactionEvent) error, filters ...dispatch.ReactionPredicate) {
	wc.router.OnReaction(handler, filters...)
}

// OnChatUpdate registers a handler for chat metadata update events.
func (wc *WebClient) OnChatUpdate(handler func(ctx context.Context, chat *types.Chat) error, filters ...dispatch.ChatUpdatePredicate) {
	wc.router.OnChatUpdate(handler, filters...)
}

// OnPresence registers a handler for user online/offline status events.
func (wc *WebClient) OnPresence(handler func(ctx context.Context, ev *types.PresenceEvent) error, filters ...dispatch.PresencePredicate) {
	wc.router.OnPresence(handler, filters...)
}

// OnTyping registers a handler for user typing indicator events.
func (wc *WebClient) OnTyping(handler func(ctx context.Context, ev *types.TypingEvent) error, filters ...dispatch.TypingPredicate) {
	wc.router.OnTyping(handler, filters...)
}

// OnDisconnect registers a handler called when the client disconnects.
func (wc *WebClient) OnDisconnect(handler func(ctx context.Context, err error)) {
	wc.router.OnDisconnect(handler)
}

// OnRaw registers a handler for low-level event frames not consumed by a
// typed event handler.
func (wc *WebClient) OnRaw(handler func(ctx context.Context, event *types.RawEvent) error, filters ...dispatch.EventPredicate) {
	wc.router.OnEvent(handler, filters...)
}

func (wc *WebClient) OnError(handler func(ctx context.Context, err error)) {
	wc.router.OnError(handler)
}
func (wc *WebClient) OnErrorScoped(scope dispatch.ErrorScope, handler func(context.Context, error)) {
	wc.router.OnErrorScoped(scope, handler)
}
func (wc *WebClient) IncludeRouter(router *dispatch.Router) { wc.router.Include(router) }

func (wc *WebClient) Connect(ctx context.Context) error { return wc.Start(ctx) }
func (wc *WebClient) Stop() error                       { return wc.Close() }
func (wc *WebClient) IsConnected() bool {
	wc.mu.RLock()
	conn := wc.conn
	wc.mu.RUnlock()
	return conn != nil && conn.IsOpen()
}
func (wc *WebClient) Relogin(ctx context.Context) error {
	if err := clearStoredSessionToken(wc.store); err != nil {
		return err
	}
	wc.cfg.Token = ""
	wc.mu.RLock()
	conn, started := wc.conn, wc.started
	wc.mu.RUnlock()
	if started && conn != nil {
		conn.Fail(errors.New("relogin requested"))
		return nil
	}
	return wc.Start(ctx)
}

// SetInteractive changes the interactive/presence flag used by the next
// WebSocket login and reconnects.
func (wc *WebClient) SetInteractive(online bool) {
	wc.mu.Lock()
	wc.cfg.Interactive = online
	wc.mu.Unlock()
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
		return nil, protocol.NewApiError(inbound)
	}

	return inbound.Payload, nil
}

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

		if wc.cfg.Relogin && isInvalidLoginToken(err) {
			_ = clearStoredSessionToken(wc.store)
			wc.cfg.Token = ""
			continue
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
	// Max's production WebSocket endpoint expects the same binary MessagePack
	// framing as v0.0.1 for SESSION_INIT and the QR flow. Text/JSON frames are
	// supported by transport as an opt-in compatibility mode, but must not be
	// used by the Max WebClient handshake.
	wsOpts.TextFrames = false
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
		&connection.Config{
			Interactive:    wc.cfg.Interactive,
			RequestTimeout: wc.cfg.RequestTimeout,
		},
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

	// Restore the session before SESSION_INIT so the same device ID is sent
	// during the handshake, exactly as PyMax does.
	sessInfo, err := wc.store.LoadSession()
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("failed to load session: %w", err)
	}
	if wc.cfg.DeviceID == "" && sessInfo != nil && sessInfo.DeviceID != "" {
		wc.cfg.DeviceID = sessInfo.DeviceID
	}
	if sessInfo != nil {
		restoreSessionUserAgent(wc.cfg, sessInfo.UserAgent)
	}
	if wc.cfg.DeviceID == "" {
		wc.cfg.DeviceID = randomHex(8)
	}

	// 1. Session Init. WebHandshakePayload in PyMax includes both the
	// browser user-agent and device ID; sending only deviceId creates a
	// partially initialized web session and can prevent QR issuance.
	_, err = wc.Invoke(ctx, protocol.OpSessionInit, map[string]interface{}{
		"userAgent": defaultWebUserAgent(wc.cfg),
		"deviceId":  wc.cfg.DeviceID,
	})
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("session init failed: %w", err)
	}

	// 2. Auth or Load token
	token := wc.cfg.Token
	if token == "" && sessInfo != nil {
		token = sessInfo.Token
	}

	if token == "" {
		qrFlow := wc.cfg.QrAuthFlow
		if qrFlow == nil {
			qrFlow = auth.NewQrAuthFlow(nil, nil)
		}
		authRes, err := qrFlow.Authenticate(ctx, wc)
		if err != nil {
			_ = connManager.Close()
			return fmt.Errorf("qr authentication failed: %w", err)
		}
		token = authRes.Token
		if err := wc.store.SaveSession(&session.SessionInfo{
			Token: token, DeviceID: wc.cfg.DeviceID, UserAgent: sessionUserAgent(wc.cfg),
		}); err != nil {
			_ = connManager.Close()
			return fmt.Errorf("save authenticated web session: %w", err)
		}
	}

	webSync := session.SyncState{
		ChatsSync: -1, ContactsSync: -1, DraftsSync: -1,
		PresenceSync: -1, ConfigHash: session.DefaultConfigHash,
	}
	if sessInfo != nil {
		webSync = sessInfo.Sync
		if webSync.ConfigHash == "" {
			webSync.ConfigHash = session.DefaultConfigHash
		}
	}
	webSync = resolveSync(webSync, wc.cfg.Sync)

	// 3. Login. Keep this payload equivalent to PyMax's WebSyncPayload.
	loginRes, err := wc.Invoke(ctx, protocol.OpLogin, map[string]interface{}{
		"token":        token,
		"chatsCount":   40,
		"interactive":  wc.cfg.Interactive,
		"chatsSync":    webSync.ChatsSync,
		"contactsSync": webSync.ContactsSync,
		"presenceSync": webSync.PresenceSync,
		"draftsSync":   webSync.DraftsSync,
	})
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("web login failed: %w", err)
	}
	if flags, ok := loginRes["login2Flags"].(map[string]interface{}); ok {
		configEnabled, _ := flags["configEnabled"].(bool)
		contactEnabled, _ := flags["contactEnabled"].(bool)
		profileEnabled, _ := flags["profileEnabled"].(bool)
		if configEnabled || contactEnabled || profileEnabled {
			contactsSync := int64(-1)
			if contactEnabled {
				contactsSync = webSync.ContactsSync
			}
			login2Res, login2Err := wc.Invoke(ctx, protocol.OpLogin2, map[string]interface{}{"needProfile": profileEnabled, "contactsSync": contactsSync, "configHash": webSync.ConfigHash})
			if login2Err != nil {
				_ = connManager.Close()
				return fmt.Errorf("web login2 failed: %w", login2Err)
			}
			for key, value := range login2Res {
				loginRes[key] = value
			}
		}
	}
	if newToken, ok := loginRes["token"].(string); ok && newToken != "" && newToken != token {
		oldToken := token
		token = newToken
		if err := wc.store.UpdateToken(oldToken, token); err != nil {
			_ = connManager.Close()
			return fmt.Errorf("persist refreshed web token: %w", err)
		}
	}
	wc.cfg.Token = token
	if syncTime, ok := extractInt64(loginRes["time"]); ok {
		webSync.ChatsSync = syncTime
		webSync.ContactsSync = syncTime
		webSync.DraftsSync = syncTime
		webSync.PresenceSync = syncTime
	}
	if config, ok := loginRes["config"].(map[string]interface{}); ok {
		if hash := types.StringValue(config["hash"]); hash != "" {
			webSync.ConfigHash = hash
		}
	}
	if err := wc.store.SaveSession(&session.SessionInfo{
		Token: token, DeviceID: wc.cfg.DeviceID, Sync: webSync, UserAgent: sessionUserAgent(wc.cfg),
	}); err != nil {
		return fmt.Errorf("save web session: %w", err)
	}

	me := &types.User{}
	if profileData, ok := loginRes["profile"].(map[string]interface{}); ok {
		parsed := types.ParseUserPayload(profileData)
		parsed.Bind(wc.Users)
		me = &parsed
	} else if userData, ok := loginRes["user"].(map[string]interface{}); ok {
		parsed := types.ParseUserPayload(userData)
		parsed.Bind(wc.Users)
		me = &parsed
	}
	for _, key := range []string{"contacts", "contactInfos"} {
		if rawUsers, ok := loginRes[key].([]interface{}); ok {
			parsed := make([]types.User, 0, len(rawUsers))
			for _, raw := range rawUsers {
				if m, ok := raw.(map[string]interface{}); ok {
					user := types.ParseUserPayload(m)
					user.Bind(wc.Users)
					parsed = append(parsed, user)
				}
			}
			wc.Users.SeedCache(parsed)
		}
	}
	chatCache := make(map[int64]types.Chat)
	if rawChats, ok := loginRes["chats"].([]interface{}); ok {
		parsedChats := make([]types.Chat, 0, len(rawChats))
		for _, raw := range rawChats {
			if m, ok := raw.(map[string]interface{}); ok {
				chat := types.ParseChatPayload(m)
				chat.Bind(wc.Messages, wc.Chats)
				chatCache[chat.ID] = chat
				parsedChats = append(parsedChats, chat)
			}
		}
		wc.Chats.SeedCache(parsedChats)
	}
	messageCache := make(map[int64][]types.Message)
	if rawMessages, ok := loginRes["messages"].(map[string]interface{}); ok {
		for rawChatID, value := range rawMessages {
			chatID, _ := types.Int64Value(rawChatID)
			if list, ok := value.([]interface{}); ok {
				for _, raw := range list {
					if m, ok := raw.(map[string]interface{}); ok {
						msg := types.ParseMessagePayload(m)
						msg.Bind(wc.Messages)
						if msg.ChatID == 0 {
							msg.ChatID = chatID
						}
						messageCache[chatID] = append(messageCache[chatID], *msg)
					}
				}
			}
		}
	}
	wc.mu.Lock()
	wc.Me = me
	wc.ChatCache = chatCache
	wc.MessageCache = messageCache
	wc.mu.Unlock()
	if wc.cfg.ClientSessionID <= 0 {
		wc.cfg.ClientSessionID = randomClientSessionID()
	}
	if wc.cfg.Telemetry {
		wc.Telemetry.Start(ctx, wc.cfg.ClientSessionID)
		defer wc.Telemetry.Stop()
	}

	wc.router.DispatchStart(ctx)

	select {
	case <-ctx.Done():
		_ = connManager.Close()
		wc.router.DispatchDisconnect(ctx, ctx.Err())
		return ctx.Err()
	case disErr := <-disconnectCh:
		_ = connManager.Close()
		wc.router.DispatchDisconnect(ctx, disErr)
		return disErr
	}
}

func defaultWebUserAgent(cfg *Config) map[string]interface{} {
	// PyMax's MobileUserAgentPayload.to_web_payload selects only the web
	// aliases and never mutates the caller's model. Copy the values here so a
	// caller-provided Config.UserAgent remains reusable for later handshakes.
	const defaultHeader = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"
	source := mobileUserAgent(cfg)
	userAgent := make(map[string]interface{}, 9)
	for _, key := range []string{
		"deviceType", "locale", "deviceLocale", "osVersion", "deviceName",
		"headerUserAgent", "appVersion", "screen", "timezone",
	} {
		if value, ok := source[key]; ok {
			userAgent[key] = value
		}
	}
	if value, ok := userAgent["headerUserAgent"].(string); !ok || value == "" {
		userAgent["headerUserAgent"] = defaultHeader
	}
	return userAgent
}

// parseMessage extracts a full types.Message from a raw payload map for WebClient.
func (wc *WebClient) parseMessage(payload map[string]interface{}) *types.Message {
	msg := types.ParseMessagePayload(payload).Bind(wc.Messages)
	if msg.Time == 0 {
		msg.Time = time.Now().UnixMilli()
	}

	wc.mu.RLock()
	me := wc.Me
	wc.mu.RUnlock()
	if me != nil && msg.SenderID != 0 && msg.SenderID == me.ID {
		msg.IsOutgoing = true
	}

	return msg
}

// handleEvent dispatches inbound server push frames to registered handlers.
func (wc *WebClient) handleEvent(frame *protocol.InboundFrame) {
	ctx := wc.ctx
	if ctx == nil {
		return
	}
	defer wc.router.DispatchEvent(ctx, &types.RawEvent{Type: frame.Opcode.String(), Opcode: uint16(frame.Opcode), Payload: frame.Payload})

	switch frame.Opcode {
	case protocol.OpNotifAttach:
		wc.Uploads.NotifyReady(frame.Payload)

	case protocol.OpNotifMark:
		ev := &types.MessageReadEvent{}
		ev.SetAsUnread, _ = frame.Payload["setAsUnread"].(bool)
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		ev.MessageID, _ = extractInt64(frame.Payload["messageId"])
		ev.Mark, _ = extractInt64(frame.Payload["mark"])
		wc.router.DispatchMessageRead(ctx, ev)

	case protocol.OpNotifContact:
		user := types.ParseUserPayload(frame.Payload)
		user.Bind(wc.Users)
		wc.Users.SeedCache([]types.User{user})
		wc.router.DispatchUserUpdate(ctx, &types.UserUpdateEvent{User: user})

	case protocol.OpNotifMessage:
		msg := wc.parseMessage(frame.Payload)
		if msg.Status == types.MessageStatusEdited {
			wc.router.DispatchMessageEdit(ctx, msg)
		} else if msg.Status == types.MessageStatusRemoved {
			wc.router.DispatchMessageDeleteEvent(ctx, &types.MessageDeleteEvent{ChatID: msg.ChatID, MessageIDs: []int64{msg.ID}, Message: msg})
		} else if msg.ChatID != 0 || msg.ID != 0 {
			wc.router.DispatchMessage(ctx, msg)
		}

	case protocol.OpChatHistory:
		if msgList, ok := frame.Payload["messages"].([]interface{}); ok {
			for _, item := range msgList {
				if mData, ok := item.(map[string]interface{}); ok {
					msg := wc.parseMessage(mData)
					wc.router.DispatchMessage(ctx, msg)
				}
			}
		} else {
			msg := wc.parseMessage(frame.Payload)
			if msg.ID != 0 {
				wc.router.DispatchMessage(ctx, msg)
			}
		}

	case protocol.OpMsgEdit:
		msg := wc.parseMessage(frame.Payload)
		wc.router.DispatchMessageEdit(ctx, msg)

	case protocol.OpMsgDelete, protocol.OpNotifMsgDelete:
		wc.router.DispatchMessageDeleteEvent(ctx, types.ParseMessageDeletePayload(frame.Payload))

	case protocol.OpNotifMsgReactionsChanged, protocol.OpNotifMsgYouReacted:
		ev := &types.ReactionEvent{}
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.MessageID, _ = extractInt64(frame.Payload["messageId"])
		ev.MessageIDRaw = types.StringValue(frame.Payload["messageId"])
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
		if total, ok := extractInt64(frame.Payload["totalCount"]); ok {
			ev.TotalCount = int(total)
		}
		if counters, ok := frame.Payload["counters"].([]interface{}); ok {
			for _, item := range counters {
				if m, ok := item.(map[string]interface{}); ok {
					count, _ := extractInt64(m["count"])
					ev.Counters = append(ev.Counters, types.ReactionCounter{Reaction: types.StringValue(m["reaction"]), Count: int(count)})
				}
			}
		}
		wc.router.DispatchReaction(ctx, ev)

	case protocol.OpNotifChat:
		chat := types.ParseChatPayload(frame.Payload)
		chat.Bind(wc.Messages, wc.Chats)
		if chat.ID != 0 {
			wc.mu.Lock()
			wc.ChatCache[chat.ID] = chat
			wc.mu.Unlock()
		}
		wc.router.DispatchChatUpdate(ctx, &chat)

	case protocol.OpNotifPresence:
		ev := &types.PresenceEvent{}
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if p, ok := frame.Payload["presence"].(map[string]interface{}); ok {
			ev.Presence = &types.Presence{Status: types.StringValue(p["status"])}
			ev.Presence.StatusCode, _ = extractInt64(p["status"])
			ev.Presence.Seen, _ = extractInt64(p["seen"])
			ev.Presence.LastSeen, _ = extractInt64(p["lastSeen"])
			ev.Presence.Online, _ = p["online"].(bool)
			ev.Online = ev.Presence.Online || ev.Presence.Status == "online" || ev.Presence.Status == "ONLINE"
		}
		if online, ok := frame.Payload["online"].(bool); ok {
			ev.Online = online
		} else if status, ok := frame.Payload["status"].(string); ok {
			ev.Online = (status == "online" || status == "ONLINE")
		}
		wc.router.DispatchPresence(ctx, ev)

	case protocol.OpNotifTyping:
		ev := &types.TypingEvent{}
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if ev.UserID == 0 {
			ev.UserID, _ = extractInt64(frame.Payload["sender"])
		}
		wc.router.DispatchTyping(ctx, ev)

	default:
	}
}

// Close disconnects the WebSocket client.
func (wc *WebClient) Close() error {
	if wc.Telemetry != nil {
		wc.Telemetry.Stop()
	}
	wc.mu.Lock()
	if !wc.started {
		wc.mu.Unlock()
		return nil
	}
	wc.started = false
	cancel, conn := wc.cancel, wc.conn
	wc.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	var errs []error
	if conn != nil {
		if err := conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if store, ok := wc.store.(session.ExtendedStore); ok {
		if err := store.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
