package messages

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"gomax/pkg/api"
	"gomax/pkg/protocol"
	"gomax/pkg/types"
)

// MessageService handles messages, reactions, mass-looking (reading), and history.
type MessageService struct {
	invoker api.Invoker
	prevCID int64
}

// NewMessageService creates a new MessageService instance.
func NewMessageService(invoker api.Invoker) *MessageService {
	return &MessageService{
		invoker: invoker,
		prevCID: time.Now().UnixMilli(),
	}
}

func (s *MessageService) nextCID() int64 {
	now := time.Now().UnixMilli()
	for {
		prev := atomic.LoadInt64(&s.prevCID)
		next := now
		if next <= prev {
			next = prev + 1
		}
		if atomic.CompareAndSwapInt64(&s.prevCID, prev, next) {
			return next
		}
	}
}

// SendMessage sends a text message with optional attachments to a chat.
func (s *MessageService) SendMessage(ctx context.Context, chatID int64, text string, replyToMsgID int64, attaches []types.Attachment) (*types.Message, error) {
	cid := s.nextCID()

	msgPayload := map[string]interface{}{
		"cid":  cid,
		"text": text,
	}
	if replyToMsgID > 0 {
		msgPayload["replyTo"] = replyToMsgID
	}
	if len(attaches) > 0 {
		var rawAttaches []interface{}
		for _, a := range attaches {
			rawAttaches = append(rawAttaches, map[string]interface{}{
				"type":  a.Type,
				"token": a.Token,
				"url":   a.URL,
			})
		}
		msgPayload["attaches"] = rawAttaches
	}

	payload := map[string]interface{}{
		"chatId":  chatID,
		"message": msgPayload,
		"notify":  true,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpMsgSend, payload)
	if err != nil {
		return nil, fmt.Errorf("send message failed: %w", err)
	}

	msg := &types.Message{
		CID:      cid,
		ChatID:   chatID,
		Text:     text,
		Time:     time.Now().Unix(),
		IsOutgoing: true,
	}

	if msgData, ok := res["message"].(map[string]interface{}); ok {
		if id, ok := msgData["id"].(int64); ok {
			msg.ID = id
		} else if idF, ok := msgData["id"].(float64); ok {
			msg.ID = int64(idF)
		}
	}

	return msg, nil
}

// AddReaction adds an emoji reaction to a message.
func (s *MessageService) AddReaction(ctx context.Context, chatID int64, messageID int64, reaction string) error {
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
		"reaction": map[string]interface{}{
			"id": reaction,
		},
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpMsgReaction, payload)
	if err != nil {
		return fmt.Errorf("add reaction failed: %w", err)
	}
	return nil
}

// RemoveReaction removes a previously placed reaction.
func (s *MessageService) RemoveReaction(ctx context.Context, chatID int64, messageID int64, reaction string) error {
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpMsgCancelReaction, payload)
	if err != nil {
		return fmt.Errorf("remove reaction failed: %w", err)
	}
	return nil
}

// ReadMessages marks specific messages as read (mass-look / views).
func (s *MessageService) ReadMessages(ctx context.Context, chatID int64, messageIDs []int64) error {
	payload := map[string]interface{}{
		"chatId":     chatID,
		"messageIds": messageIDs,
		"type":       "READ",
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpChatMark, payload)
	if err != nil {
		return fmt.Errorf("read messages failed: %w", err)
	}
	return nil
}

// ReadChat marks a chat as read up to a message ID.
func (s *MessageService) ReadChat(ctx context.Context, chatID int64, markID int64) error {
	payload := map[string]interface{}{
		"chatId": chatID,
		"mark":   markID,
		"type":   "READ",
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpChatMark, payload)
	if err != nil {
		return fmt.Errorf("read chat failed: %w", err)
	}
	return nil
}

// GetChatHistory fetches chat message history.
func (s *MessageService) GetChatHistory(ctx context.Context, chatID int64, fromTime int64, count int) ([]types.Message, error) {
	if count <= 0 {
		count = 50
	}
	payload := map[string]interface{}{
		"chatId": chatID,
		"count":  count,
	}
	if fromTime > 0 {
		payload["time"] = fromTime
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpChatHistory, payload)
	if err != nil {
		return nil, fmt.Errorf("get chat history failed: %w", err)
	}

	var messages []types.Message
	if rawList, ok := res["messages"].([]interface{}); ok {
		for _, item := range rawList {
			if m, ok := item.(map[string]interface{}); ok {
				var msg types.Message
				msg.ChatID = chatID
				if id, ok := m["id"].(int64); ok {
					msg.ID = id
				} else if idF, ok := m["id"].(float64); ok {
					msg.ID = int64(idF)
				}
				if text, ok := m["text"].(string); ok {
					msg.Text = text
				}
				if sender, ok := m["sender"].(int64); ok {
					msg.SenderID = sender
				} else if senderF, ok := m["sender"].(float64); ok {
					msg.SenderID = int64(senderF)
				}
				messages = append(messages, msg)
			}
		}
	}
	return messages, nil
}

// GetHistory retrieves message history for a chat (alias for GetChatHistory matching API specification).
func (s *MessageService) GetHistory(ctx context.Context, chatID int64, fromTime int64, count int) ([]types.Message, error) {
	return s.GetChatHistory(ctx, chatID, fromTime, count)
}

// EditMessage edits an existing message text.
func (s *MessageService) EditMessage(ctx context.Context, chatID int64, messageID int64, newText string) error {
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
		"text":      newText,
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpMsgEdit, payload)
	if err != nil {
		return fmt.Errorf("edit message failed: %w", err)
	}
	return nil
}

// DeleteMessage deletes a message.
func (s *MessageService) DeleteMessage(ctx context.Context, chatID int64, messageID int64, forAll bool) error {
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
		"forAll":    forAll,
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpMsgDelete, payload)
	if err != nil {
		return fmt.Errorf("delete message failed: %w", err)
	}
	return nil
}

// ForwardMessages forwards messages from one chat to another.
func (s *MessageService) ForwardMessages(ctx context.Context, toChatID int64, fromChatID int64, messageIDs []int64) error {
	payload := map[string]interface{}{
		"toChatId":   toChatID,
		"fromChatId": fromChatID,
		"messageIds": messageIDs,
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpMsgSend, payload)
	if err != nil {
		return fmt.Errorf("forward messages failed: %w", err)
	}
	return nil
}

// PinMessage pins a message to the top of the chat.
func (s *MessageService) PinMessage(ctx context.Context, chatID int64, messageID int64) error {
	payload := map[string]interface{}{
		"chatId":        chatID,
		"pinMessageId":  messageID,
		"notifyPin":     true,
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpChatUpdate, payload)
	if err != nil {
		return fmt.Errorf("pin message failed: %w", err)
	}
	return nil
}

// VotePoll votes for option(s) in a poll.
func (s *MessageService) VotePoll(ctx context.Context, chatID int64, messageID int64, pollID int64, optionIDs []int) error {
	payload := map[string]interface{}{
		"chatId":     chatID,
		"messageId":  messageID,
		"pollId":     pollID,
		"answersIds": optionIDs,
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpSendVote, payload)
	if err != nil {
		return fmt.Errorf("vote poll failed: %w", err)
	}
	return nil
}

// GetMessages retrieves messages by their IDs.
func (s *MessageService) GetMessages(ctx context.Context, chatID int64, messageIDs []int64) ([]types.Message, error) {
	payload := map[string]interface{}{
		"chatId":     chatID,
		"messageIds": messageIDs,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpMsgGet, payload)
	if err != nil {
		return nil, fmt.Errorf("get messages failed: %w", err)
	}

	var messages []types.Message
	if rawList, ok := res["messages"].([]interface{}); ok {
		for _, item := range rawList {
			if m, ok := item.(map[string]interface{}); ok {
				var msg types.Message
				msg.ChatID = chatID
				if id, ok := m["id"].(int64); ok {
					msg.ID = id
				} else if idF, ok := m["id"].(float64); ok {
					msg.ID = int64(idF)
				}
				if text, ok := m["text"].(string); ok {
					msg.Text = text
				}
				if sender, ok := m["sender"].(int64); ok {
					msg.SenderID = sender
				} else if senderF, ok := m["sender"].(float64); ok {
					msg.SenderID = int64(senderF)
				}
				messages = append(messages, msg)
			}
		}
	}
	return messages, nil
}

// GetMessage retrieves a single message by ID.
func (s *MessageService) GetMessage(ctx context.Context, chatID int64, messageID int64) (*types.Message, error) {
	msgs, err := s.GetMessages(ctx, chatID, []int64{messageID})
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message %d not found", messageID)
	}
	return &msgs[0], nil
}

// FetchHistory is an alias for GetChatHistory matching PyMax fetch_history.
func (s *MessageService) FetchHistory(ctx context.Context, chatID int64, fromTime int64, count int) ([]types.Message, error) {
	return s.GetChatHistory(ctx, chatID, fromTime, count)
}

// GetReactions retrieves reaction information for specific messages.
func (s *MessageService) GetReactions(ctx context.Context, chatID int64, messageIDs []int64) (map[int64][]types.ReactionInfo, error) {
	payload := map[string]interface{}{
		"chatId":     chatID,
		"messageIds": messageIDs,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpMsgGetReactions, payload)
	if err != nil {
		return nil, fmt.Errorf("get reactions failed: %w", err)
	}

	result := make(map[int64][]types.ReactionInfo)
	if rawReactions, ok := res["messagesReactions"].(map[string]interface{}); ok {
		for k, v := range rawReactions {
			var msgID int64
			fmt.Sscanf(k, "%d", &msgID)
			if rList, ok := v.([]interface{}); ok {
				var infoList []types.ReactionInfo
				for _, rItem := range rList {
					if rMap, ok := rItem.(map[string]interface{}); ok {
						var ri types.ReactionInfo
						if r, ok := rMap["reaction"].(string); ok {
							ri.Reaction = r
						}
						if c, ok := rMap["count"].(int); ok {
							ri.Count = c
						} else if cF, ok := rMap["count"].(float64); ok {
							ri.Count = int(cF)
						}
						if self, ok := rMap["self"].(bool); ok {
							ri.Self = self
						}
						infoList = append(infoList, ri)
					}
				}
				result[msgID] = infoList
			}
		}
	}
	return result, nil
}

// GetVideoByID requests video playback info.
func (s *MessageService) GetVideoByID(ctx context.Context, chatID int64, messageID int64, videoID int64) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
		"videoId":   videoID,
	}
	return s.invoker.Invoke(ctx, protocol.OpVideoPlay, payload)
}

// GetFileByID requests file download metadata.
func (s *MessageService) GetFileByID(ctx context.Context, chatID int64, messageID int64, fileID int64) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
		"fileId":    fileID,
	}
	return s.invoker.Invoke(ctx, protocol.OpFileDownload, payload)
}
