// Package gomax provides an idiomatic, high-performance Go client for the Max API,
// ported directly from the Python PyMax library.
package gomax

import (
	"gomax/pkg/client"
	"gomax/pkg/types"
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

	// RegistrationConfig holds registration profile names.
	RegistrationConfig = types.RegistrationConfig
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
