package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

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

// InMemoryStore stores session in RAM.
type InMemoryStore struct {
	mu      sync.RWMutex
	session *SessionInfo
}

// NewInMemoryStore creates an in-memory session store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{}
}

func (s *InMemoryStore) SaveSession(info *SessionInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = info
	return nil
}

func (s *InMemoryStore) LoadSession() (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.session, nil
}

func (s *InMemoryStore) UpdateToken(phone, newToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		s.session.Token = newToken
	}
	return nil
}

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
