package types

// EventType represents notification push event types from server.
type EventType string

const (
	EventMessageNew    EventType = "MESSAGE_NEW"
	EventMessageUpdate EventType = "MESSAGE_UPDATE"
	EventMessageDelete EventType = "MESSAGE_DELETE"
	EventChatUpdate    EventType = "CHAT_UPDATE"
	EventUserPresence  EventType = "USER_PRESENCE"
)

// RawEvent represents a low-level incoming event frame.
type RawEvent struct {
	Type    string                 `json:"type" msgpack:"type"`
	Opcode  uint16                 `json:"opcode" msgpack:"opcode"`
	Payload map[string]interface{} `json:"payload" msgpack:"payload"`
}
