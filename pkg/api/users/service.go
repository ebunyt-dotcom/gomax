package users

import (
	"context"
	"fmt"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// UserService handles user profiles, contacts, and searches.
type UserService struct {
	invoker api.Invoker
}

// NewUserService creates a new UserService instance.
func NewUserService(invoker api.Invoker) *UserService {
	return &UserService{invoker: invoker}
}

// GetUser retrieves detailed user profile information by ID.
func (s *UserService) GetUser(ctx context.Context, userID int64) (*types.User, error) {
	payload := map[string]interface{}{
		"userId": userID,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpContactInfo, payload)
	if err != nil {
		return nil, fmt.Errorf("get user failed: %w", err)
	}

	user := &types.User{ID: userID}
	if uData, ok := res["user"].(map[string]interface{}); ok {
		if fn, ok := uData["firstName"].(string); ok {
			user.FirstName = fn
		}
		if ln, ok := uData["lastName"].(string); ok {
			user.LastName = ln
		}
		if phone, ok := uData["phone"].(string); ok {
			user.Phone = phone
		}
	}
	return user, nil
}

// GetCachedUser is kept as an explicit Go API hook. GoMax currently leaves
// caching to the caller, so it performs one regular lookup.
func (s *UserService) GetCachedUser(ctx context.Context, userID int64) (*types.User, error) {
	return s.GetUser(ctx, userID)
}

// FetchUsers is the uncached batch lookup counterpart to GetUsers.
func (s *UserService) FetchUsers(ctx context.Context, userIDs []int64) ([]types.User, error) {
	return s.GetUsers(ctx, userIDs)
}

// GetUsers retrieves multiple user profiles in batch.
func (s *UserService) GetUsers(ctx context.Context, userIDs []int64) ([]types.User, error) {
	payload := map[string]interface{}{
		"userIds": userIDs,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpContactInfo, payload)
	if err != nil {
		return nil, fmt.Errorf("get users failed: %w", err)
	}

	var users []types.User
	if uList, ok := res["users"].([]interface{}); ok {
		for _, item := range uList {
			if uData, ok := item.(map[string]interface{}); ok {
				var u types.User
				if id, ok := uData["id"].(int64); ok {
					u.ID = id
				} else if idF, ok := uData["id"].(float64); ok {
					u.ID = int64(idF)
				}
				if fn, ok := uData["firstName"].(string); ok {
					u.FirstName = fn
				}
				if ln, ok := uData["lastName"].(string); ok {
					u.LastName = ln
				}
				users = append(users, u)
			}
		}
	}
	return users, nil
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
				var u types.User
				if id, ok := uData["id"].(int64); ok {
					u.ID = id
				} else if idF, ok := uData["id"].(float64); ok {
					u.ID = int64(idF)
				}
				if fn, ok := uData["firstName"].(string); ok {
					u.FirstName = fn
				}
				users = append(users, u)
			}
		}
	}
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

	var contacts []types.User
	if cList, ok := res["contacts"].([]interface{}); ok {
		for _, item := range cList {
			if cData, ok := item.(map[string]interface{}); ok {
				var u types.User
				if id, ok := cData["id"].(int64); ok {
					u.ID = id
				} else if idF, ok := cData["id"].(float64); ok {
					u.ID = int64(idF)
				}
				if fn, ok := cData["firstName"].(string); ok {
					u.FirstName = fn
				}
				contacts = append(contacts, u)
			}
		}
	}
	return contacts, nil
}

// GetSelf retrieves current account profile details.
func (s *UserService) GetSelf(ctx context.Context) (*types.User, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpProfile, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("get self profile failed: %w", err)
	}

	user := &types.User{}
	if uData, ok := res["profile"].(map[string]interface{}); ok {
		if id, ok := uData["id"].(int64); ok {
			user.ID = id
		} else if idF, ok := uData["id"].(float64); ok {
			user.ID = int64(idF)
		}
		if fn, ok := uData["firstName"].(string); ok {
			user.FirstName = fn
		}
		if phone, ok := uData["phone"].(string); ok {
			user.Phone = phone
		}
	}
	return user, nil
}

// SessionItem represents an active device session.
type SessionItem struct {
	ID         int64  `json:"id"`
	Device     string `json:"device"`
	Location   string `json:"location"`
	Client     string `json:"client"`
	IP         string `json:"ip"`
	LastActive int64  `json:"last_active"`
}

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
				var si SessionItem
				if id, ok := sData["id"].(int64); ok {
					si.ID = id
				} else if idF, ok := sData["id"].(float64); ok {
					si.ID = int64(idF)
				}
				if dev, ok := sData["device"].(string); ok {
					si.Device = dev
				}
				if ip, ok := sData["ip"].(string); ok {
					si.IP = ip
				}
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
	payload := map[string]interface{}{
		"password": password,
		"hint":     hint,
		"email":    email,
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpAuthSet2Fa, payload)
	if err != nil {
		return fmt.Errorf("set 2fa failed: %w", err)
	}
	return nil
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
		if v, ok := data["id"].(int64); ok {
			user.ID = v
		}
		if v, ok := data["firstName"].(string); ok {
			user.FirstName = v
		}
		if v, ok := data["lastName"].(string); ok {
			user.LastName = v
		}
	}
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
	return nil
}

// ImportContacts imports phone/name pairs into the account contacts.
func (s *UserService) ImportContacts(ctx context.Context, contacts map[string]string) ([]types.User, error) {
	contactList := make(map[string]interface{}, len(contacts))
	for phone, firstName := range contacts {
		contactList[phone] = map[string]interface{}{"firstName": firstName}
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpSync, map[string]interface{}{"contactList": contactList})
	if err != nil {
		return nil, fmt.Errorf("import contacts failed: %w", err)
	}
	return usersFromResponse(res), nil
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

	user := &types.User{Phone: phone}
	src := res
	if uData, ok := res["user"].(map[string]interface{}); ok {
		src = uData
	} else if uData, ok := res["contact"].(map[string]interface{}); ok {
		src = uData
	}
	if id, ok := src["id"].(int64); ok {
		user.ID = id
	} else if idF, ok := src["id"].(float64); ok {
		user.ID = int64(idF)
	}
	if fn, ok := src["firstName"].(string); ok {
		user.FirstName = fn
	}
	if ln, ok := src["lastName"].(string); ok {
		user.LastName = ln
	}
	return user, nil
}

func usersFromResponse(res map[string]interface{}) []types.User {
	var out []types.User
	list, _ := res["contacts"].([]interface{})
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		u := types.User{Phone: stringOrEmpty(m["phone"])}
		if v, ok := m["id"].(int64); ok {
			u.ID = v
		} else if v, ok := m["id"].(float64); ok {
			u.ID = int64(v)
		}
		if v, ok := m["firstName"].(string); ok {
			u.FirstName = v
		}
		out = append(out, u)
	}
	return out
}

func stringOrEmpty(value interface{}) string { v, _ := value.(string); return v }
