package telemetry

import (
	"math/rand"
	"testing"

	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

func TestPayloadBuildersMatchPyMaxShape(t *testing.T) {
	login := LoginEvent(42, 7)
	if login.Event != "login" || login.Type != "PERF" || login.UserID != 42 || login.SessionID != 7 {
		t.Fatalf("unexpected login event: %#v", login)
	}
	rng := rand.New(rand.NewSource(1))
	nav := NavigationEvent(Snapshot{UserID: 42, Chats: []types.Chat{{ID: 99, Type: types.ChatTypeChat}}}, 8, ScreenChats, ScreenChat, 123, 1, rng)
	if nav.Params["source_type"] != 2 || nav.Params["source_id"] != int64(99) {
		t.Fatalf("unexpected chat source: %#v", nav.Params)
	}
	if OpenChatEvent(42, 8, rng).Event != "open_chat_to_render" {
		t.Fatal("wrong open-chat event")
	}
}
