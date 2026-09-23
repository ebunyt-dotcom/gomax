package selfapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// SelfService handles the current account's profile, settings, and folder management.
// It mirrors pymax.api.self.service.SelfService.
type SelfService struct {
	invoker             api.Invoker
	userActions         types.UserActions
	onTokenChanged      func(oldToken, newToken string) error
	onConfigHashChanged func(hash string) error
}

// SetUserActions binds convenience methods on returned profile values.
func (s *SelfService) SetUserActions(actions types.UserActions) { s.userActions = actions }

func (s *SelfService) SetSessionHooks(tokenHook func(string, string) error, hashHook func(string) error) {
	s.onTokenChanged = tokenHook
	s.onConfigHashChanged = hashHook
}

type interactiveController interface {
	SetInteractive(bool)
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

	user := types.ParseUserPayload(res)
	user.Bind(s.userActions)
	return &user, nil
}

// ChangeProfile updates the current account's first name, last name, description, and/or photo.
// Set photoToken to "" to leave the avatar unchanged.
func (s *SelfService) ChangeProfile(ctx context.Context, firstName, lastName, description, photoToken string) error {
	_, err := s.ChangeProfileResult(ctx, firstName, lastName, description, photoToken)
	return err
}

// ChangeProfileResult updates the profile and returns the contact from Max's response.
func (s *SelfService) ChangeProfileResult(ctx context.Context, firstName, lastName, description, photoToken string) (*types.User, error) {
	payload := map[string]interface{}{
		"firstName":  firstName,
		"avatarType": "USER_AVATAR",
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

	res, err := s.invoker.Invoke(ctx, protocol.OpProfile, payload)
	if err != nil {
		return nil, fmt.Errorf("change profile failed: %w", err)
	}
	user := types.ParseUserPayload(res)
	user.Bind(s.userActions)
	return &user, nil
}

// SetPresence changes the interactive flag used by the owning client.
func (s *SelfService) SetPresence(online bool) {
	if controller, ok := s.invoker.(interactiveController); ok {
		controller.SetInteractive(online)
	}
}

// RequestProfilePhotoUploadURL requests an upload slot for the account avatar.
func (s *SelfService) RequestProfilePhotoUploadURL(ctx context.Context) (string, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpPhotoUpload, map[string]interface{}{"count": 1, "profile": true})
	if err != nil {
		return "", fmt.Errorf("request profile photo upload url failed: %w", err)
	}
	if url, ok := res["url"].(string); ok {
		return url, nil
	}
	return "", fmt.Errorf("profile photo upload response did not contain url")
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
	res, err := s.invoker.Invoke(ctx, protocol.OpSessionsClose, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("close all sessions failed: %w", err)
	}
	newToken, _ := res["token"].(string)
	if newToken == "" {
		return fmt.Errorf("close all sessions failed: response did not contain a replacement token")
	}
	if s.onTokenChanged != nil {
		if err := s.onTokenChanged("", newToken); err != nil {
			return fmt.Errorf("persist replacement token: %w", err)
		}
	}
	return nil
}

// GetFolders returns all configured chat folders (filters).
func (s *SelfService) GetFolders(ctx context.Context) (*types.FolderList, error) {
	return s.GetFoldersFromSync(ctx, 0)
}

func (s *SelfService) GetFoldersFromSync(ctx context.Context, folderSync int64) (*types.FolderList, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpFoldersGet, map[string]interface{}{
		"folderSync": folderSync,
	})
	if err != nil {
		return nil, fmt.Errorf("get folders failed: %w", err)
	}

	fl := &types.FolderList{}
	fl.Sync, _ = types.Int64Value(res["folderSync"])
	if fl.Sync == 0 {
		fl.Sync, _ = types.Int64Value(res["sync"])
	}
	if order, ok := res["foldersOrder"].([]interface{}); ok {
		for _, item := range order {
			if id, ok := item.(string); ok {
				fl.FoldersOrder = append(fl.FoldersOrder, id)
			}
		}
	}
	fl.AllFilterExcludeFolders, _ = res["allFilterExcludeFolders"].([]interface{})

	if rawFolders, ok := res["folders"].([]interface{}); ok {
		for _, item := range rawFolders {
			if fData, ok := item.(map[string]interface{}); ok {
				fl.Folders = append(fl.Folders, parseFolder(fData))
			}
		}
	}
	return fl, nil
}

// CreateFolder creates a new chat folder with the given title and included chat IDs.
func (s *SelfService) CreateFolder(ctx context.Context, title string, chatInclude []int64) (*types.Folder, error) {
	update, err := s.CreateFolderUpdate(ctx, title, chatInclude, nil)
	if err != nil || update == nil {
		return nil, err
	}
	return update.Folder, nil
}

// CreateFolderUpdate exposes the complete FolderUpdate response and custom filters.
func (s *SelfService) CreateFolderUpdate(ctx context.Context, title string, chatInclude []int64, filters []interface{}) (*types.FolderUpdate, error) {
	if filters == nil {
		filters = []interface{}{}
	}
	payload := map[string]interface{}{
		"id":      newFolderID(),
		"title":   title,
		"include": chatInclude,
		"filters": filters,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpFoldersUpdate, payload)
	if err != nil {
		return nil, fmt.Errorf("create folder failed: %w", err)
	}

	return parseFolderUpdate(res), nil
}

// UpdateFolderWithOptions is the full PyMax folder update variant.
func (s *SelfService) UpdateFolderWithOptions(ctx context.Context, folderID, title string, chatInclude []int64, filters, options []interface{}) (*types.Folder, error) {
	update, err := s.UpdateFolderResult(ctx, folderID, title, chatInclude, filters, options)
	if err != nil || update == nil {
		return nil, err
	}
	return update.Folder, nil
}

// UpdateFolderResult returns the complete folder order and sync marker.
func (s *SelfService) UpdateFolderResult(ctx context.Context, folderID, title string, chatInclude []int64, filters, options []interface{}) (*types.FolderUpdate, error) {
	if chatInclude == nil {
		chatInclude = []int64{}
	}
	if filters == nil {
		filters = []interface{}{}
	}
	if options == nil {
		options = []interface{}{}
	}
	payload := map[string]interface{}{
		"id": folderID, "title": title, "include": chatInclude,
		"filters": filters, "options": options,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpFoldersUpdate, payload)
	if err != nil {
		return nil, fmt.Errorf("update folder failed: %w", err)
	}
	return parseFolderUpdate(res), nil
}

// UpdateFolder updates an existing folder's title and included chats.
func (s *SelfService) UpdateFolder(ctx context.Context, folderID, title string, chatInclude []int64) (*types.Folder, error) {
	return s.UpdateFolderWithOptions(ctx, folderID, title, chatInclude, []interface{}{}, []interface{}{})
}

// DeleteFolder removes a chat folder by its ID.
func (s *SelfService) DeleteFolder(ctx context.Context, folderID string) error {
	_, err := s.DeleteFolderResult(ctx, folderID)
	return err
}

// DeleteFolderResult returns the updated folder order and sync marker.
func (s *SelfService) DeleteFolderResult(ctx context.Context, folderID string) (*types.FolderUpdate, error) {
	payload := map[string]interface{}{
		"folderIds": []string{folderID},
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpFoldersDelete, payload)
	if err != nil {
		return nil, fmt.Errorf("delete folder failed: %w", err)
	}
	return parseFolderUpdate(res), nil
}

// ChangeProfileSettings updates privacy settings. The map keys must be the
// protocol names (for example SEARCH_BY_PHONE or HIDDEN).
func (s *SelfService) ChangeProfileSettings(ctx context.Context, settings map[string]interface{}) error {
	res, err := s.invoker.Invoke(ctx, protocol.OpConfig, map[string]interface{}{
		"settings": map[string]interface{}{"user": settings},
	})
	if err != nil {
		return fmt.Errorf("change profile settings failed: %w", err)
	}
	hash, _ := res["hash"].(string)
	if hash != "" && s.onConfigHashChanged != nil {
		if err := s.onConfigHashChanged(hash); err != nil {
			return fmt.Errorf("persist config hash: %w", err)
		}
	}
	return nil
}

// ChangePrivacySettings is the typed form of ChangeProfileSettings.
func (s *SelfService) ChangePrivacySettings(ctx context.Context, settings types.PrivacySettingsUpdate) error {
	return s.ChangeProfileSettings(ctx, settings.Payload())
}

func newFolderID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "folder-unknown"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}

func parseFolder(raw map[string]interface{}) types.Folder {
	folder := types.Folder{ID: types.StringValue(raw["id"]), Title: types.StringValue(raw["title"])}
	folder.SourceID, _ = types.Int64Value(raw["sourceId"])
	folder.UpdateTime, _ = types.Int64Value(raw["updateTime"])
	folder.Options, _ = raw["options"].([]interface{})
	folder.Filters, _ = raw["filters"].([]interface{})
	if include, ok := raw["include"].([]interface{}); ok {
		for _, value := range include {
			if chatID, ok := types.Int64Value(value); ok {
				folder.Include = append(folder.Include, chatID)
			}
		}
	}
	return folder
}

func parseFolderUpdate(raw map[string]interface{}) *types.FolderUpdate {
	update := &types.FolderUpdate{}
	update.FolderSync, _ = types.Int64Value(raw["folderSync"])
	if order, ok := raw["foldersOrder"].([]interface{}); ok {
		for _, value := range order {
			if id, ok := value.(string); ok {
				update.FoldersOrder = append(update.FoldersOrder, id)
			}
		}
	}
	if folder, ok := raw["folder"].(map[string]interface{}); ok {
		parsed := parseFolder(folder)
		update.Folder = &parsed
	}
	return update
}
