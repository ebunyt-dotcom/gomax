package selfapi

import (
	"context"
	"fmt"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// SelfService handles the current account's profile, settings, and folder management.
// It mirrors pymax.api.self.service.SelfService.
type SelfService struct {
	invoker api.Invoker
}

// NewSelfService creates a new SelfService instance.
func NewSelfService(invoker api.Invoker) *SelfService {
	return &SelfService{invoker: invoker}
}

// GetSelf retrieves the current user's profile.
func (s *SelfService) GetSelf(ctx context.Context) (*types.User, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpProfile, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("get self failed: %w", err)
	}

	user := &types.User{}
	uData := res
	if profile, ok := res["profile"].(map[string]interface{}); ok {
		uData = profile
		if contact, ok := profile["contact"].(map[string]interface{}); ok {
			uData = contact
		}
	}

	if id, ok := uData["id"].(int64); ok {
		user.ID = id
	} else if idF, ok := uData["id"].(float64); ok {
		user.ID = int64(idF)
	}
	if fn, ok := uData["firstName"].(string); ok {
		user.FirstName = fn
	}
	if ln, ok := uData["lastName"].(string); ok {
		user.LastName = ln
	}
	if phone, ok := uData["phone"].(string); ok {
		user.Phone = phone
	}
	if bio, ok := uData["description"].(string); ok {
		user.Bio = bio
	}
	return user, nil
}

// ChangeProfile updates the current account's first name, last name, description, and/or photo.
// Set photoToken to "" to leave the avatar unchanged.
func (s *SelfService) ChangeProfile(ctx context.Context, firstName, lastName, description, photoToken string) error {
	payload := map[string]interface{}{
		"firstName": firstName,
	}
	if lastName != "" {
		payload["lastName"] = lastName
	}
	if description != "" {
		payload["description"] = description
	}
	if photoToken != "" {
		payload["photoToken"] = photoToken
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpProfile, payload)
	if err != nil {
		return fmt.Errorf("change profile failed: %w", err)
	}
	return nil
}

// Logout terminates the current session on the server.
func (s *SelfService) Logout(ctx context.Context) error {
	_, err := s.invoker.Invoke(ctx, protocol.OpLogout, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}
	return nil
}

// CloseAllSessions terminates all other active device sessions, keeping the current one.
func (s *SelfService) CloseAllSessions(ctx context.Context) error {
	_, err := s.invoker.Invoke(ctx, protocol.OpSessionsClose, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("close all sessions failed: %w", err)
	}
	return nil
}

// GetFolders returns all configured chat folders (filters).
func (s *SelfService) GetFolders(ctx context.Context) (*types.FolderList, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpFoldersGet, map[string]interface{}{
		"folderSync": 0,
	})
	if err != nil {
		return nil, fmt.Errorf("get folders failed: %w", err)
	}

	fl := &types.FolderList{}
	if syncVal, ok := res["sync"].(float64); ok {
		fl.Sync = int64(syncVal)
	} else if syncVal, ok := res["sync"].(int64); ok {
		fl.Sync = syncVal
	}

	if rawFolders, ok := res["folders"].([]interface{}); ok {
		for _, item := range rawFolders {
			if fData, ok := item.(map[string]interface{}); ok {
				folder := types.Folder{}
				if id, ok := fData["id"].(string); ok {
					folder.ID = id
				}
				if title, ok := fData["title"].(string); ok {
					folder.Title = title
				}
				if includeRaw, ok := fData["include"].([]interface{}); ok {
					for _, v := range includeRaw {
						if chatID, ok := v.(float64); ok {
							folder.Include = append(folder.Include, int64(chatID))
						} else if chatID, ok := v.(int64); ok {
							folder.Include = append(folder.Include, chatID)
						}
					}
				}
				fl.Folders = append(fl.Folders, folder)
			}
		}
	}
	return fl, nil
}

// CreateFolder creates a new chat folder with the given title and included chat IDs.
func (s *SelfService) CreateFolder(ctx context.Context, title string, chatInclude []int64) (*types.Folder, error) {
	payload := map[string]interface{}{
		"title":   title,
		"include": chatInclude,
		"filters": []interface{}{},
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpFoldersUpdate, payload)
	if err != nil {
		return nil, fmt.Errorf("create folder failed: %w", err)
	}

	folder := &types.Folder{Title: title}
	if fData, ok := res["folder"].(map[string]interface{}); ok {
		if id, ok := fData["id"].(string); ok {
			folder.ID = id
		}
	}
	return folder, nil
}

// UpdateFolder updates an existing folder's title and included chats.
func (s *SelfService) UpdateFolder(ctx context.Context, folderID, title string, chatInclude []int64) (*types.Folder, error) {
	payload := map[string]interface{}{
		"id":      folderID,
		"title":   title,
		"include": chatInclude,
		"filters": []interface{}{},
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpFoldersUpdate, payload)
	if err != nil {
		return nil, fmt.Errorf("update folder failed: %w", err)
	}

	folder := &types.Folder{ID: folderID, Title: title}
	if fData, ok := res["folder"].(map[string]interface{}); ok {
		if id, ok := fData["id"].(string); ok {
			folder.ID = id
		}
		if t, ok := fData["title"].(string); ok {
			folder.Title = t
		}
	}
	return folder, nil
}

// DeleteFolder removes a chat folder by its ID.
func (s *SelfService) DeleteFolder(ctx context.Context, folderID string) error {
	payload := map[string]interface{}{
		"folderIds": []string{folderID},
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpFoldersDelete, payload)
	if err != nil {
		return fmt.Errorf("delete folder failed: %w", err)
	}
	return nil
}
