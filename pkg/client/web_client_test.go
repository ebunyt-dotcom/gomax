package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/gorilla/websocket"
)

func TestDefaultWebUserAgentDoesNotMutateCustomUserAgent(t *testing.T) {
	custom := map[string]interface{}{
		"deviceType":      "WEB",
		"locale":          "en",
		"deviceLocale":    "en-US",
		"osVersion":       "Linux",
		"deviceName":      "Firefox",
		"headerUserAgent": "custom-agent",
		"appVersion":      "custom-version",
		"screen":          "100x100 1.0x",
		"timezone":        "UTC",
		"buildNumber":     123,
	}
	cfg := &Config{UserAgent: custom}

	got := defaultWebUserAgent(cfg)
	if got["headerUserAgent"] != "custom-agent" {
		t.Fatalf("custom header user agent was not preserved: %#v", got)
	}
	if got["deviceName"] != "Firefox" || got["appVersion"] != "custom-version" {
		t.Fatalf("custom web user agent fields were not preserved: %#v", got)
	}
	if _, ok := got["buildNumber"]; ok {
		t.Fatalf("mobile-only buildNumber leaked into web payload: %#v", got)
	}
	if custom["buildNumber"] != 123 {
		t.Fatalf("caller-provided user agent was mutated: %#v", custom)
	}
}

func TestWebClientSessionInitUsesBinaryMessagePackFrames(t *testing.T) {
	type observation struct {
		messageType int
		data        []byte
		err         error
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	observed := make(chan observation, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			observed <- observation{err: err}
			return
		}
		defer ws.Close()

		messageType, data, err := ws.ReadMessage()
		observed <- observation{messageType: messageType, data: data, err: err}
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = "ws" + strings.TrimPrefix(server.URL, "http")
	cfg.PersistSession = false
	cfg.Reconnect = false
	cfg.Token = "test-token"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = NewWebClient(cfg).Start(ctx)

	var got observation
	select {
	case got = <-observed:
	case <-ctx.Done():
		t.Fatalf("server did not receive SESSION_INIT: %v", ctx.Err())
	}
	if got.err != nil {
		t.Fatalf("server failed to read SESSION_INIT: %v", got.err)
	}
	if got.messageType != websocket.BinaryMessage {
		t.Fatalf("SESSION_INIT must use a binary WebSocket message, got type %d", got.messageType)
	}

	wsProtocol, err := protocol.NewWsProtocol(true)
	if err != nil {
		t.Fatalf("create binary WebSocket protocol: %v", err)
	}
	frame, err := wsProtocol.Decode(got.data)
	if err != nil {
		t.Fatalf("decode binary SESSION_INIT: %v", err)
	}
	if frame.Opcode != protocol.OpSessionInit || frame.Cmd != protocol.CmdRequest {
		t.Fatalf("unexpected SESSION_INIT frame: cmd=%v opcode=%v", frame.Cmd, frame.Opcode)
	}
	if _, ok := frame.Payload["deviceId"].(string); !ok {
		t.Fatalf("SESSION_INIT deviceId missing from payload: %#v", frame.Payload)
	}
	if _, ok := frame.Payload["userAgent"].(map[string]interface{}); !ok {
		t.Fatalf("SESSION_INIT userAgent missing from payload: %#v", frame.Payload)
	}
}
