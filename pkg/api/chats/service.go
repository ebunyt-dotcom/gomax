package chats

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// ChatService handles all chat, channel and group related operations.
type ChatService struct {
	invoker        api.Invoker
	messageActions types.MessageActions
	mu             sync.RWMutex
	cache          map[int64]types.Chat
}

// NewChatService creates a new ChatService instance.
func NewChatService(invoker api.Invoker) *ChatService {
	return &ChatService{invoker: invoker, cache: make(map[int64]types.Chat)}
}

// SetMessageActions binds message convenience methods on returned chats.
func (s *ChatService) SetMessageActions(actions types.MessageActions) {
	s.mu.Lock()
	s.messageActions = actions
	for id, chat := range s.cache {
		chat.Bind(actions, s)
		s.cache[id] = chat
	}
	s.mu.Unlock()
}

func (s *ChatService) cacheChat(chat types.Chat) types.Chat {
	s.mu.RLock()
	messages := s.messageActions
	s.mu.RUnlock()
	chat.Bind(messages, s)
	if chat.ID == 0 {
		return chat
	}
	s.mu.Lock()
	s.cache[chat.ID] = chat
	s.mu.Unlock()
	return chat
}

func (s *ChatService) cachedChat(chatID int64) (types.Chat, bool) {
	s.mu.RLock()
	chat, ok := s.cache[chatID]
	s.mu.RUnlock()
	return chat, ok
}

func (s *ChatService) removeCachedChat(chatID int64) {
	s.mu.Lock()
	delete(s.cache, chatID)
	s.mu.Unlock()
}

// SeedCache adds chats received during LOGIN/LOGIN2 synchronization.
func (s *ChatService) SeedCache(chats []types.Chat) {
	for _, chat := range chats {
		s.cacheChat(chat)
	}
}

// JoinChat joins a public or private chat by invite link or hash.
func (s *ChatService) JoinChat(ctx context.Context, link string) (*types.Chat, error) {
	payload := map[string]interface{}{
		"link": link,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatJoin, payload)
	if err != nil {
		return nil, fmt.Errorf("join chat failed: %w", err)
	}

	chat := types.ParseChatPayload(res)
	if chat.InviteLink == "" {
		chat.InviteLink = link
	}
	chat = s.cacheChat(chat)
	return &chat, nil
}

// InviteUsersToGroup adds users to a group chat by their user IDs.
func (s *ChatService) InviteUsersToGroup(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error {
	_, err := s.InviteUsersToGroupResult(ctx, chatID, userIDs, showHistory)
	return err
}

// InviteUsersToGroupResult adds users and returns the updated chat when Max includes it.
func (s *ChatService) InviteUsersToGroupResult(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) (*types.Chat, error) {
	payload := map[string]interface{}{
		"chatId":      chatID,
		"userIds":     userIDs,
		"showHistory": showHistory,
		"operation":   "add",
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatMembersUpdate, payload)
	if err != nil {
		return nil, fmt.Errorf("invite users to group failed: %w", err)
	}
	if _, ok := res["chat"].(map[string]interface{}); ok {
		chat := types.ParseChatPayload(res)
		chat = s.cacheChat(chat)
		return &chat, nil
	}
	return nil, nil
}

// InviteUsersToChannel adds users to a channel by user IDs.
func (s *ChatService) InviteUsersToChannel(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error {
	_, err := s.InviteUsersToChannelResult(ctx, chatID, userIDs, showHistory)
	return err
}

// InviteUsersToChannelResult adds users and returns an updated channel when present.
func (s *ChatService) InviteUsersToChannelResult(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) (*types.Chat, error) {
	return s.InviteUsersToGroupResult(ctx, chatID, userIDs, showHistory)
}

// JoinGroup joins a group using the same invite-link protocol as JoinChat.
func (s *ChatService) JoinGroup(ctx context.Context, link string) (*types.Chat, error) {
	processed, ok := processJoinLink(link)
	if !ok {
		return nil, fmt.Errorf("join group failed: invalid group link")
	}
	return s.JoinChat(ctx, processed)
}

// JoinChannel joins a channel using an invite link.
func (s *ChatService) JoinChannel(ctx context.Context, link string) (*types.Chat, error) {
	if processed, ok := processJoinLink(link); ok {
		link = processed
	}
	return s.JoinChat(ctx, link)
}

// ResolveGroupByLink resolves an invite link without joining the chat.
func (s *ChatService) ResolveGroupByLink(ctx context.Context, link string) (*types.Chat, error) {
	processed, ok := processJoinLink(link)
	if !ok {
		return nil, fmt.Errorf("resolve group link failed: invalid group link")
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpLinkInfo, map[string]interface{}{"link": processed})
	if err != nil {
		return nil, fmt.Errorf("resolve group link failed: %w", err)
	}
	if raw, ok := res["chat"].(map[string]interface{}); ok {
		chat := s.cacheChat(types.ParseChatPayload(raw))
		return &chat, nil
	}
	return nil, nil
}

// RemoveUsersFromGroup kicks users from a group.
func (s *ChatService) RemoveUsersFromGroup(ctx context.Context, chatID int64, userIDs []int64, cleanMsgPeriod int) error {
	payload := map[string]interface{}{
		"chatId":         chatID,
		"userIds":        userIDs,
		"cleanMsgPeriod": cleanMsgPeriod,
		"operation":      "remove",
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpChatMembersUpdate, payload)
	if err != nil {
		return fmt.Errorf("remove users from group failed: %w", err)
	}
	return nil
}

// CreateGroup creates a new group chat with initial participants.
func (s *ChatService) CreateGroup(ctx context.Context, name string, participantIDs []int64, notify bool) (*types.Chat, error) {
	chat, _, err := s.CreateGroupWithMessage(ctx, name, participantIDs, notify)
	return chat, err
}

// CreateGroupWithMessage returns both objects produced by Max's MSG_SEND response.
func (s *ChatService) CreateGroupWithMessage(ctx context.Context, name string, participantIDs []int64, notify bool) (*types.Chat, *types.Message, error) {
	nowMs := time.Now().UnixMilli()
	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"cid": nowMs,
			"attaches": []interface{}{
				map[string]interface{}{
					"_type":    "CONTROL",
					"event":    "new",
					"chatType": "CHAT",
					"title":    name,
					"userIds":  participantIDs,
				},
			},
		},
		"notify": notify,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpMsgSend, payload)
	if err != nil {
		return nil, nil, fmt.Errorf("create group failed: %w", err)
	}
	chatData, ok := res["chat"].(map[string]interface{})
	if !ok {
		return nil, nil, nil
	}
	chat := types.ParseChatPayload(chatData)
	chat = s.cacheChat(chat)
	message := types.ParseMessagePayload(res)
	return &chat, message, nil
}

// LeaveChat leaves a group chat or channel.
func (s *ChatService) LeaveChat(ctx context.Context, chatID int64) error {
	payload := map[string]interface{}{
		"chatId": chatID,
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpChatLeave, payload)
	if err != nil {
		return fmt.Errorf("leave chat failed: %w", err)
	}
	s.removeCachedChat(chatID)
	return nil
}

// DeleteChat deletes a conversation completely.
func (s *ChatService) DeleteChat(ctx context.Context, chatID int64) error {
	return s.DeleteChatWithOptions(ctx, chatID, time.Now().UnixMilli(), true)
}

// DeleteChatWithOptions exposes PyMax's last-event and for-all controls.
func (s *ChatService) DeleteChatWithOptions(ctx context.Context, chatID, lastEventTime int64, forAll bool) error {
	payload := map[string]interface{}{
		"chatId":        chatID,
		"lastEventTime": lastEventTime,
		"forAll":        forAll,
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpChatDelete, payload)
	if err != nil {
		return fmt.Errorf("delete chat failed: %w", err)
	}
	s.removeCachedChat(chatID)
	return nil
}

// GroupSettings is the complete partial-update settings set supported by PyMax.
type GroupSettings struct {
	OnlyOwnerCanChangeIconTitle *bool
	AllCanPinMessage            *bool
	OnlyAdminCanAddMember       *bool
	OnlyAdminCanCall            *bool
	MembersCanSeePrivateLink    *bool
}

// ChangeGroupSettingsFull updates only non-nil options.
func (s *ChatService) ChangeGroupSettingsFull(ctx context.Context, chatID int64, settings GroupSettings) error {
	options := make(map[string]bool)
	for key, value := range map[string]*bool{
		"ONLY_OWNER_CAN_CHANGE_ICON_TITLE": settings.OnlyOwnerCanChangeIconTitle,
		"ALL_CAN_PIN_MESSAGE":              settings.AllCanPinMessage,
		"ONLY_ADMIN_CAN_ADD_MEMBER":        settings.OnlyAdminCanAddMember,
		"ONLY_ADMIN_CAN_CALL":              settings.OnlyAdminCanCall,
		"MEMBERS_CAN_SEE_PRIVATE_LINK":     settings.MembersCanSeePrivateLink,
	} {
		if value != nil {
			options[key] = *value
		}
	}
	return s.ChangeGroupSettingsWithOptions(ctx, chatID, options)
}

// ChangeGroupSettingsAction implements types.ChatActions.
func (s *ChatService) ChangeGroupSettingsAction(ctx context.Context, chatID int64, settings types.ChatSettings) error {
	return s.ChangeGroupSettingsFull(ctx, chatID, GroupSettings{
		OnlyOwnerCanChangeIconTitle: settings.OnlyOwnerCanChangeIconTitle,
		AllCanPinMessage:            settings.AllCanPinMessage,
		OnlyAdminCanAddMember:       settings.OnlyAdminCanAddMember,
		OnlyAdminCanCall:            settings.OnlyAdminCanCall,
		MembersCanSeePrivateLink:    settings.MembersCanSeePrivateLink,
	})
}

// ChangeGroupSettings updates permissions and settings for a group chat.
func (s *ChatService) ChangeGroupSettings(ctx context.Context, chatID int64, allCanPin bool, onlyAdminCanAdd bool) error {
	payload := map[string]interface{}{
		"chatId": chatID,
		"options": map[string]interface{}{
			"ALL_CAN_PIN_MESSAGE":       allCanPin,
			"ONLY_ADMIN_CAN_ADD_MEMBER": onlyAdminCanAdd,
		},
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpChatUpdate, payload)
	if err != nil {
		return fmt.Errorf("change group settings failed: %w", err)
	}
	return nil
}

// ChangeGroupSettingsWithOptions updates every optional group setting. Nil
// values are omitted, which preserves the PyMax partial-update semantics.
func (s *ChatService) ChangeGroupSettingsWithOptions(ctx context.Context, chatID int64, options map[string]bool) error {
	normalized := make(map[string]bool, len(options))
	aliases := map[string]string{
		"onlyOwnerCanChangeIconTitle": "ONLY_OWNER_CAN_CHANGE_ICON_TITLE",
		"allCanPinMessage":            "ALL_CAN_PIN_MESSAGE",
		"onlyAdminCanAddMember":       "ONLY_ADMIN_CAN_ADD_MEMBER",
		"onlyAdminCanCall":            "ONLY_ADMIN_CAN_CALL",
		"membersCanSeePrivateLink":    "MEMBERS_CAN_SEE_PRIVATE_LINK",
	}
	for key, value := range options {
		if alias, ok := aliases[key]; ok {
			key = alias
		}
		normalized[key] = value
	}
	payload := map[string]interface{}{"chatId": chatID, "options": normalized}
	if _, err := s.invoker.Invoke(ctx, protocol.OpChatUpdate, payload); err != nil {
		return fmt.Errorf("change group settings failed: %w", err)
	}
	return nil
}

// ChangeGroupProfile updates the group title/description and optional photo token.
func (s *ChatService) ChangeGroupProfile(ctx context.Context, chatID int64, name, description, photoToken string) error {
	payload := map[string]interface{}{"chatId": chatID}
	if name != "" {
		payload["theme"] = name
	}
	if description != "" {
		payload["description"] = description
	}
	if photoToken != "" {
		payload["photoToken"] = photoToken
	}
	if _, err := s.invoker.Invoke(ctx, protocol.OpChatUpdate, payload); err != nil {
		return fmt.Errorf("change group profile failed: %w", err)
	}
	return nil
}

// GetChatMembers returns the list of members in a chat.
func (s *ChatService) GetChatMembers(ctx context.Context, chatID int64, count int, marker string) ([]types.Member, string, error) {
	if count <= 0 {
		count = 50
	}
	markerValue := 0
	if marker != "" {
		if parsed, err := strconv.Atoi(marker); err == nil {
			markerValue = parsed
		}
	}
	payload := map[string]interface{}{
		"chatId": chatID,
		"type":   "MEMBER",
		"count":  count,
		"marker": markerValue,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatMembers, payload)
	if err != nil {
		return nil, "", fmt.Errorf("get chat members failed: %w", err)
	}

	var members []types.Member
	var nextMarker string

	if value, ok := chatInt64(res["marker"]); ok {
		nextMarker = strconv.FormatInt(value, 10)
	}

	if membersList, ok := res["members"].([]interface{}); ok {
		for _, item := range membersList {
			if m, ok := item.(map[string]interface{}); ok {
				members = append(members, parseMember(m))
			}
		}
	}

	return members, nextMarker, nil
}

// GetChatMembersPage is the typed pagination variant used by PyMax.
func (s *ChatService) GetChatMembersPage(ctx context.Context, chatID int64, marker, count int) ([]types.Member, int, error) {
	if count <= 0 {
		count = 50
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpChatMembers, map[string]interface{}{
		"type": "MEMBER", "chatId": chatID, "marker": marker, "count": count,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get chat members failed: %w", err)
	}
	var members []types.Member
	if list, ok := res["members"].([]interface{}); ok {
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				members = append(members, parseMember(m))
			}
		}
	}
	next, _ := chatInt64(res["marker"])
	return members, int(next), nil
}

// FetchChats retrieves dialogs and active chats list.
func (s *ChatService) FetchChats(ctx context.Context, count int, marker string) ([]types.Chat, string, error) {
	_ = count // retained for source compatibility; Max accepts only marker here.
	markerValue := time.Now().UnixMilli()
	if marker != "" {
		if parsed, err := strconv.ParseInt(marker, 10, 64); err == nil {
			markerValue = parsed
		}
	}
	payload := map[string]interface{}{"marker": markerValue}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatsList, payload)
	if err != nil {
		return nil, "", fmt.Errorf("fetch chats failed: %w", err)
	}

	var chats []types.Chat
	var nextMarker string
	if value, ok := chatInt64(res["marker"]); ok {
		nextMarker = strconv.FormatInt(value, 10)
	}

	if chatsList, ok := res["chats"].([]interface{}); ok {
		for _, item := range chatsList {
			if c, ok := item.(map[string]interface{}); ok {
				chats = append(chats, s.cacheChat(types.ParseChatPayload(c)))
			}
		}
	}

	return chats, nextMarker, nil
}

// GetChats retrieves chat metadata for the requested IDs.
func (s *ChatService) GetChats(ctx context.Context, chatIDs []int64) ([]types.Chat, error) {
	found := make(map[int64]types.Chat, len(chatIDs))
	missing := make([]int64, 0, len(chatIDs))
	for _, chatID := range chatIDs {
		if chat, ok := s.cachedChat(chatID); ok {
			found[chatID] = chat
		} else {
			missing = append(missing, chatID)
		}
	}
	if len(missing) > 0 {
		res, err := s.invoker.Invoke(ctx, protocol.OpChatInfo, map[string]interface{}{"chatIds": missing})
		if err != nil {
			return nil, fmt.Errorf("get chats failed: %w", err)
		}
		if list, ok := res["chats"].([]interface{}); ok {
			for _, item := range list {
				if m, ok := item.(map[string]interface{}); ok {
					chat := s.cacheChat(types.ParseChatPayload(m))
					found[chat.ID] = chat
				}
			}
		}
	}
	out := make([]types.Chat, 0, len(chatIDs))
	for _, chatID := range chatIDs {
		if chat, ok := found[chatID]; ok {
			out = append(out, chat)
		}
	}
	return out, nil
}

// GetChat is a concise alias for GetChatInfo.
func (s *ChatService) GetChat(ctx context.Context, chatID int64) (*types.Chat, error) {
	chats, err := s.GetChats(ctx, []int64{chatID})
	if err != nil {
		return nil, err
	}
	if len(chats) == 0 {
		return nil, fmt.Errorf("chat %d was not found in response", chatID)
	}
	return &chats[0], nil
}

// LeaveGroup and LeaveChannel are explicit aliases matching PyMax's API.
func (s *ChatService) LeaveGroup(ctx context.Context, chatID int64) error {
	return s.LeaveChat(ctx, chatID)
}

// LeaveChannel leaves a channel.
func (s *ChatService) LeaveChannel(ctx context.Context, chatID int64) error {
	return s.LeaveChat(ctx, chatID)
}

// GetJoinRequests returns pending membership requests for a chat.
func (s *ChatService) GetJoinRequests(ctx context.Context, chatID int64, count int) ([]types.Member, error) {
	if count <= 0 {
		count = 100
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpChatMembers, map[string]interface{}{
		"chatId": chatID, "type": "JOIN_REQUEST", "count": count,
	})
	if err != nil {
		return nil, fmt.Errorf("get join requests failed: %w", err)
	}
	var out []types.Member
	if list, ok := res["members"].([]interface{}); ok {
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, parseMember(m))
			}
		}
	}
	return out, nil
}

// ConfirmJoinRequests approves pending membership requests.
func (s *ChatService) ConfirmJoinRequests(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error {
	return s.memberRequestAction(ctx, chatID, userIDs, "add", &showHistory)
}

// ConfirmJoinRequest approves one pending membership request.
func (s *ChatService) ConfirmJoinRequest(ctx context.Context, chatID, userID int64, showHistory bool) error {
	return s.ConfirmJoinRequests(ctx, chatID, []int64{userID}, showHistory)
}

// DeclineJoinRequests rejects pending membership requests.
func (s *ChatService) DeclineJoinRequests(ctx context.Context, chatID int64, userIDs []int64) error {
	return s.memberRequestAction(ctx, chatID, userIDs, "remove", nil)
}

// DeclineJoinRequest rejects one pending membership request.
func (s *ChatService) DeclineJoinRequest(ctx context.Context, chatID, userID int64) error {
	return s.DeclineJoinRequests(ctx, chatID, []int64{userID})
}

func (s *ChatService) memberRequestAction(ctx context.Context, chatID int64, userIDs []int64, operation string, showHistory *bool) error {
	payload := map[string]interface{}{"chatId": chatID, "userIds": userIDs, "type": "JOIN_REQUEST", "operation": operation}
	if showHistory != nil {
		payload["showHistory"] = *showHistory
	}
	if _, err := s.invoker.Invoke(ctx, protocol.OpChatMembersUpdate, payload); err != nil {
		return fmt.Errorf("join request action failed: %w", err)
	}
	return nil
}

// AddAdmin grants channel/group administrator permissions represented as the
// protocol's bit mask.
func (s *ChatService) AddAdmin(ctx context.Context, chatID, userID int64, permissions int) error {
	_, err := s.invoker.Invoke(ctx, protocol.OpChatMembersUpdate, map[string]interface{}{
		"chatId": chatID, "userIds": []int64{userID}, "type": "ADMIN", "operation": "add", "permissions": permissions,
	})
	if err != nil {
		return fmt.Errorf("add admin failed: %w", err)
	}
	return nil
}

// ReworkInviteLink regenerates the invitation link for a chat.
func (s *ChatService) ReworkInviteLink(ctx context.Context, chatID int64) (string, error) {
	payload := map[string]interface{}{
		"chatId":            chatID,
		"revokePrivateLink": true,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpChatUpdate, payload)
	if err != nil {
		return "", fmt.Errorf("rework invite link failed: %w", err)
	}
	if link, ok := res["link"].(string); ok {
		return link, nil
	}
	return "", nil
}

// ReworkInviteLinkChat returns the updated chat model, as PyMax does.
func (s *ChatService) ReworkInviteLinkChat(ctx context.Context, chatID int64) (*types.Chat, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpChatUpdate, map[string]interface{}{
		"chatId": chatID, "revokePrivateLink": true,
	})
	if err != nil {
		return nil, fmt.Errorf("rework invite link failed: %w", err)
	}
	if raw, ok := res["chat"].(map[string]interface{}); ok {
		chat := s.cacheChat(types.ParseChatPayload(raw))
		return &chat, nil
	}
	chat := s.cacheChat(types.ParseChatPayload(res))
	return &chat, nil
}

// FetchChatsFromMarker uses PyMax's millisecond integer marker.
func (s *ChatService) FetchChatsFromMarker(ctx context.Context, marker int64) ([]types.Chat, error) {
	if marker == 0 {
		marker = time.Now().UnixMilli()
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpChatsList, map[string]interface{}{"marker": marker})
	if err != nil {
		return nil, fmt.Errorf("fetch chats failed: %w", err)
	}
	var out []types.Chat
	if list, ok := res["chats"].([]interface{}); ok {
		for _, item := range list {
			if raw, ok := item.(map[string]interface{}); ok {
				out = append(out, s.cacheChat(types.ParseChatPayload(raw)))
			}
		}
	}
	return out, nil
}

// GetChatInfo retrieves metadata (title, type, members count, etc.) for a specific chat.
func (s *ChatService) GetChatInfo(ctx context.Context, chatID int64) (*types.Chat, error) {
	payload := map[string]interface{}{
		"chatIds": []int64{chatID},
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpChatInfo, payload)
	if err != nil {
		return nil, fmt.Errorf("get chat info failed: %w", err)
	}

	chat := &types.Chat{ID: chatID}
	src := res
	if list, ok := res["chats"].([]interface{}); ok && len(list) > 0 {
		if first, ok := list[0].(map[string]interface{}); ok {
			src = first
		}
	}
	if cData, ok := res["chat"].(map[string]interface{}); ok {
		src = cData
	}
	if id, ok := src["id"].(int64); ok {
		chat.ID = id
	} else if idF, ok := src["id"].(float64); ok {
		chat.ID = int64(idF)
	}
	if title, ok := src["title"].(string); ok {
		chat.Title = title
	}
	if desc, ok := src["description"].(string); ok {
		chat.Description = desc
	}
	if isChannel, ok := src["isChannel"].(bool); ok {
		chat.IsChannel = isChannel
	}
	if isPublic, ok := src["isPublic"].(bool); ok {
		chat.IsPublic = isPublic
	}
	if count, ok := src["membersCount"].(float64); ok {
		chat.MembersCount = int(count)
	}
	parsed := types.ParseChatPayload(src)
	if parsed.ID == 0 {
		parsed.ID = chat.ID
	}
	parsed = s.cacheChat(parsed)
	return &parsed, nil
}

// PublicSearch searches public channels and groups by query string.
func (s *ChatService) PublicSearch(ctx context.Context, query string, count int) ([]types.Chat, error) {
	if count <= 0 {
		count = 20
	}
	payload := map[string]interface{}{
		"query": query,
		"count": count,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpPublicSearch, payload)
	if err != nil {
		return nil, fmt.Errorf("public search failed: %w", err)
	}

	var chats []types.Chat
	if rawList, ok := res["chats"].([]interface{}); ok {
		for _, item := range rawList {
			if cData, ok := item.(map[string]interface{}); ok {
				chat := types.Chat{}
				if id, ok := cData["id"].(int64); ok {
					chat.ID = id
				} else if idF, ok := cData["id"].(float64); ok {
					chat.ID = int64(idF)
				}
				if title, ok := cData["title"].(string); ok {
					chat.Title = title
				}
				if desc, ok := cData["description"].(string); ok {
					chat.Description = desc
				}
				chats = append(chats, chat)
			}
		}
	}
	return chats, nil
}

func chatInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	case uint64:
		return int64(v), true
	case string:
		var n int64
		if _, err := fmt.Sscan(v, &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

func mustChatInt64(value interface{}) int64 { n, _ := chatInt64(value); return n }
func stringValue(value interface{}) string  { v, _ := value.(string); return v }

func chatFromMap(m map[string]interface{}) *types.Chat {
	chat := types.ParseChatPayload(m)
	return &chat
}

func processJoinLink(link string) (string, bool) {
	index := strings.Index(link, "join/")
	if index < 0 {
		return "", false
	}
	return link[index:], true
}

func parseMember(raw map[string]interface{}) types.Member {
	member := types.Member{UserID: mustChatInt64(raw["userId"]), Role: stringValue(raw["role"])}
	if contact, ok := raw["contact"].(map[string]interface{}); ok {
		parsed := types.ParseUserPayload(contact)
		member.Contact = &parsed
		if member.UserID == 0 {
			member.UserID = parsed.ID
		}
	}
	if presence, ok := raw["presence"].(map[string]interface{}); ok {
		parsed := &types.Presence{}
		parsed.Seen, _ = types.Int64Value(presence["seen"])
		parsed.StatusCode, _ = types.Int64Value(presence["status"])
		member.Presence = parsed
	}
	return member
}
