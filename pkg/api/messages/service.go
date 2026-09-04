package messages

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
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
	if text == "" && len(attaches) == 0 {
		return nil, fmt.Errorf("send message failed: either text or attachments must be provided")
	}

	cid := s.nextCID()

	msgPayload := map[string]interface{}{
		"cid":      cid,
		"text":     text,
		"elements": []interface{}{},
		"attaches": []interface{}{},
	}
	if replyToMsgID > 0 {
		msgPayload["link"] = map[string]interface{}{
			"type":      "REPLY",
			"messageId": replyToMsgID,
		}
	}
	if len(attaches) > 0 {
		var rawAttaches []interface{}
		for _, a := range attaches {
			rawAttaches = append(rawAttaches, attachmentPayload(a))
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
		CID:        cid,
		ChatID:     chatID,
		Text:       text,
		Time:       time.Now().Unix(),
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

// GetReactions returns reaction summaries keyed by message ID.
func (s *MessageService) GetReactions(ctx context.Context, chatID int64, messageIDs []int64) (map[int64][]types.ReactionInfo, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpMsgGetReactions, map[string]interface{}{
		"chatId": chatID, "messageIds": messageIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get reactions failed: %w", err)
	}

	out := make(map[int64][]types.ReactionInfo)
	if raw, ok := res["messagesReactions"].(map[string]interface{}); ok {
		for key, value := range raw {
			var id int64
			if _, scanErr := fmt.Sscan(key, &id); scanErr != nil {
				continue
			}
			if list, ok := value.([]interface{}); ok {
				for _, item := range list {
					if m, ok := item.(map[string]interface{}); ok {
						out[id] = append(out[id], reactionInfoFromMap(m))
					}
				}
			} else if m, ok := value.(map[string]interface{}); ok {
				out[id] = append(out[id], reactionInfoFromMap(m))
			}
		}
	}
	return out, nil
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
		"chatId":       chatID,
		"forward":      0,
		"backward":     count,
		"backwardTime": 0,
		"forwardTime":  0,
		"from":         fromTime,
		"itemType":     "REGULAR",
		"getChat":      false,
		"getMessages":  true,
		"interactive":  false,
	}
	if fromTime <= 0 {
		payload["from"] = time.Now().UnixMilli()
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
	if newText == "" {
		return fmt.Errorf("edit message failed: text must not be empty")
	}
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
		"text":      newText,
		"elements":  []interface{}{},
		"attachments": []interface{}{},
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
		"chatId":     chatID,
		"messageIds": []int64{messageID},
		"forMe":      !forAll,
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpMsgDelete, payload)
	if err != nil {
		return fmt.Errorf("delete message failed: %w", err)
	}
	return nil
}

// GetMessages retrieves several messages in one request.
func (s *MessageService) GetMessages(ctx context.Context, chatID int64, messageIDs []int64) ([]types.Message, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpMsgGet, map[string]interface{}{
		"chatId": chatID, "messageIds": messageIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get messages failed: %w", err)
	}
	return parseMessages(res, chatID), nil
}

// GetVideoByID resolves a playable video URL from a message attachment.
func (s *MessageService) GetVideoByID(ctx context.Context, chatID, messageID, videoID int64) (types.Attachment, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpVideoPlay, map[string]interface{}{
		"chatId": chatID, "messageId": messageID, "videoId": videoID,
	})
	if err != nil {
		return types.Attachment{}, fmt.Errorf("get video failed: %w", err)
	}
	return attachmentFromMap(res), nil
}

// GetFileByID resolves a downloadable file URL from a message attachment.
func (s *MessageService) GetFileByID(ctx context.Context, chatID, messageID, fileID int64) (types.Attachment, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpFileDownload, map[string]interface{}{
		"chatId": chatID, "messageId": messageID, "fileId": fileID,
	})
	if err != nil {
		return types.Attachment{}, fmt.Errorf("get file failed: %w", err)
	}
	return attachmentFromMap(res), nil
}

// ForwardMessages forwards messages from one chat to another.
func (s *MessageService) ForwardMessages(ctx context.Context, toChatID int64, fromChatID int64, messageIDs []int64) error {
	payload := map[string]interface{}{"chatId": toChatID, "notify": true}
	if len(messageIDs) == 1 {
		payload["message"] = map[string]interface{}{
			"cid":  -s.nextCID(),
			"link": map[string]interface{}{"type": "FORWARD", "messageId": fmt.Sprint(messageIDs[0]), "chatId": fromChatID},
		}
	} else {
		payload["fromChatId"] = fromChatID
		payload["messageIds"] = messageIDs
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
		"chatId":       chatID,
		"pinMessageId": messageID,
		"notifyPin":    true,
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

// GetMessage retrieves a single message by its ID within a chat.
func (s *MessageService) GetMessage(ctx context.Context, chatID, messageID int64) (*types.Message, error) {
	payload := map[string]interface{}{
		"chatId":     chatID,
		"messageIds": []int64{messageID},
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpMsgGet, payload)
	if err != nil {
		return nil, fmt.Errorf("get message failed: %w", err)
	}
	if messages := parseMessages(res, chatID); len(messages) > 0 {
		return &messages[0], nil
	}

	msg := &types.Message{ChatID: chatID}
	src := res
	if mData, ok := res["message"].(map[string]interface{}); ok {
		src = mData
	}
	if id, ok := src["id"].(int64); ok {
		msg.ID = id
	} else if idF, ok := src["id"].(float64); ok {
		msg.ID = int64(idF)
	}
	if text, ok := src["text"].(string); ok {
		msg.Text = text
	}
	if sender, ok := src["sender"].(int64); ok {
		msg.SenderID = sender
	} else if senderF, ok := src["sender"].(float64); ok {
		msg.SenderID = int64(senderF)
	}
	if ts, ok := src["time"].(float64); ok {
		msg.Time = int64(ts)
	}
	return msg, nil
}

// ForwardMessage forwards one message and returns the server-created message.
func (s *MessageService) ForwardMessage(ctx context.Context, chatID int64, messageID int64, sourceChatID int64, notify bool) (*types.Message, error) {
	if sourceChatID == 0 {
		sourceChatID = chatID
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpMsgSend, map[string]interface{}{
		"chatId": chatID, "notify": notify,
		"message": map[string]interface{}{
			"cid":  -s.nextCID(),
			"link": map[string]interface{}{"type": "FORWARD", "messageId": fmt.Sprint(messageID), "chatId": sourceChatID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("forward message failed: %w", err)
	}
	msg := parseMessages(map[string]interface{}{"messages": []interface{}{res["message"]}}, chatID)
	if len(msg) == 0 {
		return &types.Message{ChatID: chatID}, nil
	}
	return &msg[0], nil
}

// ReadMessage marks one message as read using the protocol's READ_MESSAGE action.
func (s *MessageService) ReadMessage(ctx context.Context, messageID int64, chatID int64) error {
	_, err := s.invoker.Invoke(ctx, protocol.OpChatMark, map[string]interface{}{
		"type": "READ_MESSAGE", "chatId": chatID, "messageId": messageID, "mark": time.Now().UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("read message failed: %w", err)
	}
	return nil
}

func parseMessages(res map[string]interface{}, chatID int64) []types.Message {
	var result []types.Message
	list, ok := res["messages"].([]interface{})
	if !ok {
		return result
	}
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		msg := types.Message{ChatID: chatID}
		if v, ok := int64Value(m["id"]); ok {
			msg.ID = v
		}
		if v, ok := int64Value(m["cid"]); ok {
			msg.CID = v
		}
		if v, ok := int64Value(m["chatId"]); ok {
			msg.ChatID = v
		}
		if v, ok := int64Value(m["sender"]); ok {
			msg.SenderID = v
		}
		if v, ok := m["senderId"].(int64); ok {
			msg.SenderID = v
		}
		if v, ok := m["text"].(string); ok {
			msg.Text = v
		}
		if v, ok := int64Value(m["time"]); ok {
			msg.Time = v
		}
		result = append(result, msg)
	}
	return result
}

func int64Value(value interface{}) (int64, bool) {
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
		var parsed int64
		if _, err := fmt.Sscan(v, &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func reactionInfoFromMap(m map[string]interface{}) types.ReactionInfo {
	var info types.ReactionInfo
	if v, ok := m["reaction"].(string); ok {
		info.Reaction = v
	}
	if v, ok := int64Value(m["count"]); ok {
		info.Count = int(v)
	}
	if v, ok := m["self"].(bool); ok {
		info.Self = v
	}
	return info
}

func attachmentFromMap(m map[string]interface{}) types.Attachment {
	if nested, ok := m["video"].(map[string]interface{}); ok {
		m = nested
	}
	if nested, ok := m["file"].(map[string]interface{}); ok {
		m = nested
	}
	var a types.Attachment
	if v, ok := m["url"].(string); ok {
		a.URL = v
	}
	if v, ok := m["token"].(string); ok {
		a.Token = v
	}
	if v, ok := m["id"].(string); ok {
		a.ID = v
	}
	if v, ok := m["name"].(string); ok {
		a.FileName = v
	}
	if v, ok := m["fileName"].(string); ok {
		a.FileName = v
	}
	if v, ok := int64Value(m["size"]); ok {
		a.FileSize = v
	}
	if v, ok := int64Value(m["duration"]); ok {
		a.Duration = int(v)
	}
	return a
}

func attachmentPayload(a types.Attachment) map[string]interface{} {
	wireType := a.Type
	if a.Type == types.AttachmentVoice {
		wireType = types.AttachmentAudio
	}
	if a.Type == types.AttachmentVideoNote {
		wireType = types.AttachmentVideo
	}
	payload := map[string]interface{}{"_type": wireType}
	if a.Token != "" {
		if a.Type == types.AttachmentPhoto {
			payload["photoToken"] = a.Token
		} else {
			payload["token"] = a.Token
		}
	}
	if a.ID != "" {
		if id, err := strconv.ParseInt(a.ID, 10, 64); err == nil {
			switch a.Type {
			case types.AttachmentVideo, types.AttachmentVideoNote:
				payload["videoId"] = id
			case types.AttachmentAudio, types.AttachmentVoice:
				payload["audioId"] = id
			case types.AttachmentFile:
				payload["fileId"] = id
			default:
				payload["id"] = a.ID
			}
		} else {
			payload["id"] = a.ID
		}
	}
	if a.Type == types.AttachmentVideoNote {
		payload["videoType"] = 1
	}
	if a.Duration > 0 {
		payload["duration"] = a.Duration
	}
	return payload
}
