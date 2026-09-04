package chats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// ChatService handles all chat, channel and group related operations.
type ChatService struct {
	invoker api.Invoker
}

// NewChatService creates a new ChatService instance.
func NewChatService(invoker api.Invoker) *ChatService {
	return &ChatService{invoker: invoker}
}

// JoinChat joins a public or private chat by invite link or hash.
func (s *ChatService) JoinChat(ctx context.Context, link string) (*types.Chat, error) {
	cleanLink := link
	if idx := strings.Index(link, "join/"); idx != -1 {
		cleanLink = link[idx:]
	}

	payload := map[string]interface{}{
		"link": cleanLink,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatJoin, payload)
	if err != nil {
		return nil, fmt.Errorf("join chat failed: %w", err)
	}

	chat := &types.Chat{
		InviteLink: link,
	}
	if chatData, ok := res["chat"].(map[string]interface{}); ok {
		if id, ok := chatData["id"].(int64); ok {
			chat.ID = id
		} else if idFloat, ok := chatData["id"].(float64); ok {
			chat.ID = int64(idFloat)
		}
		if title, ok := chatData["title"].(string); ok {
			chat.Title = title
		}
	}
	return chat, nil
}

// InviteUsersToGroup adds users to a group chat by their user IDs.
func (s *ChatService) InviteUsersToGroup(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error {
	payload := map[string]interface{}{
		"chatId":      chatID,
		"userIds":     userIDs,
		"showHistory": showHistory,
		"operation":   "ADD",
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpChatMembersUpdate, payload)
	if err != nil {
		return fmt.Errorf("invite users to group failed: %w", err)
	}
	return nil
}

// InviteUsersToChannel adds users to a channel by user IDs.
func (s *ChatService) InviteUsersToChannel(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error {
	return s.InviteUsersToGroup(ctx, chatID, userIDs, showHistory)
}

// JoinGroup joins a group using the same invite-link protocol as JoinChat.
func (s *ChatService) JoinGroup(ctx context.Context, link string) (*types.Chat, error) {
	return s.JoinChat(ctx, link)
}

// JoinChannel joins a channel using an invite link.
func (s *ChatService) JoinChannel(ctx context.Context, link string) (*types.Chat, error) {
	return s.JoinChat(ctx, link)
}

// ResolveGroupByLink resolves an invite link without joining the chat.
func (s *ChatService) ResolveGroupByLink(ctx context.Context, link string) (*types.Chat, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpLinkInfo, map[string]interface{}{"link": link})
	if err != nil {
		return nil, fmt.Errorf("resolve group link failed: %w", err)
	}
	if raw, ok := res["chat"].(map[string]interface{}); ok {
		return chatFromMap(raw), nil
	}
	return chatFromMap(res), nil
}

// RemoveUsersFromGroup kicks users from a group.
func (s *ChatService) RemoveUsersFromGroup(ctx context.Context, chatID int64, userIDs []int64, cleanMsgPeriod int) error {
	payload := map[string]interface{}{
		"chatId":         chatID,
		"userIds":        userIDs,
		"cleanMsgPeriod": cleanMsgPeriod,
		"operation":      "REMOVE",
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpChatMembersUpdate, payload)
	if err != nil {
		return fmt.Errorf("remove users from group failed: %w", err)
	}
	return nil
}

// CreateGroup creates a new group chat with initial participants.
func (s *ChatService) CreateGroup(ctx context.Context, name string, participantIDs []int64, notify bool) (*types.Chat, error) {
	nowMs := time.Now().UnixMilli()
	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"cid": nowMs,
			"attaches": []interface{}{
				map[string]interface{}{
					"_type":    "CONTROL",
					"event":    "NEW",
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
		return nil, fmt.Errorf("create group failed: %w", err)
	}

	chat := &types.Chat{
		Title: name,
	}
	if chatData, ok := res["chat"].(map[string]interface{}); ok {
		if id, ok := chatData["id"].(int64); ok {
			chat.ID = id
		} else if idFloat, ok := chatData["id"].(float64); ok {
			chat.ID = int64(idFloat)
		}
	}
	return chat, nil
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
	return nil
}

// DeleteChat deletes a conversation completely.
func (s *ChatService) DeleteChat(ctx context.Context, chatID int64) error {
	payload := map[string]interface{}{
		"chatId":        chatID,
		"lastEventTime": time.Now().UnixMilli(),
		"forAll":        true,
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpChatDelete, payload)
	if err != nil {
		return fmt.Errorf("delete chat failed: %w", err)
	}
	return nil
}

// ChangeGroupSettings updates permissions and settings for a group chat.
func (s *ChatService) ChangeGroupSettings(ctx context.Context, chatID int64, allCanPin bool, onlyAdminCanAdd bool) error {
	payload := map[string]interface{}{
		"chatId": chatID,
		"options": map[string]interface{}{
			"allCanPinMessage":      allCanPin,
			"onlyAdminCanAddMember": onlyAdminCanAdd,
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
	payload := map[string]interface{}{"chatId": chatID, "options": options}
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
	payload := map[string]interface{}{
		"chatId": chatID,
		"type":   "MEMBER",
		"count":  count,
	}
	if marker != "" {
		payload["marker"] = marker
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatMembers, payload)
	if err != nil {
		return nil, "", fmt.Errorf("get chat members failed: %w", err)
	}

	var members []types.Member
	var nextMarker string

	if mStr, ok := res["marker"].(string); ok {
		nextMarker = mStr
	}

	if membersList, ok := res["members"].([]interface{}); ok {
		for _, item := range membersList {
			if m, ok := item.(map[string]interface{}); ok {
				var mem types.Member
				if uid, ok := m["userId"].(int64); ok {
					mem.UserID = uid
				} else if uidF, ok := m["userId"].(float64); ok {
					mem.UserID = int64(uidF)
				}
				if role, ok := m["role"].(string); ok {
					mem.Role = role
				}
				members = append(members, mem)
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
				member := types.Member{}
				if v, ok := chatInt64(m["userId"]); ok {
					member.UserID = v
				}
				if v, ok := m["role"].(string); ok {
					member.Role = v
				}
				members = append(members, member)
			}
		}
	}
	next, _ := chatInt64(res["marker"])
	return members, int(next), nil
}

// FetchChats retrieves dialogs and active chats list.
func (s *ChatService) FetchChats(ctx context.Context, count int, marker string) ([]types.Chat, string, error) {
	if count <= 0 {
		count = 40
	}
	payload := map[string]interface{}{
		"count": count,
	}
	if marker != "" {
		payload["marker"] = marker
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatsList, payload)
	if err != nil {
		return nil, "", fmt.Errorf("fetch chats failed: %w", err)
	}

	var chats []types.Chat
	var nextMarker string
	if mStr, ok := res["marker"].(string); ok {
		nextMarker = mStr
	}

	if chatsList, ok := res["chats"].([]interface{}); ok {
		for _, item := range chatsList {
			if c, ok := item.(map[string]interface{}); ok {
				var chat types.Chat
				if id, ok := c["id"].(int64); ok {
					chat.ID = id
				} else if idF, ok := c["id"].(float64); ok {
					chat.ID = int64(idF)
				}
				if title, ok := c["title"].(string); ok {
					chat.Title = title
				}
				chats = append(chats, chat)
			}
		}
	}

	return chats, nextMarker, nil
}

// GetChats retrieves chat metadata for the requested IDs.
func (s *ChatService) GetChats(ctx context.Context, chatIDs []int64) ([]types.Chat, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpChatInfo, map[string]interface{}{"chatIds": chatIDs})
	if err != nil {
		return nil, fmt.Errorf("get chats failed: %w", err)
	}
	var out []types.Chat
	if list, ok := res["chats"].([]interface{}); ok {
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, *chatFromMap(m))
			}
		}
	}
	return out, nil
}

// GetChat is a concise alias for GetChatInfo.
func (s *ChatService) GetChat(ctx context.Context, chatID int64) (*types.Chat, error) {
	return s.GetChatInfo(ctx, chatID)
}

// LeaveGroup and LeaveChannel are explicit aliases matching PyMax's API.
func (s *ChatService) LeaveGroup(ctx context.Context, chatID int64) error {
	return s.LeaveChat(ctx, chatID)
}
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
				out = append(out, types.Member{UserID: mustChatInt64(m["userId"]), Role: stringValue(m["role"])})
			}
		}
	}
	return out, nil
}

// ConfirmJoinRequests approves pending membership requests.
func (s *ChatService) ConfirmJoinRequests(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error {
	return s.memberRequestAction(ctx, chatID, userIDs, "ADD", &showHistory)
}

// ConfirmJoinRequest approves one pending membership request.
func (s *ChatService) ConfirmJoinRequest(ctx context.Context, chatID, userID int64, showHistory bool) error {
	return s.ConfirmJoinRequests(ctx, chatID, []int64{userID}, showHistory)
}

// DeclineJoinRequests rejects pending membership requests.
func (s *ChatService) DeclineJoinRequests(ctx context.Context, chatID int64, userIDs []int64) error {
	return s.memberRequestAction(ctx, chatID, userIDs, "REMOVE", nil)
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
		return chatFromMap(raw), nil
	}
	return chatFromMap(res), nil
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
				out = append(out, *chatFromMap(raw))
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
	return chat, nil
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
	chat := &types.Chat{}
	if v, ok := chatInt64(m["id"]); ok {
		chat.ID = v
	}
	if v, ok := m["title"].(string); ok {
		chat.Title = v
	}
	if v, ok := m["description"].(string); ok {
		chat.Description = v
	}
	if v, ok := m["isChannel"].(bool); ok {
		chat.IsChannel = v
	}
	if v, ok := m["isPublic"].(bool); ok {
		chat.IsPublic = v
	}
	if v, ok := chatInt64(m["membersCount"]); ok {
		chat.MembersCount = int(v)
	}
	return chat
}
