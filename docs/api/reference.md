# Полный публичный API

Этот раздел — единый список экспортируемого API GoMax. Здесь указаны точные
сигнатуры и назначение. Рабочие примеры вынесены на страницы соответствующих
сервисов.

## Как пользоваться справочником

| Что нужно сделать | Где смотреть |
|---|---|
| Быстро начать проект | [Начало работы](../getting-started.md) |
| Выбрать SMS или QR | [Авторизация](../authentication/sms.md) |
| Найти готовый сценарий | [Задачи](../tasks/index.md) |
| Найти метод сервиса | Страницы [Messages](messages.md), [Chats](chats.md), [Users](users.md) и другие |
| Узнать точную сигнатуру | Этот справочник |
| Посмотреть поля результата | [Типы и данные](types.md) |
| Собрать свой транспорт или event loop | [Низкоуровневый API](low-level.md) |

Все сетевые методы принимают первым аргументом `context.Context`. Каждый
возвращаемый `error` нужно проверять. Значения `chatID`, `userID` и
`messageID` имеют тип `int64`.

## Корневой пакет `gomax`

### Конструкторы и конфигурация

```go
func NewClient(cfg *Config) *Client
func NewWebClient(cfg *Config) *WebClient
func DefaultConfig() *Config
func NewSmsAuthFlow(codeProvider CodeProvider, passwordProvider PasswordProvider) *SmsAuthFlow
func NewQrAuthFlow(handler QrHandler, passwordProvider PasswordProvider) *QrAuthFlow
```

- `NewClient` создаёт TCP-клиент с SMS-авторизацией.
- `NewWebClient` создаёт WebSocket-клиент с QR-авторизацией.
- `DefaultConfig` возвращает базовую конфигурацию.
- `NewSmsAuthFlow` и `NewQrAuthFlow` позволяют заменить ввод кода, пароля и QR.

### Алиасы типов и константы

Корневой пакет переэкспортирует основные типы: `Config`, `Client`,
`WebClient`, `RegistrationConfig`, `NonRecoverableError`, `Message`, `Chat`, `User`, `Member`, `Attachment`, `Folder`,
`FolderList`, `Poll`, `PollOption`, `ReactionInfo`, `AuthResult` и типы
событий.

Константы вложений: `AttachmentPhoto`, `AttachmentVideo`, `AttachmentAudio`,
`AttachmentFile`, `AttachmentVoice`, `AttachmentVideoNote`, `AttachmentPoll`,
`AttachmentSticker`.

Константы чатов: `ChatTypeDialog`, `ChatTypeChat`, `ChatTypeChannel`.

Подробные поля перечислены в [типах и данных](types.md), а параметры
конфигурации — в [конфигурации](../configuration.md).

## `Client`

`Client` и `WebClient` имеют одинаковые сервисы и обработчики. Отличие:
`Client` подключается по TCP и обычно входит по SMS, а `WebClient` подключается
по WebSocket и обычно входит по QR.

```go
func (c *Client) Start(ctx context.Context) error
func (c *Client) Close() error
func (c *Client) SetInteractive(online bool)
func (c *Client) Invoke(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error)
func (c *Client) CallsSeed() int64
func (c *Client) GetCallsSeed() int64
func (c *Client) GetDeviceID() string
```

`Start` подключается, выполняет авторизацию и запускает обработку событий.
`Close` завершает соединение. `Invoke` нужен для ручного RPC-вызова; в обычном
коде используйте сервисы ниже. `CallsSeed`, `GetCallsSeed` и `GetDeviceID`
относятся к мобильному TCP-клиенту.

```go
type NonRecoverableError struct {
    Err error
}

func (e *NonRecoverableError) Error() string
func (e *NonRecoverableError) Unwrap() error
```

Эта ошибка означает, что повторное подключение не поможет. Проверяйте её через
`errors.As`, если нужно отличать окончательную ошибку от временного обрыва.

## `WebClient`

```go
func (c *WebClient) Start(ctx context.Context) error
func (c *WebClient) Close() error
func (c *WebClient) SetInteractive(online bool)
func (c *WebClient) Invoke(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error)
```

## Обработчики событий

Эти методы есть у `Client` и `WebClient`:

```go
func (c *Client) OnStart(handler func(context.Context) error)
func (c *Client) OnMessage(handler func(context.Context, *types.Message) error)
func (c *Client) OnMessageEdit(handler func(context.Context, *types.Message) error)
func (c *Client) OnMessageDelete(handler func(context.Context, int64, int64) error)
func (c *Client) OnMessageRead(handler func(context.Context, *types.MessageReadEvent) error)
func (c *Client) OnUserUpdate(handler func(context.Context, *types.UserUpdateEvent) error)
func (c *Client) OnReaction(handler func(context.Context, *types.ReactionEvent) error)
func (c *Client) OnChatUpdate(handler func(context.Context, *types.Chat) error)
func (c *Client) OnPresence(handler func(context.Context, *types.PresenceEvent) error)
func (c *Client) OnTyping(handler func(context.Context, *types.TypingEvent) error)
func (c *Client) OnDisconnect(handler func(context.Context, error))
func (c *Client) OnRaw(handler func(context.Context, *types.RawEvent) error)
```

У `WebClient` сигнатуры те же, меняется только получатель `*WebClient`.
Подробные примеры: [события](../dispatch/events.md).

## `client.Messages`

Полное описание: [сообщения](messages.md).

```go
func (s *messages.MessageService) SendMessage(ctx context.Context, chatID int64, text string, replyToMsgID int64, attaches []types.Attachment) (*types.Message, error)
func (s *messages.MessageService) EditMessage(ctx context.Context, chatID int64, messageID int64, newText string) error
func (s *messages.MessageService) DeleteMessage(ctx context.Context, chatID int64, messageID int64, forAll bool) error
func (s *messages.MessageService) ForwardMessage(ctx context.Context, chatID int64, messageID int64, sourceChatID int64, notify bool) (*types.Message, error)
func (s *messages.MessageService) ForwardMessages(ctx context.Context, toChatID int64, fromChatID int64, messageIDs []int64) error
func (s *messages.MessageService) PinMessage(ctx context.Context, chatID int64, messageID int64) error
func (s *messages.MessageService) GetChatHistory(ctx context.Context, chatID int64, fromTime int64, count int) ([]types.Message, error)
func (s *messages.MessageService) GetHistory(ctx context.Context, chatID int64, fromTime int64, count int) ([]types.Message, error)
func (s *messages.MessageService) GetMessages(ctx context.Context, chatID int64, messageIDs []int64) ([]types.Message, error)
func (s *messages.MessageService) GetMessage(ctx context.Context, chatID, messageID int64) (*types.Message, error)
func (s *messages.MessageService) GetVideoByID(ctx context.Context, chatID, messageID, videoID int64) (types.Attachment, error)
func (s *messages.MessageService) GetFileByID(ctx context.Context, chatID, messageID, fileID int64) (types.Attachment, error)
func (s *messages.MessageService) AddReaction(ctx context.Context, chatID int64, messageID int64, reaction string) error
func (s *messages.MessageService) RemoveReaction(ctx context.Context, chatID int64, messageID int64, reaction string) error
func (s *messages.MessageService) GetReactions(ctx context.Context, chatID int64, messageIDs []int64) (map[int64][]types.ReactionInfo, error)
func (s *messages.MessageService) ReadMessage(ctx context.Context, messageID int64, chatID int64) error
func (s *messages.MessageService) ReadMessages(ctx context.Context, chatID int64, messageIDs []int64) error
func (s *messages.MessageService) ReadChat(ctx context.Context, chatID int64, markID int64) error
func (s *messages.MessageService) VotePoll(ctx context.Context, chatID int64, messageID int64, pollID int64, optionIDs []int) error
```

## `client.Chats`

Полное описание: [чаты и каналы](chats.md).

```go
func (s *chats.ChatService) JoinChat(ctx context.Context, link string) (*types.Chat, error)
func (s *chats.ChatService) JoinGroup(ctx context.Context, link string) (*types.Chat, error)
func (s *chats.ChatService) JoinChannel(ctx context.Context, link string) (*types.Chat, error)
func (s *chats.ChatService) ResolveGroupByLink(ctx context.Context, link string) (*types.Chat, error)
func (s *chats.ChatService) CreateGroup(ctx context.Context, name string, participantIDs []int64, notify bool) (*types.Chat, error)
func (s *chats.ChatService) InviteUsersToGroup(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error
func (s *chats.ChatService) InviteUsersToChannel(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error
func (s *chats.ChatService) RemoveUsersFromGroup(ctx context.Context, chatID int64, userIDs []int64, cleanMsgPeriod int) error
func (s *chats.ChatService) GetChatMembers(ctx context.Context, chatID int64, count int, marker string) ([]types.Member, string, error)
func (s *chats.ChatService) GetChatMembersPage(ctx context.Context, chatID int64, marker, count int) ([]types.Member, int, error)
func (s *chats.ChatService) GetJoinRequests(ctx context.Context, chatID int64, count int) ([]types.Member, error)
func (s *chats.ChatService) ConfirmJoinRequests(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error
func (s *chats.ChatService) ConfirmJoinRequest(ctx context.Context, chatID, userID int64, showHistory bool) error
func (s *chats.ChatService) DeclineJoinRequests(ctx context.Context, chatID int64, userIDs []int64) error
func (s *chats.ChatService) DeclineJoinRequest(ctx context.Context, chatID, userID int64) error
func (s *chats.ChatService) AddAdmin(ctx context.Context, chatID, userID int64, permissions int) error
func (s *chats.ChatService) ChangeGroupSettings(ctx context.Context, chatID int64, allCanPin bool, onlyAdminCanAdd bool) error
func (s *chats.ChatService) ChangeGroupSettingsWithOptions(ctx context.Context, chatID int64, options map[string]bool) error
func (s *chats.ChatService) ChangeGroupProfile(ctx context.Context, chatID int64, name, description, photoToken string) error
func (s *chats.ChatService) ReworkInviteLink(ctx context.Context, chatID int64) (string, error)
func (s *chats.ChatService) ReworkInviteLinkChat(ctx context.Context, chatID int64) (*types.Chat, error)
func (s *chats.ChatService) LeaveChat(ctx context.Context, chatID int64) error
func (s *chats.ChatService) LeaveGroup(ctx context.Context, chatID int64) error
func (s *chats.ChatService) LeaveChannel(ctx context.Context, chatID int64) error
func (s *chats.ChatService) DeleteChat(ctx context.Context, chatID int64) error
func (s *chats.ChatService) FetchChats(ctx context.Context, count int, marker string) ([]types.Chat, string, error)
func (s *chats.ChatService) FetchChatsFromMarker(ctx context.Context, marker int64) ([]types.Chat, error)
func (s *chats.ChatService) GetChats(ctx context.Context, chatIDs []int64) ([]types.Chat, error)
func (s *chats.ChatService) GetChat(ctx context.Context, chatID int64) (*types.Chat, error)
func (s *chats.ChatService) GetChatInfo(ctx context.Context, chatID int64) (*types.Chat, error)
func (s *chats.ChatService) PublicSearch(ctx context.Context, query string, count int) ([]types.Chat, error)
```

## `client.Users`

Полное описание: [пользователи и контакты](users.md).

```go
func (s *users.UserService) GetUser(ctx context.Context, userID int64) (*types.User, error)
func (s *users.UserService) GetUsers(ctx context.Context, userIDs []int64) ([]types.User, error)
func (s *users.UserService) FetchUsers(ctx context.Context, userIDs []int64) ([]types.User, error)
func (s *users.UserService) GetCachedUser(ctx context.Context, userID int64) (*types.User, error)
func (s *users.UserService) SearchUsers(ctx context.Context, query string) ([]types.User, error)
func (s *users.UserService) SearchByPhone(ctx context.Context, phone string) (*types.User, error)
func (s *users.UserService) GetUserByPhone(ctx context.Context, phone string) (*types.User, error)
func (s *users.UserService) GetSelf(ctx context.Context) (*types.User, error)
func (s *users.UserService) GetContacts(ctx context.Context) ([]types.User, error)
func (s *users.UserService) GetChatID(_ context.Context, firstUserID, secondUserID int64) int64
func (s *users.UserService) AddContact(ctx context.Context, userID int64, firstName, lastName, phone string) error
func (s *users.UserService) AddContactByID(ctx context.Context, contactID int64) (*types.User, error)
func (s *users.UserService) UpdateContact(ctx context.Context, userID int64, firstName, lastName string) error
func (s *users.UserService) RemoveContact(ctx context.Context, contactID int64) error
func (s *users.UserService) ImportContacts(ctx context.Context, contacts map[string]string) ([]types.User, error)
func (s *users.UserService) GetSessions(ctx context.Context) ([]users.SessionItem, error)
func (s *users.UserService) GetActiveSessions(ctx context.Context) ([]users.SessionItem, error)
func (s *users.UserService) CloseSession(ctx context.Context, sessionID int64) error
func (s *users.UserService) Set2FA(ctx context.Context, password, hint, email string) error
```

## `client.Uploads`

Полное описание: [загрузки](uploads.md).

```go
func (s *uploads.UploadService) UploadPhoto(ctx context.Context, data []byte, fileName string) (*types.Attachment, error)
func (s *uploads.UploadService) UploadPhotoWithOptions(ctx context.Context, data []byte, fileName string, profile bool) (*types.Attachment, error)
func (s *uploads.UploadService) UploadVideo(ctx context.Context, data []byte, fileName string, duration int) (*types.Attachment, error)
func (s *uploads.UploadService) UploadVoice(ctx context.Context, data []byte, duration int) (*types.Attachment, error)
func (s *uploads.UploadService) UploadFile(ctx context.Context, data []byte, fileName string) (*types.Attachment, error)
func uploads.DecodeThumbhash(value string) ([]byte, error)
func (s *uploads.UploadService) NotifyReady(payload map[string]interface{})
func (s *uploads.UploadService) NotifyVideoReady(id int64)
func (s *uploads.UploadService) NotifyVoiceReady(id int64)
func (s *uploads.UploadService) NotifyFileReady(id int64)
```

`Notify*` — внутренние сигналы завершения обработки загрузки. В обычном
приложении их вызывать не нужно.

## `client.Self`

Полное описание: [профиль и настройки](self.md).

```go
func (s *selfapi.SelfService) GetSelf(ctx context.Context) (*types.User, error)
func (s *selfapi.SelfService) ChangeProfile(ctx context.Context, firstName, lastName, description, photoToken string) error
func (s *selfapi.SelfService) RequestProfilePhotoUploadURL(ctx context.Context) (string, error)
func (s *selfapi.SelfService) SetPresence(online bool)
func (s *selfapi.SelfService) ChangeProfileSettings(ctx context.Context, settings map[string]interface{}) error
func (s *selfapi.SelfService) GetFolders(ctx context.Context) (*types.FolderList, error)
func (s *selfapi.SelfService) CreateFolder(ctx context.Context, title string, chatInclude []int64) (*types.Folder, error)
func (s *selfapi.SelfService) UpdateFolder(ctx context.Context, folderID, title string, chatInclude []int64) (*types.Folder, error)
func (s *selfapi.SelfService) UpdateFolderWithOptions(ctx context.Context, folderID, title string, chatInclude []int64, filters, options []interface{}) (*types.Folder, error)
func (s *selfapi.SelfService) DeleteFolder(ctx context.Context, folderID string) error
func (s *selfapi.SelfService) CloseAllSessions(ctx context.Context) error
func (s *selfapi.SelfService) Logout(ctx context.Context) error
```

## `client.Auth`

Полное описание: [ручная авторизация](auth.md). Эти методы возвращают данные
протокола; для обычного входа используйте `client.Start`.

```go
func (s *authapi.AuthService) RequestCode(ctx context.Context, phone string, mode []byte) (map[string]interface{}, error)
func (s *authapi.AuthService) SendCode(ctx context.Context, token, code string) (map[string]interface{}, error)
func (s *authapi.AuthService) CheckPassword(ctx context.Context, trackID, password string) (map[string]interface{}, error)
func (s *authapi.AuthService) ConfirmRegistration(ctx context.Context, firstName, lastName, token string) (map[string]interface{}, error)
func (s *authapi.AuthService) RequestQR(ctx context.Context) (map[string]interface{}, error)
func (s *authapi.AuthService) CheckQR(ctx context.Context, trackID string) (map[string]interface{}, error)
func (s *authapi.AuthService) ConfirmQR(ctx context.Context, trackID string) (map[string]interface{}, error)
func (s *authapi.AuthService) ApproveQR(ctx context.Context, qrLink string) error
func (s *authapi.AuthService) CreateAuthTrack(ctx context.Context) (string, error)
func (s *authapi.AuthService) SetPassword(ctx context.Context, trackID, password string) error
func (s *authapi.AuthService) SetHint(ctx context.Context, trackID, hint string) error
func (s *authapi.AuthService) RequestEmailCode(ctx context.Context, trackID, email string) error
func (s *authapi.AuthService) VerifyEmailCode(ctx context.Context, trackID, code string) error
func (s *authapi.AuthService) CommitTwoFactor(ctx context.Context, trackID, password, hint string, capabilities []string) error
```

## `client.Bots`

Полное описание: [Bot Web App](bots.md).

```go
func (s *bots.BotsService) GetInitData(ctx context.Context, botID, chatID int64, startParam string) (*types.InitData, error)
```

## Auth providers и flow

```go
func auth.GetDeviceID() string
func auth.GetCallsSeed() int64

type CodeProvider interface {
    GetCode(context.Context) (string, error)
}

type PasswordProvider interface {
    GetPassword(context.Context) (string, error)
}

type PasswordProviderHint interface {
    GetPasswordWithHint(context.Context, string) (string, error)
}

type QrHandler interface {
    HandleQr(context.Context, string) error
}

type DeviceInfoProvider interface {
    GetDeviceID() string
    GetCallsSeed() int64
}

func (f *auth.SmsAuthFlow) Authenticate(ctx context.Context, invoker api.Invoker, phone string) (*auth.AuthResult, error)
func (f *auth.QrAuthFlow) Authenticate(ctx context.Context, invoker api.Invoker) (*auth.AuthResult, error)

func (p *auth.ConsoleCodeProvider) GetCode(ctx context.Context) (string, error)
func (p *auth.ConsolePasswordProvider) GetPassword(ctx context.Context) (string, error)
func (p *auth.ConsolePasswordProvider) GetPasswordWithHint(ctx context.Context, hint string) (string, error)
func (h *auth.ConsoleQrHandler) HandleQr(ctx context.Context, qrURL string) error
```

Если provider или handler не задан (`nil`), используются консольные реализации
из `pkg/auth`: `ConsoleCodeProvider`, `ConsolePasswordProvider` и
`ConsoleQrHandler`.

## Session stores

Полное описание: [сессия в JSON](../session/file.md), [сессия в RAM](../session/memory.md), [сессия в SQLite](../session/sqlite.md).

```go
func session.NewFileStore(workDir, sessionName string) *session.FileStore
func session.NewInMemoryStore() *session.InMemoryStore
func session.NewSqliteStore(db *sql.DB) (*session.SqliteStore, error)
```

Все встроенные store реализуют `session.Store`:

```go
type Store interface {
    SaveSession(*SessionInfo) error
    LoadSession() (*SessionInfo, error)
    UpdateToken(phone, newToken string) error
}
```

Расширенный интерфейс `session.ExtendedStore` добавляет поиск по телефону и
устройству, удаление сессий и `Close`. Конкретные методы перечислены на
странице [сессии в JSON](../session/file.md).

## Расширение и низкий уровень

Конструкторы сервисов, `dispatch.Router`, `protocol`, `connection`,
`transport` и `fingerprint` вынесены в отдельную страницу:
[Низкоуровневый API](low-level.md).

## Что не является публичным API

В справочник намеренно не входят неэкспортируемые helper-функции (`parse...`,
`handle...`, внутренние waiter-ы и преобразователи). Они могут измениться без
обратной совместимости и недоступны пользователю пакета.
