package messages

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/formatting"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// MessageService handles messages, reactions, mass-looking (reading), and history.
type MessageService struct {
	invoker             api.Invoker
	prevCID             int64
	waitAttachmentReady func(context.Context, []types.Attachment) error
}

// SetAttachmentReadyWaiter connects the upload notification waiters used when
// Max temporarily rejects a voice or video-note as attachment.not.ready.
func (s *MessageService) SetAttachmentReadyWaiter(waiter func(context.Context, []types.Attachment) error) {
	s.waitAttachmentReady = waiter
}

func (s *MessageService) invokeWithAttachmentRetry(ctx context.Context, op protocol.Opcode, payload map[string]interface{}, attachments []types.Attachment) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, op, payload)
	if err == nil || s.waitAttachmentReady == nil {
		return res, err
	}
	var apiErr *protocol.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorStr != "attachment.not.ready" {
		return nil, err
	}
	if err := s.waitAttachmentReady(ctx, attachments); err != nil {
		return nil, err
	}
	return s.invoker.Invoke(ctx, op, payload)
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
	return s.SendMessageWithOptions(ctx, chatID, text, attaches, SendOptions{Notify: true, ReplyToMessageID: replyToMsgID})
}

// SendMessageAction implements types.MessageActions for bound domain models.
func (s *MessageService) SendMessageAction(ctx context.Context, chatID int64, text string, attaches []types.Attachment, opts types.MessageActionOptions) (*types.Message, error) {
	return s.SendMessageWithOptions(ctx, chatID, text, attaches, SendOptions{
		Notify: opts.Notify, ReplyToMessageID: opts.ReplyToMessageID,
		SendAt: opts.SendAt, NotifySender: opts.NotifySender,
	})
}

type SendOptions struct {
	Notify           bool
	ReplyToMessageID int64
	Elements         []map[string]any
	SendAt           time.Time
	NotifySender     bool
}

func (s *MessageService) SendMessageWithOptions(ctx context.Context, chatID int64, text string, attaches []types.Attachment, opts SendOptions) (*types.Message, error) {
	if text == "" && len(attaches) == 0 {
		return nil, fmt.Errorf("send message failed: either text or attachments must be provided")
	}

	cid := s.nextCID()

	if text != "" && opts.Elements == nil {
		text, opts.Elements = formatMarkdown(text)
	}
	msgPayload := map[string]interface{}{
		"cid":      cid,
		"text":     text,
		"elements": opts.Elements,
		"attaches": []interface{}{},
	}
	if opts.ReplyToMessageID > 0 {
		msgPayload["link"] = map[string]interface{}{
			"type":      "REPLY",
			"messageId": opts.ReplyToMessageID,
		}
	}
	if !opts.SendAt.IsZero() {
		msgPayload["delayedAttributes"] = map[string]interface{}{
			"timeToFire": opts.SendAt.UnixMilli(), "notifySender": opts.NotifySender,
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
		"notify":  opts.Notify,
	}

	res, err := s.invokeWithAttachmentRetry(ctx, protocol.OpMsgSend, payload, attaches)
	if err != nil {
		return nil, fmt.Errorf("send message failed: %w", err)
	}

	msg := &types.Message{
		CID:        cid,
		ChatID:     chatID,
		Text:       text,
		Time:       time.Now().UnixMilli(),
		IsOutgoing: true,
	}

	messageData := res
	if nested, ok := res["message"].(map[string]interface{}); ok {
		messageData = nested
	}
	if len(messageData) > 0 {
		parsed := types.ParseMessagePayload(messageData)
		if parsed.ID != 0 || parsed.CID != 0 || parsed.Text != "" {
			msg = parsed
			if msg.ChatID == 0 {
				msg.ChatID = chatID
			}
			if msg.CID == 0 {
				msg.CID = cid
			}
		}
	}

	return msg.Bind(s), nil
}

// AddReaction adds an emoji reaction to a message.
func (s *MessageService) AddReaction(ctx context.Context, chatID int64, messageID int64, reaction string) error {
	_, err := s.AddReactionInfo(ctx, chatID, messageID, reaction)
	return err
}

// AddReactionInfo adds a reaction and returns the updated server summary.
func (s *MessageService) AddReactionInfo(ctx context.Context, chatID int64, messageID int64, reaction string) (*types.ReactionInfo, error) {
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
		"reaction": map[string]interface{}{
			"reactionType": "EMOJI",
			"id":           reaction,
		},
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpMsgReaction, payload)
	if err != nil {
		return nil, fmt.Errorf("add reaction failed: %w", err)
	}
	if raw, ok := res["reactionInfo"].(map[string]interface{}); ok {
		info := reactionInfoFromMap(raw)
		return &info, nil
	}
	return nil, nil
}

// RemoveReaction removes a previously placed reaction.
func (s *MessageService) RemoveReaction(ctx context.Context, chatID int64, messageID int64, reaction string) error {
	_, err := s.RemoveReactionInfo(ctx, chatID, messageID)
	return err
}

// RemoveReactionInfo removes the current user's reaction and returns the updated summary.
func (s *MessageService) RemoveReactionInfo(ctx context.Context, chatID int64, messageID int64) (*types.ReactionInfo, error) {
	payload := map[string]interface{}{
		"chatId":    chatID,
		"messageId": messageID,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpMsgCancelReaction, payload)
	if err != nil {
		return nil, fmt.Errorf("remove reaction failed: %w", err)
	}
	if raw, ok := res["reactionInfo"].(map[string]interface{}); ok {
		info := reactionInfoFromMap(raw)
		return &info, nil
	}
	return nil, nil
}

// GetReactions returns reaction summaries keyed by message ID.
func (s *MessageService) GetReactions(ctx context.Context, chatID int64, messageIDs []int64) (map[int64][]types.ReactionInfo, error) {
	details, err := s.GetReactionDetails(ctx, chatID, messageIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]types.ReactionInfo, len(details))
	for id, info := range details {
		out[id] = []types.ReactionInfo{info}
	}
	return out, nil
}

// GetReactionDetails returns the protocol-native reaction summary per message.
func (s *MessageService) GetReactionDetails(ctx context.Context, chatID int64, messageIDs []int64) (map[int64]types.ReactionInfo, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpMsgGetReactions, map[string]interface{}{
		"chatId": chatID, "messageIds": messageIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get reactions failed: %w", err)
	}

	out := make(map[int64]types.ReactionInfo)
	if raw, ok := res["messagesReactions"].(map[string]interface{}); ok {
		for key, value := range raw {
			var id int64
			if _, scanErr := fmt.Sscan(key, &id); scanErr != nil {
				continue
			}
			if m, ok := value.(map[string]interface{}); ok {
				out[id] = reactionInfoFromMap(m)
			}
		}
	}
	return out, nil
}

// ReadMessages marks specific messages as read (mass-look / views).
func (s *MessageService) ReadMessages(ctx context.Context, chatID int64, messageIDs []int64) error {
	for _, messageID := range messageIDs {
		if _, err := s.ReadMessageState(ctx, messageID, chatID); err != nil {
			return err
		}
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

	return s.parseMessages(res, chatID), nil
}

// FetchHistoryAction implements PyMax's full history surface for bound chats.
func (s *MessageService) FetchHistoryAction(ctx context.Context, chatID int64, opts types.HistoryActionOptions) ([]types.Message, error) {
	if opts.Backward == 0 && opts.Forward == 0 {
		opts.Backward = 40
	}
	if opts.FromTime == 0 {
		opts.FromTime = time.Now().UnixMilli()
	}
	if opts.ItemType == "" {
		opts.ItemType = "REGULAR"
	}
	if !opts.GetMessagesSet {
		opts.GetMessages = true
	}
	payload := map[string]interface{}{
		"chatId": chatID, "forward": opts.Forward, "backward": opts.Backward,
		"backwardTime": opts.BackwardTime, "forwardTime": opts.ForwardTime,
		"from": opts.FromTime, "itemType": opts.ItemType, "getChat": opts.GetChat,
		"getMessages": opts.GetMessages, "interactive": opts.Interactive,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpChatHistory, payload)
	if err != nil {
		return nil, fmt.Errorf("fetch history failed: %w", err)
	}
	return s.parseMessages(res, chatID), nil
}

// FetchHistory is the direct service counterpart to PyMax fetch_history.
func (s *MessageService) FetchHistory(ctx context.Context, chatID int64, opts types.HistoryActionOptions) ([]types.Message, error) {
	return s.FetchHistoryAction(ctx, chatID, opts)
}

// GetHistory retrieves message history for a chat (alias for GetChatHistory matching API specification).
func (s *MessageService) GetHistory(ctx context.Context, chatID int64, fromTime int64, count int) ([]types.Message, error) {
	return s.GetChatHistory(ctx, chatID, fromTime, count)
}

// EditMessage edits an existing message text.
func (s *MessageService) EditMessage(ctx context.Context, chatID int64, messageID int64, newText string) error {
	return s.EditMessageWithAttachments(ctx, chatID, messageID, newText, nil, nil)
}

func (s *MessageService) EditMessageWithAttachments(ctx context.Context, chatID, messageID int64, text string, elements []map[string]any, attachments []types.Attachment) error {
	_, err := s.EditMessageResultWithAttachments(ctx, chatID, messageID, text, elements, attachments)
	return err
}

// EditMessageResultWithAttachments edits a message and returns the updated message.
func (s *MessageService) EditMessageResultWithAttachments(ctx context.Context, chatID, messageID int64, text string, elements []map[string]any, attachments []types.Attachment) (*types.Message, error) {
	if text == "" && len(attachments) == 0 {
		return nil, fmt.Errorf("edit message failed: text or attachments must be provided")
	}
	if text != "" && elements == nil {
		text, elements = formatMarkdown(text)
	}
	rawAttachments := make([]interface{}, 0, len(attachments))
	for _, attachment := range attachments {
		rawAttachments = append(rawAttachments, attachmentPayload(attachment))
	}
	payload := map[string]interface{}{
		"chatId":      chatID,
		"messageId":   messageID,
		"text":        text,
		"elements":    elements,
		"attachments": rawAttachments,
	}

	res, err := s.invokeWithAttachmentRetry(ctx, protocol.OpMsgEdit, payload, attachments)
	if err != nil {
		return nil, fmt.Errorf("edit message failed: %w", err)
	}
	messageData := res
	if nested, ok := res["message"].(map[string]interface{}); ok {
		messageData = nested
	}
	msg := types.ParseMessagePayload(messageData)
	if msg.ChatID == 0 {
		msg.ChatID = chatID
	}
	return msg.Bind(s), nil
}

// DeleteMessage deletes a message.
func (s *MessageService) DeleteMessage(ctx context.Context, chatID int64, messageID int64, forAll bool) error {
	return s.DeleteMessages(ctx, chatID, []int64{messageID}, forAll)
}

func (s *MessageService) DeleteMessages(ctx context.Context, chatID int64, messageIDs []int64, forAll bool) error {
	payload := map[string]interface{}{
		"chatId":     chatID,
		"messageIds": messageIDs,
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
	return s.parseMessages(res, chatID), nil
}

// GetVideoByID resolves a playable video URL from a message attachment.
func (s *MessageService) GetVideoByID(ctx context.Context, chatID int64, messageID any, videoID int64) (types.Attachment, error) {
	request, err := s.GetVideoRequestByID(ctx, chatID, messageID, videoID)
	if err != nil {
		return types.Attachment{}, err
	}
	return types.Attachment{Type: types.AttachmentVideo, ID: strconv.FormatInt(videoID, 10), URL: request.URL, Raw: map[string]any{"EXTERNAL": request.External, "cache": request.Cache}}, nil
}

// GetVideoRequestByID returns PyMax's typed playback response and selects the
// highest MP4_<quality> URL when the server omits a direct url field.
func (s *MessageService) GetVideoRequestByID(ctx context.Context, chatID int64, messageID any, videoID int64) (*types.VideoRequest, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpVideoPlay, map[string]interface{}{
		"chatId": chatID, "messageId": messageID, "videoId": videoID,
	})
	if err != nil {
		return nil, fmt.Errorf("get video failed: %w", err)
	}
	request := &types.VideoRequest{External: res["EXTERNAL"]}
	request.Cache, _ = res["cache"].(bool)
	request.URL = types.StringValue(res["url"])
	bestQuality := -1
	if request.URL == "" {
		for key, raw := range res {
			if !strings.HasPrefix(strings.ToUpper(key), "MP4_") {
				continue
			}
			quality, parseErr := strconv.Atoi(strings.TrimPrefix(strings.ToUpper(key), "MP4_"))
			url := types.StringValue(raw)
			if parseErr == nil && quality > bestQuality && url != "" {
				bestQuality, request.URL = quality, url
			}
		}
	}
	if request.URL == "" {
		request.URL = types.StringValue(res["dynamicUrl"])
	}
	return request, nil
}

// GetFileByID resolves a downloadable file URL from a message attachment.
func (s *MessageService) GetFileByID(ctx context.Context, chatID int64, messageID any, fileID int64) (types.Attachment, error) {
	request, err := s.GetFileRequestByID(ctx, chatID, messageID, fileID)
	if err != nil {
		return types.Attachment{}, err
	}
	return types.Attachment{Type: types.AttachmentFile, ID: strconv.FormatInt(fileID, 10), URL: request.URL, Unsafe: request.Unsafe}, nil
}

// GetFileRequestByID returns PyMax's typed download response.
func (s *MessageService) GetFileRequestByID(ctx context.Context, chatID int64, messageID any, fileID int64) (*types.FileRequest, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpFileDownload, map[string]interface{}{
		"chatId": chatID, "messageId": messageID, "fileId": fileID,
	})
	if err != nil {
		return nil, fmt.Errorf("get file failed: %w", err)
	}
	request := &types.FileRequest{URL: types.StringValue(res["url"])}
	request.Unsafe, _ = res["unsafe"].(bool)
	return request, nil
}

// ForwardMessages forwards messages from one chat to another.
func (s *MessageService) ForwardMessages(ctx context.Context, toChatID int64, fromChatID int64, messageIDs []int64) error {
	for _, messageID := range messageIDs {
		if _, err := s.ForwardMessage(ctx, toChatID, messageID, fromChatID, true); err != nil {
			return err
		}
	}
	return nil
}

// PinMessage pins a message to the top of the chat.
func (s *MessageService) PinMessage(ctx context.Context, chatID int64, messageID int64) error {
	return s.PinMessageWithNotify(ctx, chatID, messageID, true)
}

func (s *MessageService) PinMessageWithNotify(ctx context.Context, chatID, messageID int64, notify bool) error {
	payload := map[string]interface{}{
		"chatId":       chatID,
		"pinMessageId": messageID,
		"notifyPin":    notify,
	}

	_, err := s.invoker.Invoke(ctx, protocol.OpChatUpdate, payload)
	if err != nil {
		return fmt.Errorf("pin message failed: %w", err)
	}
	return nil
}

// VotePoll votes for option(s) in a poll.
func (s *MessageService) VotePoll(ctx context.Context, chatID int64, messageID int64, pollID int64, optionIDs []int) error {
	_, err := s.VotePollState(ctx, chatID, messageID, pollID, optionIDs)
	return err
}

// VotePollState submits a vote and returns the current poll state.
func (s *MessageService) VotePollState(ctx context.Context, chatID int64, messageID int64, pollID int64, optionIDs []int) (*types.PollState, error) {
	payload := map[string]interface{}{
		"chatId":     chatID,
		"messageId":  messageID,
		"pollId":     pollID,
		"answersIds": optionIDs,
	}

	res, err := s.invoker.Invoke(ctx, protocol.OpSendVote, payload)
	if err != nil {
		return nil, fmt.Errorf("vote poll failed: %w", err)
	}
	state, _ := res["state"].(map[string]interface{})
	return types.ParsePollState(state), nil
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
	if messages := s.parseMessages(res, chatID); len(messages) > 0 {
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
	return msg.Bind(s), nil
}

// ForwardMessage forwards one message and returns the server-created message.
func (s *MessageService) ForwardMessage(ctx context.Context, chatID int64, messageID any, sourceChatID int64, notify bool) (*types.Message, error) {
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
	msg := s.parseMessages(map[string]interface{}{"messages": []interface{}{res["message"]}}, chatID)
	if len(msg) == 0 {
		return (&types.Message{ChatID: chatID}).Bind(s), nil
	}
	return &msg[0], nil
}

// ReadMessage marks one message as read using the protocol's READ_MESSAGE action.
func (s *MessageService) ReadMessage(ctx context.Context, messageID any, chatID int64) error {
	_, err := s.ReadMessageState(ctx, messageID, chatID)
	return err
}

// ReadMessageState marks one message as read and returns unread/mark state.
func (s *MessageService) ReadMessageState(ctx context.Context, messageID any, chatID int64) (*types.ReadState, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpChatMark, map[string]interface{}{
		"type": "READ_MESSAGE", "chatId": chatID, "messageId": messageID, "mark": time.Now().UnixMilli(),
	})
	if err != nil {
		return nil, fmt.Errorf("read message failed: %w", err)
	}
	unread, _ := int64Value(res["unread"])
	mark, _ := int64Value(res["mark"])
	return &types.ReadState{Unread: int(unread), Mark: mark}, nil
}

func formatMarkdown(text string) (string, []map[string]any) {
	cleanText, elements := formatting.FormatMarkdown(text)
	wireElements := make([]map[string]any, 0, len(elements))
	for _, element := range elements {
		wire := map[string]any{"type": element.Type, "from": element.From, "length": element.Length}
		if element.Attributes != nil {
			wire["attributes"] = map[string]any{"url": element.Attributes.URL}
		}
		wireElements = append(wireElements, wire)
	}
	return cleanText, wireElements
}

func (s *MessageService) parseMessages(res map[string]interface{}, chatID int64) []types.Message {
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
		msg := types.ParseMessagePayload(m)
		if msg.ChatID == 0 {
			msg.ChatID = chatID
		}
		msg.Bind(s)
		result = append(result, *msg)
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
	if v, ok := int64Value(m["totalCount"]); ok {
		info.TotalCount = int(v)
	}
	if v, ok := m["yourReaction"].(string); ok {
		info.YourReaction = v
		info.Self = true
	}
	if list, ok := m["counters"].([]interface{}); ok {
		for _, item := range list {
			if counter, ok := item.(map[string]interface{}); ok {
				count, _ := int64Value(counter["count"])
				info.Counters = append(info.Counters, types.ReactionCounter{Reaction: types.StringValue(counter["reaction"]), Count: int(count)})
			}
		}
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
		switch a.Type {
		case types.AttachmentPhoto:
			payload["photoToken"] = a.Token
		case types.AttachmentFile:
			// File attachments are identified solely by fileId.
		default:
			payload["token"] = a.Token
		}
	}
	if a.ID != "" {
		if id, err := strconv.ParseInt(a.ID, 10, 64); err == nil {
			switch a.Type {
			case types.AttachmentVideo:
				payload["videoId"] = id
			case types.AttachmentVideoNote:
				if a.Token == "" {
					payload["videoId"] = id
				}
			case types.AttachmentAudio, types.AttachmentVoice:
				if a.Token == "" {
					payload["audioId"] = id
				}
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
	} else if a.Type == types.AttachmentVideo {
		payload["videoType"] = 0
	}
	if a.Type == types.AttachmentVoice {
		if len(a.Wave) > 0 {
			payload["wave"] = a.Wave
		} else {
			payload["wave"] = make([]byte, 80)
		}
	}
	if len(a.ThumbHash) > 0 {
		payload["thumbhash"] = a.ThumbHash
	}
	if a.Duration > 0 {
		payload["duration"] = a.Duration
	}
	return payload
}
