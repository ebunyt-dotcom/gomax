package dispatch

import (
	"context"
	"sync"

	"gomax/pkg/types"
)

// MessageHandler handles incoming messages.
type MessageHandler func(ctx context.Context, msg *types.Message) error

// StartHandler handles client ready event.
type StartHandler func(ctx context.Context) error

// EventHandler handles raw events.
type EventHandler func(ctx context.Context, event *types.RawEvent) error

// MessagePredicate is a filter function that determines if a message should be dispatched.
type MessagePredicate func(msg *types.Message) bool

type filteredHandler struct {
	predicates []MessagePredicate
	handler    MessageHandler
}

// Router stores registered event handlers and supports predicate filters.
type Router struct {
	mu               sync.RWMutex
	filteredHandlers []filteredHandler
	startHandlers    []StartHandler
	eventHandlers    []EventHandler
}

// NewRouter creates a new event router.
func NewRouter() *Router {
	return &Router{}
}

// OnMessage registers an unconditional message handler.
func (r *Router) OnMessage(handler MessageHandler, filters ...MessagePredicate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filteredHandlers = append(r.filteredHandlers, filteredHandler{
		predicates: filters,
		handler:    handler,
	})
}

// OnStart registers a handler called when client is connected and ready.
func (r *Router) OnStart(handler StartHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.startHandlers = append(r.startHandlers, handler)
}

// OnEvent registers a generic event handler.
func (r *Router) OnEvent(handler EventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventHandlers = append(r.eventHandlers, handler)
}

// DispatchMessage routes a message event to handlers matching all filter predicates.
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

// DispatchEvent routes a raw event.
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
