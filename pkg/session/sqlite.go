package session

import (
	"database/sql"
	"encoding/json"
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
	CREATE TABLE IF NOT EXISTS max_sessions_v2 (
		token TEXT PRIMARY KEY,
		phone TEXT NOT NULL DEFAULT '',
		device_id TEXT NOT NULL DEFAULT '',
		mt_instance_id TEXT NOT NULL DEFAULT '',
		chats_sync INTEGER NOT NULL DEFAULT -1,
		contacts_sync INTEGER NOT NULL DEFAULT -1,
		drafts_sync INTEGER NOT NULL DEFAULT -1,
		presence_sync INTEGER NOT NULL DEFAULT -1,
		config_hash TEXT NOT NULL DEFAULT '',
		user_agent TEXT
	);`
	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("create max_sessions table failed: %w", err)
	}
	// Import legacy phone-keyed rows once, without deleting the old table.
	_, _ = db.Exec(`INSERT OR IGNORE INTO max_sessions_v2
		(token, phone, device_id, mt_instance_id, chats_sync, contacts_sync, drafts_sync, presence_sync, config_hash)
		SELECT token, phone, device_id, mt_instance_id, chats_sync, contacts_sync, drafts_sync, presence_sync, config_hash FROM max_sessions`)

	return &SqliteStore{db: db}, nil
}

// SaveSession inserts or updates session credentials in SQLite.
func (s *SqliteStore) SaveSession(info *SessionInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userAgent, err := json.Marshal(info.UserAgent)
	if err != nil {
		return err
	}
	query := `
	INSERT INTO max_sessions_v2 (token, phone, device_id, mt_instance_id, chats_sync, contacts_sync, drafts_sync, presence_sync, config_hash, user_agent)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(token) DO UPDATE SET
		phone = excluded.phone,
		device_id = excluded.device_id,
		mt_instance_id = excluded.mt_instance_id,
		chats_sync = excluded.chats_sync,
		contacts_sync = excluded.contacts_sync,
		drafts_sync = excluded.drafts_sync,
		presence_sync = excluded.presence_sync,
		config_hash = excluded.config_hash,
		user_agent = excluded.user_agent;`

	_, err = s.db.Exec(query,
		info.Token,
		info.Phone,
		info.DeviceID,
		info.MTInstanceID,
		info.Sync.ChatsSync,
		info.Sync.ContactsSync,
		info.Sync.DraftsSync,
		info.Sync.PresenceSync,
		info.Sync.ConfigHash,
		string(userAgent),
	)
	return err
}

// LoadSession loads the most recent active session from SQLite.
func (s *SqliteStore) LoadSession() (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT COALESCE(phone, ''), COALESCE(token, ''), COALESCE(device_id, ''), COALESCE(mt_instance_id, ''), COALESCE(chats_sync, -1), COALESCE(contacts_sync, -1), COALESCE(drafts_sync, -1), COALESCE(presence_sync, -1), COALESCE(config_hash, ''), user_agent FROM max_sessions_v2 LIMIT 1;`
	row := s.db.QueryRow(query)

	var info SessionInfo
	var rawUserAgent sql.NullString
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
		&rawUserAgent,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if rawUserAgent.Valid && rawUserAgent.String != "" && rawUserAgent.String != "null" {
		_ = json.Unmarshal([]byte(rawUserAgent.String), &info.UserAgent)
	}
	return &info, nil
}

// UpdateToken rotates an authorization token while preserving session state.
func (s *SqliteStore) UpdateToken(oldToken, newToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec(`UPDATE max_sessions_v2 SET token = ? WHERE token = ?;`, newToken, oldToken)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return fmt.Errorf("session: token to replace was not found")
	}
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
	row := s.db.QueryRow(`SELECT COALESCE(phone, ''), COALESCE(token, ''), COALESCE(device_id, ''), COALESCE(mt_instance_id, ''), COALESCE(chats_sync, -1), COALESCE(contacts_sync, -1), COALESCE(drafts_sync, -1), COALESCE(presence_sync, -1), COALESCE(config_hash, ''), user_agent FROM max_sessions_v2 WHERE `+where+` LIMIT 1`, arg)
	var info SessionInfo
	var rawUserAgent sql.NullString
	if err := row.Scan(&info.Phone, &info.Token, &info.DeviceID, &info.MTInstanceID, &info.Sync.ChatsSync, &info.Sync.ContactsSync, &info.Sync.DraftsSync, &info.Sync.PresenceSync, &info.Sync.ConfigHash, &rawUserAgent); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if rawUserAgent.Valid && rawUserAgent.String != "" && rawUserAgent.String != "null" {
		_ = json.Unmarshal([]byte(rawUserAgent.String), &info.UserAgent)
	}
	return &info, nil
}

// DeleteSession deletes one session by its token.
func (s *SqliteStore) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM max_sessions_v2 WHERE token = ?`, token)
	return err
}

// DeleteAllSessions removes all persisted sessions.
func (s *SqliteStore) DeleteAllSessions() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM max_sessions_v2`)
	return err
}

// Close closes the database owned by the caller. It is provided for Store
// symmetry; callers should not reuse the *sql.DB after calling it.
func (s *SqliteStore) Close() error { return s.db.Close() }
