package client

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/session"
	"github.com/ebunyt-dotcom/gomax/pkg/transport"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
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
	Self     *selfapi.SelfService

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
	wc.Self = selfapi.NewSelfService(wc)

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

// OnMessageEdit registers a handler for message edit events.
func (wc *WebClient) OnMessageEdit(handler func(ctx context.Context, msg *types.Message) error) {
	wc.router.OnMessageEdit(handler)
}

// OnMessageDelete registers a handler for message delete events.
func (wc *WebClient) OnMessageDelete(handler func(ctx context.Context, chatID, msgID int64) error) {
	wc.router.OnMessageDelete(handler)
}

// OnReaction registers a handler for reaction add/remove events.
func (wc *WebClient) OnReaction(handler func(ctx context.Context, ev *types.ReactionEvent) error) {
	wc.router.OnReaction(handler)
}

// OnChatUpdate registers a handler for chat metadata update events.
func (wc *WebClient) OnChatUpdate(handler func(ctx context.Context, chat *types.Chat) error) {
	wc.router.OnChatUpdate(handler)
}

// OnPresence registers a handler for user online/offline status events.
func (wc *WebClient) OnPresence(handler func(ctx context.Context, ev *types.PresenceEvent) error) {
	wc.router.OnPresence(handler)
}

// OnTyping registers a handler for user typing indicator events.
func (wc *WebClient) OnTyping(handler func(ctx context.Context, ev *types.TypingEvent) error) {
	wc.router.OnTyping(handler)
}

// OnDisconnect registers a handler called when the client disconnects.
func (wc *WebClient) OnDisconnect(handler func(ctx context.Context, err error)) {
	wc.router.OnDisconnect(handler)
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
		src := profileData
		if contact, ok := profileData["contact"].(map[string]interface{}); ok {
			src = contact
		}
		if id, ok := extractInt64(src["id"]); ok {
			wc.Me.ID = id
		}
		if fn, ok := src["firstName"].(string); ok {
			wc.Me.FirstName = fn
		}
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

// parseMessage extracts a full types.Message from a raw payload map for WebClient.
func (wc *WebClient) parseMessage(payload map[string]interface{}) *types.Message {
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

	wc.mu.RLock()
	me := wc.Me
	wc.mu.RUnlock()
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
func (wc *WebClient) handleEvent(frame *protocol.InboundFrame) {
	ctx := wc.ctx
	if ctx == nil {
		return
	}

	switch frame.Opcode {
	case protocol.OpMsgSend, protocol.OpNotifMessage:
		msg := wc.parseMessage(frame.Payload)
		if msg.ChatID != 0 || msg.ID != 0 {
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
		chatID, _ := extractInt64(frame.Payload["chatId"])
		msgID, _ := extractInt64(frame.Payload["messageId"])
		if msgID == 0 {
			msgID, _ = extractInt64(frame.Payload["id"])
		}
		wc.router.DispatchMessageDelete(ctx, chatID, msgID)

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
		wc.router.DispatchReaction(ctx, ev)

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
		wc.router.DispatchChatUpdate(ctx, chat)

	case protocol.OpContactPresence, protocol.OpNotifPresence:
		ev := &types.PresenceEvent{}
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if online, ok := frame.Payload["online"].(bool); ok {
			ev.Online = online
		} else if status, ok := frame.Payload["status"].(string); ok {
			ev.Online = (status == "online" || status == "ONLINE")
		}
		wc.router.DispatchPresence(ctx, ev)

	case protocol.OpMsgTyping, protocol.OpNotifTyping:
		ev := &types.TypingEvent{}
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if ev.UserID == 0 {
			ev.UserID, _ = extractInt64(frame.Payload["sender"])
		}
		wc.router.DispatchTyping(ctx, ev)

	default:
		if frame.Cmd == protocol.CmdEvent {
			wc.router.DispatchEvent(ctx, &types.RawEvent{
				Opcode:  uint16(frame.Opcode),
				Payload: frame.Payload,
			})
		}
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
