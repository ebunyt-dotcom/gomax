package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	authapi "github.com/ebunyt-dotcom/gomax/pkg/api/auth"
	"github.com/ebunyt-dotcom/gomax/pkg/api/bots"
	"github.com/ebunyt-dotcom/gomax/pkg/api/chats"
	"github.com/ebunyt-dotcom/gomax/pkg/api/messages"
	selfapi "github.com/ebunyt-dotcom/gomax/pkg/api/selfapi"
	sessionapi "github.com/ebunyt-dotcom/gomax/pkg/api/sessionapi"
	"github.com/ebunyt-dotcom/gomax/pkg/api/uploads"
	"github.com/ebunyt-dotcom/gomax/pkg/api/users"
	"github.com/ebunyt-dotcom/gomax/pkg/auth"
	"github.com/ebunyt-dotcom/gomax/pkg/connection"
	"github.com/ebunyt-dotcom/gomax/pkg/dispatch"
	"github.com/ebunyt-dotcom/gomax/pkg/fingerprint"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/session"
	"github.com/ebunyt-dotcom/gomax/pkg/telemetry"
	"github.com/ebunyt-dotcom/gomax/pkg/transport"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// Config configures client network endpoints and session behavior.
type Config struct {
	Phone          string
	WorkDir        string
	SessionName    string
	Host           string
	Port           int
	URL            string // WebSocket URL
	UseSSL         bool
	Proxy          string
	DeviceID       string
	MtInstanceID   string
	Token          string
	PersistSession bool
	Reconnect      bool
	Relogin        bool
	ReconnectDelay time.Duration
	RequestTimeout time.Duration
	// PasswordMaxAttempts is nil for unlimited attempts. Zero or a negative
	// value rejects a password challenge immediately, matching PyMax.
	PasswordMaxAttempts         *int
	LogLevel                    string
	Telemetry                   bool
	Interactive                 bool
	UploadTimeout               time.Duration
	DisableUploadTimeout        bool
	ProtocolVersion             uint8
	ClientSessionID             int
	RestoreUserAgentFromSession bool
	DisableUserAgentRestore     bool
	Sync                        SyncOverrides

	// Device/user-agent fields mirror PyMax's ExtraConfig and can be used to
	// override the built-in Android profile sent during handshake and login.
	DeviceType      string
	AppVersion      string
	BuildNumber     int
	OSVersion       string
	Timezone        string
	Screen          string
	Locale          string
	DeviceLocale    string
	DeviceName      string
	Arch            string
	PushDeviceType  string
	Release         int
	HeaderUserAgent string
	UserAgent       map[string]interface{}
	Fingerprint     *fingerprint.ApkBuildFingerprint

	Registration *RegistrationConfig

	Store session.Store

	// AuthFlow allows callers to provide a custom SMS/2FA flow. A pointer is
	// used so the value returned by auth.NewSmsAuthFlow can be assigned
	// directly and so the flow can retain provider state between retries.
	AuthFlow *auth.SmsAuthFlow
	// QrAuthFlow is used by WebClient. It lives here because Config is shared
	// by Client and WebClient while the concrete flow remains optional.
	QrAuthFlow *auth.QrAuthFlow
}

// SyncOverrides selectively replaces sync markers loaded from the session.
type SyncOverrides struct {
	ChatsSync    *int64
	ContactsSync *int64
	DraftsSync   *int64
	PresenceSync *int64
	ConfigHash   *string
}

// RegistrationConfig contains the profile fields required to finish a new
// account login when SMS verification returns a registration token.
type RegistrationConfig struct {
	FirstName string
	LastName  string
}

// DefaultConfig returns default client configuration matching PyMax.
func DefaultConfig() *Config {
	return &Config{
		Host:                        "api2.oneme.ru",
		Port:                        443,
		URL:                         "wss://api.oneme.ru/websocket",
		UseSSL:                      true,
		WorkDir:                     "cache",
		SessionName:                 "main.json",
		PersistSession:              true,
		Reconnect:                   true,
		Relogin:                     true,
		ReconnectDelay:              time.Second,
		RequestTimeout:              30 * time.Second,
		PasswordMaxAttempts:         nil,
		LogLevel:                    "INFO",
		Telemetry:                   true,
		Interactive:                 true,
		UploadTimeout:               15 * time.Minute,
		ProtocolVersion:             protocol.VersionTcp,
		RestoreUserAgentFromSession: true,
		DeviceType:                  "ANDROID",
		AppVersion:                  "26.25.0",
		BuildNumber:                 6790,
		OSVersion:                   "Android 14",
		Timezone:                    "Europe/Moscow",
		Screen:                      "405dpi 405dpi 1080x2400",
		Locale:                      "ru",
		DeviceLocale:                "ru",
		DeviceName:                  "Samsung SM-A536B",
		Arch:                        "arm64-v8a",
		PushDeviceType:              "GCM",
	}
}

func applyDefaults(cfg *Config, web bool) {
	defaults := DefaultConfig()
	if cfg.WorkDir == "" {
		cfg.WorkDir = defaults.WorkDir
	}
	if cfg.SessionName == "" {
		cfg.SessionName = defaults.SessionName
	}
	if cfg.DisableUploadTimeout || cfg.UploadTimeout < 0 {
		cfg.UploadTimeout = 0
	} else if cfg.UploadTimeout == 0 {
		cfg.UploadTimeout = defaults.UploadTimeout
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaults.LogLevel
	}
	if cfg.ProtocolVersion == 0 {
		cfg.ProtocolVersion = defaults.ProtocolVersion
	}
	if !cfg.DisableUserAgentRestore {
		cfg.RestoreUserAgentFromSession = true
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaults.RequestTimeout
	}
	if cfg.ReconnectDelay <= 0 {
		cfg.ReconnectDelay = defaults.ReconnectDelay
	}
	if cfg.AppVersion == "" {
		cfg.AppVersion = defaults.AppVersion
	}
	if cfg.BuildNumber == 0 {
		cfg.BuildNumber = defaults.BuildNumber
	}
	if cfg.DeviceType == "" {
		cfg.DeviceType = defaults.DeviceType
	}
	if cfg.OSVersion == "" {
		cfg.OSVersion = defaults.OSVersion
	}
	if cfg.Timezone == "" {
		cfg.Timezone = defaults.Timezone
	}
	if cfg.Screen == "" {
		cfg.Screen = defaults.Screen
	}
	if cfg.Locale == "" {
		cfg.Locale = defaults.Locale
	}
	if cfg.DeviceLocale == "" {
		cfg.DeviceLocale = defaults.DeviceLocale
	}
	if cfg.DeviceName == "" {
		cfg.DeviceName = defaults.DeviceName
	}
	if cfg.Arch == "" {
		cfg.Arch = defaults.Arch
	}
	if cfg.PushDeviceType == "" {
		cfg.PushDeviceType = defaults.PushDeviceType
	}
	if web {
		cfg.DeviceType = "WEB"
		cfg.AppVersion = "26.8.4"
		cfg.BuildNumber = 0
		cfg.OSVersion = "Linux"
		cfg.Screen = "1080x1920 1.0x"
		cfg.DeviceName = "Chrome"
		cfg.PushDeviceType = ""
	}
}

func mobileUserAgent(cfg *Config) map[string]interface{} {
	if cfg.UserAgent != nil {
		return cfg.UserAgent
	}
	userAgent := map[string]interface{}{
		"deviceType":   cfg.DeviceType,
		"appVersion":   cfg.AppVersion,
		"osVersion":    cfg.OSVersion,
		"timezone":     cfg.Timezone,
		"screen":       cfg.Screen,
		"locale":       cfg.Locale,
		"deviceLocale": cfg.DeviceLocale,
		"buildNumber":  cfg.BuildNumber,
		"deviceName":   cfg.DeviceName,
		"arch":         cfg.Arch,
	}
	if cfg.PushDeviceType != "" {
		userAgent["pushDeviceType"] = cfg.PushDeviceType
	}
	if cfg.Release != 0 {
		userAgent["release"] = cfg.Release
	}
	if cfg.HeaderUserAgent != "" {
		userAgent["headerUserAgent"] = cfg.HeaderUserAgent
	}
	return userAgent
}

// Client is the primary high-level TCP client for Max API.
type Client struct {
	cfg       *Config
	conn      *connection.ConnectionManager
	store     session.Store
	router    *dispatch.Router
	fpGen     *fingerprint.FingerprintGenerator
	callsSeed int64

	Messages  *messages.MessageService
	Auth      *authapi.AuthService
	Chats     *chats.ChatService
	Users     *users.UserService
	Uploads   *uploads.UploadService
	Bots      *bots.BotsService
	Self      *selfapi.SelfService
	Session   *sessionapi.Service
	Telemetry *telemetry.Service

	Me           *types.User
	ChatCache    map[int64]types.Chat
	MessageCache map[int64][]types.Message
	mu           sync.RWMutex
	started      bool
	ctx          context.Context
	cancel       context.CancelFunc
}

// CallsSeed returns the callsSeed received from the handshake.
func (c *Client) CallsSeed() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.callsSeed
}

// GetDeviceID returns the client device ID.
func (c *Client) GetDeviceID() string {
	return c.cfg.DeviceID
}

// GetCallsSeed returns the handshake callsSeed.
func (c *Client) GetCallsSeed() int64 {
	return c.CallsSeed()
}

// SetInteractive changes the interactive/presence flag used by the next
// session and reconnects. It mirrors PyMax SelfService.set_presence.
func (c *Client) SetInteractive(online bool) {
	c.mu.Lock()
	c.cfg.Interactive = online
	c.mu.Unlock()
}

// NonRecoverableError indicates an error that should terminate client.Start immediately without reconnecting.
type NonRecoverableError struct {
	Err error
}

// Error returns the underlying non-recoverable error message.
func (e *NonRecoverableError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error for errors.Is and errors.As.
func (e *NonRecoverableError) Unwrap() error {
	return e.Err
}

func nonRecoverable(err error) error {
	if err == nil {
		return nil
	}
	return &NonRecoverableError{Err: err}
}

func extractInt64(val any) (int64, bool) {
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int16:
		return int64(v), true
	case int8:
		return int64(v), true
	case uint64:
		return int64(v), true
	case uint:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint8:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, true
		}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, true
		}
	}
	return 0, false
}

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "a1b2c3d4e5f60718"
	}
	return hex.EncodeToString(bytes)
}

func randomClientSessionID() int {
	value, err := rand.Int(rand.Reader, big.NewInt(70))
	if err != nil {
		return 1
	}
	return int(value.Int64()) + 1
}

func (c *Client) logf(format string, args ...any) {
	switch strings.ToUpper(c.cfg.LogLevel) {
	case "OFF", "NONE", "FATAL", "ERROR", "WARN", "WARNING":
		return
	default:
		log.Printf("[gomax] "+format, args...)
	}
}

// NewClient creates a new Max TCP client matching pymax.Client.
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.Host == "" {
		cfg.Host = "api2.oneme.ru"
	}
	if cfg.Port == 0 {
		cfg.Port = 443
	}
	applyDefaults(cfg, false)
	var store session.Store
	if !cfg.PersistSession {
		store = session.NewInMemoryStore()
	} else if cfg.Store != nil {
		store = cfg.Store
	} else {
		store = session.NewFileStore(cfg.WorkDir, cfg.SessionName)
	}

	fingerprintData := cfg.Fingerprint
	if fingerprintData == nil {
		if resolved, err := fingerprint.NewVersionCatalog().Resolve(cfg.AppVersion); err == nil {
			fingerprintData = resolved
		} else {
			fingerprintData = fingerprint.DefaultFingerprint()
		}
	}
	c := &Client{
		cfg:          cfg,
		store:        store,
		router:       dispatch.NewRouter(),
		fpGen:        fingerprint.NewFingerprintGenerator(fingerprintData),
		ChatCache:    make(map[int64]types.Chat),
		MessageCache: make(map[int64][]types.Message),
	}

	c.Messages = messages.NewMessageService(c)
	c.Auth = authapi.NewAuthService(c)
	c.Chats = chats.NewChatService(c)
	c.Users = users.NewUserService(c)
	c.Chats.SetMessageActions(c.Messages)
	c.Uploads = uploads.NewUploadServiceWithOptions(c, uploads.Options{
		Timeout: cfg.UploadTimeout, ProxyURL: cfg.Proxy,
		UserAgent: fmt.Sprintf("OKMessages/%s (%s; %s; %s)", cfg.AppVersion, cfg.OSVersion, cfg.DeviceName, cfg.Screen),
	})
	c.Messages.SetAttachmentReadyWaiter(c.Uploads.WaitForMessageAttachments)
	c.Bots = bots.NewBotsService(c)
	c.Self = selfapi.NewSelfService(c)
	c.Self.SetUserActions(c.Users)
	c.Session = sessionapi.NewService(c)
	c.Telemetry = telemetry.NewService(c, c.telemetrySnapshot)
	c.Self.SetSessionHooks(
		func(_ string, newToken string) error { return updateStoredToken(c.store, c.cfg, newToken) },
		func(hash string) error { return updateStoredConfigHash(c.store, hash) },
	)

	return c
}

func (c *Client) telemetrySnapshot() telemetry.Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	chats := make([]types.Chat, 0, len(c.ChatCache))
	for _, chat := range c.ChatCache {
		chats = append(chats, chat)
	}
	userID := int64(0)
	if c.Me != nil {
		userID = c.Me.ID
	}
	return telemetry.Snapshot{Ready: c.started && c.conn != nil && c.conn.IsOpen() && c.Me != nil, UserID: userID, Chats: chats}
}

// OnMessage registers an incoming message listener.
func (c *Client) OnMessage(handler func(ctx context.Context, msg *types.Message) error, filters ...dispatch.MessagePredicate) {
	c.router.OnMessage(handler, filters...)
}

// OnStart registers an on_start listener executed after successful login.
func (c *Client) OnStart(handler func(ctx context.Context) error) {
	c.router.OnStart(handler)
}

// OnMessageEdit registers a handler for message edit events.
func (c *Client) OnMessageEdit(handler func(ctx context.Context, msg *types.Message) error, filters ...dispatch.MessagePredicate) {
	c.router.OnMessageEdit(handler, filters...)
}

// OnMessageDelete registers a handler for message delete events.
func (c *Client) OnMessageDelete(handler func(ctx context.Context, chatID, msgID int64) error) {
	c.router.OnMessageDelete(handler)
}

func (c *Client) OnMessageDeleteEvent(handler func(ctx context.Context, ev *types.MessageDeleteEvent) error, filters ...dispatch.MessageDeletePredicate) {
	c.router.OnMessageDeleteEvent(handler, filters...)
}

// OnMessageRead registers a read-marker handler.
func (c *Client) OnMessageRead(handler func(ctx context.Context, ev *types.MessageReadEvent) error, filters ...dispatch.MessageReadPredicate) {
	c.router.OnMessageRead(handler, filters...)
}

// OnUserUpdate registers a contact/profile update handler.
func (c *Client) OnUserUpdate(handler func(ctx context.Context, ev *types.UserUpdateEvent) error, filters ...dispatch.UserUpdatePredicate) {
	c.router.OnUserUpdate(handler, filters...)
}

// OnReaction registers a handler for reaction add/remove events.
func (c *Client) OnReaction(handler func(ctx context.Context, ev *types.ReactionEvent) error, filters ...dispatch.ReactionPredicate) {
	c.router.OnReaction(handler, filters...)
}

// OnChatUpdate registers a handler for chat metadata update events.
func (c *Client) OnChatUpdate(handler func(ctx context.Context, chat *types.Chat) error, filters ...dispatch.ChatUpdatePredicate) {
	c.router.OnChatUpdate(handler, filters...)
}

// OnPresence registers a handler for user online/offline status events.
func (c *Client) OnPresence(handler func(ctx context.Context, ev *types.PresenceEvent) error, filters ...dispatch.PresencePredicate) {
	c.router.OnPresence(handler, filters...)
}

// OnTyping registers a handler for user typing indicator events.
func (c *Client) OnTyping(handler func(ctx context.Context, ev *types.TypingEvent) error, filters ...dispatch.TypingPredicate) {
	c.router.OnTyping(handler, filters...)
}

// OnDisconnect registers a handler called when the client disconnects.
func (c *Client) OnDisconnect(handler func(ctx context.Context, err error)) {
	c.router.OnDisconnect(handler)
}

// OnRaw registers a handler for low-level event frames not consumed by a
// typed event handler.
func (c *Client) OnRaw(handler func(ctx context.Context, event *types.RawEvent) error, filters ...dispatch.EventPredicate) {
	c.router.OnEvent(handler, filters...)
}

func updateStoredToken(store session.Store, cfg *Config, newToken string) error {
	info, err := store.LoadSession()
	if err != nil {
		return err
	}
	oldToken := ""
	if info != nil {
		oldToken = info.Token
	}
	if oldToken == "" {
		if info == nil {
			info = &session.SessionInfo{Phone: cfg.Phone, DeviceID: cfg.DeviceID, MTInstanceID: cfg.MtInstanceID}
		}
		info.Token = newToken
		if err := store.SaveSession(info); err != nil {
			return err
		}
	} else if err := store.UpdateToken(oldToken, newToken); err != nil {
		return err
	}
	cfg.Token = newToken
	return nil
}

func updateStoredConfigHash(store session.Store, hash string) error {
	info, err := store.LoadSession()
	if err != nil || info == nil {
		return err
	}
	info.Sync.ConfigHash = hash
	return store.SaveSession(info)
}

func sessionUserAgent(cfg *Config) *session.UserAgentPayload {
	return &session.UserAgentPayload{DeviceType: cfg.DeviceType, AppVersion: cfg.AppVersion, BuildNumber: cfg.BuildNumber, OSVersion: cfg.OSVersion, Timezone: cfg.Timezone, Screen: cfg.Screen, PushDeviceType: cfg.PushDeviceType, Arch: cfg.Arch, Locale: cfg.Locale, DeviceName: cfg.DeviceName, DeviceLocale: cfg.DeviceLocale, Release: cfg.Release, HeaderUserAgent: cfg.HeaderUserAgent}
}

func restoreSessionUserAgent(cfg *Config, stored *session.UserAgentPayload) {
	if stored == nil || cfg.UserAgent != nil || !cfg.RestoreUserAgentFromSession || stored.DeviceType != cfg.DeviceType {
		return
	}
	if stored.DeviceType != "" {
		cfg.DeviceType = stored.DeviceType
	}
	if stored.OSVersion != "" {
		cfg.OSVersion = stored.OSVersion
	}
	if stored.DeviceName != "" {
		cfg.DeviceName = stored.DeviceName
	}
	if stored.Timezone != "" {
		cfg.Timezone = stored.Timezone
	}
	if stored.Screen != "" {
		cfg.Screen = stored.Screen
	}
	if stored.PushDeviceType != "" {
		cfg.PushDeviceType = stored.PushDeviceType
	}
	if stored.Arch != "" {
		cfg.Arch = stored.Arch
	}
	if stored.Locale != "" {
		cfg.Locale = stored.Locale
	}
	if stored.DeviceLocale != "" {
		cfg.DeviceLocale = stored.DeviceLocale
	}
	if stored.Release != 0 {
		cfg.Release = stored.Release
	}
	if stored.HeaderUserAgent != "" {
		cfg.HeaderUserAgent = stored.HeaderUserAgent
	}
}

func resolveSync(saved session.SyncState, overrides SyncOverrides) session.SyncState {
	if overrides.ChatsSync != nil {
		saved.ChatsSync = *overrides.ChatsSync
	}
	if overrides.ContactsSync != nil {
		saved.ContactsSync = *overrides.ContactsSync
	}
	if overrides.DraftsSync != nil {
		saved.DraftsSync = *overrides.DraftsSync
	}
	if overrides.PresenceSync != nil {
		saved.PresenceSync = *overrides.PresenceSync
	}
	if overrides.ConfigHash != nil {
		saved.ConfigHash = *overrides.ConfigHash
	}
	return saved
}

func (c *Client) OnError(handler func(ctx context.Context, err error)) { c.router.OnError(handler) }
func (c *Client) OnErrorScoped(scope dispatch.ErrorScope, handler func(context.Context, error)) {
	c.router.OnErrorScoped(scope, handler)
}
func (c *Client) IncludeRouter(router *dispatch.Router) { c.router.Include(router) }

func (c *Client) Connect(ctx context.Context) error { return c.Start(ctx) }
func (c *Client) Stop() error                       { return c.Close() }
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	return conn != nil && conn.IsOpen()
}
func (c *Client) Relogin(ctx context.Context) error {
	if err := clearStoredSessionToken(c.store); err != nil {
		return err
	}
	c.cfg.Token = ""
	c.mu.RLock()
	conn, started := c.conn, c.started
	c.mu.RUnlock()
	if started && conn != nil {
		conn.Fail(errors.New("relogin requested"))
		return nil
	}
	return c.Start(ctx)
}

// Invoke implements the api.Invoker interface for RPC commands.
func (c *Client) Invoke(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil || !conn.IsOpen() {
		return nil, errors.New("client: connection is not open")
	}

	inbound, err := conn.SendRequest(ctx, op, payload)
	if err != nil {
		return nil, err
	}

	if inbound.Cmd == protocol.CmdError {
		return nil, protocol.NewApiError(inbound)
	}

	return inbound.Payload, nil
}

// Start connects to Max server, handles auth, and begins processing events.
// If Reconnect is enabled, it automatically re-establishes connection on drops.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return errors.New("client already started")
	}
	c.started = true
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.mu.Unlock()

	defer func() {
		_ = c.Close()
	}()

	// Validate initial-auth credentials before opening a network connection.
	// A saved session may supply both fields, so consult the store first.
	stored, err := c.store.LoadSession()
	if err != nil {
		return nonRecoverable(fmt.Errorf("failed to load session: %w", err))
	}
	hasToken := c.cfg.Token != ""
	hasPhone := c.cfg.Phone != ""
	if stored != nil {
		hasToken = hasToken || stored.Token != ""
		hasPhone = hasPhone || stored.Phone != ""
	}
	if !hasToken && !hasPhone {
		return nonRecoverable(errors.New("phone number required for initial authentication"))
	}

	for {
		err := c.runSession(c.ctx)
		if err == nil || errors.Is(err, context.Canceled) || c.ctx.Err() != nil {
			return nil
		}

		if c.cfg.Relogin && isInvalidLoginToken(err) {
			_ = clearStoredSessionToken(c.store)
			c.cfg.Token = ""
			continue
		}
		var nre *NonRecoverableError
		if errors.As(err, &nre) {
			return err
		}

		if !c.cfg.Reconnect {
			return err
		}

		delay := c.cfg.ReconnectDelay
		if delay <= 0 {
			delay = 3 * time.Second
		}

		select {
		case <-c.ctx.Done():
			return nil
		case <-time.After(delay):
			// retry reconnecting
		}
	}
}

// clearStoredSessionToken supports both the current ExtendedStore contract and
// legacy custom Store implementations. Keeping a token-less record preserves
// the device identity while guaranteeing that the next run authenticates.
func clearStoredSessionToken(store session.Store) error {
	info, err := store.LoadSession()
	if err != nil || info == nil {
		return err
	}
	if extended, ok := store.(session.ExtendedStore); ok {
		return extended.DeleteSession(info.Token)
	}
	info.Token = ""
	return store.SaveSession(info)
}

func (c *Client) runSession(ctx context.Context) error {
	tcpOpts := transport.DefaultTCPOptions()
	if c.cfg.Host != "" {
		tcpOpts.Host = c.cfg.Host
	}
	if c.cfg.Port != 0 {
		tcpOpts.Port = c.cfg.Port
	}
	tcpOpts.UseSSL = c.cfg.UseSSL
	tcpOpts.ProxyURL = c.cfg.Proxy

	tcpTransport := transport.NewTCPTransport(tcpOpts)
	reader := connection.NewTCPReader(tcpTransport)

	tcpProto, err := protocol.NewTcpProtocolVersion(c.cfg.ProtocolVersion)
	if err != nil {
		return fmt.Errorf("init protocol failed: %w", err)
	}

	disconnectCh := make(chan error, 1)

	connManager := connection.NewConnectionManager(
		reader,
		tcpTransport,
		tcpProto,
		&connection.Config{
			Interactive:    c.cfg.Interactive,
			RequestTimeout: c.cfg.RequestTimeout,
		},
		func(err error) {
			select {
			case disconnectCh <- err:
			default:
			}
		},
		func(event *protocol.InboundFrame) {
			c.handleEvent(event)
		},
	)

	c.mu.Lock()
	c.conn = connManager
	c.mu.Unlock()

	c.logf("Connecting to Max server at %s:%d (SSL=%v)...", tcpOpts.Host, tcpOpts.Port, tcpOpts.UseSSL)
	if err := connManager.Start(ctx); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	c.logf("Connected to Max server successfully")

	if c.fpGen == nil {
		c.fpGen = fingerprint.NewFingerprintGenerator(fingerprint.DefaultFingerprint())
	}
	if c.fpGen.VersionName() != "" && c.fpGen.VersionName() != c.cfg.AppVersion {
		return nonRecoverable(fmt.Errorf("fingerprint version %s does not match configured app version %s", c.fpGen.VersionName(), c.cfg.AppVersion))
	}

	// Load session info to reuse deviceId and mt_instanceid if previously saved
	sessInfo, err := c.store.LoadSession()
	if err != nil {
		_ = connManager.Close()
		return nonRecoverable(fmt.Errorf("failed to load session: %w", err))
	}

	if sessInfo != nil {
		restoreSessionUserAgent(c.cfg, sessInfo.UserAgent)
		if c.cfg.DeviceID == "" && sessInfo.DeviceID != "" {
			c.cfg.DeviceID = sessInfo.DeviceID
		}
		if c.cfg.MtInstanceID == "" && sessInfo.MTInstanceID != "" {
			c.cfg.MtInstanceID = sessInfo.MTInstanceID
		}
		if c.cfg.Phone == "" && sessInfo.Phone != "" {
			c.cfg.Phone = sessInfo.Phone
		}
	}
	if c.cfg.DeviceID == "" {
		c.cfg.DeviceID = randomHex(8)
	}
	if c.cfg.MtInstanceID == "" {
		c.cfg.MtInstanceID = randomHex(8)
	}

	// 1. Session Init Handshake
	clientSessionID := c.cfg.ClientSessionID
	if clientSessionID <= 0 {
		clientSessionID = randomClientSessionID()
		c.cfg.ClientSessionID = clientSessionID
	}

	userAgent := mobileUserAgent(c.cfg)

	initPayload := map[string]interface{}{
		"mt_instanceid":   c.cfg.MtInstanceID,
		"userAgent":       userAgent,
		"clientSessionId": clientSessionID,
		"deviceId":        c.cfg.DeviceID,
	}

	c.logf("Sending mobile session handshake (SESSION_INIT, deviceId=%s, mt_instanceid=%s, clientSessionId=%d)...",
		c.cfg.DeviceID, c.cfg.MtInstanceID, clientSessionID)
	initRes, err := c.Invoke(ctx, protocol.OpSessionInit, initPayload)
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("session init failed: %w", err)
	}

	// Validate response does not contain an API validation error
	if errVal, ok := initRes["error"]; ok && errVal != nil {
		_ = connManager.Close()
		msg := initRes["message"]
		if msg == nil {
			msg = initRes["localizedMessage"]
		}
		return nonRecoverable(fmt.Errorf("session init validation error from server: %v (message: %v)", errVal, msg))
	}
	if errVal, ok := initRes["err"]; ok && errVal != nil {
		_ = connManager.Close()
		return nonRecoverable(fmt.Errorf("session init error from server: %v", errVal))
	}

	var callsSeed int64
	if cs, ok := extractInt64(initRes["callsSeed"]); ok {
		callsSeed = cs
	} else if cs, ok := extractInt64(initRes["calls_seed"]); ok {
		callsSeed = cs
	}

	c.mu.Lock()
	c.callsSeed = callsSeed
	c.mu.Unlock()

	c.logf("Handshake completed successfully (callsSeed=%d)", callsSeed)

	// 2. Load or execute Auth
	token := c.cfg.Token
	if token == "" && sessInfo != nil {
		token = sessInfo.Token
	}

	if token == "" {
		if c.cfg.Phone == "" {
			_ = connManager.Close()
			return nonRecoverable(errors.New("phone number required for initial authentication"))
		}
		c.logf("No active session token found; starting SMS authentication for %s...", c.cfg.Phone)
		smsFlow := c.cfg.AuthFlow
		if smsFlow == nil {
			smsFlow = auth.NewSmsAuthFlow(nil, nil)
		}
		if smsFlow.CodeProvider == nil {
			smsFlow.CodeProvider = &auth.ConsoleCodeProvider{}
		}
		if smsFlow.PasswordProvider == nil {
			smsFlow.PasswordProvider = &auth.ConsolePasswordProvider{}
		}
		smsFlow.DeviceID = c.cfg.DeviceID
		smsFlow.CallsSeed = callsSeed
		if smsFlow.FpGen == nil {
			smsFlow.FpGen = c.fpGen
		}
		if smsFlow.Arch == "" {
			smsFlow.Arch = c.cfg.Arch
		}
		smsFlow.PasswordMaxAttempts = c.cfg.PasswordMaxAttempts
		smsFlow.LogLevel = c.cfg.LogLevel

		authRes, err := smsFlow.Authenticate(ctx, c, c.cfg.Phone)
		if err != nil {
			_ = connManager.Close()
			return nonRecoverable(fmt.Errorf("authentication failed: %w", err))
		}
		token = authRes.Token
		if authRes.IsRegister {
			if c.cfg.Registration == nil || c.cfg.Registration.FirstName == "" {
				_ = connManager.Close()
				return nonRecoverable(errors.New("registration profile is required for a registration token"))
			}
			registration, regErr := c.Auth.ConfirmRegistration(ctx,
				c.cfg.Registration.FirstName, c.cfg.Registration.LastName, token)
			if regErr != nil {
				_ = connManager.Close()
				return nonRecoverable(regErr)
			}
			if registeredToken, ok := registration["token"].(string); ok && registeredToken != "" {
				token = registeredToken
			}
		}
		if err := c.store.SaveSession(&session.SessionInfo{
			Token:        token,
			Phone:        c.cfg.Phone,
			DeviceID:     c.cfg.DeviceID,
			MTInstanceID: c.cfg.MtInstanceID,
			UserAgent:    sessionUserAgent(c.cfg),
		}); err != nil {
			_ = connManager.Close()
			return nonRecoverable(fmt.Errorf("save authenticated session: %w", err))
		}
		c.logf("Session credentials saved to store")
	} else {
		c.logf("Resuming session for device %s...", c.cfg.DeviceID)
	}

	// 3. Login
	c.logf("Logging in with session token...")
	syncState := session.SyncState{
		ChatsSync:    -1,
		ContactsSync: -1,
		DraftsSync:   -1,
		PresenceSync: -1,
		ConfigHash:   session.DefaultConfigHash,
	}
	if sessInfo != nil {
		syncState = sessInfo.Sync
		if syncState.ConfigHash == "" {
			syncState.ConfigHash = session.DefaultConfigHash
		}
	}
	syncState = resolveSync(syncState, c.cfg.Sync)
	loginPayload := map[string]interface{}{
		"token":        token,
		"deviceId":     c.cfg.DeviceID,
		"userAgent":    userAgent,
		"chatsSync":    syncState.ChatsSync,
		"contactsSync": syncState.ContactsSync,
		"presenceSync": syncState.PresenceSync,
		"draftsSync":   syncState.DraftsSync,
		"interactive":  c.cfg.Interactive,
		// These fields are part of PyMax's mobile SyncPayload. In particular,
		// exp.chatsCountGroups is a small MessagePack binary value, not a map.
		"exp":        map[string]interface{}{"chatsCountGroups": []byte{0x0a, 0x32}},
		"configHash": syncState.ConfigHash,
	}
	if callsSeed != 0 {
		fp, fpErr := c.fpGen.GenerateFingerprint(c.cfg.DeviceID, callsSeed, c.cfg.Arch)
		if fpErr != nil {
			return nonRecoverable(fpErr)
		}
		if len(fp) > 0 {
			loginPayload["chatCacheFingerprint"] = fp
		}
	}

	loginRes, err := c.Invoke(ctx, protocol.OpLogin, loginPayload)
	if err != nil {
		_ = connManager.Close()
		return fmt.Errorf("login failed: %w", err)
	}

	// Validate login response for server rejection
	if errVal, ok := loginRes["error"]; ok && errVal != nil {
		_ = connManager.Close()
		msg := loginRes["message"]
		if msg == nil {
			msg = loginRes["localizedMessage"]
		}
		return nonRecoverable(fmt.Errorf("login rejected by server: %v (message: %v)", errVal, msg))
	}

	if flags, ok := loginRes["login2Flags"].(map[string]interface{}); ok {
		configEnabled, _ := flags["configEnabled"].(bool)
		contactEnabled, _ := flags["contactEnabled"].(bool)
		profileEnabled, _ := flags["profileEnabled"].(bool)
		if configEnabled || contactEnabled || profileEnabled {
			contactsSync := int64(-1)
			if contactEnabled {
				contactsSync = syncState.ContactsSync
			}
			login2Res, login2Err := c.Invoke(ctx, protocol.OpLogin2, map[string]interface{}{
				"needProfile": profileEnabled, "contactsSync": contactsSync, "configHash": syncState.ConfigHash,
			})
			if login2Err != nil {
				_ = connManager.Close()
				return fmt.Errorf("login2 failed: %w", login2Err)
			}
			for key, value := range login2Res {
				loginRes[key] = value
			}
		}
	}

	// If token refreshed
	if newToken, ok := loginRes["token"].(string); ok && newToken != "" && newToken != token {
		oldToken := token
		token = newToken
		if err := c.store.UpdateToken(oldToken, token); err != nil {
			return fmt.Errorf("persist refreshed token: %w", err)
		}
	}
	c.cfg.Token = token
	if syncTime, ok := extractInt64(loginRes["time"]); ok {
		syncState.ChatsSync = syncTime
		syncState.ContactsSync = syncTime
		syncState.DraftsSync = syncTime
		syncState.PresenceSync = syncTime
	}
	if config, ok := loginRes["config"].(map[string]interface{}); ok {
		if hash := types.StringValue(config["hash"]); hash != "" {
			syncState.ConfigHash = hash
		}
	}
	if err := c.store.SaveSession(&session.SessionInfo{
		Token:        token,
		Phone:        c.cfg.Phone,
		DeviceID:     c.cfg.DeviceID,
		MTInstanceID: c.cfg.MtInstanceID,
		UserAgent:    sessionUserAgent(c.cfg),
		Sync:         syncState,
	}); err != nil {
		return fmt.Errorf("save login session: %w", err)
	}

	// Resolve the initial state off-lock, then publish it atomically. Event
	// delivery starts with the connection manager, so it may overlap the tail
	// of login on a fast server.
	me := &types.User{}
	if profileData, ok := loginRes["profile"].(map[string]interface{}); ok {
		parsed := types.ParseUserPayload(profileData)
		parsed.Bind(c.Users)
		me = &parsed
	} else if userMap, ok := loginRes["user"].(map[string]interface{}); ok {
		parsed := types.ParseUserPayload(userMap)
		parsed.Bind(c.Users)
		me = &parsed
	}
	for _, key := range []string{"contacts", "contactInfos"} {
		if rawUsers, ok := loginRes[key].([]interface{}); ok {
			parsed := make([]types.User, 0, len(rawUsers))
			for _, raw := range rawUsers {
				if m, ok := raw.(map[string]interface{}); ok {
					user := types.ParseUserPayload(m)
					user.Bind(c.Users)
					parsed = append(parsed, user)
				}
			}
			c.Users.SeedCache(parsed)
		}
	}
	chatCache := make(map[int64]types.Chat)
	if rawChats, ok := loginRes["chats"].([]interface{}); ok {
		parsedChats := make([]types.Chat, 0, len(rawChats))
		for _, raw := range rawChats {
			if m, ok := raw.(map[string]interface{}); ok {
				chat := types.ParseChatPayload(m)
				chat.Bind(c.Messages, c.Chats)
				chatCache[chat.ID] = chat
				parsedChats = append(parsedChats, chat)
			}
		}
		c.Chats.SeedCache(parsedChats)
	}
	messageCache := make(map[int64][]types.Message)
	if rawMessages, ok := loginRes["messages"].(map[string]interface{}); ok {
		for rawChatID, value := range rawMessages {
			chatID, _ := types.Int64Value(rawChatID)
			if list, ok := value.([]interface{}); ok {
				for _, raw := range list {
					if m, ok := raw.(map[string]interface{}); ok {
						msg := types.ParseMessagePayload(m)
						msg.Bind(c.Messages)
						if msg.ChatID == 0 {
							msg.ChatID = chatID
						}
						messageCache[chatID] = append(messageCache[chatID], *msg)
					}
				}
			}
		}
	}
	c.mu.Lock()
	c.Me = me
	c.ChatCache = chatCache
	c.MessageCache = messageCache
	c.mu.Unlock()

	c.logf("Login successful! Logged in as '%s' (ID: %d)", me.FirstName, me.ID)
	if c.cfg.Telemetry {
		c.Telemetry.Start(ctx, clientSessionID)
		defer c.Telemetry.Stop()
	}

	// Dispatch OnStart hooks
	c.router.DispatchStart(ctx)

	// Await disconnect or cancellation
	select {
	case <-ctx.Done():
		_ = connManager.Close()
		c.router.DispatchDisconnect(ctx, ctx.Err())
		return ctx.Err()
	case disErr := <-disconnectCh:
		_ = connManager.Close()
		c.router.DispatchDisconnect(ctx, disErr)
		return disErr
	}
}

// parseMessage extracts a full types.Message from a raw payload map.
// msgPayload may be the top-level frame payload or a nested "message" sub-object.
func (c *Client) parseMessage(payload map[string]interface{}) *types.Message {
	msg := types.ParseMessagePayload(payload).Bind(c.Messages)
	if msg.Time == 0 {
		msg.Time = time.Now().UnixMilli()
	}

	// Mark as outgoing if sender matches logged-in user
	c.mu.RLock()
	me := c.Me
	c.mu.RUnlock()
	if me != nil && msg.SenderID != 0 && msg.SenderID == me.ID {
		msg.IsOutgoing = true
	}

	return msg
}

func isInvalidLoginToken(err error) bool {
	var apiErr *protocol.ApiError
	if !errors.As(err, &apiErr) || apiErr.Opcode != protocol.OpLogin {
		return false
	}
	return apiErr.ErrorStr == "FAIL_LOGIN_TOKEN" || apiErr.ErrorStr == "FAIL_LOGOUT_ALL" || apiErr.Message == "FAIL_LOGIN_TOKEN" || apiErr.Message == "FAIL_LOGOUT_ALL"
}

// handleEvent dispatches inbound server push frames to registered handlers.
func (c *Client) handleEvent(frame *protocol.InboundFrame) {
	ctx := c.ctx
	if ctx == nil {
		return
	}
	defer c.router.DispatchEvent(ctx, &types.RawEvent{Type: frame.Opcode.String(), Opcode: uint16(frame.Opcode), Payload: frame.Payload})

	switch frame.Opcode {
	case protocol.OpNotifAttach:
		c.Uploads.NotifyReady(frame.Payload)

	case protocol.OpNotifMark:
		ev := &types.MessageReadEvent{}
		ev.SetAsUnread, _ = frame.Payload["setAsUnread"].(bool)
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		ev.MessageID, _ = extractInt64(frame.Payload["messageId"])
		ev.Mark, _ = extractInt64(frame.Payload["mark"])
		c.router.DispatchMessageRead(ctx, ev)

	case protocol.OpNotifContact:
		user := types.ParseUserPayload(frame.Payload)
		user.Bind(c.Users)
		c.Users.SeedCache([]types.User{user})
		c.router.DispatchUserUpdate(ctx, &types.UserUpdateEvent{User: user})

	// Incoming new message
	case protocol.OpNotifMessage:
		msg := c.parseMessage(frame.Payload)
		if msg.Status == types.MessageStatusEdited {
			c.router.DispatchMessageEdit(ctx, msg)
		} else if msg.Status == types.MessageStatusRemoved {
			c.router.DispatchMessageDeleteEvent(ctx, &types.MessageDeleteEvent{ChatID: msg.ChatID, MessageIDs: []int64{msg.ID}, Message: msg})
		} else if msg.ChatID != 0 || msg.ID != 0 {
			c.router.DispatchMessage(ctx, msg)
		}

	// Bulk history push during sync
	case protocol.OpChatHistory:
		if msgList, ok := frame.Payload["messages"].([]interface{}); ok {
			for _, item := range msgList {
				if mData, ok := item.(map[string]interface{}); ok {
					msg := c.parseMessage(mData)
					c.router.DispatchMessage(ctx, msg)
				}
			}
		} else {
			// Single message in history event
			msg := c.parseMessage(frame.Payload)
			if msg.ID != 0 {
				c.router.DispatchMessage(ctx, msg)
			}
		}

	// Message edited
	case protocol.OpMsgEdit:
		msg := c.parseMessage(frame.Payload)
		c.router.DispatchMessageEdit(ctx, msg)

	// Single message deleted
	case protocol.OpMsgDelete, protocol.OpNotifMsgDelete:
		c.router.DispatchMessageDeleteEvent(ctx, types.ParseMessageDeletePayload(frame.Payload))

	// Reaction changed on a message
	case protocol.OpNotifMsgReactionsChanged, protocol.OpNotifMsgYouReacted:
		ev := &types.ReactionEvent{}
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.MessageID, _ = extractInt64(frame.Payload["messageId"])
		ev.MessageIDRaw = types.StringValue(frame.Payload["messageId"])
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if rData, ok := frame.Payload["reaction"].(map[string]interface{}); ok {
			if id, ok := rData["id"].(string); ok {
				ev.Reaction = id
			}
		} else if r, ok := frame.Payload["reaction"].(string); ok {
			ev.Reaction = r
		}
		if removed, ok := frame.Payload["removed"].(bool); ok {
			ev.Removed = removed
		}
		if total, ok := extractInt64(frame.Payload["totalCount"]); ok {
			ev.TotalCount = int(total)
		}
		if counters, ok := frame.Payload["counters"].([]interface{}); ok {
			for _, item := range counters {
				if m, ok := item.(map[string]interface{}); ok {
					count, _ := extractInt64(m["count"])
					ev.Counters = append(ev.Counters, types.ReactionCounter{Reaction: types.StringValue(m["reaction"]), Count: int(count)})
				}
			}
		}
		c.router.DispatchReaction(ctx, ev)

	// Chat metadata updated
	case protocol.OpNotifChat:
		chat := types.ParseChatPayload(frame.Payload)
		chat.Bind(c.Messages, c.Chats)
		if chat.ID != 0 {
			c.mu.Lock()
			c.ChatCache[chat.ID] = chat
			c.mu.Unlock()
		}
		c.router.DispatchChatUpdate(ctx, &chat)

	// User presence / online status changed
	case protocol.OpNotifPresence:
		ev := &types.PresenceEvent{}
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if p, ok := frame.Payload["presence"].(map[string]interface{}); ok {
			ev.Presence = &types.Presence{Status: types.StringValue(p["status"])}
			ev.Presence.StatusCode, _ = extractInt64(p["status"])
			ev.Presence.Seen, _ = extractInt64(p["seen"])
			ev.Presence.LastSeen, _ = extractInt64(p["lastSeen"])
			ev.Presence.Online, _ = p["online"].(bool)
			ev.Online = ev.Presence.Online || ev.Presence.Status == "online" || ev.Presence.Status == "ONLINE"
		}
		if online, ok := frame.Payload["online"].(bool); ok {
			ev.Online = online
		} else if status, ok := frame.Payload["status"].(string); ok {
			ev.Online = (status == "online" || status == "ONLINE")
		}
		c.router.DispatchPresence(ctx, ev)

	// User typing indicator
	case protocol.OpNotifTyping:
		ev := &types.TypingEvent{}
		ev.ChatID, _ = extractInt64(frame.Payload["chatId"])
		ev.UserID, _ = extractInt64(frame.Payload["userId"])
		if ev.UserID == 0 {
			ev.UserID, _ = extractInt64(frame.Payload["sender"])
		}
		c.router.DispatchTyping(ctx, ev)

	// Any other server push event — route as raw
	default:
	}
}

// Close disconnects and terminates client session.
func (c *Client) Close() error {
	if c.Telemetry != nil {
		c.Telemetry.Stop()
	}
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = false
	cancel, conn := c.cancel, c.conn
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	var errs []error
	if conn != nil {
		if err := conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if store, ok := c.store.(session.ExtendedStore); ok {
		if err := store.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
