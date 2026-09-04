package dispatch

import (
	"context"
	"sync"

	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// MessageHandler handles incoming new messages.
type MessageHandler func(ctx context.Context, msg *types.Message) error

// MessageEditHandler handles message edit events.
type MessageEditHandler func(ctx context.Context, msg *types.Message) error

// MessageDeleteHandler handles message delete events.
type MessageDeleteHandler func(ctx context.Context, chatID, msgID int64) error

// ReactionHandler handles reaction add/remove events.
type ReactionHandler func(ctx context.Context, ev *types.ReactionEvent) error

// ChatUpdateHandler handles chat metadata update events.
type ChatUpdateHandler func(ctx context.Context, chat *types.Chat) error

// PresenceHandler handles user online/offline status events.
type PresenceHandler func(ctx context.Context, ev *types.PresenceEvent) error

// TypingHandler handles user typing indicator events.
type TypingHandler func(ctx context.Context, ev *types.TypingEvent) error

// DisconnectHandler handles client disconnect events.
type DisconnectHandler func(ctx context.Context, err error)

// StartHandler handles client ready event.
type StartHandler func(ctx context.Context) error

// EventHandler handles raw unrecognized events.
type EventHandler func(ctx context.Context, event *types.RawEvent) error

// MessagePredicate is a filter function that determines if a message should be dispatched.
type MessagePredicate func(msg *types.Message) bool

type filteredHandler struct {
	predicates []MessagePredicate
	handler    MessageHandler
}

// Router stores registered event handlers and supports predicate filters.
type Router struct {
	mu sync.RWMutex

	filteredHandlers    []filteredHandler
	messageEditHandlers []MessageEditHandler
	messageDelHandlers  []MessageDeleteHandler
	reactionHandlers    []ReactionHandler
	chatUpdateHandlers  []ChatUpdateHandler
	presenceHandlers    []PresenceHandler
	typingHandlers      []TypingHandler
	disconnectHandlers  []DisconnectHandler
	startHandlers       []StartHandler
	eventHandlers       []EventHandler
}

// NewRouter creates a new event router.
func NewRouter() *Router {
	return &Router{}
}

// OnMessage registers a message handler with optional filter predicates.
func (r *Router) OnMessage(handler MessageHandler, filters ...MessagePredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filteredHandlers = append(r.filteredHandlers, filteredHandler{
		predicates: filters,
		handler:    handler,
	})
}

// OnMessageEdit registers a handler for message edit events.
func (r *Router) OnMessageEdit(handler MessageEditHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageEditHandlers = append(r.messageEditHandlers, handler)
}

// OnMessageDelete registers a handler for message delete events.
func (r *Router) OnMessageDelete(handler MessageDeleteHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageDelHandlers = append(r.messageDelHandlers, handler)
}

// OnReaction registers a handler for reaction add/remove events.
func (r *Router) OnReaction(handler ReactionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reactionHandlers = append(r.reactionHandlers, handler)
}

// OnChatUpdate registers a handler for chat metadata update events.
func (r *Router) OnChatUpdate(handler ChatUpdateHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chatUpdateHandlers = append(r.chatUpdateHandlers, handler)
}

// OnPresence registers a handler for user online/offline presence events.
func (r *Router) OnPresence(handler PresenceHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.presenceHandlers = append(r.presenceHandlers, handler)
}

// OnTyping registers a handler for user typing indicator events.
func (r *Router) OnTyping(handler TypingHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.typingHandlers = append(r.typingHandlers, handler)
}

// OnDisconnect registers a handler called when the client disconnects.
func (r *Router) OnDisconnect(handler DisconnectHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disconnectHandlers = append(r.disconnectHandlers, handler)
}

// OnStart registers a handler called when client is connected and ready.
func (r *Router) OnStart(handler StartHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.startHandlers = append(r.startHandlers, handler)
}

// OnEvent registers a generic raw event handler for unrecognized event types.
func (r *Router) OnEvent(handler EventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventHandlers = append(r.eventHandlers, handler)
}

// DispatchMessage routes a message event to all handlers matching predicates.
func (r *Router) DispatchMessage(ctx context.Context, msg *types.Message) {
	r.mu.RLock()
	handlers := make([]filteredHandler, len(r.filteredHandlers))
	copy(handlers, r.filteredHandlers)
	r.mu.RUnlock()

	for _, fh := range handlers {
		match := true
		for _, pred := range fh.predicates {
			if !pred(msg) {
				match = false
				break
			}
		}
		if match {
			go func(h MessageHandler) {
				_ = h(ctx, msg)
			}(fh.handler)
		}
	}
}

// DispatchMessageEdit routes an edit event to all registered handlers.
func (r *Router) DispatchMessageEdit(ctx context.Context, msg *types.Message) {
	r.mu.RLock()
	handlers := make([]MessageEditHandler, len(r.messageEditHandlers))
	copy(handlers, r.messageEditHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler MessageEditHandler) {
			_ = handler(ctx, msg)
		}(h)
	}
}

// DispatchMessageDelete routes a delete event to all registered handlers.
func (r *Router) DispatchMessageDelete(ctx context.Context, chatID, msgID int64) {
	r.mu.RLock()
	handlers := make([]MessageDeleteHandler, len(r.messageDelHandlers))
	copy(handlers, r.messageDelHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler MessageDeleteHandler) {
			_ = handler(ctx, chatID, msgID)
		}(h)
	}
}

// DispatchReaction routes a reaction event to all registered handlers.
func (r *Router) DispatchReaction(ctx context.Context, ev *types.ReactionEvent) {
	r.mu.RLock()
	handlers := make([]ReactionHandler, len(r.reactionHandlers))
	copy(handlers, r.reactionHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler ReactionHandler) {
			_ = handler(ctx, ev)
		}(h)
	}
}

// DispatchChatUpdate routes a chat update event to all registered handlers.
func (r *Router) DispatchChatUpdate(ctx context.Context, chat *types.Chat) {
	r.mu.RLock()
	handlers := make([]ChatUpdateHandler, len(r.chatUpdateHandlers))
	copy(handlers, r.chatUpdateHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler ChatUpdateHandler) {
			_ = handler(ctx, chat)
		}(h)
	}
}

// DispatchPresence routes a presence event to all registered handlers.
func (r *Router) DispatchPresence(ctx context.Context, ev *types.PresenceEvent) {
	r.mu.RLock()
	handlers := make([]PresenceHandler, len(r.presenceHandlers))
	copy(handlers, r.presenceHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler PresenceHandler) {
			_ = handler(ctx, ev)
		}(h)
	}
}

// DispatchTyping routes a typing event to all registered handlers.
func (r *Router) DispatchTyping(ctx context.Context, ev *types.TypingEvent) {
	r.mu.RLock()
	handlers := make([]TypingHandler, len(r.typingHandlers))
	copy(handlers, r.typingHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler TypingHandler) {
			_ = handler(ctx, ev)
		}(h)
	}
}

// DispatchDisconnect calls all disconnect handlers synchronously (called at shutdown).
func (r *Router) DispatchDisconnect(ctx context.Context, err error) {
	r.mu.RLock()
	handlers := make([]DisconnectHandler, len(r.disconnectHandlers))
	copy(handlers, r.disconnectHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		h(ctx, err)
	}
}

// DispatchStart routes the start event to all registered start handlers.
func (r *Router) DispatchStart(ctx context.Context) {
	r.mu.RLock()
	handlers := make([]StartHandler, len(r.startHandlers))
	copy(handlers, r.startHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler StartHandler) {
			_ = handler(ctx)
		}(h)
	}
}

// DispatchEvent routes a raw unrecognized event to all registered raw event handlers.
func (r *Router) DispatchEvent(ctx context.Context, event *types.RawEvent) {
	r.mu.RLock()
	handlers := make([]EventHandler, len(r.eventHandlers))
	copy(handlers, r.eventHandlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler EventHandler) {
			_ = handler(ctx, event)
		}(h)
	}
}
