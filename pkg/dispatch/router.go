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

// MessageDeleteEventHandler handles the complete normalized delete event.
type MessageDeleteEventHandler func(ctx context.Context, ev *types.MessageDeleteEvent) error

// MessageReadHandler handles server read-marker notifications.
type MessageReadHandler func(ctx context.Context, ev *types.MessageReadEvent) error

// UserUpdateHandler handles contact/profile updates.
type UserUpdateHandler func(ctx context.Context, ev *types.UserUpdateEvent) error

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

// ErrorHandler receives errors returned by asynchronous event handlers.
type ErrorHandler func(ctx context.Context, err error)

// ErrorScope controls which handler failures are visible to an error handler.
type ErrorScope uint8

const (
	// ErrorScopeGlobal receives failures from this router and its descendants.
	ErrorScopeGlobal ErrorScope = iota
	// ErrorScopeLocal receives failures only from handlers owned by this router.
	ErrorScopeLocal
)

// Predicate filters an event before its handler is invoked.
type Predicate[T any] func(event *T) bool

type MessagePredicate = Predicate[types.Message]
type MessageDeletePredicate = Predicate[types.MessageDeleteEvent]
type MessageReadPredicate = Predicate[types.MessageReadEvent]
type UserUpdatePredicate = Predicate[types.UserUpdateEvent]
type ReactionPredicate = Predicate[types.ReactionEvent]
type ChatUpdatePredicate = Predicate[types.Chat]
type PresencePredicate = Predicate[types.PresenceEvent]
type TypingPredicate = Predicate[types.TypingEvent]
type EventPredicate = Predicate[types.RawEvent]

type filteredEntry[T any] struct {
	predicates []Predicate[T]
	handler    func(context.Context, *T) error
}

type errorEntry struct {
	scope   ErrorScope
	handler ErrorHandler
}

// Router stores registered event handlers and supports predicate filters.
type Router struct {
	mu sync.RWMutex

	messageHandlers         []filteredEntry[types.Message]
	messageEditHandlers     []filteredEntry[types.Message]
	messageDelHandlers      []MessageDeleteHandler
	messageDelEventHandlers []filteredEntry[types.MessageDeleteEvent]
	messageReadHandlers     []filteredEntry[types.MessageReadEvent]
	userUpdateHandlers      []filteredEntry[types.UserUpdateEvent]
	reactionHandlers        []filteredEntry[types.ReactionEvent]
	chatUpdateHandlers      []filteredEntry[types.Chat]
	presenceHandlers        []filteredEntry[types.PresenceEvent]
	typingHandlers          []filteredEntry[types.TypingEvent]
	disconnectHandlers      []DisconnectHandler
	startHandlers           []StartHandler
	eventHandlers           []filteredEntry[types.RawEvent]
	errorHandlers           []errorEntry
	children                []*Router
}

// NewRouter creates a new event router.
func NewRouter() *Router {
	return &Router{}
}

// OnMessage registers a message handler with optional filter predicates.
func (r *Router) OnMessage(handler MessageHandler, filters ...MessagePredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageHandlers = append(r.messageHandlers, filteredEntry[types.Message]{
		predicates: filters,
		handler:    func(ctx context.Context, event *types.Message) error { return handler(ctx, event) },
	})
}

// OnMessageEdit registers a handler for message edit events.
func (r *Router) OnMessageEdit(handler MessageEditHandler, filters ...MessagePredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageEditHandlers = append(r.messageEditHandlers, filteredEntry[types.Message]{predicates: filters, handler: func(ctx context.Context, event *types.Message) error { return handler(ctx, event) }})
}

// OnMessageDelete registers a handler for message delete events.
func (r *Router) OnMessageDelete(handler MessageDeleteHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageDelHandlers = append(r.messageDelHandlers, handler)
}

// OnMessageDeleteEvent registers a complete delete-event handler.
func (r *Router) OnMessageDeleteEvent(handler MessageDeleteEventHandler, filters ...MessageDeletePredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageDelEventHandlers = append(r.messageDelEventHandlers, filteredEntry[types.MessageDeleteEvent]{predicates: filters, handler: func(ctx context.Context, event *types.MessageDeleteEvent) error { return handler(ctx, event) }})
}

// OnMessageRead registers a read-marker handler.
func (r *Router) OnMessageRead(handler MessageReadHandler, filters ...MessageReadPredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageReadHandlers = append(r.messageReadHandlers, filteredEntry[types.MessageReadEvent]{predicates: filters, handler: func(ctx context.Context, event *types.MessageReadEvent) error { return handler(ctx, event) }})
}

// OnUserUpdate registers a contact/profile update handler.
func (r *Router) OnUserUpdate(handler UserUpdateHandler, filters ...UserUpdatePredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userUpdateHandlers = append(r.userUpdateHandlers, filteredEntry[types.UserUpdateEvent]{predicates: filters, handler: func(ctx context.Context, event *types.UserUpdateEvent) error { return handler(ctx, event) }})
}

// OnReaction registers a handler for reaction add/remove events.
func (r *Router) OnReaction(handler ReactionHandler, filters ...ReactionPredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reactionHandlers = append(r.reactionHandlers, filteredEntry[types.ReactionEvent]{predicates: filters, handler: func(ctx context.Context, event *types.ReactionEvent) error { return handler(ctx, event) }})
}

// OnChatUpdate registers a handler for chat metadata update events.
func (r *Router) OnChatUpdate(handler ChatUpdateHandler, filters ...ChatUpdatePredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chatUpdateHandlers = append(r.chatUpdateHandlers, filteredEntry[types.Chat]{predicates: filters, handler: func(ctx context.Context, event *types.Chat) error { return handler(ctx, event) }})
}

// OnPresence registers a handler for user online/offline presence events.
func (r *Router) OnPresence(handler PresenceHandler, filters ...PresencePredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.presenceHandlers = append(r.presenceHandlers, filteredEntry[types.PresenceEvent]{predicates: filters, handler: func(ctx context.Context, event *types.PresenceEvent) error { return handler(ctx, event) }})
}

// OnTyping registers a handler for user typing indicator events.
func (r *Router) OnTyping(handler TypingHandler, filters ...TypingPredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.typingHandlers = append(r.typingHandlers, filteredEntry[types.TypingEvent]{predicates: filters, handler: func(ctx context.Context, event *types.TypingEvent) error { return handler(ctx, event) }})
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
func (r *Router) OnEvent(handler EventHandler, filters ...EventPredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventHandlers = append(r.eventHandlers, filteredEntry[types.RawEvent]{predicates: filters, handler: func(ctx context.Context, event *types.RawEvent) error { return handler(ctx, event) }})
}

func (r *Router) OnError(handler ErrorHandler) {
	r.OnErrorScoped(ErrorScopeGlobal, handler)
}

// OnErrorScoped registers an error handler with PyMax-compatible visibility.
func (r *Router) OnErrorScoped(scope ErrorScope, handler ErrorHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errorHandlers = append(r.errorHandlers, errorEntry{scope: scope, handler: handler})
}

// Include attaches another router as a live child. Handlers registered on the
// child after inclusion are visible immediately, matching PyMax's router tree.
func (r *Router) Include(other *Router) {
	if other == nil || other == r || other.contains(r) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.children = append(r.children, other)
}

func (r *Router) childrenSnapshot() []*Router {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]*Router(nil), r.children...)
}

func (r *Router) contains(target *Router) bool {
	if r == target {
		return true
	}
	for _, child := range r.childrenSnapshot() {
		if child.contains(target) {
			return true
		}
	}
	return false
}

func (r *Router) dispatchError(ctx context.Context, err error, failed *Router) {
	if err == nil {
		return
	}
	r.dispatchErrorTree(ctx, err, failed)
}

func (r *Router) dispatchErrorTree(ctx context.Context, err error, failed *Router) {
	r.mu.RLock()
	handlers := append([]errorEntry(nil), r.errorHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	for _, entry := range handlers {
		if entry.scope == ErrorScopeGlobal || r == failed {
			entry.handler(ctx, err)
		}
	}
	for _, child := range children {
		child.dispatchErrorTree(ctx, err, failed)
	}
}

// DispatchMessage routes a message event to all handlers matching predicates.
func (r *Router) DispatchMessage(ctx context.Context, msg *types.Message) {
	r.dispatchMessage(r, ctx, msg)
}

func (r *Router) dispatchMessage(root *Router, ctx context.Context, msg *types.Message) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.Message](nil), r.messageHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, msg, handlers)
	for _, child := range children {
		child.dispatchMessage(root, ctx, msg)
	}
}

// DispatchMessageEdit routes an edit event to all registered handlers.
func (r *Router) DispatchMessageEdit(ctx context.Context, msg *types.Message) {
	r.dispatchMessageEdit(r, ctx, msg)
}

func (r *Router) dispatchMessageEdit(root *Router, ctx context.Context, msg *types.Message) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.Message](nil), r.messageEditHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, msg, handlers)
	for _, child := range children {
		child.dispatchMessageEdit(root, ctx, msg)
	}
}

// DispatchMessageDelete routes a delete event to all registered handlers.
func (r *Router) DispatchMessageDelete(ctx context.Context, chatID, msgID int64) {
	r.DispatchMessageDeleteEvent(ctx, &types.MessageDeleteEvent{ChatID: chatID, MessageIDs: []int64{msgID}})
}

// DispatchMessageDeleteEvent routes a normalized delete event and also calls
// legacy per-message handlers once for each ID.
func (r *Router) DispatchMessageDeleteEvent(ctx context.Context, event *types.MessageDeleteEvent) {
	r.dispatchMessageDeleteEvent(r, ctx, event)
}

func (r *Router) dispatchMessageDeleteEvent(root *Router, ctx context.Context, event *types.MessageDeleteEvent) {
	r.mu.RLock()
	handlers := make([]MessageDeleteHandler, len(r.messageDelHandlers))
	copy(handlers, r.messageDelHandlers)
	eventHandlers := append([]filteredEntry[types.MessageDeleteEvent](nil), r.messageDelEventHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()

	dispatchFiltered(root, r, ctx, event, eventHandlers)
	for _, messageID := range event.MessageIDs {
		for _, handler := range handlers {
			go func(h MessageDeleteHandler, id int64) {
				root.dispatchError(ctx, h(ctx, event.ChatID, id), r)
			}(handler, messageID)
		}
	}
	for _, child := range children {
		child.dispatchMessageDeleteEvent(root, ctx, event)
	}
}

// DispatchMessageRead routes a read-marker event.
func (r *Router) DispatchMessageRead(ctx context.Context, ev *types.MessageReadEvent) {
	r.dispatchMessageRead(r, ctx, ev)
}

func (r *Router) dispatchMessageRead(root *Router, ctx context.Context, ev *types.MessageReadEvent) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.MessageReadEvent](nil), r.messageReadHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, ev, handlers)
	for _, child := range children {
		child.dispatchMessageRead(root, ctx, ev)
	}
}

// DispatchUserUpdate routes a contact/profile event.
func (r *Router) DispatchUserUpdate(ctx context.Context, ev *types.UserUpdateEvent) {
	r.dispatchUserUpdate(r, ctx, ev)
}

func (r *Router) dispatchUserUpdate(root *Router, ctx context.Context, ev *types.UserUpdateEvent) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.UserUpdateEvent](nil), r.userUpdateHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, ev, handlers)
	for _, child := range children {
		child.dispatchUserUpdate(root, ctx, ev)
	}
}

// DispatchReaction routes a reaction event to all registered handlers.
func (r *Router) DispatchReaction(ctx context.Context, ev *types.ReactionEvent) {
	r.dispatchReaction(r, ctx, ev)
}

func (r *Router) dispatchReaction(root *Router, ctx context.Context, ev *types.ReactionEvent) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.ReactionEvent](nil), r.reactionHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, ev, handlers)
	for _, child := range children {
		child.dispatchReaction(root, ctx, ev)
	}
}

// DispatchChatUpdate routes a chat update event to all registered handlers.
func (r *Router) DispatchChatUpdate(ctx context.Context, chat *types.Chat) {
	r.dispatchChatUpdate(r, ctx, chat)
}

func (r *Router) dispatchChatUpdate(root *Router, ctx context.Context, chat *types.Chat) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.Chat](nil), r.chatUpdateHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, chat, handlers)
	for _, child := range children {
		child.dispatchChatUpdate(root, ctx, chat)
	}
}

// DispatchPresence routes a presence event to all registered handlers.
func (r *Router) DispatchPresence(ctx context.Context, ev *types.PresenceEvent) {
	r.dispatchPresence(r, ctx, ev)
}

func (r *Router) dispatchPresence(root *Router, ctx context.Context, ev *types.PresenceEvent) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.PresenceEvent](nil), r.presenceHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, ev, handlers)
	for _, child := range children {
		child.dispatchPresence(root, ctx, ev)
	}
}

// DispatchTyping routes a typing event to all registered handlers.
func (r *Router) DispatchTyping(ctx context.Context, ev *types.TypingEvent) {
	r.dispatchTyping(r, ctx, ev)
}

func (r *Router) dispatchTyping(root *Router, ctx context.Context, ev *types.TypingEvent) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.TypingEvent](nil), r.typingHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, ev, handlers)
	for _, child := range children {
		child.dispatchTyping(root, ctx, ev)
	}
}

// DispatchDisconnect calls all disconnect handlers synchronously (called at shutdown).
func (r *Router) DispatchDisconnect(ctx context.Context, err error) {
	r.dispatchDisconnect(ctx, err)
}

func (r *Router) dispatchDisconnect(ctx context.Context, err error) {
	r.mu.RLock()
	handlers := make([]DisconnectHandler, len(r.disconnectHandlers))
	copy(handlers, r.disconnectHandlers)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()

	for _, h := range handlers {
		h(ctx, err)
	}
	for _, child := range children {
		child.dispatchDisconnect(ctx, err)
	}
}

// DispatchStart routes the start event to all registered start handlers.
func (r *Router) DispatchStart(ctx context.Context) {
	r.dispatchStart(r, ctx)
}

func (r *Router) dispatchStart(root *Router, ctx context.Context) {
	r.mu.RLock()
	handlers := make([]StartHandler, len(r.startHandlers))
	copy(handlers, r.startHandlers)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()

	for _, h := range handlers {
		go func(handler StartHandler) {
			root.dispatchError(ctx, handler(ctx), r)
		}(h)
	}
	for _, child := range children {
		child.dispatchStart(root, ctx)
	}
}

// DispatchEvent routes a raw unrecognized event to all registered raw event handlers.
func (r *Router) DispatchEvent(ctx context.Context, event *types.RawEvent) {
	r.dispatchEvent(r, ctx, event)
}

func (r *Router) dispatchEvent(root *Router, ctx context.Context, event *types.RawEvent) {
	r.mu.RLock()
	handlers := append([]filteredEntry[types.RawEvent](nil), r.eventHandlers...)
	children := append([]*Router(nil), r.children...)
	r.mu.RUnlock()
	dispatchFiltered(root, r, ctx, event, handlers)
	for _, child := range children {
		child.dispatchEvent(root, ctx, event)
	}
}

func dispatchFiltered[T any](root, owner *Router, ctx context.Context, event *T, handlers []filteredEntry[T]) {
	for _, entry := range handlers {
		matches := true
		for _, predicate := range entry.predicates {
			if !predicate(event) {
				matches = false
				break
			}
		}
		if matches {
			go func(handler func(context.Context, *T) error) {
				root.dispatchError(ctx, handler(ctx, event), owner)
			}(entry.handler)
		}
	}
}
