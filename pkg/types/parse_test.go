package types

import "testing"

func TestParseMessagePayloadKeepsRichFields(t *testing.T) {
	msg := ParseMessagePayload(map[string]any{
		"id": "42", "chatId": int32(9), "status": "EDITED", "text": "x",
		"attaches":          []any{map[string]any{"_type": "AUDIO", "audioId": uint16(3), "wave": []byte{1, 2}}},
		"delayedAttributes": map[string]any{"timeToFire": int64(7), "notifySender": true},
	})
	if msg.ID != 42 || msg.ChatID != 9 || msg.Status != MessageStatusEdited {
		t.Fatalf("message=%#v", msg)
	}
	if len(msg.Attachments) != 1 || msg.Attachments[0].ID != "3" || len(msg.Attachments[0].Wave) != 2 {
		t.Fatalf("attachments=%#v", msg.Attachments)
	}
	if msg.DelayedAttributes == nil || msg.DelayedAttributes.TimeToFire != 7 {
		t.Fatalf("delayed=%#v", msg.DelayedAttributes)
	}
}

func TestParseMessagePayloadPreservesNullableUnionFields(t *testing.T) {
	msg := ParseMessagePayload(map[string]any{
		"id": int64(1), "time": int64(2), "type": "USER",
		"prevMessageId": "opaque-id", "ttl": true, "unread": int64(3),
		"mark": int64(4), "options": int64(7),
		"elements": []any{map[string]any{"type": "STRONG", "from": 1, "length": 2}},
		"link": map[string]any{
			"type": "FORWARD", "chatId": int64(9), "chatAccessType": "PRIVATE",
			"chatIconUrl": "icon", "message": map[string]any{"id": int64(8), "time": int64(1), "type": "USER"},
		},
	})
	if msg.PrevMessageID != "opaque-id" || msg.TTL == nil || !*msg.TTL || msg.Unread == nil || *msg.Unread != 3 || msg.Mark == nil || *msg.Mark != 4 {
		t.Fatalf("nullable fields were not preserved: %#v", msg)
	}
	if msg.Options != int64(7) || len(msg.Elements) != 1 || msg.Elements[0].Type != "STRONG" {
		t.Fatalf("union/elements were not preserved: %#v", msg)
	}
	if msg.Link == nil || msg.Link.Type != LinkForward || msg.Link.ChatAccessType != AccessPrivate || msg.Link.ChatIconURL != "icon" || msg.Link.Message == nil || msg.Link.Message.ID != 8 {
		t.Fatalf("link was not parsed: %#v", msg.Link)
	}
}
