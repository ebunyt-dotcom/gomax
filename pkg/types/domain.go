package types

import "time"

// ChatType represents the kind of chat dialog.
type ChatType string

const (
	ChatTypeDialog  ChatType = "DIALOG"
	ChatTypeChat    ChatType = "CHAT"
	ChatTypeChannel ChatType = "CHANNEL"
)

// AccessType is the visibility mode of a group or channel.
type AccessType string

const (
	AccessPublic  AccessType = "PUBLIC"
	AccessPrivate AccessType = "PRIVATE"
	AccessSecret  AccessType = "SECRET"
)

// LinkType identifies a reply or forward source.
type LinkType string

const (
	LinkReply   LinkType = "REPLY"
	LinkForward LinkType = "FORWARD"
)

// Chat represents a chat conversation, group or channel in Max.
type Chat struct {
	ID                       int64                    `json:"id" msgpack:"id"`
	Type                     ChatType                 `json:"type" msgpack:"type"`
	Title                    string                   `json:"title,omitempty" msgpack:"title,omitempty"`
	Description              string                   `json:"description,omitempty" msgpack:"description,omitempty"`
	Icon                     string                   `json:"icon,omitempty" msgpack:"icon,omitempty"`
	MembersCount             int                      `json:"members_count,omitempty" msgpack:"members_count,omitempty"`
	OwnerID                  int64                    `json:"owner_id,omitempty" msgpack:"owner_id,omitempty"`
	CreatedAt                time.Time                `json:"created_at,omitempty" msgpack:"created_at,omitempty"`
	PinnedMsgID              int64                    `json:"pinned_message_id,omitempty" msgpack:"pinned_message_id,omitempty"`
	IsChannel                bool                     `json:"is_channel,omitempty" msgpack:"is_channel,omitempty"`
	IsPublic                 bool                     `json:"is_public,omitempty" msgpack:"is_public,omitempty"`
	InviteLink               string                   `json:"invite_link,omitempty" msgpack:"invite_link,omitempty"`
	Status                   string                   `json:"status,omitempty" msgpack:"status,omitempty"`
	Participants             map[int64]int64          `json:"participants,omitempty" msgpack:"participants,omitempty"`
	BaseRawIconURL           string                   `json:"baseRawIconUrl,omitempty" msgpack:"baseRawIconUrl,omitempty"`
	BaseIconURL              string                   `json:"baseIconUrl,omitempty" msgpack:"baseIconUrl,omitempty"`
	LastMessage              *Message                 `json:"lastMessage,omitempty" msgpack:"lastMessage,omitempty"`
	LastEventTime            int64                    `json:"lastEventTime,omitempty" msgpack:"lastEventTime,omitempty"`
	LastDelayedUpdateTime    int64                    `json:"lastDelayedUpdateTime,omitempty" msgpack:"lastDelayedUpdateTime,omitempty"`
	LastFireDelayedErrorTime int64                    `json:"lastFireDelayedErrorTime,omitempty" msgpack:"lastFireDelayedErrorTime,omitempty"`
	Created                  int64                    `json:"created,omitempty" msgpack:"created,omitempty"`
	NewMessages              int                      `json:"newMessages,omitempty" msgpack:"newMessages,omitempty"`
	Access                   AccessType               `json:"access,omitempty" msgpack:"access,omitempty"`
	Restrictions             int64                    `json:"restrictions,omitempty" msgpack:"restrictions,omitempty"`
	PinnedMessage            *Message                 `json:"pinnedMessage,omitempty" msgpack:"pinnedMessage,omitempty"`
	Options                  any                      `json:"options,omitempty" msgpack:"options,omitempty"`
	JoinTime                 int64                    `json:"joinTime,omitempty" msgpack:"joinTime,omitempty"`
	InvitedBy                int64                    `json:"invitedBy,omitempty" msgpack:"invitedBy,omitempty"`
	Modified                 int64                    `json:"modified,omitempty" msgpack:"modified,omitempty"`
	MessagesCount            int                      `json:"messagesCount,omitempty" msgpack:"messagesCount,omitempty"`
	HasBots                  *bool                    `json:"hasBots,omitempty" msgpack:"hasBots,omitempty"`
	PrevMessageID            int64                    `json:"prevMessageId,omitempty" msgpack:"prevMessageId,omitempty"`
	AdminParticipants        map[int64]map[string]any `json:"adminParticipants,omitempty" msgpack:"adminParticipants,omitempty"`
	Admins                   []int64                  `json:"admins,omitempty" msgpack:"admins,omitempty"`
	CID                      int64                    `json:"cid,omitempty" msgpack:"cid,omitempty"`
	Raw                      map[string]any           `json:"raw,omitempty" msgpack:"raw,omitempty"`
	messageActions           MessageActions
	chatActions              ChatActions
}

type Name struct {
	Name      string `json:"name,omitempty" msgpack:"name,omitempty"`
	FirstName string `json:"firstName,omitempty" msgpack:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty" msgpack:"lastName,omitempty"`
	Type      string `json:"type,omitempty" msgpack:"type,omitempty"`
}

type HandshakeResponse struct {
	CallsSeed     int64 `json:"callsSeed,omitempty" msgpack:"callsSeed,omitempty"`
	AppUpdateType int64 `json:"app-update-type,omitempty" msgpack:"app-update-type,omitempty"`
}

type MobileUserAgent struct {
	DeviceType      string `json:"deviceType" msgpack:"deviceType"`
	AppVersion      string `json:"appVersion" msgpack:"appVersion"`
	OSVersion       string `json:"osVersion" msgpack:"osVersion"`
	Timezone        string `json:"timezone" msgpack:"timezone"`
	Screen          string `json:"screen" msgpack:"screen"`
	PushDeviceType  string `json:"pushDeviceType,omitempty" msgpack:"pushDeviceType,omitempty"`
	Arch            string `json:"arch,omitempty" msgpack:"arch,omitempty"`
	Locale          string `json:"locale" msgpack:"locale"`
	BuildNumber     int    `json:"buildNumber,omitempty" msgpack:"buildNumber,omitempty"`
	DeviceName      string `json:"deviceName" msgpack:"deviceName"`
	DeviceLocale    string `json:"deviceLocale" msgpack:"deviceLocale"`
	Release         int    `json:"release,omitempty" msgpack:"release,omitempty"`
	HeaderUserAgent string `json:"headerUserAgent,omitempty" msgpack:"headerUserAgent,omitempty"`
}

// Session represents an active account session/device.
type Session struct {
	ID           any    `json:"id,omitempty" msgpack:"id,omitempty"`
	Device       string `json:"device,omitempty" msgpack:"device,omitempty"`
	DeviceID     string `json:"deviceId,omitempty" msgpack:"deviceId,omitempty"`
	Current      bool   `json:"current,omitempty" msgpack:"current,omitempty"`
	UserAgent    string `json:"userAgent,omitempty" msgpack:"userAgent,omitempty"`
	AppVersion   string `json:"appVersion,omitempty" msgpack:"appVersion,omitempty"`
	DeviceName   string `json:"deviceName,omitempty" msgpack:"deviceName,omitempty"`
	DeviceType   string `json:"deviceType,omitempty" msgpack:"deviceType,omitempty"`
	Platform     string `json:"platform,omitempty" msgpack:"platform,omitempty"`
	Location     string `json:"location,omitempty" msgpack:"location,omitempty"`
	Client       string `json:"client,omitempty" msgpack:"client,omitempty"`
	IP           string `json:"ip,omitempty" msgpack:"ip,omitempty"`
	Created      int64  `json:"created,omitempty" msgpack:"created,omitempty"`
	Updated      int64  `json:"updated,omitempty" msgpack:"updated,omitempty"`
	LastActivity int64  `json:"lastActivity,omitempty" msgpack:"lastActivity,omitempty"`
	Options      any    `json:"options,omitempty" msgpack:"options,omitempty"`
}

// User represents a user contact or profile.
type User struct {
	ID               int64          `json:"id" msgpack:"id"`
	Phone            string         `json:"phone,omitempty" msgpack:"phone,omitempty"`
	FirstName        string         `json:"first_name,omitempty" msgpack:"first_name,omitempty"`
	LastName         string         `json:"last_name,omitempty" msgpack:"last_name,omitempty"`
	Nickname         string         `json:"nickname,omitempty" msgpack:"nickname,omitempty"`
	AvatarURL        string         `json:"avatar_url,omitempty" msgpack:"avatar_url,omitempty"`
	Bio              string         `json:"bio,omitempty" msgpack:"bio,omitempty"`
	IsBot            bool           `json:"is_bot,omitempty" msgpack:"is_bot,omitempty"`
	IsContact        bool           `json:"is_contact,omitempty" msgpack:"is_contact,omitempty"`
	IsMutual         bool           `json:"is_mutual,omitempty" msgpack:"is_mutual,omitempty"`
	IsVerified       bool           `json:"is_verified,omitempty" msgpack:"is_verified,omitempty"`
	AccountStatus    int64          `json:"accountStatus,omitempty" msgpack:"accountStatus,omitempty"`
	RegistrationTime int64          `json:"registrationTime,omitempty" msgpack:"registrationTime,omitempty"`
	Country          string         `json:"country,omitempty" msgpack:"country,omitempty"`
	BaseRawURL       string         `json:"baseRawUrl,omitempty" msgpack:"baseRawUrl,omitempty"`
	BaseURL          string         `json:"baseUrl,omitempty" msgpack:"baseUrl,omitempty"`
	Names            []Name         `json:"names,omitempty" msgpack:"names,omitempty"`
	Options          []string       `json:"options,omitempty" msgpack:"options,omitempty"`
	PhotoID          int64          `json:"photoId,omitempty" msgpack:"photoId,omitempty"`
	UpdateTime       int64          `json:"updateTime,omitempty" msgpack:"updateTime,omitempty"`
	Status           string         `json:"status,omitempty" msgpack:"status,omitempty"`
	Gender           any            `json:"gender,omitempty" msgpack:"gender,omitempty"`
	Link             any            `json:"link,omitempty" msgpack:"link,omitempty"`
	WebApp           any            `json:"webApp,omitempty" msgpack:"webApp,omitempty"`
	MenuButton       map[string]any `json:"menuButton,omitempty" msgpack:"menuButton,omitempty"`
	Raw              map[string]any `json:"raw,omitempty" msgpack:"raw,omitempty"`
	actions          UserActions
}

// Member represents a chat/group participant.
type Member struct {
	UserID    int64     `json:"user_id" msgpack:"user_id"`
	Role      string    `json:"role" msgpack:"role"` // "ADMIN", "MEMBER", "OWNER"
	JoinedAt  int64     `json:"joined_at,omitempty" msgpack:"joined_at,omitempty"`
	InvitedBy int64     `json:"invited_by,omitempty" msgpack:"invited_by,omitempty"`
	Contact   *User     `json:"contact,omitempty" msgpack:"contact,omitempty"`
	Presence  *Presence `json:"presence,omitempty" msgpack:"presence,omitempty"`
}

// ReactionInfo represents reactions attached to a message.
type ReactionInfo struct {
	// Legacy flattened fields are kept for source compatibility.
	Reaction     string            `json:"reaction,omitempty" msgpack:"reaction,omitempty"`
	Count        int               `json:"count,omitempty" msgpack:"count,omitempty"`
	Self         bool              `json:"self,omitempty" msgpack:"self,omitempty"`
	TotalCount   int               `json:"totalCount,omitempty" msgpack:"totalCount,omitempty"`
	Counters     []ReactionCounter `json:"counters,omitempty" msgpack:"counters,omitempty"`
	YourReaction string            `json:"yourReaction,omitempty" msgpack:"yourReaction,omitempty"`
}

type ReactionCounter struct {
	Count    int    `json:"count" msgpack:"count"`
	Reaction string `json:"reaction" msgpack:"reaction"`
}

// ReadState is returned after a READ_MESSAGE operation.
type ReadState struct {
	Unread int   `json:"unread" msgpack:"unread"`
	Mark   int64 `json:"mark" msgpack:"mark"`
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
	AttachmentContact   AttachmentType = "CONTACT"
	AttachmentCall      AttachmentType = "CALL"
	AttachmentControl   AttachmentType = "CONTROL"
	AttachmentKeyboard  AttachmentType = "INLINE_KEYBOARD"
	AttachmentShare     AttachmentType = "SHARE"
	AttachmentUnknown   AttachmentType = "UNKNOWN"
)

// Attachment represents an uploaded or received media attachment.
type Attachment struct {
	Type                AttachmentType      `json:"type" msgpack:"type"`
	ID                  string              `json:"id,omitempty" msgpack:"id,omitempty"`
	URL                 string              `json:"url,omitempty" msgpack:"url,omitempty"`
	FileName            string              `json:"file_name,omitempty" msgpack:"file_name,omitempty"`
	FileSize            int64               `json:"file_size,omitempty" msgpack:"file_size,omitempty"`
	Duration            int                 `json:"duration,omitempty" msgpack:"duration,omitempty"`
	Width               int                 `json:"width,omitempty" msgpack:"width,omitempty"`
	Height              int                 `json:"height,omitempty" msgpack:"height,omitempty"`
	Token               string              `json:"token,omitempty" msgpack:"token,omitempty"`
	Wave                []byte              `json:"wave,omitempty" msgpack:"wave,omitempty"`
	ThumbHash           []byte              `json:"thumbHash,omitempty" msgpack:"thumbHash,omitempty"`
	VideoType           int                 `json:"videoType,omitempty" msgpack:"videoType,omitempty"`
	Poll                *Poll               `json:"poll,omitempty" msgpack:"poll,omitempty"`
	BaseURL             string              `json:"baseUrl,omitempty" msgpack:"baseUrl,omitempty"`
	PreviewData         []byte              `json:"previewData,omitempty" msgpack:"previewData,omitempty"`
	Thumbnail           string              `json:"thumbnail,omitempty" msgpack:"thumbnail,omitempty"`
	TranscriptionStatus TranscriptionStatus `json:"transcriptionStatus,omitempty" msgpack:"transcriptionStatus,omitempty"`
	Unsafe              bool                `json:"unsafe,omitempty" msgpack:"unsafe,omitempty"`
	AuthorType          string              `json:"authorType,omitempty" msgpack:"authorType,omitempty"`
	LottieURL           string              `json:"lottieUrl,omitempty" msgpack:"lottieUrl,omitempty"`
	StickerID           int64               `json:"stickerId,omitempty" msgpack:"stickerId,omitempty"`
	Tags                []string            `json:"tags,omitempty" msgpack:"tags,omitempty"`
	SetID               int64               `json:"setId,omitempty" msgpack:"setId,omitempty"`
	StickerType         string              `json:"stickerType,omitempty" msgpack:"stickerType,omitempty"`
	Audio               bool                `json:"audio,omitempty" msgpack:"audio,omitempty"`
	ContactID           int64               `json:"contactId,omitempty" msgpack:"contactId,omitempty"`
	FirstName           string              `json:"firstName,omitempty" msgpack:"firstName,omitempty"`
	LastName            string              `json:"lastName,omitempty" msgpack:"lastName,omitempty"`
	Name                string              `json:"name,omitempty" msgpack:"name,omitempty"`
	PhotoURL            string              `json:"photoUrl,omitempty" msgpack:"photoUrl,omitempty"`
	ConversationID      any                 `json:"conversationId,omitempty" msgpack:"conversationId,omitempty"`
	ContactIDs          []int64             `json:"contactIds,omitempty" msgpack:"contactIds,omitempty"`
	CallType            CallType            `json:"callType,omitempty" msgpack:"callType,omitempty"`
	HangupType          HangupType          `json:"hangupType,omitempty" msgpack:"hangupType,omitempty"`
	Event               string              `json:"event,omitempty" msgpack:"event,omitempty"`
	Title               string              `json:"title,omitempty" msgpack:"title,omitempty"`
	Description         string              `json:"description,omitempty" msgpack:"description,omitempty"`
	Image               map[string]any      `json:"image,omitempty" msgpack:"image,omitempty"`
	Keyboard            map[string]any      `json:"keyboard,omitempty" msgpack:"keyboard,omitempty"`
	Raw                 map[string]any      `json:"raw,omitempty" msgpack:"raw,omitempty"`
}

// Named aliases mirror PyMax's attachment model names while retaining a
// single discriminated Go representation.
type PhotoAttachment = Attachment
type VideoAttachment = Attachment
type AudioAttachment = Attachment
type FileAttachment = Attachment
type StickerAttachment = Attachment
type ContactAttachment = Attachment
type CallAttachment = Attachment
type ControlAttachment = Attachment
type ShareAttachment = Attachment
type InlineKeyboardAttachment = Attachment
type PollAttachment = Poll
type UnknownAttachment = Attachment

type TranscriptionStatus string

const (
	TranscriptionFailed        TranscriptionStatus = "FAILED"
	TranscriptionMediaNotReady TranscriptionStatus = "MEDIA_NOT_READY"
	TranscriptionNotSupported  TranscriptionStatus = "NOT_SUPPORTED"
	TranscriptionProcessing    TranscriptionStatus = "PROCESSING"
	TranscriptionSuccess       TranscriptionStatus = "SUCCESS"
	TranscriptionUnknown       TranscriptionStatus = "UNKNOWN"
)

type PollFlags int64

const (
	PollAnonymous   PollFlags = 1
	PollMultiselect PollFlags = 2
	PollRevote      PollFlags = 4
	PollClosed      PollFlags = 8
	PollQuiz        PollFlags = 16
	PollCanForward  PollFlags = 32
)

type CallType string

const (
	CallAudio CallType = "AUDIO"
	CallVideo CallType = "VIDEO"
)

type HangupType string

const (
	HangupMissed   HangupType = "MISSED"
	HangupRejected HangupType = "REJECTED"
	HangupCanceled HangupType = "CANCELED"
	HangupHungup   HangupType = "HUNGUP"
)

type VideoRequest struct {
	External any    `json:"EXTERNAL,omitempty" msgpack:"EXTERNAL,omitempty"`
	Cache    bool   `json:"cache" msgpack:"cache"`
	URL      string `json:"url,omitempty" msgpack:"url,omitempty"`
}

type FileRequest struct {
	Unsafe bool   `json:"unsafe" msgpack:"unsafe"`
	URL    string `json:"url" msgpack:"url"`
}

type ContactInfo struct {
	Phone     string `json:"phone" msgpack:"phone"`
	FirstName string `json:"firstName" msgpack:"firstName"`
	LastName  string `json:"lastName,omitempty" msgpack:"lastName,omitempty"`
}

// PollOption represents a single choice in a poll.
type PollOption struct {
	ID    int    `json:"id" msgpack:"id"`
	Text  string `json:"text" msgpack:"text"`
	Votes int    `json:"votes,omitempty" msgpack:"votes,omitempty"`
}

// PollAnswer is the wire representation used by Max for poll choices.
type PollAnswer struct {
	Text     string `json:"text" msgpack:"text"`
	AnswerID *int   `json:"answerId,omitempty" msgpack:"answerId,omitempty"`
}

// Poll represents a poll object in a message.
type Poll struct {
	ID           string       `json:"id" msgpack:"id"`
	PollID       int64        `json:"pollId,omitempty" msgpack:"pollId,omitempty"`
	Title        string       `json:"title,omitempty" msgpack:"title,omitempty"`
	Answers      []PollAnswer `json:"answers,omitempty" msgpack:"answers,omitempty"`
	Settings     int64        `json:"settings,omitempty" msgpack:"settings,omitempty"`
	Version      int64        `json:"version,omitempty" msgpack:"version,omitempty"`
	State        *PollState   `json:"state,omitempty" msgpack:"state,omitempty"`
	Question     string       `json:"question" msgpack:"question"`
	Options      []PollOption `json:"options" msgpack:"options"`
	Multiple     bool         `json:"multiple,omitempty" msgpack:"multiple,omitempty"`
	Anonymous    bool         `json:"anonymous,omitempty" msgpack:"anonymous,omitempty"`
	Closed       bool         `json:"closed,omitempty" msgpack:"closed,omitempty"`
	SelectedOpts []int        `json:"selected_options,omitempty" msgpack:"selected_options,omitempty"`
}

type PollVote struct {
	Timestamp int64 `json:"timestamp" msgpack:"timestamp"`
	UserID    int64 `json:"userId" msgpack:"userId"`
}

type PollResult struct {
	AnswerID  int        `json:"answerId" msgpack:"answerId"`
	VoteCount int        `json:"voteCount" msgpack:"voteCount"`
	Votes     []PollVote `json:"votes" msgpack:"votes"`
	Rate      int        `json:"rate" msgpack:"rate"`
	Options   int        `json:"options" msgpack:"options"`
}

type PollState struct {
	Total           int          `json:"total" msgpack:"total"`
	Result          []PollResult `json:"result,omitempty" msgpack:"result,omitempty"`
	VoterPreviewIDs []int64      `json:"voterPreviewIds" msgpack:"voterPreviewIds"`
}

// Message represents a text or media message in Max.
type Message struct {
	ID                int64              `json:"id" msgpack:"id"`
	CID               int64              `json:"cid,omitempty" msgpack:"cid,omitempty"` // Client message sequence ID
	ChatID            int64              `json:"chat_id" msgpack:"chat_id"`
	SenderID          int64              `json:"sender_id" msgpack:"sender_id"`
	Text              string             `json:"text,omitempty" msgpack:"text,omitempty"`
	Time              int64              `json:"time" msgpack:"time"`
	EditedAt          int64              `json:"edited_at,omitempty" msgpack:"edited_at,omitempty"`
	ReplyToMsgID      int64              `json:"reply_to,omitempty" msgpack:"reply_to,omitempty"`
	Attachments       []Attachment       `json:"attachments,omitempty" msgpack:"attachments,omitempty"`
	Reactions         []ReactionInfo     `json:"reactions,omitempty" msgpack:"reactions,omitempty"`
	ReactionInfo      *ReactionInfo      `json:"reactionInfo,omitempty" msgpack:"reactionInfo,omitempty"`
	IsOutgoing        bool               `json:"is_outgoing,omitempty" msgpack:"is_outgoing,omitempty"`
	IsPinned          bool               `json:"is_pinned,omitempty" msgpack:"is_pinned,omitempty"`
	IsDeleted         bool               `json:"is_deleted,omitempty" msgpack:"is_deleted,omitempty"`
	Type              string             `json:"type,omitempty" msgpack:"type,omitempty"`
	Status            MessageStatus      `json:"status,omitempty" msgpack:"status,omitempty"`
	PrevMessageID     any                `json:"prevMessageId,omitempty" msgpack:"prevMessageId,omitempty"`
	TTL               *bool              `json:"ttl,omitempty" msgpack:"ttl,omitempty"`
	Unread            *int               `json:"unread,omitempty" msgpack:"unread,omitempty"`
	Mark              *int64             `json:"mark,omitempty" msgpack:"mark,omitempty"`
	Elements          []Element          `json:"elements,omitempty" msgpack:"elements,omitempty"`
	DelayedAttributes *DelayedAttributes `json:"delayedAttributes,omitempty" msgpack:"delayedAttributes,omitempty"`
	Link              *MessageLink       `json:"link,omitempty" msgpack:"link,omitempty"`
	Stats             map[string]any     `json:"stats,omitempty" msgpack:"stats,omitempty"`
	Options           any                `json:"options,omitempty" msgpack:"options,omitempty"`
	Raw               map[string]any     `json:"raw,omitempty" msgpack:"raw,omitempty"`
	actions           MessageActions
}

type MessageStatus string

const (
	MessageStatusEdited  MessageStatus = "EDITED"
	MessageStatusRemoved MessageStatus = "REMOVED"
)

type DelayedAttributes struct {
	TimeToFire      int64 `json:"timeToFire" msgpack:"timeToFire"`
	NotifySender    bool  `json:"notifySender" msgpack:"notifySender"`
	NotifyOpponents bool  `json:"notifyOpponents" msgpack:"notifyOpponents"`
}

type ElementAttributes struct {
	URL string `json:"url,omitempty" msgpack:"url,omitempty"`
}
type Element struct {
	Type       string             `json:"type" msgpack:"type"`
	From       int                `json:"from,omitempty" msgpack:"from,omitempty"`
	Length     int                `json:"length,omitempty" msgpack:"length,omitempty"`
	Attributes *ElementAttributes `json:"attributes,omitempty" msgpack:"attributes,omitempty"`
}

type MessageLink struct {
	Type           LinkType   `json:"type" msgpack:"type"`
	ChatID         int64      `json:"chatId,omitempty" msgpack:"chatId,omitempty"`
	ChatName       string     `json:"chatName,omitempty" msgpack:"chatName,omitempty"`
	ChatLink       string     `json:"chatLink,omitempty" msgpack:"chatLink,omitempty"`
	ChatAccessType AccessType `json:"chatAccessType,omitempty" msgpack:"chatAccessType,omitempty"`
	ChatIconURL    string     `json:"chatIconUrl,omitempty" msgpack:"chatIconUrl,omitempty"`
	Message        *Message   `json:"message,omitempty" msgpack:"message,omitempty"`
}

// MaxAPIError is the typed error body embedded in some successful protocol
// envelopes. Transport/protocol failures use protocol.ApiError.
type MaxAPIError struct {
	Error            string `json:"error" msgpack:"error"`
	Message          string `json:"message" msgpack:"message"`
	Title            string `json:"title,omitempty" msgpack:"title,omitempty"`
	LocalizedMessage string `json:"localizedMessage,omitempty" msgpack:"localizedMessage,omitempty"`
}

// ReactionEvent represents a reaction add/remove notification push from the server.
type ReactionEvent struct {
	ChatID       int64             `json:"chat_id" msgpack:"chat_id"`
	MessageID    int64             `json:"message_id" msgpack:"message_id"`
	UserID       int64             `json:"user_id" msgpack:"user_id"`
	Reaction     string            `json:"reaction" msgpack:"reaction"`
	Removed      bool              `json:"removed" msgpack:"removed"`
	MessageIDRaw string            `json:"messageIdRaw,omitempty" msgpack:"messageIdRaw,omitempty"`
	Counters     []ReactionCounter `json:"counters,omitempty" msgpack:"counters,omitempty"`
	TotalCount   int               `json:"totalCount,omitempty" msgpack:"totalCount,omitempty"`
}

// PresenceEvent represents a user online/offline status change notification.
type PresenceEvent struct {
	UserID   int64     `json:"user_id" msgpack:"user_id"`
	Online   bool      `json:"online" msgpack:"online"`
	Presence *Presence `json:"presence,omitempty" msgpack:"presence,omitempty"`
}

type Presence struct {
	Status     string `json:"status,omitempty" msgpack:"status,omitempty"`
	LastSeen   int64  `json:"lastSeen,omitempty" msgpack:"lastSeen,omitempty"`
	Online     bool   `json:"online,omitempty" msgpack:"online,omitempty"`
	Seen       int64  `json:"seen,omitempty" msgpack:"seen,omitempty"`
	StatusCode int64  `json:"statusCode,omitempty" msgpack:"statusCode,omitempty"`
}

// Folder represents a chat filter/folder used to organize dialogs.
type Folder struct {
	SourceID   int64         `json:"sourceId,omitempty" msgpack:"sourceId,omitempty"`
	ID         string        `json:"id" msgpack:"id"`
	Title      string        `json:"title" msgpack:"title"`
	Include    []int64       `json:"include" msgpack:"include"`
	Options    []interface{} `json:"options,omitempty" msgpack:"options,omitempty"`
	Filters    []interface{} `json:"filters,omitempty" msgpack:"filters,omitempty"`
	UpdateTime int64         `json:"updateTime,omitempty" msgpack:"updateTime,omitempty"`
}

// FolderList represents a paginated list of chat folders with a sync marker.
type FolderList struct {
	Folders                 []Folder      `json:"folders" msgpack:"folders"`
	Sync                    int64         `json:"folderSync" msgpack:"folderSync"`
	FoldersOrder            []string      `json:"foldersOrder,omitempty" msgpack:"foldersOrder,omitempty"`
	AllFilterExcludeFolders []interface{} `json:"allFilterExcludeFolders,omitempty" msgpack:"allFilterExcludeFolders,omitempty"`
}

type FolderUpdate struct {
	FoldersOrder []string `json:"foldersOrder,omitempty" msgpack:"foldersOrder,omitempty"`
	Folder       *Folder  `json:"folder,omitempty" msgpack:"folder,omitempty"`
	FolderSync   int64    `json:"folderSync" msgpack:"folderSync"`
}

// PrivacyAccess controls who can access a profile capability.
type PrivacyAccess string

const (
	PrivacyAccessAll      PrivacyAccess = "ALL"
	PrivacyAccessContacts PrivacyAccess = "CONTACTS"
	PrivacyAccessNobody   PrivacyAccess = "_NONE_"
)

// PrivacySettingsUpdate contains optional account privacy changes. Pointer
// fields distinguish an omitted setting from an explicit false value.
type PrivacySettingsUpdate struct {
	SearchByPhone         *PrivacyAccess
	IncomingCalls         *PrivacyAccess
	ChatInvites           *PrivacyAccess
	PhoneNumberVisibility *PrivacyAccess
	HideOnlineStatus      *bool
	SafeContentOnly       *bool
}

func (settings PrivacySettingsUpdate) Payload() map[string]any {
	payload := make(map[string]any)
	if settings.SearchByPhone != nil {
		payload["SEARCH_BY_PHONE"] = *settings.SearchByPhone
	}
	if settings.IncomingCalls != nil {
		payload["INCOMING_CALL"] = *settings.IncomingCalls
	}
	if settings.ChatInvites != nil {
		payload["CHATS_INVITE"] = *settings.ChatInvites
	}
	if settings.PhoneNumberVisibility != nil {
		payload["PHONE_NUMBER_PRIVACY"] = *settings.PhoneNumberVisibility
	}
	if settings.HideOnlineStatus != nil {
		payload["HIDDEN"] = *settings.HideOnlineStatus
	}
	if settings.SafeContentOnly != nil {
		payload["CONTENT_LEVEL_ACCESS"] = *settings.SafeContentOnly
	}
	return payload
}

// TypingEvent represents a user typing/action indicator in a chat.
type TypingEvent struct {
	ChatID int64 `json:"chat_id" msgpack:"chat_id"`
	UserID int64 `json:"user_id" msgpack:"user_id"`
}
