package chats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gomax/pkg/api"
	"gomax/pkg/protocol"
	"gomax/pkg/types"
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

	_, err := s.invoker.Invoke(ctx, protocol.OpChatMembers, payload)
	if err != nil {
		return fmt.Errorf("invite users to group failed: %w", err)
	}
	return nil
}

// InviteUsersToChannel adds users to a channel by user IDs.
func (s *ChatService) InviteUsersToChannel(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error {
	return s.InviteUsersToGroup(ctx, chatID, userIDs, showHistory)
}

// RemoveUsersFromGroup kicks users from a group.
func (s *ChatService) RemoveUsersFromGroup(ctx context.Context, chatID int64, userIDs []int64, cleanMsgPeriod int) error {
	payload := map[string]interface{}{
		"chatId":         chatID,
		"userIds":        userIDs,
		"cleanMsgPeriod": cleanMsgPeriod,
		"operation":      "REMOVE",
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpChatMembers, payload)
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
					"title":   name,
					"userIds": participantIDs,
				},
			},
		},
		"notify": notify,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatCreate, payload)
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
		"chatId": chatID,
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
			"allCanPinMessage":         allCanPin,
			"onlyAdminCanAddMember":    onlyAdminCanAdd,
		},
	}
	_, err := s.invoker.Invoke(ctx, protocol.OpChatUpdate, payload)
	if err != nil {
		return fmt.Errorf("change group settings failed: %w", err)
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

// ReworkInviteLink regenerates the invitation link for a chat.
func (s *ChatService) ReworkInviteLink(ctx context.Context, chatID int64) (string, error) {
	payload := map[string]interface{}{
		"chatId": chatID,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpChatCheckLink, payload)
	if err != nil {
		return "", fmt.Errorf("rework invite link failed: %w", err)
	}
	if link, ok := res["link"].(string); ok {
		return link, nil
	}
	return "", nil
}
