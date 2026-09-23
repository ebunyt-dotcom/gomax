package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Int64Value accepts every numeric representation produced by JSON and
// MessagePack decoders, plus decimal string IDs used by WebSocket payloads.
func Int64Value(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), uint64(v) <= uint64(^uint64(0)>>1)
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func StringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		if _, ok := Int64Value(value); ok {
			return fmt.Sprint(value)
		}
		return ""
	}
}

func ParseUserPayload(src map[string]any) User {
	if nested, ok := src["profile"].(map[string]any); ok {
		src = nested
	}
	if nested, ok := src["contact"].(map[string]any); ok {
		src = nested
	}
	user := User{Raw: src}
	user.ID, _ = Int64Value(src["id"])
	user.Phone = StringValue(src["phone"])
	user.FirstName = StringValue(src["firstName"])
	user.LastName = StringValue(src["lastName"])
	user.Nickname = StringValue(src["nickname"])
	user.AvatarURL = StringValue(src["avatarUrl"])
	user.Bio = StringValue(src["description"])
	user.IsBot, _ = src["isBot"].(bool)
	user.IsContact, _ = src["isContact"].(bool)
	user.IsMutual, _ = src["isMutual"].(bool)
	user.IsVerified, _ = src["isVerified"].(bool)
	user.AccountStatus, _ = Int64Value(src["accountStatus"])
	user.RegistrationTime, _ = Int64Value(src["registrationTime"])
	user.Country = StringValue(src["country"])
	user.BaseRawURL = StringValue(src["baseRawUrl"])
	user.BaseURL = StringValue(src["baseUrl"])
	if user.AvatarURL == "" {
		user.AvatarURL = user.BaseURL
	}
	user.PhotoID, _ = Int64Value(src["photoId"])
	user.UpdateTime, _ = Int64Value(src["updateTime"])
	user.Status = StringValue(src["status"])
	user.Gender = src["gender"]
	user.Link = src["link"]
	user.WebApp = src["webApp"]
	user.MenuButton, _ = src["menuButton"].(map[string]any)
	if options, ok := src["options"].([]interface{}); ok {
		for _, option := range options {
			user.Options = append(user.Options, StringValue(option))
		}
	}
	if names, ok := src["names"].([]interface{}); ok {
		for _, value := range names {
			if name, ok := value.(map[string]any); ok {
				user.Names = append(user.Names, Name{Name: StringValue(name["name"]), FirstName: StringValue(name["firstName"]), LastName: StringValue(name["lastName"]), Type: StringValue(name["type"])})
			}
		}
	}
	if user.FirstName == "" && len(user.Names) > 0 {
		user.FirstName, user.LastName = user.Names[0].FirstName, user.Names[0].LastName
	}
	return user
}

func ParseChatPayload(src map[string]any) Chat {
	if nested, ok := src["chat"].(map[string]any); ok {
		src = nested
	}
	chat := Chat{Raw: src}
	chat.ID, _ = Int64Value(src["id"])
	chat.Type = ChatType(StringValue(src["type"]))
	chat.Title = StringValue(src["title"])
	chat.Description = StringValue(src["description"])
	chat.Icon = StringValue(src["icon"])
	if v, ok := Int64Value(src["membersCount"]); ok {
		chat.MembersCount = int(v)
	}
	chat.OwnerID, _ = Int64Value(src["owner"])
	if chat.OwnerID == 0 {
		chat.OwnerID, _ = Int64Value(src["ownerId"])
	}
	if v, ok := Int64Value(src["participantsCount"]); ok {
		chat.MembersCount = int(v)
	}
	chat.PinnedMsgID, _ = Int64Value(src["pinnedMessageId"])
	chat.IsChannel, _ = src["isChannel"].(bool)
	chat.IsPublic, _ = src["isPublic"].(bool)
	chat.InviteLink = StringValue(src["inviteLink"])
	if chat.InviteLink == "" {
		chat.InviteLink = StringValue(src["link"])
	}
	chat.Status = StringValue(src["status"])
	chat.BaseRawIconURL = StringValue(src["baseRawIconUrl"])
	chat.BaseIconURL = StringValue(src["baseIconUrl"])
	if chat.Icon == "" {
		chat.Icon = chat.BaseIconURL
	}
	chat.LastEventTime, _ = Int64Value(src["lastEventTime"])
	chat.LastDelayedUpdateTime, _ = Int64Value(src["lastDelayedUpdateTime"])
	chat.LastFireDelayedErrorTime, _ = Int64Value(src["lastFireDelayedErrorTime"])
	chat.Created, _ = Int64Value(src["created"])
	if v, ok := Int64Value(src["newMessages"]); ok {
		chat.NewMessages = int(v)
	}
	chat.Access = AccessType(StringValue(src["access"]))
	chat.Restrictions, _ = Int64Value(src["restrictions"])
	chat.Options = src["options"]
	chat.JoinTime, _ = Int64Value(src["joinTime"])
	chat.InvitedBy, _ = Int64Value(src["invitedBy"])
	chat.Modified, _ = Int64Value(src["modified"])
	if v, ok := Int64Value(src["messagesCount"]); ok {
		chat.MessagesCount = int(v)
	}
	if v, ok := src["hasBots"].(bool); ok {
		chat.HasBots = &v
	}
	chat.PrevMessageID, _ = Int64Value(src["prevMessageId"])
	chat.CID, _ = Int64Value(src["cid"])
	if message, ok := src["lastMessage"].(map[string]any); ok {
		chat.LastMessage = ParseMessagePayload(message)
	}
	if message, ok := src["pinnedMessage"].(map[string]any); ok {
		chat.PinnedMessage = ParseMessagePayload(message)
	}
	if admins, ok := src["admins"].([]interface{}); ok {
		for _, value := range admins {
			if id, ok := Int64Value(value); ok {
				chat.Admins = append(chat.Admins, id)
			}
		}
	}
	chat.Participants = int64Map(src["participants"])
	chat.AdminParticipants = nestedInt64Map(src["adminParticipants"])
	return chat
}

func int64Map(value any) map[int64]int64 {
	out := make(map[int64]int64)
	switch values := value.(type) {
	case map[string]any:
		for key, raw := range values {
			id, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				continue
			}
			if v, ok := Int64Value(raw); ok {
				out[id] = v
			}
		}
	case map[int64]interface{}:
		for key, raw := range values {
			if v, ok := Int64Value(raw); ok {
				out[key] = v
			}
		}
	}
	return out
}

func nestedInt64Map(value any) map[int64]map[string]any {
	out := make(map[int64]map[string]any)
	if values, ok := value.(map[string]any); ok {
		for key, raw := range values {
			id, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				continue
			}
			if nested, ok := raw.(map[string]any); ok {
				out[id] = nested
			}
		}
	}
	return out
}

func ParseAttachmentPayload(src map[string]any) Attachment {
	original := src
	attachmentType := StringValue(src["_type"])
	if attachmentType == "" {
		attachmentType = StringValue(src["type"])
	}
	for _, key := range []string{"photo", "video", "audio", "file", "sticker", "contact", "call", "control", "keyboard", "share", "poll"} {
		if nested, ok := src[key].(map[string]any); ok {
			src = nested
			if attachmentType == "" {
				attachmentType = key
			}
			break
		}
	}
	a := Attachment{Type: AttachmentType(strings.ToUpper(attachmentType)), Raw: original}
	if a.Type == "" {
		a.Type = AttachmentUnknown
	}
	a.URL = StringValue(src["url"])
	a.BaseURL = StringValue(src["baseUrl"])
	a.Thumbnail = StringValue(src["thumbnail"])
	a.Token = StringValue(src["token"])
	if a.Token == "" {
		a.Token = StringValue(src["photoToken"])
	}
	for _, key := range []string{"id", "photoId", "videoId", "audioId", "fileId"} {
		if v := StringValue(src[key]); v != "" {
			a.ID = v
			break
		}
		if v, ok := Int64Value(src[key]); ok {
			a.ID = strconv.FormatInt(v, 10)
			break
		}
	}
	a.FileName = StringValue(src["fileName"])
	if a.FileName == "" {
		a.FileName = StringValue(src["name"])
	}
	a.FileSize, _ = Int64Value(src["size"])
	if v, ok := Int64Value(src["duration"]); ok {
		a.Duration = int(v)
	}
	if v, ok := Int64Value(src["width"]); ok {
		a.Width = int(v)
	}
	if v, ok := Int64Value(src["height"]); ok {
		a.Height = int(v)
	}
	if v, ok := Int64Value(src["videoType"]); ok {
		a.VideoType = int(v)
	}
	if v, ok := src["wave"].([]byte); ok {
		a.Wave = append([]byte(nil), v...)
	} else if v := StringValue(src["wave"]); v != "" {
		a.Wave = []byte(v)
	}
	if v, ok := src["previewData"].([]byte); ok {
		a.PreviewData = append([]byte(nil), v...)
	}
	a.TranscriptionStatus = TranscriptionStatus(StringValue(src["transcriptionStatus"]))
	a.Unsafe, _ = src["unsafe"].(bool)
	a.AuthorType = StringValue(src["authorType"])
	a.LottieURL = StringValue(src["lottieUrl"])
	a.StickerID, _ = Int64Value(src["stickerId"])
	if tags, ok := src["tags"].([]any); ok {
		for _, tag := range tags {
			a.Tags = append(a.Tags, StringValue(tag))
		}
	}
	a.SetID, _ = Int64Value(src["setId"])
	a.StickerType = StringValue(src["stickerType"])
	a.Audio, _ = src["audio"].(bool)
	a.ContactID, _ = Int64Value(src["contactId"])
	a.FirstName = StringValue(src["firstName"])
	a.LastName = StringValue(src["lastName"])
	a.Name = StringValue(src["name"])
	a.PhotoURL = StringValue(src["photoUrl"])
	a.ConversationID = src["conversationId"]
	if ids, ok := src["contactIds"].([]any); ok {
		for _, raw := range ids {
			if id, ok := Int64Value(raw); ok {
				a.ContactIDs = append(a.ContactIDs, id)
			}
		}
	}
	a.CallType = CallType(StringValue(src["callType"]))
	a.HangupType = HangupType(StringValue(src["hangupType"]))
	a.Event = StringValue(src["event"])
	a.Title = StringValue(src["title"])
	a.Description = StringValue(src["description"])
	a.Image, _ = src["image"].(map[string]any)
	a.Keyboard, _ = src["keyboard"].(map[string]any)
	if v, ok := src["thumbHash"].([]byte); ok {
		a.ThumbHash = append([]byte(nil), v...)
	} else if v, ok := src["thumbhash"].([]byte); ok {
		a.ThumbHash = append([]byte(nil), v...)
	}
	if a.Type == AttachmentPoll || a.Type == AttachmentType("poll") {
		a.Type = AttachmentPoll
		a.Poll = parsePoll(src)
	}
	return a
}

func parsePoll(src map[string]any) *Poll {
	poll := &Poll{Title: StringValue(src["title"]), Question: StringValue(src["question"])}
	poll.PollID, _ = Int64Value(src["pollId"])
	if poll.PollID == 0 {
		poll.PollID, _ = Int64Value(src["id"])
	}
	poll.ID = StringValue(src["id"])
	if poll.ID == "" && poll.PollID != 0 {
		poll.ID = strconv.FormatInt(poll.PollID, 10)
	}
	poll.Settings, _ = Int64Value(src["settings"])
	poll.Version, _ = Int64Value(src["version"])
	poll.Multiple, _ = src["multiple"].(bool)
	poll.Anonymous, _ = src["anonymous"].(bool)
	poll.Closed, _ = src["closed"].(bool)
	if rawAnswers, ok := src["answers"].([]any); ok {
		for _, raw := range rawAnswers {
			answerMap, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			answer := PollAnswer{Text: StringValue(answerMap["text"])}
			if id, ok := Int64Value(answerMap["answerId"]); ok {
				value := int(id)
				answer.AnswerID = &value
				poll.Options = append(poll.Options, PollOption{ID: value, Text: answer.Text})
			}
			poll.Answers = append(poll.Answers, answer)
		}
	}
	if state, ok := src["state"].(map[string]any); ok {
		poll.State = ParsePollState(state)
	}
	return poll
}

// ParsePollState decodes the state returned by poll attachments and votes.
func ParsePollState(src map[string]any) *PollState {
	state := &PollState{}
	if total, ok := Int64Value(src["total"]); ok {
		state.Total = int(total)
	}
	if ids, ok := src["voterPreviewIds"].([]any); ok {
		for _, raw := range ids {
			if id, ok := Int64Value(raw); ok {
				state.VoterPreviewIDs = append(state.VoterPreviewIDs, id)
			}
		}
	}
	if results, ok := src["result"].([]any); ok {
		for _, raw := range results {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			result := PollResult{}
			if value, ok := Int64Value(m["answerId"]); ok {
				result.AnswerID = int(value)
			}
			if value, ok := Int64Value(m["voteCount"]); ok {
				result.VoteCount = int(value)
			}
			if value, ok := Int64Value(m["rate"]); ok {
				result.Rate = int(value)
			}
			if value, ok := Int64Value(m["options"]); ok {
				result.Options = int(value)
			}
			if votes, ok := m["votes"].([]any); ok {
				for _, rawVote := range votes {
					voteMap, ok := rawVote.(map[string]any)
					if !ok {
						continue
					}
					vote := PollVote{}
					vote.Timestamp, _ = Int64Value(voteMap["timestamp"])
					vote.UserID, _ = Int64Value(voteMap["userId"])
					result.Votes = append(result.Votes, vote)
				}
			}
			state.Result = append(state.Result, result)
		}
	}
	return state
}

func ParseMessagePayload(src map[string]any) *Message {
	if nested, ok := src["message"].(map[string]any); ok {
		src = nested
	}
	msg := &Message{Raw: src}
	msg.ID, _ = Int64Value(src["id"])
	msg.CID, _ = Int64Value(src["cid"])
	msg.ChatID, _ = Int64Value(src["chatId"])
	msg.SenderID, _ = Int64Value(src["sender"])
	if msg.SenderID == 0 {
		msg.SenderID, _ = Int64Value(src["senderId"])
	}
	msg.Text = StringValue(src["text"])
	msg.Time, _ = Int64Value(src["time"])
	msg.EditedAt, _ = Int64Value(src["editedAt"])
	msg.PrevMessageID = src["prevMessageId"]
	if value, ok := src["ttl"].(bool); ok {
		msg.TTL = &value
	}
	if value, ok := Int64Value(src["mark"]); ok {
		msg.Mark = &value
	}
	msg.Type = StringValue(src["type"])
	msg.Status = MessageStatus(StringValue(src["status"]))
	msg.IsDeleted = msg.Status == MessageStatusRemoved
	if value, ok := Int64Value(src["unread"]); ok {
		unread := int(value)
		msg.Unread = &unread
	}
	msg.Stats, _ = src["stats"].(map[string]any)
	msg.Options = src["options"]
	if reactionInfo, ok := src["reactionInfo"].(map[string]any); ok {
		msg.ReactionInfo = parseReactionInfo(reactionInfo)
		msg.Reactions = append(msg.Reactions, *msg.ReactionInfo)
	}
	if link, ok := src["link"].(map[string]any); ok {
		msg.Link = &MessageLink{Type: LinkType(StringValue(link["type"])), ChatName: StringValue(link["chatName"]), ChatLink: StringValue(link["chatLink"]), ChatAccessType: AccessType(StringValue(link["chatAccessType"])), ChatIconURL: StringValue(link["chatIconUrl"])}
		msg.Link.ChatID, _ = Int64Value(link["chatId"])
		if linkedMessage, ok := link["message"].(map[string]any); ok {
			msg.Link.Message = ParseMessagePayload(linkedMessage)
		}
		if id, ok := Int64Value(link["messageId"]); ok {
			msg.ReplyToMsgID = id
		}
	}
	if delayed, ok := src["delayedAttributes"].(map[string]any); ok {
		msg.DelayedAttributes = &DelayedAttributes{}
		msg.DelayedAttributes.TimeToFire, _ = Int64Value(delayed["timeToFire"])
		msg.DelayedAttributes.NotifySender, _ = delayed["notifySender"].(bool)
		msg.DelayedAttributes.NotifyOpponents, _ = delayed["notifyOpponents"].(bool)
	}
	if raw, ok := src["elements"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				element := Element{Type: StringValue(m["type"])}
				if value, ok := Int64Value(m["from"]); ok {
					element.From = int(value)
				}
				if value, ok := Int64Value(m["length"]); ok {
					element.Length = int(value)
				}
				if attrs, ok := m["attributes"].(map[string]any); ok {
					element.Attributes = &ElementAttributes{URL: StringValue(attrs["url"])}
				}
				msg.Elements = append(msg.Elements, element)
			}
		}
	}
	for _, key := range []string{"attaches", "attachments"} {
		if raw, ok := src[key].([]any); ok {
			for _, item := range raw {
				if m, ok := item.(map[string]any); ok {
					msg.Attachments = append(msg.Attachments, ParseAttachmentPayload(m))
				}
			}
			break
		}
	}
	return msg
}

func parseReactionInfo(src map[string]any) *ReactionInfo {
	info := &ReactionInfo{YourReaction: StringValue(src["yourReaction"])}
	if count, ok := Int64Value(src["totalCount"]); ok {
		info.TotalCount = int(count)
	}
	if counters, ok := src["counters"].([]any); ok {
		for _, raw := range counters {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			count, _ := Int64Value(m["count"])
			info.Counters = append(info.Counters, ReactionCounter{Count: int(count), Reaction: StringValue(m["reaction"])})
		}
	}
	return info
}

// ParseMessageDeletePayload accepts opcode 142's chat/messageIds shape and
// opcode 128's chatId/message shape.
func ParseMessageDeletePayload(src map[string]any) *MessageDeleteEvent {
	event := &MessageDeleteEvent{}
	event.ChatID, _ = Int64Value(src["chatId"])
	event.TTL, _ = src["ttl"].(bool)
	if chatData, ok := src["chat"].(map[string]any); ok {
		chat := ParseChatPayload(chatData)
		event.Chat = &chat
		if event.ChatID == 0 {
			event.ChatID = chat.ID
		}
	}
	if messageData, ok := src["message"].(map[string]any); ok {
		event.Message = ParseMessagePayload(messageData)
		if event.ChatID == 0 {
			event.ChatID = event.Message.ChatID
		}
		if event.Message.ID != 0 {
			event.MessageIDs = append(event.MessageIDs, event.Message.ID)
		}
	}
	if ids, ok := src["messageIds"].([]interface{}); ok {
		for _, rawID := range ids {
			if id, ok := Int64Value(rawID); ok {
				event.MessageIDs = append(event.MessageIDs, id)
			}
		}
	} else if ids, ok := src["messageIds"].([]int64); ok {
		event.MessageIDs = append(event.MessageIDs, ids...)
	}
	if len(event.MessageIDs) == 0 {
		for _, key := range []string{"messageId", "id"} {
			if id, ok := Int64Value(src[key]); ok && id != 0 {
				event.MessageIDs = append(event.MessageIDs, id)
				break
			}
		}
	}
	return event
}
