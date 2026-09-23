package dispatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

func TestHandlerErrorsReachOnError(t *testing.T) {
	router := NewRouter()
	want := errors.New("handler failed")
	got := make(chan error, 1)
	router.OnError(func(_ context.Context, err error) { got <- err })
	router.OnMessage(func(context.Context, *types.Message) error { return want })
	router.DispatchMessage(context.Background(), &types.Message{})
	select {
	case err := <-got:
		if !errors.Is(err, want) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("error handler was not called")
	}
}

func TestIncludeKeepsLiveRouterTree(t *testing.T) {
	child, parent := NewRouter(), NewRouter()
	called := make(chan struct{}, 1)
	parent.Include(child)
	child.OnMessage(func(context.Context, *types.Message) error { called <- struct{}{}; return nil })
	parent.DispatchMessage(context.Background(), &types.Message{})
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("included handler was not called")
	}
}

func TestErrorScopesAcrossRouterTree(t *testing.T) {
	child, parent := NewRouter(), NewRouter()
	parent.Include(child)
	want := errors.New("child failed")
	global := make(chan error, 1)
	parentLocal := make(chan error, 1)
	childLocal := make(chan error, 1)
	parent.OnError(func(_ context.Context, err error) { global <- err })
	parent.OnErrorScoped(ErrorScopeLocal, func(_ context.Context, err error) { parentLocal <- err })
	child.OnErrorScoped(ErrorScopeLocal, func(_ context.Context, err error) { childLocal <- err })
	child.OnMessage(func(context.Context, *types.Message) error { return want })

	parent.DispatchMessage(context.Background(), &types.Message{})
	for name, ch := range map[string]<-chan error{"global": global, "child local": childLocal} {
		select {
		case err := <-ch:
			if !errors.Is(err, want) {
				t.Fatalf("%s error=%v", name, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s error handler was not called", name)
		}
	}
	select {
	case err := <-parentLocal:
		t.Fatalf("parent local handler unexpectedly received %v", err)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestTypedEventFilters(t *testing.T) {
	router := NewRouter()
	called := make(chan int64, 1)
	router.OnChatUpdate(func(_ context.Context, chat *types.Chat) error {
		called <- chat.ID
		return nil
	}, func(chat *types.Chat) bool { return chat.ID == 2 })
	router.DispatchChatUpdate(context.Background(), &types.Chat{ID: 1})
	router.DispatchChatUpdate(context.Background(), &types.Chat{ID: 2})
	select {
	case id := <-called:
		if id != 2 {
			t.Fatalf("filtered handler received chat %d", id)
		}
	case <-time.After(time.Second):
		t.Fatal("matching filtered handler was not called")
	}
}
