package types

import "time"

// ChatType represents the kind of chat dialog.
type ChatType string

const (
	ChatTypeDialog  ChatType = "DIALOG"
	ChatTypeChat    ChatType = "CHAT"
	ChatTypeChannel ChatType = "CHANNEL"
)

// Chat represents a chat conversation, group or channel in Max.
type Chat struct {
	ID           int64     `json:"id" msgpack:"id"`
	Type         ChatType  `json:"type" msgpack:"type"`
	Title        string    `json:"title,omitempty" msgpack:"title,omitempty"`
	Description  string    `json:"description,omitempty" msgpack:"description,omitempty"`
	Icon         string    `json:"icon,omitempty" msgpack:"icon,omitempty"`
	MembersCount int       `json:"members_count,omitempty" msgpack:"members_count,omitempty"`
	OwnerID      int64     `json:"owner_id,omitempty" msgpack:"owner_id,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty" msgpack:"created_at,omitempty"`
	PinnedMsgID  int64     `json:"pinned_message_id,omitempty" msgpack:"pinned_message_id,omitempty"`
	IsChannel    bool      `json:"is_channel,omitempty" msgpack:"is_channel,omitempty"`
	IsPublic     bool      `json:"is_public,omitempty" msgpack:"is_public,omitempty"`
	InviteLink   string    `json:"invite_link,omitempty" msgpack:"invite_link,omitempty"`
}

// User represents a user contact or profile.
type User struct {
	ID         int64  `json:"id" msgpack:"id"`
	Phone      string `json:"phone,omitempty" msgpack:"phone,omitempty"`
	FirstName  string `json:"first_name,omitempty" msgpack:"first_name,omitempty"`
	LastName   string `json:"last_name,omitempty" msgpack:"last_name,omitempty"`
	Nickname   string `json:"nickname,omitempty" msgpack:"nickname,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty" msgpack:"avatar_url,omitempty"`
	Bio        string `json:"bio,omitempty" msgpack:"bio,omitempty"`
	IsBot      bool   `json:"is_bot,omitempty" msgpack:"is_bot,omitempty"`
	IsContact  bool   `json:"is_contact,omitempty" msgpack:"is_contact,omitempty"`
	IsMutual   bool   `json:"is_mutual,omitempty" msgpack:"is_mutual,omitempty"`
	IsVerified bool   `json:"is_verified,omitempty" msgpack:"is_verified,omitempty"`
}

// Member represents a chat/group participant.
type Member struct {
	UserID    int64  `json:"user_id" msgpack:"user_id"`
	Role      string `json:"role" msgpack:"role"` // "ADMIN", "MEMBER", "OWNER"
	JoinedAt  int64  `json:"joined_at,omitempty" msgpack:"joined_at,omitempty"`
	InvitedBy int64  `json:"invited_by,omitempty" msgpack:"invited_by,omitempty"`
}

// ReactionInfo represents reactions attached to a message.
type ReactionInfo struct {
	Reaction string `json:"reaction" msgpack:"reaction"`
	Count    int    `json:"count" msgpack:"count"`
	Self     bool   `json:"self,omitempty" msgpack:"self,omitempty"`
}

// AttachmentType defines the kind of media attachment.
type AttachmentType string

const (
	AttachmentPhoto AttachmentType = "PHOTO"
	AttachmentVideo AttachmentType = "VIDEO"
	AttachmentAudio AttachmentType = "AUDIO"
	AttachmentFile  AttachmentType = "FILE"
	// Voice and video-note are Go convenience types; attachmentPayload maps
	// them to PyMax's AUDIO and VIDEO wire types respectively.
	AttachmentVoice     AttachmentType = "VOICE"
	AttachmentVideoNote AttachmentType = "VIDEO_NOTE"
	AttachmentPoll      AttachmentType = "POLL"
	AttachmentSticker   AttachmentType = "STICKER"
)

// Attachment represents an uploaded or received media attachment.
type Attachment struct {
	Type     AttachmentType `json:"type" msgpack:"type"`
	ID       string         `json:"id,omitempty" msgpack:"id,omitempty"`
	URL      string         `json:"url,omitempty" msgpack:"url,omitempty"`
	FileName string         `json:"file_name,omitempty" msgpack:"file_name,omitempty"`
	FileSize int64          `json:"file_size,omitempty" msgpack:"file_size,omitempty"`
	Duration int            `json:"duration,omitempty" msgpack:"duration,omitempty"`
	Width    int            `json:"width,omitempty" msgpack:"width,omitempty"`
	Height   int            `json:"height,omitempty" msgpack:"height,omitempty"`
	Token    string         `json:"token,omitempty" msgpack:"token,omitempty"`
}

// PollOption represents a single choice in a poll.
type PollOption struct {
	ID    int    `json:"id" msgpack:"id"`
	Text  string `json:"text" msgpack:"text"`
	Votes int    `json:"votes,omitempty" msgpack:"votes,omitempty"`
}

// Poll represents a poll object in a message.
type Poll struct {
	ID           string       `json:"id" msgpack:"id"`
	Question     string       `json:"question" msgpack:"question"`
	Options      []PollOption `json:"options" msgpack:"options"`
	Multiple     bool         `json:"multiple,omitempty" msgpack:"multiple,omitempty"`
	Anonymous    bool         `json:"anonymous,omitempty" msgpack:"anonymous,omitempty"`
	Closed       bool         `json:"closed,omitempty" msgpack:"closed,omitempty"`
	SelectedOpts []int        `json:"selected_options,omitempty" msgpack:"selected_options,omitempty"`
}

// Message represents a text or media message in Max.
type Message struct {
	ID           int64          `json:"id" msgpack:"id"`
	CID          int64          `json:"cid,omitempty" msgpack:"cid,omitempty"` // Client message sequence ID
	ChatID       int64          `json:"chat_id" msgpack:"chat_id"`
	SenderID     int64          `json:"sender_id" msgpack:"sender_id"`
	Text         string         `json:"text,omitempty" msgpack:"text,omitempty"`
	Time         int64          `json:"time" msgpack:"time"`
	EditedAt     int64          `json:"edited_at,omitempty" msgpack:"edited_at,omitempty"`
	ReplyToMsgID int64          `json:"reply_to,omitempty" msgpack:"reply_to,omitempty"`
	Attachments  []Attachment   `json:"attachments,omitempty" msgpack:"attachments,omitempty"`
	Reactions    []ReactionInfo `json:"reactions,omitempty" msgpack:"reactions,omitempty"`
	IsOutgoing   bool           `json:"is_outgoing,omitempty" msgpack:"is_outgoing,omitempty"`
	IsPinned     bool           `json:"is_pinned,omitempty" msgpack:"is_pinned,omitempty"`
	IsDeleted    bool           `json:"is_deleted,omitempty" msgpack:"is_deleted,omitempty"`
}

// ReactionEvent represents a reaction add/remove notification push from the server.
type ReactionEvent struct {
	ChatID    int64  `json:"chat_id" msgpack:"chat_id"`
	MessageID int64  `json:"message_id" msgpack:"message_id"`
	UserID    int64  `json:"user_id" msgpack:"user_id"`
	Reaction  string `json:"reaction" msgpack:"reaction"`
	Removed   bool   `json:"removed" msgpack:"removed"`
}

// PresenceEvent represents a user online/offline status change notification.
type PresenceEvent struct {
	UserID int64 `json:"user_id" msgpack:"user_id"`
	Online bool  `json:"online" msgpack:"online"`
}

// Folder represents a chat filter/folder used to organize dialogs.
type Folder struct {
	ID      string  `json:"id" msgpack:"id"`
	Title   string  `json:"title" msgpack:"title"`
	Include []int64 `json:"include" msgpack:"include"`
}

// FolderList represents a paginated list of chat folders with a sync marker.
type FolderList struct {
	Folders []Folder `json:"folders" msgpack:"folders"`
	Sync    int64    `json:"sync" msgpack:"sync"`
}

// TypingEvent represents a user typing/action indicator in a chat.
type TypingEvent struct {
	ChatID int64 `json:"chat_id" msgpack:"chat_id"`
	UserID int64 `json:"user_id" msgpack:"user_id"`
}
