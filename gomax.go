// Package gomax provides an idiomatic Go client for the unofficial Max API.
//
// The package follows the protocol and public behavior of PyMax while exposing
// Go-native clients, services, typed events, session stores, and media helpers.
// Use NewClient for TCP/SMS login or NewWebClient for WebSocket/QR login.
//
// Basic usage:
//
//	cfg := gomax.DefaultConfig()
//	cfg.Phone = "+79990000000"
//	client := gomax.NewClient(cfg)
//	err := client.Start(context.Background())
//
// See the full guide at https://ebunyt-dotcom.github.io/gomax/.
package gomax

import (
	authapi "github.com/ebunyt-dotcom/gomax/pkg/api/auth"
	authflow "github.com/ebunyt-dotcom/gomax/pkg/auth"
	"github.com/ebunyt-dotcom/gomax/pkg/client"
	"github.com/ebunyt-dotcom/gomax/pkg/dispatch"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/session"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// Primary client constructors
type (
	// Client provides a high-level TCP client for Max API.
	Client = client.Client

	// WebClient provides a WebSocket client with QR login.
	WebClient = client.WebClient

	// Config configures client endpoints, auth, and persistence.
	Config        = client.Config
	SyncOverrides = client.SyncOverrides
	// RegistrationConfig contains the profile fields used to finish a new account registration.
	RegistrationConfig = client.RegistrationConfig
	// NonRecoverableError marks an error after which reconnecting is not useful.
	NonRecoverableError = client.NonRecoverableError
	// ApiError is the typed error returned by the Max protocol.
	ApiError = protocol.ApiError
	// SyncState contains persisted incremental-sync markers.
	SyncState = session.SyncState

	// Message represents a text/media message.
	Message = types.Message

	// Chat represents a group, channel, or dialog.
	Chat = types.Chat

	// User represents a user contact or profile.
	User = types.User

	// Member represents a chat member.
	Member = types.Member

	// ReactionInfo represents reactions on a message.
	ReactionInfo    = types.ReactionInfo
	ReactionCounter = types.ReactionCounter

	// Attachment represents a media attachment.
	Attachment               = types.Attachment
	PhotoAttachment          = types.PhotoAttachment
	VideoAttachment          = types.VideoAttachment
	AudioAttachment          = types.AudioAttachment
	FileAttachment           = types.FileAttachment
	StickerAttachment        = types.StickerAttachment
	ContactAttachment        = types.ContactAttachment
	CallAttachment           = types.CallAttachment
	ControlAttachment        = types.ControlAttachment
	ShareAttachment          = types.ShareAttachment
	InlineKeyboardAttachment = types.InlineKeyboardAttachment
	UnknownAttachment        = types.UnknownAttachment
	VideoRequest             = types.VideoRequest
	FileRequest              = types.FileRequest
	ContactInfo              = types.ContactInfo
	TranscriptionStatus      = types.TranscriptionStatus
	PollFlags                = types.PollFlags
	CallType                 = types.CallType
	HangupType               = types.HangupType

	// AttachmentType classifies the kind of media attachment.
	AttachmentType = types.AttachmentType

	// ReactionEvent represents a reaction add/remove notification.
	ReactionEvent      = types.ReactionEvent
	RawEvent           = types.RawEvent
	MessageReadEvent   = types.MessageReadEvent
	MessageDeleteEvent = types.MessageDeleteEvent
	UserUpdateEvent    = types.UserUpdateEvent

	// PresenceEvent represents a user online/offline status notification.
	PresenceEvent = types.PresenceEvent

	// TypingEvent represents a user typing indicator.
	TypingEvent = types.TypingEvent

	// Folder represents a chat filter/folder.
	Folder = types.Folder

	// FolderList represents a list of chat folders.
	FolderList   = types.FolderList
	FolderUpdate = types.FolderUpdate

	// PollOption represents a poll answer choice.
	PollOption = types.PollOption

	// Poll represents a poll message object.
	Poll                        = types.Poll
	PollState                   = types.PollState
	PollAnswer                  = types.PollAnswer
	PollVote                    = types.PollVote
	PollResult                  = types.PollResult
	ReadState                   = types.ReadState
	Element                     = types.Element
	ElementAttributes           = types.ElementAttributes
	DelayedAttributes           = types.DelayedAttributes
	MessageLink                 = types.MessageLink
	Name                        = types.Name
	Presence                    = types.Presence
	HandshakeResponse           = types.HandshakeResponse
	MobileUserAgent             = types.MobileUserAgent
	Session                     = types.Session
	StartAuthResponse           = types.StartAuthResponse
	CheckCodeResponse           = types.CheckCodeResponse
	CheckPasswordResponse       = types.CheckPasswordResponse
	RequestQRResponse           = types.RequestQRResponse
	CheckQRResponse             = types.CheckQRResponse
	ConfirmRegistrationResponse = types.ConfirmRegistrationResponse
	LoginResponse               = types.LoginResponse
	Login2Response              = types.Login2Response
	Login2Flags                 = types.Login2Flags
	AuthToken                   = types.AuthToken
	TokenAttrs                  = types.TokenAttrs
	PasswordChallenge           = types.PasswordChallenge
	QRStatus                    = types.QRStatus
	Profile                     = types.Profile
	MaxAPIError                 = types.MaxAPIError
	PrivacyAccess               = types.PrivacyAccess
	PrivacySettingsUpdate       = types.PrivacySettingsUpdate
	MessageActionOptions        = types.MessageActionOptions
	HistoryActionOptions        = types.HistoryActionOptions
	ChatSettings                = types.ChatSettings

	// ChatType classifies the chat kind.
	ChatType        = types.ChatType
	AccessType      = types.AccessType
	LinkType        = types.LinkType
	MessageStatus   = types.MessageStatus
	ErrorScope      = dispatch.ErrorScope
	TwoFactorAction = authapi.TwoFactorAction

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

const (
	TwoFactorSetPassword     = authapi.TwoFactorSetPassword
	TwoFactorUpdatePassword  = authapi.TwoFactorUpdatePassword
	TwoFactorRestorePassword = authapi.TwoFactorRestorePassword
	TwoFactorHint            = authapi.TwoFactorHint
	TwoFactorEmail           = authapi.TwoFactorEmail
	TwoFactorRemove          = authapi.TwoFactorRemove
)

const (
	MessageStatusEdited  = types.MessageStatusEdited
	MessageStatusRemoved = types.MessageStatusRemoved
	PollAnonymous        = types.PollAnonymous
	PollMultiselect      = types.PollMultiselect
	PollRevote           = types.PollRevote
	PollClosed           = types.PollClosed
	PollQuiz             = types.PollQuiz
	PollCanForward       = types.PollCanForward
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
	AttachmentContact   = types.AttachmentContact
	AttachmentCall      = types.AttachmentCall
	AttachmentControl   = types.AttachmentControl
	AttachmentKeyboard  = types.AttachmentKeyboard
	AttachmentShare     = types.AttachmentShare
	AttachmentUnknown   = types.AttachmentUnknown
)

const (
	AccessPublic  = types.AccessPublic
	AccessPrivate = types.AccessPrivate
	AccessSecret  = types.AccessSecret
	LinkReply     = types.LinkReply
	LinkForward   = types.LinkForward
)

const (
	ErrorScopeGlobal = dispatch.ErrorScopeGlobal
	ErrorScopeLocal  = dispatch.ErrorScopeLocal
)

const (
	PrivacyAccessAll      = types.PrivacyAccessAll
	PrivacyAccessContacts = types.PrivacyAccessContacts
	PrivacyAccessNobody   = types.PrivacyAccessNobody
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
