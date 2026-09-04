package users

import (
	"context"
	"fmt"

	"gomax/pkg/api"
	"gomax/pkg/protocol"
	"gomax/pkg/types"
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
	ID        int64  `json:"id"`
	Device    string `json:"device"`
	Location  string `json:"location"`
	Client    string `json:"client"`
	IP        string `json:"ip"`
	LastActive int64 `json:"last_active"`
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

