package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// DefaultConfigHash is the server's initial configuration marker used by
// PyMax when no configuration has been synchronized yet.
const DefaultConfigHash = "00000000-0000000000000000-00000000-0000000000000000-0000000000000000-0-0000000000000000-00000000"

// SyncState holds incremental sync tokens.
type SyncState struct {
	ChatsSync    int64  `json:"chats_sync"`
	ContactsSync int64  `json:"contacts_sync"`
	DraftsSync   int64  `json:"drafts_sync"`
	PresenceSync int64  `json:"presence_sync"`
	ConfigHash   string `json:"config_hash"`
}

// UserAgentPayload stores client device headers.
type UserAgentPayload struct {
	DeviceType  string `json:"device_type"`
	AppVersion  string `json:"app_version"`
	BuildNumber int    `json:"build_number"`
	OSVersion   string `json:"os_version"`
	DeviceName  string `json:"device_name"`
}

// SessionInfo stores persisted credentials and session state.
type SessionInfo struct {
	Token        string            `json:"token"`
	DeviceID     string            `json:"device_id"`
	Phone        string            `json:"phone"`
	MTInstanceID string            `json:"mt_instance_id,omitempty"`
	UserAgent    *UserAgentPayload `json:"user_agent,omitempty"`
	Sync         SyncState         `json:"sync"`
}

// Store defines persistence operations for session state.
type Store interface {
	SaveSession(info *SessionInfo) error
	LoadSession() (*SessionInfo, error)
	UpdateToken(phone, newToken string) error
}

// ExtendedStore is the optional full session-store contract implemented by
// the built-in stores. Store remains small so custom stores written against
// earlier GoMax versions continue to work.
type ExtendedStore interface {
	Store
	LoadSessionByDeviceID(deviceID string) (*SessionInfo, error)
	LoadSessionByPhone(phone string) (*SessionInfo, error)
	DeleteSession(token string) error
	DeleteAllSessions() error
	Close() error
}

// InMemoryStore stores session in RAM.
type InMemoryStore struct {
	mu      sync.RWMutex
	session *SessionInfo
}

// NewInMemoryStore creates an in-memory session store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{}
}

// SaveSession replaces the in-memory session.
func (s *InMemoryStore) SaveSession(info *SessionInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = info
	return nil
}

// LoadSession returns the in-memory session, or nil when none is saved.
func (s *InMemoryStore) LoadSession() (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.session, nil
}

// UpdateToken changes the token of the in-memory session.
func (s *InMemoryStore) UpdateToken(phone, newToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		s.session.Token = newToken
	}
	return nil
}

// LoadSessionByDeviceID returns the session for deviceID.
func (s *InMemoryStore) LoadSessionByDeviceID(deviceID string) (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.session == nil || s.session.DeviceID != deviceID {
		return nil, nil
	}
	copy := *s.session
	return &copy, nil
}

// LoadSessionByPhone returns the session for phone.
func (s *InMemoryStore) LoadSessionByPhone(phone string) (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.session == nil || s.session.Phone != phone {
		return nil, nil
	}
	copy := *s.session
	return &copy, nil
}

// DeleteSession removes the in-memory session when token matches.
func (s *InMemoryStore) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil && (token == "" || s.session.Token == token) {
		s.session = nil
	}
	return nil
}

// DeleteAllSessions removes the in-memory session.
func (s *InMemoryStore) DeleteAllSessions() error { return s.DeleteSession("") }

// Close releases no resources for the in-memory store.
func (s *InMemoryStore) Close() error             { return nil }

// FileStore persists session data to a JSON file.
type FileStore struct {
	mu       sync.RWMutex
	filePath string
}

// NewFileStore creates a file-backed session store in workDir with sessionName.
func NewFileStore(workDir, sessionName string) *FileStore {
	if sessionName == "" {
		sessionName = "session.json"
	}
	return &FileStore{
		filePath: filepath.Join(workDir, sessionName),
	}
}

// SaveSession writes the session as a private JSON file.
func (s *FileStore) SaveSession(info *SessionInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0600)
}

// LoadSession reads the JSON session file, returning nil when it does not exist.
func (s *FileStore) LoadSession() (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var info SessionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// UpdateToken updates the token in the JSON session file.
func (s *FileStore) UpdateToken(phone, newToken string) error {
	info, err := s.LoadSession()
	if err != nil {
		return err
	}
	if info == nil {
		info = &SessionInfo{Phone: phone}
	}
	info.Token = newToken
	return s.SaveSession(info)
}

// LoadSessionByDeviceID reads the JSON session when its device ID matches.
func (s *FileStore) LoadSessionByDeviceID(deviceID string) (*SessionInfo, error) {
	info, err := s.LoadSession()
	if err != nil || info == nil || info.DeviceID != deviceID {
		return nil, err
	}
	return info, nil
}

// LoadSessionByPhone reads the JSON session when its phone matches.
func (s *FileStore) LoadSessionByPhone(phone string) (*SessionInfo, error) {
	info, err := s.LoadSession()
	if err != nil || info == nil || info.Phone != phone {
		return nil, err
	}
	return info, nil
}

// DeleteSession removes the JSON session when token matches.
func (s *FileStore) DeleteSession(token string) error {
	info, err := s.LoadSession()
	if err != nil || info == nil || (token != "" && info.Token != token) {
		if info == nil && err == nil {
			return nil
		}
		return err
	}
	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DeleteAllSessions removes the JSON session file.
func (s *FileStore) DeleteAllSessions() error { return s.DeleteSession("") }

// Close releases no resources for the file store.
func (s *FileStore) Close() error             { return nil }
