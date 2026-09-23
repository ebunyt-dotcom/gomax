package types

// EventType represents notification push event types from server.
type EventType string

const (
	EventMessageNew    EventType = "MESSAGE_NEW"
	EventMessageUpdate EventType = "MESSAGE_UPDATE"
	EventMessageDelete EventType = "MESSAGE_DELETE"
	EventMessageRead   EventType = "MESSAGE_READ"
	EventTyping        EventType = "TYPING"
	EventPresence      EventType = "PRESENCE"
	EventReaction      EventType = "REACTION_UPDATE"
	EventChatUpdate    EventType = "CHAT_UPDATE"
	EventUserUpdate    EventType = "USER_UPDATE"
	EventVideoReady    EventType = "VIDEO_READY"
	EventFileReady     EventType = "FILE_READY"
	EventVoiceReady    EventType = "VOICE_READY"
	EventRaw           EventType = "RAW"
)

// MessageReadEvent describes a server read marker notification.
type MessageReadEvent struct {
	SetAsUnread bool  `json:"setAsUnread" msgpack:"setAsUnread"`
	ChatID      int64 `json:"chatId" msgpack:"chatId"`
	UserID      int64 `json:"userId" msgpack:"userId"`
	Mark        int64 `json:"mark" msgpack:"mark"`
	// MessageID is retained for compatibility with older GoMax handlers.
	MessageID int64 `json:"messageId,omitempty" msgpack:"messageId,omitempty"`
}

// MessageDeleteEvent normalizes both mobile bulk-delete and WebSocket
// single-message delete notifications.
type MessageDeleteEvent struct {
	MessageIDs []int64  `json:"messageIds" msgpack:"messageIds"`
	ChatID     int64    `json:"chatId" msgpack:"chatId"`
	Chat       *Chat    `json:"chat,omitempty" msgpack:"chat,omitempty"`
	Message    *Message `json:"message,omitempty" msgpack:"message,omitempty"`
	TTL        bool     `json:"ttl,omitempty" msgpack:"ttl,omitempty"`
}

// UserUpdateEvent describes a changed contact/profile payload.
type UserUpdateEvent struct {
	User User `json:"user" msgpack:"user"`
}

// VideoUploadSignal, FileUploadSignal and VoiceUploadSignal are emitted by
// NOTIF_ATTACH after media processing has completed.
type VideoUploadSignal struct {
	VideoID int64 `json:"video_id" msgpack:"video_id"`
}
type FileUploadSignal struct {
	FileID int64 `json:"file_id" msgpack:"file_id"`
}
type VoiceUploadSignal struct {
	AudioID int64 `json:"audio_id" msgpack:"audio_id"`
}

// RawEvent represents a low-level incoming event frame.
type RawEvent struct {
	Type    string                 `json:"type" msgpack:"type"`
	Opcode  uint16                 `json:"opcode" msgpack:"opcode"`
	Payload map[string]interface{} `json:"payload" msgpack:"payload"`
}
