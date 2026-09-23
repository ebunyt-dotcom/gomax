package messages

import (
	"context"
	"testing"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

type recordingInvoker struct {
	op      protocol.Opcode
	payload map[string]interface{}
	result  map[string]interface{}
}

func (m *recordingInvoker) Invoke(_ context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	m.op = op
	m.payload, _ = payload.(map[string]interface{})
	if m.result != nil {
		return m.result, nil
	}
	return map[string]interface{}{}, nil
}

func TestAttachmentPayloadMatchesPyMaxTokenRules(t *testing.T) {
	voice := attachmentPayload(types.Attachment{Type: types.AttachmentVoice, ID: "12", Token: "voice-token", Duration: 3})
	if _, exists := voice["audioId"]; exists || voice["token"] != "voice-token" || voice["_type"] != types.AttachmentAudio {
		t.Fatalf("voice=%#v", voice)
	}
	note := attachmentPayload(types.Attachment{Type: types.AttachmentVideoNote, ID: "13", Token: "note-token", ThumbHash: []byte{1}})
	if _, exists := note["videoId"]; exists || note["videoType"] != 1 || note["thumbhash"] == nil {
		t.Fatalf("video note=%#v", note)
	}
	video := attachmentPayload(types.Attachment{Type: types.AttachmentVideo, ID: "14", Token: "video-token"})
	if video["videoId"] != int64(14) || video["videoType"] != 0 {
		t.Fatalf("video=%#v", video)
	}
}

func TestSendMessageParsesDirectResponseAndMarkdown(t *testing.T) {
	inv := &recordingInvoker{result: map[string]interface{}{"id": int64(9), "chatId": int64(1), "text": "hello bold", "sender": int64(4)}}
	message, err := NewMessageService(inv).SendMessage(context.Background(), 1, "hello **bold**", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	wire := inv.payload["message"].(map[string]interface{})
	if message.ID != 9 || message.SenderID != 4 || wire["text"] != "hello bold" {
		t.Fatalf("message=%#v wire=%#v", message, wire)
	}
}

func TestReturnedMessageHasBoundConvenienceMethods(t *testing.T) {
	inv := &recordingInvoker{result: map[string]interface{}{"id": int64(9), "chatId": int64(1), "text": "hello"}}
	message, err := NewMessageService(inv).SendMessage(context.Background(), 1, "hello", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	inv.result = map[string]interface{}{}
	if err := message.Delete(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if inv.op != protocol.OpMsgDelete || inv.payload["forMe"] != true {
		t.Fatalf("op=%v payload=%#v", inv.op, inv.payload)
	}
}

func TestAddReactionIncludesReactionType(t *testing.T) {
	inv := &recordingInvoker{}
	if err := NewMessageService(inv).AddReaction(context.Background(), 1, 2, "heart"); err != nil {
		t.Fatal(err)
	}
	reaction, ok := inv.payload["reaction"].(map[string]interface{})
	if !ok || reaction["reactionType"] != "EMOJI" || reaction["id"] != "heart" {
		t.Fatalf("reaction=%#v", reaction)
	}
}

func TestSendMessageOptionsPreserveSchedulingAndNotify(t *testing.T) {
	inv := &recordingInvoker{}
	service := NewMessageService(inv)
	when := time.Unix(1700000000, 0)
	_, err := service.SendMessageWithOptions(context.Background(), 1, "hello", nil, SendOptions{Notify: true, SendAt: when, NotifySender: true})
	if err != nil {
		t.Fatal(err)
	}
	if inv.payload["notify"] != true {
		t.Fatalf("notify=%#v", inv.payload["notify"])
	}
	message := inv.payload["message"].(map[string]interface{})
	delayed := message["delayedAttributes"].(map[string]interface{})
	if delayed["timeToFire"] != when.UnixMilli() || delayed["notifySender"] != true {
		t.Fatalf("delayed=%#v", delayed)
	}
}
