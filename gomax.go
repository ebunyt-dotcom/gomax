// Package gomax provides an idiomatic Go client for the unofficial Max API.
//
// The package follows the protocol and public behavior of PyMax while exposing
// Go-native clients, services, typed events, session stores, and media helpers.
// Use NewClient for TCP/SMS login or NewWebClient for WebSocket/QR login.
//
// Basic usage:
//
//     cfg := gomax.DefaultConfig()
//     cfg.Phone = "+79990000000"
//     client := gomax.NewClient(cfg)
//     err := client.Start(context.Background())
//
// See the full guide at https://ebunyt-dotcom.github.io/gomax/.
package gomax

import (
	authflow "github.com/ebunyt-dotcom/gomax/pkg/auth"
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

	// InitData contains bot web-app initialization data.
	InitData = types.InitData

	// SmsAuthFlow performs phone/SMS authentication and optional 2FA.
	SmsAuthFlow = authflow.SmsAuthFlow
	// QrAuthFlow performs QR authentication and optional 2FA.
	QrAuthFlow = authflow.QrAuthFlow
	// CodeProvider supplies an SMS verification code.
	CodeProvider = authflow.CodeProvider
	// PasswordProvider supplies a 2FA password.
	PasswordProvider = authflow.PasswordProvider
	// PasswordProviderHint optionally receives a server password hint.
	PasswordProviderHint = authflow.PasswordProviderWithHint
	// QrHandler receives a QR URL and decides how to display it.
	QrHandler = authflow.QrHandler
	// AuthResult contains the token and user data returned by an auth flow.
	AuthResult = authflow.AuthResult
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

// NewSmsAuthFlow creates a configurable SMS/2FA authentication flow.
func NewSmsAuthFlow(codeProvider CodeProvider, passwordProvider PasswordProvider) *SmsAuthFlow {
	return authflow.NewSmsAuthFlow(codeProvider, passwordProvider)
}

// NewQrAuthFlow creates a configurable QR authentication flow.
func NewQrAuthFlow(handler QrHandler, passwordProvider PasswordProvider) *QrAuthFlow {
	return authflow.NewQrAuthFlow(handler, passwordProvider)
}
