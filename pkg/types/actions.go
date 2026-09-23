package types

import (
	"context"
	"errors"
	"time"
)

var ErrUnboundModel = errors.New("gomax: domain model is not bound to a client service")

// MessageActionOptions controls the optional arguments shared by Message.Reply
// and Message.Answer. Notify defaults to true unless NotifySet is true.
type MessageActionOptions struct {
	ReplyToMessageID int64
	Notify           bool
	NotifySet        bool
	SendAt           time.Time
	NotifySender     bool
}

// HistoryActionOptions mirrors PyMax's full fetch_history controls.
type HistoryActionOptions struct {
	Forward        int
	Backward       int
	BackwardTime   int64
	ForwardTime    int64
	FromTime       int64
	ItemType       string
	GetChat        bool
	GetMessages    bool
	GetMessagesSet bool
	Interactive    bool
}

// ChatSettings contains all optional group settings. Nil means omitted.
type ChatSettings struct {
	AllCanPinMessage            *bool
	OnlyOwnerCanChangeIconTitle *bool
	OnlyAdminCanAddMember       *bool
	OnlyAdminCanCall            *bool
	MembersCanSeePrivateLink    *bool
}

// MessageActions is implemented by api/messages.MessageService. It keeps the
// domain package independent from API packages while supporting PyMax's bound
// Message and Chat convenience methods.
type MessageActions interface {
	SendMessageAction(context.Context, int64, string, []Attachment, MessageActionOptions) (*Message, error)
	FetchHistoryAction(context.Context, int64, HistoryActionOptions) ([]Message, error)
	GetMessage(context.Context, int64, int64) (*Message, error)
	GetMessages(context.Context, int64, []int64) ([]Message, error)
	ForwardMessage(context.Context, int64, any, int64, bool) (*Message, error)
	PinMessageWithNotify(context.Context, int64, int64, bool) error
	EditMessageResultWithAttachments(context.Context, int64, int64, string, []map[string]any, []Attachment) (*Message, error)
	DeleteMessage(context.Context, int64, int64, bool) error
	ReadMessageState(context.Context, any, int64) (*ReadState, error)
	AddReactionInfo(context.Context, int64, int64, string) (*ReactionInfo, error)
	RemoveReactionInfo(context.Context, int64, int64) (*ReactionInfo, error)
	GetReactionDetails(context.Context, int64, []int64) (map[int64]ReactionInfo, error)
}

// ChatActions is implemented by api/chats.ChatService.
type ChatActions interface {
	LeaveGroup(context.Context, int64) error
	LeaveChannel(context.Context, int64) error
	DeleteChatWithOptions(context.Context, int64, int64, bool) error
	InviteUsersToGroupResult(context.Context, int64, []int64, bool) (*Chat, error)
	InviteUsersToChannelResult(context.Context, int64, []int64, bool) (*Chat, error)
	RemoveUsersFromGroup(context.Context, int64, []int64, int) error
	ChangeGroupSettingsAction(context.Context, int64, ChatSettings) error
	ReworkInviteLinkChat(context.Context, int64) (*Chat, error)
}

// UserActions is implemented by api/users.UserService.
type UserActions interface {
	AddContactByID(context.Context, int64) (*User, error)
	RemoveContact(context.Context, int64) error
	GetChatID(context.Context, int64, int64) int64
}

// Bind attaches message operations to this value and returns the same pointer.
func (m *Message) Bind(actions MessageActions) *Message {
	if m != nil {
		m.actions = actions
	}
	return m
}

func (m *Message) bound() (MessageActions, int64, error) {
	if m == nil || m.actions == nil {
		return nil, 0, ErrUnboundModel
	}
	if m.ChatID == 0 {
		return nil, 0, errors.New("gomax: message does not contain chat ID")
	}
	return m.actions, m.ChatID, nil
}

func actionOptions(options []MessageActionOptions) MessageActionOptions {
	result := MessageActionOptions{Notify: true}
	if len(options) > 0 {
		result = options[0]
		if !result.NotifySet {
			result.Notify = true
		}
	}
	return result
}

func (m *Message) Reply(ctx context.Context, text string, attachments []Attachment, options ...MessageActionOptions) (*Message, error) {
	actions, chatID, err := m.bound()
	if err != nil {
		return nil, err
	}
	opts := actionOptions(options)
	opts.ReplyToMessageID = m.ID
	return actions.SendMessageAction(ctx, chatID, text, attachments, opts)
}

func (m *Message) Answer(ctx context.Context, text string, attachments []Attachment, options ...MessageActionOptions) (*Message, error) {
	actions, chatID, err := m.bound()
	if err != nil {
		return nil, err
	}
	return actions.SendMessageAction(ctx, chatID, text, attachments, actionOptions(options))
}

func (m *Message) Forward(ctx context.Context, chatID int64, notify bool) (*Message, error) {
	actions, sourceChatID, err := m.bound()
	if err != nil {
		return nil, err
	}
	return actions.ForwardMessage(ctx, chatID, m.ID, sourceChatID, notify)
}

func (m *Message) Pin(ctx context.Context, notify bool) error {
	actions, chatID, err := m.bound()
	if err != nil {
		return err
	}
	return actions.PinMessageWithNotify(ctx, chatID, m.ID, notify)
}

func (m *Message) Edit(ctx context.Context, text string, attachments []Attachment) (*Message, error) {
	actions, chatID, err := m.bound()
	if err != nil {
		return nil, err
	}
	return actions.EditMessageResultWithAttachments(ctx, chatID, m.ID, text, nil, attachments)
}

func (m *Message) Delete(ctx context.Context, forMe bool) error {
	actions, chatID, err := m.bound()
	if err != nil {
		return err
	}
	return actions.DeleteMessage(ctx, chatID, m.ID, !forMe)
}

func (m *Message) Read(ctx context.Context) (*ReadState, error) {
	actions, chatID, err := m.bound()
	if err != nil {
		return nil, err
	}
	return actions.ReadMessageState(ctx, m.ID, chatID)
}

func (m *Message) React(ctx context.Context, reaction string) (*ReactionInfo, error) {
	actions, chatID, err := m.bound()
	if err != nil {
		return nil, err
	}
	return actions.AddReactionInfo(ctx, chatID, m.ID, reaction)
}

func (m *Message) Unreact(ctx context.Context) (*ReactionInfo, error) {
	actions, chatID, err := m.bound()
	if err != nil {
		return nil, err
	}
	return actions.RemoveReactionInfo(ctx, chatID, m.ID)
}

func (m *Message) GetReactions(ctx context.Context) (map[int64]ReactionInfo, error) {
	actions, chatID, err := m.bound()
	if err != nil {
		return nil, err
	}
	return actions.GetReactionDetails(ctx, chatID, []int64{m.ID})
}

// Bind attaches both service families used by Chat convenience methods.
func (c *Chat) Bind(messages MessageActions, chats ChatActions) *Chat {
	if c == nil {
		return nil
	}
	c.messageActions, c.chatActions = messages, chats
	if c.LastMessage != nil {
		c.LastMessage.Bind(messages)
	}
	if c.PinnedMessage != nil {
		c.PinnedMessage.Bind(messages)
	}
	return c
}

func (c *Chat) Answer(ctx context.Context, text string, attachments []Attachment, options ...MessageActionOptions) (*Message, error) {
	if c == nil || c.messageActions == nil {
		return nil, ErrUnboundModel
	}
	return c.messageActions.SendMessageAction(ctx, c.ID, text, attachments, actionOptions(options))
}

func (c *Chat) History(ctx context.Context, options HistoryActionOptions) ([]Message, error) {
	if c == nil || c.messageActions == nil {
		return nil, ErrUnboundModel
	}
	return c.messageActions.FetchHistoryAction(ctx, c.ID, options)
}

func (c *Chat) GetMessage(ctx context.Context, messageID int64) (*Message, error) {
	if c == nil || c.messageActions == nil {
		return nil, ErrUnboundModel
	}
	return c.messageActions.GetMessage(ctx, c.ID, messageID)
}

func (c *Chat) GetMessages(ctx context.Context, messageIDs []int64) ([]Message, error) {
	if c == nil || c.messageActions == nil {
		return nil, ErrUnboundModel
	}
	return c.messageActions.GetMessages(ctx, c.ID, messageIDs)
}

func (c *Chat) Leave(ctx context.Context) error {
	if c == nil || c.chatActions == nil {
		return ErrUnboundModel
	}
	switch c.Type {
	case ChatTypeDialog:
		return errors.New("gomax: cannot leave a dialog")
	case ChatTypeChat:
		return c.chatActions.LeaveGroup(ctx, c.ID)
	case ChatTypeChannel:
		return c.chatActions.LeaveChannel(ctx, c.ID)
	default:
		return errors.New("gomax: unknown chat type")
	}
}

func (c *Chat) Delete(ctx context.Context, forAll bool) error {
	if c == nil || c.chatActions == nil {
		return ErrUnboundModel
	}
	return c.chatActions.DeleteChatWithOptions(ctx, c.ID, c.LastEventTime, forAll)
}

func (c *Chat) Invite(ctx context.Context, userIDs []int64, showHistory bool) (*Chat, error) {
	if c == nil || c.chatActions == nil {
		return nil, ErrUnboundModel
	}
	switch c.Type {
	case ChatTypeChat:
		return c.chatActions.InviteUsersToGroupResult(ctx, c.ID, userIDs, showHistory)
	case ChatTypeChannel:
		return c.chatActions.InviteUsersToChannelResult(ctx, c.ID, userIDs, showHistory)
	default:
		return nil, errors.New("gomax: chat type does not support invitations")
	}
}

func (c *Chat) RemoveUsers(ctx context.Context, userIDs []int64, cleanMessagePeriod int) error {
	if c == nil || c.chatActions == nil {
		return ErrUnboundModel
	}
	return c.chatActions.RemoveUsersFromGroup(ctx, c.ID, userIDs, cleanMessagePeriod)
}

func (c *Chat) PinMessage(ctx context.Context, messageID int64, notify bool) error {
	if c == nil || c.messageActions == nil {
		return ErrUnboundModel
	}
	return c.messageActions.PinMessageWithNotify(ctx, c.ID, messageID, notify)
}

func (c *Chat) UpdateSettings(ctx context.Context, settings ChatSettings) error {
	if c == nil || c.chatActions == nil {
		return ErrUnboundModel
	}
	return c.chatActions.ChangeGroupSettingsAction(ctx, c.ID, settings)
}

func (c *Chat) ReworkInviteLink(ctx context.Context) (*Chat, error) {
	if c == nil || c.chatActions == nil {
		return nil, ErrUnboundModel
	}
	return c.chatActions.ReworkInviteLinkChat(ctx, c.ID)
}

func (c Chat) IsDialog() bool      { return c.Type == ChatTypeDialog }
func (c Chat) IsGroup() bool       { return c.Type == ChatTypeChat }
func (c Chat) IsChannelChat() bool { return c.Type == ChatTypeChannel }

// Bind attaches contact operations to a user value.
func (u *User) Bind(actions UserActions) *User {
	if u != nil {
		u.actions = actions
	}
	return u
}

func (u *User) AddContact(ctx context.Context) (*User, error) {
	if u == nil || u.actions == nil {
		return nil, ErrUnboundModel
	}
	return u.actions.AddContactByID(ctx, u.ID)
}

func (u *User) RemoveContact(ctx context.Context) error {
	if u == nil || u.actions == nil {
		return ErrUnboundModel
	}
	return u.actions.RemoveContact(ctx, u.ID)
}

func (u *User) GetChatID(ctx context.Context, otherUserID int64) (int64, error) {
	if u == nil || u.actions == nil {
		return 0, ErrUnboundModel
	}
	return u.actions.GetChatID(ctx, otherUserID, u.ID), nil
}
