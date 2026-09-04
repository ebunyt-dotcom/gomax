package protocol

import "fmt"

// Command represents the low-level frame category in the Max protocol.
type Command uint8

const (
	// CmdRequest indicates a client-initiated request awaiting a response.
	CmdRequest Command = 0
	// CmdResponse indicates a server response resolving a prior request by sequence number.
	CmdResponse Command = 1
	// CmdEvent indicates a server push notification/event.
	CmdEvent Command = 2
	// CmdError indicates a server error frame resolving a prior request by sequence number.
	CmdError Command = 3
)

// String returns the canonical name of the Command.
func (c Command) String() string {
	switch c {
	case CmdRequest:
		return "REQUEST"
	case CmdResponse:
		return "RESPONSE"
	case CmdEvent:
		return "EVENT"
	case CmdError:
		return "ERROR"
	default:
		return fmt.Sprintf("Command(%d)", c)
	}
}

// IsValid checks if the command is within the known 0..3 range.
func (c Command) IsValid() bool {
	return c <= CmdError
}

// Opcode identifies the RPC method or notification event in the Max protocol.
type Opcode uint16

// Full catalog of all 114 protocol opcodes matching pymax.protocol.enums.Opcode.
const (
	OpPing                         Opcode = 1   // Keepalive ping/pong heartbeat
	OpDebug                        Opcode = 2   // Protocol debugging frame
	OpReconnect                    Opcode = 3   // Server instruction to disconnect and reconnect
	OpLog                          Opcode = 5   // Remote logging / telemetry
	OpSessionInit                  Opcode = 6   // Initial session handshake (returns callsSeed)
	OpLogin2                       Opcode = 8   // Secondary sync request for extended profile and contacts
	OpProfile                      Opcode = 16  // Fetch or update user profile
	OpAuthRequest                  Opcode = 17  // Request SMS verification code for phone number
	OpAuth                         Opcode = 18  // Submit SMS verification code, returns login token
	OpLogin                        Opcode = 19  // Main session login and initial sync
	OpLogout                       Opcode = 20  // Terminate active session
	OpSync                         Opcode = 21  // Incremental state synchronization
	OpConfig                       Opcode = 22  // Client configuration parameters
	OpAuthConfirm                  Opcode = 23  // Confirm registration for newly created accounts
	OpPresetAvatars                Opcode = 25  // Retrieve default avatar presets
	OpAssetsGet                    Opcode = 26  // Retrieve user sticker / asset packs
	OpAssetsUpdate                 Opcode = 27  // Update sticker / asset metadata
	OpAssetsGetByIds               Opcode = 28  // Batch query assets by IDs
	OpAssetsAdd                    Opcode = 29  // Add new assets/stickers to collection
	OpSearchFeedback               Opcode = 31  // Send analytics feedback on search queries
	OpContactInfo                  Opcode = 32  // Query detailed contact information
	OpContactAdd                   Opcode = 33  // Add user to contacts
	OpContactUpdate                Opcode = 34  // Rename or edit contact details
	OpContactPresence              Opcode = 35  // Query online/offline status for contacts
	OpContactList                  Opcode = 36  // Fetch complete contact list
	OpContactSearch                Opcode = 37  // Search contacts by phone or name
	OpContactMutual                Opcode = 38  // Retrieve mutual contact relationships
	OpContactPhotos                Opcode = 39  // Fetch contact profile photos
	OpContactSort                  Opcode = 40  // Update contact ordering preference
	OpContactVerify                Opcode = 42  // Verify contact authenticity
	OpRemoveContactPhoto           Opcode = 43  // Remove contact avatar
	OpContactInfoByPhone           Opcode = 46  // Look up contact by phone number
	OpChatInfo                     Opcode = 48  // Query chat metadata, title, and member count
	OpChatHistory                  Opcode = 49  // Fetch message history for a chat
	OpChatMark                     Opcode = 50  // Mark chat messages as read (mass-read)
	OpChatMedia                    Opcode = 51  // Retrieve media attachments shared in chat
	OpChatDelete                   Opcode = 52  // Delete conversation / dialog
	OpChatsList                    Opcode = 53  // Retrieve paginated list of dialogs
	OpChatClear                    Opcode = 54  // Clear message history in chat
	OpChatUpdate                   Opcode = 55  // Update chat settings, title, or avatar
	OpChatCheckLink                Opcode = 56  // Verify and resolve invite link
	OpChatJoin                     Opcode = 57  // Join public or private chat via link/ID
	OpChatLeave                    Opcode = 58  // Leave group chat or channel
	OpChatMembers                  Opcode = 59  // Fetch chat participants list
	OpPublicSearch                 Opcode = 60  // Global search across public channels & groups
	OpChatPersonalConfig           Opcode = 61  // User-specific chat configuration (mute, pin)
	OpChatLivestreamInfo           Opcode = 62  // Stream metadata for live broadcasts
	OpChatCreate                   Opcode = 63  // Create new group chat or broadcast channel
	OpMsgSend                      Opcode = 64  // Send new text/attachment message
	OpMsgTyping                    Opcode = 65  // Send typing/action indicator
	OpMsgDelete                    Opcode = 66  // Delete specific message
	OpMsgEdit                      Opcode = 67  // Edit previously sent message text/attachments
	OpChatSearch                   Opcode = 68  // Search messages within a specific chat
	OpMsgSharePreview              Opcode = 70  // Generate link preview card
	OpMsgGet                       Opcode = 71  // Retrieve single message by ID
	OpMsgSearchTouch               Opcode = 72  // Update recent message search index
	OpMsgSearch                    Opcode = 73  // Global message search
	OpMsgGetStat                   Opcode = 74  // Channel message view and forward statistics
	OpChatSubscribe                Opcode = 75  // Subscribe to channel updates
	OpVideoChatStart               Opcode = 76  // Initiate group audio/video conference
	OpChatMembersUpdate            Opcode = 77  // Modify member roles or permissions
	OpVideoChatStartActive         Opcode = 78  // Join active group video call
	OpVideoChatHistory             Opcode = 79  // Video call log history
	OpPhotoUpload                  Opcode = 80  // Request photo upload URL / token
	OpStickerUpload                Opcode = 81  // Upload sticker file
	OpVideoUpload                  Opcode = 82  // Request video upload slot
	OpVideoPlay                    Opcode = 83  // Retrieve video streaming playback URL
	OpVideoChatCreateJoinLink      Opcode = 84  // Generate join URL for video room
	OpChatPinSetVisibility         Opcode = 86  // Toggle visibility of pinned message banner
	OpFileUpload                   Opcode = 87  // Request generic document/file upload endpoint
	OpFileDownload                 Opcode = 88  // Retrieve file download URL
	OpLinkInfo                     Opcode = 89  // Resolve OpenGraph metadata for external URL
	OpMsgDeleteRange               Opcode = 92  // Batch delete messages in a range
	OpSessionsInfo                 Opcode = 96  // Fetch active device sessions list
	OpSessionsClose                Opcode = 97  // Terminate another device session
	OpPhoneBindRequest             Opcode = 98  // Request SMS to link new phone number
	OpPhoneBindConfirm             Opcode = 99  // Submit code to confirm new phone number
	OpAuthLoginRestorePassword     Opcode = 101 // Initiate 2FA password recovery flow
	OpGetInboundCalls              Opcode = 103 // Query pending incoming VoIP calls
	OpAuth2FaDetails               Opcode = 104 // Query 2FA status, hint, and recovery email
	OpExternalCallback             Opcode = 105 // Handle external OAuth / SSO callbacks
	OpAuthValidatePassword         Opcode = 107 // Validate password format before setting
	OpAuthValidateHint             Opcode = 108 // Validate password hint
	OpAuthVerifyEmail              Opcode = 109 // Request verification code to 2FA email
	OpAuthCheckEmail               Opcode = 110 // Verify email code for 2FA setup
	OpAuthSet2Fa                   Opcode = 111 // Commit new 2FA password / remove 2FA
	OpAuthCreateTrack              Opcode = 112 // Create tracking ID for multi-step auth
	OpAuthCheckPassword            Opcode = 113 // Verify current 2FA password
	OpAuthLoginCheckPassword       Opcode = 115 // Submit 2FA password during login challenge
	OpAuthLoginProfileDelete       Opcode = 116 // Delete profile during login process
	OpChatComplain                 Opcode = 117 // File moderation report against a chat
	OpMsgSendCallback              Opcode = 118 // Click inline keyboard button (bot callback)
	OpSuspendBot                   Opcode = 119 // Block or unblock a bot
	OpLocationStop                 Opcode = 124 // Stop sharing live geolocation
	OpLocationSend                 Opcode = 125 // Update live geolocation coordinates
	OpLocationRequest              Opcode = 126 // Request contact's live location
	OpGetLastMentions              Opcode = 127 // Query unread @mentions across all chats
	OpNotifMessage                 Opcode = 128 // Event: Incoming message / message state update
	OpNotifTyping                  Opcode = 129 // Event: User typing indicator
	OpNotifMark                    Opcode = 130 // Event: Read receipt notification
	OpNotifContact                 Opcode = 131 // Event: Contact profile changed
	OpNotifPresence                Opcode = 132 // Event: User presence / online status changed
	OpNotifConfig                  Opcode = 134 // Event: App configuration updated
	OpNotifChat                    Opcode = 135 // Event: Chat metadata or membership updated
	OpNotifAttach                  Opcode = 136 // Event: Attachment media processing completed
	OpNotifCallStart               Opcode = 137 // Event: Incoming call alert
	OpNotifContactSort             Opcode = 139 // Event: Contact sorting order updated
	OpNotifMsgDeleteRange          Opcode = 140 // Event: Range of messages removed
	OpNotifMsgDelete               Opcode = 142 // Event: Single message deleted
	OpNotifCallbackAnswer          Opcode = 143 // Event: Bot responded to inline button press
	OpChatBotCommands              Opcode = 144 // Fetch available slash commands for bot
	OpBotInfo                      Opcode = 145 // Query bot description and capabilities
	OpNotifLocation                Opcode = 147 // Event: Live location stream update
	OpNotifLocationRequest         Opcode = 148 // Event: Request to share location
	OpNotifAssetsUpdate            Opcode = 150 // Event: User stickers/assets updated
	OpNotifDraft                   Opcode = 152 // Event: Chat message draft saved
	OpNotifDraftDiscard            Opcode = 153 // Event: Chat message draft discarded
	OpNotifMsgDelayed              Opcode = 154 // Event: Scheduled message pending dispatch
	OpNotifMsgReactionsChanged     Opcode = 155 // Event: Message reaction counts / emoji changed
	OpNotifMsgYouReacted           Opcode = 156 // Event: Self reaction acknowledged
	OpCallsToken                   Opcode = 158 // VoIP signaling token
	OpNotifProfile                 Opcode = 159 // Event: Self profile updated from another client
	OpWebAppInitData               Opcode = 160 // Initialize Mini-App session
	OpComplain                     Opcode = 161 // Submit user report
	OpComplainReasonsGet           Opcode = 162 // Query valid complaint reason codes
	OpVideoChatJoin                Opcode = 166 // Join existing group video call
	OpDraftSave                    Opcode = 176 // Persist sync draft message to cloud
	OpDraftDiscard                 Opcode = 177 // Clear cloud sync draft
	OpMsgReaction                  Opcode = 178 // Add reaction emoji to message
	OpMsgCancelReaction            Opcode = 179 // Remove reaction from message
	OpMsgGetReactions              Opcode = 180 // Query reaction summary for message
	OpMsgGetDetailedReactions      Opcode = 181 // List individual users who reacted
	OpStickerCreate                Opcode = 193 // Create custom user sticker
	OpStickerSuggest               Opcode = 194 // Retrieve suggested stickers for emoji
	OpVideoChatMembers             Opcode = 195 // List call participants
	OpChatHide                     Opcode = 196 // Hide dialog from main list
	OpChatSearchCommonParticipants Opcode = 198 // Find mutual groups with user
	OpProfileDelete                Opcode = 199 // Request account deletion
	OpProfileDeleteTime            Opcode = 200 // Query grace period countdown for deletion
	OpTranscribeMedia              Opcode = 202 // Speech-to-text transcription of voice message
	OpStoriesList                  Opcode = 208 // Fetch active user stories feed
	OpStoriesListByOwnerId         Opcode = 209 // Query stories by specific user
	OpStoriesGetByOwnerId          Opcode = 210 // Load full story item
	OpStoriesGetStats              Opcode = 211 // Query story view statistics
	OpStoriesGetDetailedStats      Opcode = 212 // Detailed story viewers list
	OpStoriesReact                 Opcode = 213 // React to story
	OpStoriesMark                  Opcode = 214 // Mark story as viewed
	OpStoriesSend                  Opcode = 215 // Upload and publish new story
	OpNotifStoriesUpdate           Opcode = 216 // Event: Stories feed updated
	OpStoriesEdit                  Opcode = 217 // Edit published story
	OpStoriesDelete                Opcode = 218 // Delete published story
	OpStoriesGetByStoryId          Opcode = 220 // Fetch specific story metadata
	OpOrgInfo                      Opcode = 256 // Corporate organization metadata
	OpChatReactionsSettingsSet     Opcode = 257 // Restrict allowed reactions in chat
	OpReactionsSettingsGetByChatId Opcode = 258 // Query reaction restrictions in chat
	OpAssetsRemove                 Opcode = 259 // Remove sticker pack from collection
	OpAssetsMove                   Opcode = 260 // Reorder sticker packs
	OpAssetsListModify             Opcode = 261 // Batch modify sticker collections
	OpFoldersGet                   Opcode = 272 // Fetch chat folder organization
	OpFoldersGetById               Opcode = 273 // Fetch single chat folder
	OpFoldersUpdate                Opcode = 274 // Create or edit chat folder
	OpFoldersReorder               Opcode = 275 // Reorder chat folders
	OpFoldersDelete                Opcode = 276 // Delete chat folder
	OpNotifFolders                 Opcode = 277 // Event: Chat folders structure changed
	OpGetQr                        Opcode = 288 // Request new QR code for login
	OpGetQrStatus                  Opcode = 289 // Poll QR scan/approval status
	OpAuthQrApprove                Opcode = 290 // Approve QR login from mobile client
	OpLoginByQr                    Opcode = 291 // Finalize login via approved QR code
	OpNotifBanners                 Opcode = 292 // Event: App announcement banners
	OpNotifTranscription           Opcode = 293 // Event: Voice-to-text async result ready
	OpChatSuggest                  Opcode = 300 // Suggested chats / channels
	OpAudioPlay                    Opcode = 301 // Audio playback URL resolution
	OpBannersGet                   Opcode = 302 // Fetch marketing / system banners
	OpMsgDelivery                  Opcode = 303 // Message delivery status confirmation
	OpSendVote                     Opcode = 304 // Cast vote in poll
	OpVotersListByAnswer           Opcode = 305 // List participants who voted for an option
	OpGetPollUpdates               Opcode = 306 // Poll results live sync
)

// Common Opcode aliases matching connection and client conventions.
const (
	OpcodePing        = OpPing
	OpcodeSessionInit = OpSessionInit
)

// opcodeNames maps Opcode numeric values to their canonical string identifiers.
var opcodeNames = map[Opcode]string{
	OpPing:                         "PING",
	OpDebug:                        "DEBUG",
	OpReconnect:                    "RECONNECT",
	OpLog:                          "LOG",
	OpSessionInit:                  "SESSION_INIT",
	OpLogin2:                       "LOGIN2",
	OpProfile:                      "PROFILE",
	OpAuthRequest:                  "AUTH_REQUEST",
	OpAuth:                         "AUTH",
	OpLogin:                        "LOGIN",
	OpLogout:                       "LOGOUT",
	OpSync:                         "SYNC",
	OpConfig:                       "CONFIG",
	OpAuthConfirm:                  "AUTH_CONFIRM",
	OpPresetAvatars:                "PRESET_AVATARS",
	OpAssetsGet:                    "ASSETS_GET",
	OpAssetsUpdate:                 "ASSETS_UPDATE",
	OpAssetsGetByIds:               "ASSETS_GET_BY_IDS",
	OpAssetsAdd:                    "ASSETS_ADD",
	OpSearchFeedback:               "SEARCH_FEEDBACK",
	OpContactInfo:                  "CONTACT_INFO",
	OpContactAdd:                   "CONTACT_ADD",
	OpContactUpdate:                "CONTACT_UPDATE",
	OpContactPresence:              "CONTACT_PRESENCE",
	OpContactList:                  "CONTACT_LIST",
	OpContactSearch:                "CONTACT_SEARCH",
	OpContactMutual:                "CONTACT_MUTUAL",
	OpContactPhotos:                "CONTACT_PHOTOS",
	OpContactSort:                  "CONTACT_SORT",
	OpContactVerify:                "CONTACT_VERIFY",
	OpRemoveContactPhoto:           "REMOVE_CONTACT_PHOTO",
	OpContactInfoByPhone:           "CONTACT_INFO_BY_PHONE",
	OpChatInfo:                     "CHAT_INFO",
	OpChatHistory:                  "CHAT_HISTORY",
	OpChatMark:                     "CHAT_MARK",
	OpChatMedia:                    "CHAT_MEDIA",
	OpChatDelete:                   "CHAT_DELETE",
	OpChatsList:                    "CHATS_LIST",
	OpChatClear:                    "CHAT_CLEAR",
	OpChatUpdate:                   "CHAT_UPDATE",
	OpChatCheckLink:                "CHAT_CHECK_LINK",
	OpChatJoin:                     "CHAT_JOIN",
	OpChatLeave:                    "CHAT_LEAVE",
	OpChatMembers:                  "CHAT_MEMBERS",
	OpPublicSearch:                 "PUBLIC_SEARCH",
	OpChatPersonalConfig:           "CHAT_PERSONAL_CONFIG",
	OpChatLivestreamInfo:           "CHAT_LIVESTREAM_INFO",
	OpChatCreate:                   "CHAT_CREATE",
	OpMsgSend:                      "MSG_SEND",
	OpMsgTyping:                    "MSG_TYPING",
	OpMsgDelete:                    "MSG_DELETE",
	OpMsgEdit:                      "MSG_EDIT",
	OpChatSearch:                   "CHAT_SEARCH",
	OpMsgSharePreview:              "MSG_SHARE_PREVIEW",
	OpMsgGet:                       "MSG_GET",
	OpMsgSearchTouch:               "MSG_SEARCH_TOUCH",
	OpMsgSearch:                    "MSG_SEARCH",
	OpMsgGetStat:                   "MSG_GET_STAT",
	OpChatSubscribe:                "CHAT_SUBSCRIBE",
	OpVideoChatStart:               "VIDEO_CHAT_START",
	OpChatMembersUpdate:            "CHAT_MEMBERS_UPDATE",
	OpVideoChatStartActive:         "VIDEO_CHAT_START_ACTIVE",
	OpVideoChatHistory:             "VIDEO_CHAT_HISTORY",
	OpPhotoUpload:                  "PHOTO_UPLOAD",
	OpStickerUpload:                "STICKER_UPLOAD",
	OpVideoUpload:                  "VIDEO_UPLOAD",
	OpVideoPlay:                    "VIDEO_PLAY",
	OpVideoChatCreateJoinLink:      "VIDEO_CHAT_CREATE_JOIN_LINK",
	OpChatPinSetVisibility:         "CHAT_PIN_SET_VISIBILITY",
	OpFileUpload:                   "FILE_UPLOAD",
	OpFileDownload:                 "FILE_DOWNLOAD",
	OpLinkInfo:                     "LINK_INFO",
	OpMsgDeleteRange:               "MSG_DELETE_RANGE",
	OpSessionsInfo:                 "SESSIONS_INFO",
	OpSessionsClose:                "SESSIONS_CLOSE",
	OpPhoneBindRequest:             "PHONE_BIND_REQUEST",
	OpPhoneBindConfirm:             "PHONE_BIND_CONFIRM",
	OpAuthLoginRestorePassword:     "AUTH_LOGIN_RESTORE_PASSWORD",
	OpGetInboundCalls:              "GET_INBOUND_CALLS",
	OpAuth2FaDetails:               "AUTH_2FA_DETAILS",
	OpExternalCallback:             "EXTERNAL_CALLBACK",
	OpAuthValidatePassword:         "AUTH_VALIDATE_PASSWORD",
	OpAuthValidateHint:             "AUTH_VALIDATE_HINT",
	OpAuthVerifyEmail:              "AUTH_VERIFY_EMAIL",
	OpAuthCheckEmail:               "AUTH_CHECK_EMAIL",
	OpAuthSet2Fa:                   "AUTH_SET_2FA",
	OpAuthCreateTrack:              "AUTH_CREATE_TRACK",
	OpAuthCheckPassword:            "AUTH_CHECK_PASSWORD",
	OpAuthLoginCheckPassword:       "AUTH_LOGIN_CHECK_PASSWORD",
	OpAuthLoginProfileDelete:       "AUTH_LOGIN_PROFILE_DELETE",
	OpChatComplain:                 "CHAT_COMPLAIN",
	OpMsgSendCallback:              "MSG_SEND_CALLBACK",
	OpSuspendBot:                   "SUSPEND_BOT",
	OpLocationStop:                 "LOCATION_STOP",
	OpLocationSend:                 "LOCATION_SEND",
	OpLocationRequest:              "LOCATION_REQUEST",
	OpGetLastMentions:              "GET_LAST_MENTIONS",
	OpNotifMessage:                 "NOTIF_MESSAGE",
	OpNotifTyping:                  "NOTIF_TYPING",
	OpNotifMark:                    "NOTIF_MARK",
	OpNotifContact:                 "NOTIF_CONTACT",
	OpNotifPresence:                "NOTIF_PRESENCE",
	OpNotifConfig:                  "NOTIF_CONFIG",
	OpNotifChat:                    "NOTIF_CHAT",
	OpNotifAttach:                  "NOTIF_ATTACH",
	OpNotifCallStart:               "NOTIF_CALL_START",
	OpNotifContactSort:             "NOTIF_CONTACT_SORT",
	OpNotifMsgDeleteRange:          "NOTIF_MSG_DELETE_RANGE",
	OpNotifMsgDelete:               "NOTIF_MSG_DELETE",
	OpNotifCallbackAnswer:          "NOTIF_CALLBACK_ANSWER",
	OpChatBotCommands:              "CHAT_BOT_COMMANDS",
	OpBotInfo:                      "BOT_INFO",
	OpNotifLocation:                "NOTIF_LOCATION",
	OpNotifLocationRequest:         "NOTIF_LOCATION_REQUEST",
	OpNotifAssetsUpdate:            "NOTIF_ASSETS_UPDATE",
	OpNotifDraft:                   "NOTIF_DRAFT",
	OpNotifDraftDiscard:            "NOTIF_DRAFT_DISCARD",
	OpNotifMsgDelayed:              "NOTIF_MSG_DELAYED",
	OpNotifMsgReactionsChanged:     "NOTIF_MSG_REACTIONS_CHANGED",
	OpNotifMsgYouReacted:           "NOTIF_MSG_YOU_REACTED",
	OpCallsToken:                   "CALLS_TOKEN",
	OpNotifProfile:                 "NOTIF_PROFILE",
	OpWebAppInitData:               "WEB_APP_INIT_DATA",
	OpComplain:                     "COMPLAIN",
	OpComplainReasonsGet:           "COMPLAIN_REASONS_GET",
	OpVideoChatJoin:                "VIDEO_CHAT_JOIN",
	OpDraftSave:                    "DRAFT_SAVE",
	OpDraftDiscard:                 "DRAFT_DISCARD",
	OpMsgReaction:                  "MSG_REACTION",
	OpMsgCancelReaction:            "MSG_CANCEL_REACTION",
	OpMsgGetReactions:              "MSG_GET_REACTIONS",
	OpMsgGetDetailedReactions:      "MSG_GET_DETAILED_REACTIONS",
	OpStickerCreate:                "STICKER_CREATE",
	OpStickerSuggest:               "STICKER_SUGGEST",
	OpVideoChatMembers:             "VIDEO_CHAT_MEMBERS",
	OpChatHide:                     "CHAT_HIDE",
	OpChatSearchCommonParticipants: "CHAT_SEARCH_COMMON_PARTICIPANTS",
	OpProfileDelete:                "PROFILE_DELETE",
	OpProfileDeleteTime:            "PROFILE_DELETE_TIME",
	OpTranscribeMedia:              "TRANSCRIBE_MEDIA",
	OpStoriesList:                  "STORIES_LIST",
	OpStoriesListByOwnerId:         "STORIES_LIST_BY_OWNER_ID",
	OpStoriesGetByOwnerId:          "STORIES_GET_BY_OWNER_ID",
	OpStoriesGetStats:              "STORIES_GET_STATS",
	OpStoriesGetDetailedStats:      "STORIES_GET_DETAILED_STATS",
	OpStoriesReact:                 "STORIES_REACT",
	OpStoriesMark:                  "STORIES_MARK",
	OpStoriesSend:                  "STORIES_SEND",
	OpNotifStoriesUpdate:           "NOTIF_STORIES_UPDATE",
	OpStoriesEdit:                  "STORIES_EDIT",
	OpStoriesDelete:                "STORIES_DELETE",
	OpStoriesGetByStoryId:          "STORIES_GET_BY_STORY_ID",
	OpOrgInfo:                      "ORG_INFO",
	OpChatReactionsSettingsSet:     "CHAT_REACTIONS_SETTINGS_SET",
	OpReactionsSettingsGetByChatId: "REACTIONS_SETTINGS_GET_BY_CHAT_ID",
	OpAssetsRemove:                 "ASSETS_REMOVE",
	OpAssetsMove:                   "ASSETS_MOVE",
	OpAssetsListModify:             "ASSETS_LIST_MODIFY",
	OpFoldersGet:                   "FOLDERS_GET",
	OpFoldersGetById:               "FOLDERS_GET_BY_ID",
	OpFoldersUpdate:                "FOLDERS_UPDATE",
	OpFoldersReorder:               "FOLDERS_REORDER",
	OpFoldersDelete:                "FOLDERS_DELETE",
	OpNotifFolders:                 "NOTIF_FOLDERS",
	OpGetQr:                        "GET_QR",
	OpGetQrStatus:                  "GET_QR_STATUS",
	OpAuthQrApprove:                "AUTH_QR_APPROVE",
	OpLoginByQr:                    "LOGIN_BY_QR",
	OpNotifBanners:                 "NOTIF_BANNERS",
	OpNotifTranscription:           "NOTIF_TRANSCRIPTION",
	OpChatSuggest:                  "CHAT_SUGGEST",
	OpAudioPlay:                    "AUDIO_PLAY",
	OpBannersGet:                   "BANNERS_GET",
	OpMsgDelivery:                  "MSG_DELIVERY",
	OpSendVote:                     "SEND_VOTE",
	OpVotersListByAnswer:           "VOTERS_LIST_BY_ANSWER",
	OpGetPollUpdates:               "GET_POLL_UPDATES",
}

// String returns the name of the Opcode or "Opcode(N)" if unknown.
func (o Opcode) String() string {
	if name, ok := opcodeNames[o]; ok {
		return name
	}
	return fmt.Sprintf("Opcode(%d)", o)
}

// IsNotification returns true if this opcode is a server-pushed asynchronous notification event.
func (o Opcode) IsNotification() bool {
	switch o {
	case OpNotifMessage, OpNotifTyping, OpNotifMark, OpNotifContact,
		OpNotifPresence, OpNotifConfig, OpNotifChat, OpNotifAttach,
		OpNotifCallStart, OpNotifContactSort, OpNotifMsgDeleteRange,
		OpNotifMsgDelete, OpNotifCallbackAnswer, OpNotifLocation,
		OpNotifLocationRequest, OpNotifAssetsUpdate, OpNotifDraft,
		OpNotifDraftDiscard, OpNotifMsgDelayed, OpNotifMsgReactionsChanged,
		OpNotifMsgYouReacted, OpNotifProfile, OpNotifStoriesUpdate,
		OpNotifFolders, OpNotifBanners, OpNotifTranscription:
		return true
	default:
		return false
	}
}

// CountOpcodes returns the total number of defined opcodes (114).
func CountOpcodes() int {
	return len(opcodeNames)
}
