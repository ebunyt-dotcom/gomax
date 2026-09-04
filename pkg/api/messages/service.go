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
