package session

import (
	"database/sql"
	"fmt"
	"sync"
)

// SqliteStore persists session data using an SQL/SQLite database connection.
type SqliteStore struct {
	mu sync.RWMutex
	db *sql.DB
}

// NewSqliteStore wraps an existing *sql.DB connection and ensures table schema exists.
func NewSqliteStore(db *sql.DB) (*SqliteStore, error) {
	query := `
	CREATE TABLE IF NOT EXISTS max_sessions (
		phone TEXT PRIMARY KEY,
		token TEXT NOT NULL,
		device_id TEXT,
		mt_instance_id TEXT,
		chats_sync INTEGER,
		contacts_sync INTEGER,
		drafts_sync INTEGER,
		presence_sync INTEGER,
		config_hash TEXT
	);`
	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("create max_sessions table failed: %w", err)
	}

	return &SqliteStore{db: db}, nil
}

// SaveSession inserts or updates session credentials in SQLite.
func (s *SqliteStore) SaveSession(info *SessionInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT INTO max_sessions (phone, token, device_id, mt_instance_id, chats_sync, contacts_sync, drafts_sync, presence_sync, config_hash)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(phone) DO UPDATE SET
		token = excluded.token,
		device_id = excluded.device_id,
		mt_instance_id = excluded.mt_instance_id,
		chats_sync = excluded.chats_sync,
		contacts_sync = excluded.contacts_sync,
		drafts_sync = excluded.drafts_sync,
		presence_sync = excluded.presence_sync,
		config_hash = excluded.config_hash;`

	_, err := s.db.Exec(query,
		info.Phone,
		info.Token,
		info.DeviceID,
		info.MTInstanceID,
		info.Sync.ChatsSync,
		info.Sync.ContactsSync,
		info.Sync.DraftsSync,
		info.Sync.PresenceSync,
		info.Sync.ConfigHash,
	)
	return err
}

// LoadSession loads the most recent active session from SQLite.
func (s *SqliteStore) LoadSession() (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT phone, token, device_id, mt_instance_id, chats_sync, contacts_sync, drafts_sync, presence_sync, config_hash FROM max_sessions LIMIT 1;`
	row := s.db.QueryRow(query)

	var info SessionInfo
	err := row.Scan(
		&info.Phone,
		&info.Token,
		&info.DeviceID,
		&info.MTInstanceID,
		&info.Sync.ChatsSync,
		&info.Sync.ContactsSync,
		&info.Sync.DraftsSync,
		&info.Sync.PresenceSync,
		&info.Sync.ConfigHash,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

// UpdateToken updates the authorization token for the specified phone number.
func (s *SqliteStore) UpdateToken(phone, newToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `UPDATE max_sessions SET token = ? WHERE phone = ?;`
	_, err := s.db.Exec(query, newToken, phone)
	return err
}

// LoadSessionByDeviceID loads a session associated with a device ID.
func (s *SqliteStore) LoadSessionByDeviceID(deviceID string) (*SessionInfo, error) {
	return s.loadWhere("device_id = ?", deviceID)
}

// LoadSessionByPhone loads a session associated with a phone number.
func (s *SqliteStore) LoadSessionByPhone(phone string) (*SessionInfo, error) {
	return s.loadWhere("phone = ?", phone)
}

func (s *SqliteStore) loadWhere(where string, arg interface{}) (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row := s.db.QueryRow(`SELECT phone, token, device_id, mt_instance_id, chats_sync, contacts_sync, drafts_sync, presence_sync, config_hash FROM max_sessions WHERE `+where+` LIMIT 1`, arg)
	var info SessionInfo
	if err := row.Scan(&info.Phone, &info.Token, &info.DeviceID, &info.MTInstanceID, &info.Sync.ChatsSync, &info.Sync.ContactsSync, &info.Sync.DraftsSync, &info.Sync.PresenceSync, &info.Sync.ConfigHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

// DeleteSession deletes one session by its token.
func (s *SqliteStore) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM max_sessions WHERE token = ?`, token)
	return err
}

// DeleteAllSessions removes all persisted sessions.
func (s *SqliteStore) DeleteAllSessions() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM max_sessions`)
	return err
}

// Close closes the database owned by the caller. It is provided for Store
// symmetry; callers should not reuse the *sql.DB after calling it.
func (s *SqliteStore) Close() error { return s.db.Close() }
