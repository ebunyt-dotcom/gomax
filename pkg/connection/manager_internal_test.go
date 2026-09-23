package connection

import (
	"testing"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
)

func TestHandleInboundDoesNotDispatchResponsesAsEvents(t *testing.T) {
	events := make(chan *protocol.InboundFrame, 1)
	m := &ConnectionManager{pending: NewPendingTracker(), onEvent: func(frame *protocol.InboundFrame) { events <- frame }}
	pending := m.pending.Create(7)
	response := &protocol.InboundFrame{Cmd: protocol.CmdResponse, Seq: 7, Opcode: protocol.OpMsgSend}
	m.handleInbound(response)
	select {
	case got := <-pending:
		if got != response {
			t.Fatal("wrong response")
		}
	case <-time.After(time.Second):
		t.Fatal("pending response not resolved")
	}
	select {
	case <-events:
		t.Fatal("RPC response was dispatched as event")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestHandleInboundDispatchesRequestPush(t *testing.T) {
	events := make(chan *protocol.InboundFrame, 1)
	m := &ConnectionManager{pending: NewPendingTracker(), onEvent: func(frame *protocol.InboundFrame) { events <- frame }}
	push := &protocol.InboundFrame{Cmd: protocol.CmdRequest, Opcode: protocol.OpNotifMessage}
	m.handleInbound(push)
	select {
	case got := <-events:
		if got != push {
			t.Fatal("wrong push")
		}
	case <-time.After(time.Second):
		t.Fatal("push was not dispatched")
	}
}
