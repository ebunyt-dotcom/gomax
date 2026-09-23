package users

import (
	"context"
	"fmt"
	"sync"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// UserService handles user profiles, contacts, and searches.
type UserService struct {
	invoker api.Invoker
	mu      sync.RWMutex
	cache   map[int64]types.User
}

// NewUserService creates a new UserService instance.
func NewUserService(invoker api.Invoker) *UserService {
	return &UserService{invoker: invoker, cache: make(map[int64]types.User)}
}

// GetUser retrieves detailed user profile information by ID.
func (s *UserService) GetUser(ctx context.Context, userID int64) (*types.User, error) {
	if cached, _ := s.GetCachedUser(ctx, userID); cached != nil {
		return cached, nil
	}
	users, err := s.FetchUsers(ctx, []int64{userID})
	if err != nil {
		return nil, fmt.Errorf("get user failed: %w", err)
	}
	if len(users) == 0 {
		return nil, nil
	}
	return &users[0], nil
}

// GetCachedUser returns a cached profile without performing network I/O.
func (s *UserService) GetCachedUser(ctx context.Context, userID int64) (*types.User, error) {
	_ = ctx
	s.mu.RLock()
	user, ok := s.cache[userID]
	s.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	copy := user
	return &copy, nil
}

// FetchUsers is the uncached batch lookup counterpart to GetUsers.
func (s *UserService) FetchUsers(ctx context.Context, userIDs []int64) ([]types.User, error) {
	if len(userIDs) == 0 {
		return []types.User{}, nil
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpContactInfo, map[string]interface{}{
		"contactIds": userIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch users failed: %w", err)
	}
	users := s.usersFromResponse(res)
	s.cacheUsers(users)
	return users, nil
}

// GetUsers retrieves multiple user profiles in batch.
func (s *UserService) GetUsers(ctx context.Context, userIDs []int64) ([]types.User, error) {
	result := make([]types.User, 0, len(userIDs))
	missing := make([]int64, 0)
	s.mu.RLock()
	for _, id := range userIDs {
		if user, ok := s.cache[id]; ok {
			result = append(result, user)
		} else {
			missing = append(missing, id)
		}
	}
	s.mu.RUnlock()
	if len(missing) > 0 {
		fetched, err := s.FetchUsers(ctx, missing)
		if err != nil {
			return nil, err
		}
		byID := make(map[int64]types.User, len(fetched))
		for _, user := range fetched {
			byID[user.ID] = user
		}
		result = result[:0]
		for _, id := range userIDs {
			if user, ok := byID[id]; ok {
				result = append(result, user)
				continue
			}
			s.mu.RLock()
			user, ok := s.cache[id]
			s.mu.RUnlock()
			if ok {
				result = append(result, user)
			}
		}
	}
	return result, nil
}

// SearchUsers searches contacts and users globally by name or query.
func (s *UserService) SearchUsers(ctx context.Context, query string) ([]types.User, error) {
	payload := map[string]interface{}{
		"query": query,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpContactSearch, payload)
	if err != nil {
		return nil, fmt.Errorf("search users failed: %w", err)
	}

	var users []types.User
	if uList, ok := res["users"].([]interface{}); ok {
		for _, item := range uList {
			if uData, ok := item.(map[string]interface{}); ok {
				user := types.ParseUserPayload(uData)
				user.Bind(s)
				users = append(users, user)
			}
		}
	}
	s.cacheUsers(users)
	return users, nil
}

// SearchByPhone resolves a user by phone number.
func (s *UserService) SearchByPhone(ctx context.Context, phone string) (*types.User, error) {
	return s.GetUserByPhone(ctx, phone)
}

// GetContacts returns the user's complete contact list.
func (s *UserService) GetContacts(ctx context.Context) ([]types.User, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpContactList, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("get contacts failed: %w", err)
	}

	contacts := s.usersFromResponse(res)
	s.cacheUsers(contacts)
	return contacts, nil
}

// GetSelf retrieves current account profile details.
func (s *UserService) GetSelf(ctx context.Context) (*types.User, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpProfile, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("get self profile failed: %w", err)
	}

	user := types.ParseUserPayload(res)
	user.Bind(s)
	s.cacheUsers([]types.User{user})
	return &user, nil
}

// SessionItem is retained for source compatibility.
type SessionItem = types.Session

// GetActiveSessions retrieves all active device sessions for this account.
func (s *UserService) GetActiveSessions(ctx context.Context) ([]SessionItem, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpSessionsInfo, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("get active sessions failed: %w", err)
	}

	var sessions []SessionItem
	if rawList, ok := res["sessions"].([]interface{}); ok {
		for _, item := range rawList {
			if sData, ok := item.(map[string]interface{}); ok {
				si := SessionItem{ID: sData["id"], Device: stringOrEmpty(sData["device"]), DeviceID: stringOrEmpty(sData["deviceId"]), UserAgent: stringOrEmpty(sData["userAgent"]), AppVersion: stringOrEmpty(sData["appVersion"]), DeviceName: stringOrEmpty(sData["deviceName"]), DeviceType: stringOrEmpty(sData["deviceType"]), Platform: stringOrEmpty(sData["platform"]), Location: stringOrEmpty(sData["location"]), Client: stringOrEmpty(sData["client"]), IP: stringOrEmpty(sData["ip"]), Options: sData["options"]}
				si.Current, _ = sData["current"].(bool)
				si.Created, _ = types.Int64Value(sData["created"])
				si.Updated, _ = types.Int64Value(sData["updated"])
				si.LastActivity, _ = types.Int64Value(sData["lastActivity"])
				sessions = append(sessions, si)
			}
		}
	}
	return sessions, nil
}

// GetSessions is the PyMax-compatible name for GetActiveSessions.
func (s *UserService) GetSessions(ctx context.Context) ([]SessionItem, error) {
	return s.GetActiveSessions(ctx)
}

// CloseSession terminates a specific remote active session by ID.
func (s *UserService) CloseSession(ctx context.Context, sessionID int64) error {
	payload := map[string]interface{}{
		"id": sessionID,
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpSessionsClose, payload)
	if err != nil {
		return fmt.Errorf("close session failed: %w", err)
	}
	return nil
}

// Set2FA configures or updates the 2FA password on the account.
func (s *UserService) Set2FA(ctx context.Context, password, hint, email string) error {
	if email != "" {
		return fmt.Errorf("set 2fa: email requires Set2FAWithEmailProvider")
	}
	return s.set2FA(ctx, password, hint, "", nil)
}

func (s *UserService) Set2FAWithEmailProvider(ctx context.Context, password, hint, email string, codeProvider func(context.Context) (string, error)) error {
	return s.set2FA(ctx, password, hint, email, codeProvider)
}

func (s *UserService) set2FA(ctx context.Context, password, hint, email string, codeProvider func(context.Context) (string, error)) error {
	res, err := s.invoker.Invoke(ctx, protocol.OpAuthCreateTrack, map[string]interface{}{"type": 0})
	if err != nil {
		return fmt.Errorf("create 2fa track: %w", err)
	}
	trackID, _ := res["trackId"].(string)
	if trackID == "" {
		return fmt.Errorf("create 2fa track: missing trackId")
	}
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthValidatePassword, map[string]interface{}{"trackId": trackID, "password": password}); err != nil {
		return err
	}
	capabilities := []int{0}
	if hint != "" {
		if _, err := s.invoker.Invoke(ctx, protocol.OpAuthValidateHint, map[string]interface{}{"trackId": trackID, "hint": hint}); err != nil {
			return err
		}
		capabilities = append(capabilities, 3)
	}
	if email != "" {
		if _, err := s.invoker.Invoke(ctx, protocol.OpAuthVerifyEmail, map[string]interface{}{"trackId": trackID, "email": email}); err != nil {
			return err
		}
		if codeProvider == nil {
			return fmt.Errorf("set 2fa: email code provider is required")
		}
		code, err := codeProvider(ctx)
		if err != nil {
			return err
		}
		if code == "" {
			return fmt.Errorf("set 2fa: email verification code is required")
		}
		if _, err := s.invoker.Invoke(ctx, protocol.OpAuthCheckEmail, map[string]interface{}{"trackId": trackID, "verifyCode": code}); err != nil {
			return err
		}
		capabilities = append(capabilities, 4)
	}
	_, err = s.invoker.Invoke(ctx, protocol.OpAuthSet2Fa, map[string]interface{}{"trackId": trackID, "password": password, "hint": hint, "expectedCapabilities": capabilities})
	return err
}

// AddContact adds a user to the contact list by user ID and optional name info.
func (s *UserService) AddContact(ctx context.Context, userID int64, firstName, lastName, phone string) error {
	payload := map[string]interface{}{
		"userId":    userID,
		"firstName": firstName,
	}
	if lastName != "" {
		payload["lastName"] = lastName
	}
	if phone != "" {
		payload["phone"] = phone
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpContactUpdate, payload)
	if err != nil {
		return fmt.Errorf("add contact failed: %w", err)
	}
	return nil
}

// AddContactByID follows PyMax's compact CONTACT_UPDATE payload.
func (s *UserService) AddContactByID(ctx context.Context, contactID int64) (*types.User, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpContactUpdate, map[string]interface{}{
		"contactId": contactID, "action": "ADD",
	})
	if err != nil {
		return nil, fmt.Errorf("add contact failed: %w", err)
	}
	user := &types.User{ID: contactID}
	if data, ok := res["contact"].(map[string]interface{}); ok {
		parsed := types.ParseUserPayload(data)
		user = &parsed
	}
	user.Bind(s)
	s.cacheUsers([]types.User{*user})
	return user, nil
}

// RemoveContact removes a user from the contact list.
func (s *UserService) RemoveContact(ctx context.Context, contactID int64) error {
	_, err := s.invoker.Invoke(ctx, protocol.OpContactUpdate, map[string]interface{}{
		"contactId": contactID, "action": "REMOVE",
	})
	if err != nil {
		return fmt.Errorf("remove contact failed: %w", err)
	}
	s.mu.Lock()
	delete(s.cache, contactID)
	s.mu.Unlock()
	return nil
}

// ImportContacts imports account contacts. contacts may be
// []types.ContactInfo (the PyMax shape) or map[string]string for compatibility
// with earlier GoMax releases.
func (s *UserService) ImportContacts(ctx context.Context, contacts any) ([]types.User, error) {
	contactList := make(map[string]interface{})
	switch values := contacts.(type) {
	case []types.ContactInfo:
		for _, contact := range values {
			if contact.Phone == "" || contact.FirstName == "" {
				return nil, fmt.Errorf("import contacts failed: phone and first name are required")
			}
			contactList[contact.Phone] = map[string]interface{}{"firstName": contact.FirstName}
		}
	case map[string]string:
		for phone, firstName := range values {
			contactList[phone] = map[string]interface{}{"firstName": firstName}
		}
	default:
		return nil, fmt.Errorf("import contacts failed: expected []types.ContactInfo or map[string]string")
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpSync, map[string]interface{}{"contactList": contactList})
	if err != nil {
		return nil, fmt.Errorf("import contacts failed: %w", err)
	}
	users := s.usersFromResponse(res)
	s.cacheUsers(users)
	return users, nil
}

// GetChatID returns the deterministic dialog ID used by Max for two users.
func (s *UserService) GetChatID(_ context.Context, firstUserID, secondUserID int64) int64 {
	return firstUserID ^ secondUserID
}

// UpdateContact renames or edits an existing contact's display name.
func (s *UserService) UpdateContact(ctx context.Context, userID int64, firstName, lastName string) error {
	payload := map[string]interface{}{
		"userId":    userID,
		"firstName": firstName,
	}
	if lastName != "" {
		payload["lastName"] = lastName
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpContactUpdate, payload)
	if err != nil {
		return fmt.Errorf("update contact failed: %w", err)
	}
	return nil
}

// GetUserByPhone looks up a user profile by their phone number.
func (s *UserService) GetUserByPhone(ctx context.Context, phone string) (*types.User, error) {
	payload := map[string]interface{}{
		"phone": phone,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpContactInfoByPhone, payload)
	if err != nil {
		return nil, fmt.Errorf("get user by phone failed: %w", err)
	}

	parsed := types.ParseUserPayload(res)
	user := &parsed
	if user.Phone == "" {
		user.Phone = phone
	}
	user.Bind(s)
	s.cacheUsers([]types.User{*user})
	return user, nil
}

func (s *UserService) usersFromResponse(res map[string]interface{}) []types.User {
	var out []types.User
	list, _ := res["contacts"].([]interface{})
	if list == nil {
		list, _ = res["users"].([]interface{})
	}
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		u := types.ParseUserPayload(m)
		u.Bind(s)
		out = append(out, u)
	}
	return out
}

func (s *UserService) cacheUsers(users []types.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range users {
		if user.ID != 0 {
			user.Bind(s)
			s.cache[user.ID] = user
		}
	}
}

// SeedCache stores users received during LOGIN/LOGIN2 synchronization.
func (s *UserService) SeedCache(users []types.User) { s.cacheUsers(users) }

func stringOrEmpty(value interface{}) string { v, _ := value.(string); return v }
