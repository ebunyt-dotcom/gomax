// Package gomax provides an idiomatic, high-performance Go client for the Max API,
// ported directly from the Python PyMax library.
package gomax

import (
	"github.com/ebunyt-dotcom/gomax/pkg/client"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// Primary client constructors
type (
	// Client provides a high-level TCP client for Max API.
	Client = client.Client

	// WebClient provides a WebSocket client with QR login.
	WebClient = client.WebClient

	// Config configures client endpoints, auth, and persistence.
	Config = client.Config

	// Message represents a text/media message.
	Message = types.Message

	// Chat represents a group, channel, or dialog.
	Chat = types.Chat

	// User represents a user contact or profile.
	User = types.User

	// Member represents a chat member.
	Member = types.Member

	// ReactionInfo represents reactions on a message.
	ReactionInfo = types.ReactionInfo

	// Attachment represents a media attachment.
	Attachment = types.Attachment

	// AttachmentType classifies the kind of media attachment.
	AttachmentType = types.AttachmentType

	// ReactionEvent represents a reaction add/remove notification.
	ReactionEvent = types.ReactionEvent

	// PresenceEvent represents a user online/offline status notification.
	PresenceEvent = types.PresenceEvent

	// TypingEvent represents a user typing indicator.
	TypingEvent = types.TypingEvent

	// Folder represents a chat filter/folder.
	Folder = types.Folder

	// FolderList represents a list of chat folders.
	FolderList = types.FolderList

	// PollOption represents a poll answer choice.
	PollOption = types.PollOption

	// Poll represents a poll message object.
	Poll = types.Poll

	// ChatType classifies the chat kind.
	ChatType = types.ChatType
)

// Attachment type constants re-exported for convenience.
const (
	AttachmentPhoto     = types.AttachmentPhoto
	AttachmentVideo     = types.AttachmentVideo
	AttachmentAudio     = types.AttachmentAudio
	AttachmentFile      = types.AttachmentFile
	AttachmentVoice     = types.AttachmentVoice
	AttachmentVideoNote = types.AttachmentVideoNote
	AttachmentPoll      = types.AttachmentPoll
	AttachmentSticker   = types.AttachmentSticker
)

// Chat type constants re-exported for convenience.
const (
	ChatTypeDialog  = types.ChatTypeDialog
	ChatTypeChat    = types.ChatTypeChat
	ChatTypeChannel = types.ChatTypeChannel
)

// NewClient creates a new TCP Max client matching PyMax Client.
func NewClient(cfg *Config) *Client {
	return client.NewClient(cfg)
}

// NewWebClient creates a new WebSocket Max client matching PyMax WebClient.
func NewWebClient(cfg *Config) *WebClient {
	return client.NewWebClient(cfg)
}

// DefaultConfig returns default client configuration.
func DefaultConfig() *Config {
	return client.DefaultConfig()
}
