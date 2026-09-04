# Полный публичный API

Этот файл — быстрый справочник публичных функций GoMax. Подробное объяснение и примеры находятся на страницах сервисов.

## Корневой пакет gomax

| Функция | Назначение |
|---|---|
| NewClient | Создать TCP/SMS-клиент |
| NewWebClient | Создать WebSocket/QR-клиент |
| DefaultConfig | Получить конфигурацию с рабочими defaults |
| NewSmsAuthFlow | Создать SMS/2FA flow |
| NewQrAuthFlow | Создать QR/2FA flow |

## Client и WebClient

Оба клиента поддерживают:

| Метод | Назначение |
|---|---|
| Start | Подключиться, авторизоваться и запустить цикл событий |
| Close | Закрыть соединение |
| Invoke | Выполнить низкоуровневый RPC-вызов |
| SetInteractive | Изменить признак активности/presence для следующего login/reconnect |
| OnStart | Обработчик готовности клиента |
| OnMessage | Новое сообщение |
| OnMessageEdit | Изменение сообщения |
| OnMessageDelete | Удаление сообщения |
| OnMessageRead | Read-marker |
| OnUserUpdate | Изменение контакта или профиля |
| OnReaction | Реакция |
| OnChatUpdate | Изменение чата |
| OnPresence | Online-статус |
| OnTyping | Индикатор печати |
| OnDisconnect | Закрытие или ошибка соединения |
| OnRaw | Нераспознанное событие |

Только Client дополнительно предоставляет CallsSeed, GetCallsSeed и GetDeviceID.

## client.Messages

SendMessage, GetChatHistory, GetHistory, GetMessages, GetMessage, EditMessage, DeleteMessage, ForwardMessage, ForwardMessages, PinMessage, AddReaction, RemoveReaction, GetReactions, ReadMessage, ReadMessages, ReadChat, VotePoll, GetVideoByID, GetFileByID.

Подробности: [сообщения](messages.md).

## client.Chats

JoinChat, JoinGroup, JoinChannel, ResolveGroupByLink, CreateGroup, InviteUsersToGroup, InviteUsersToChannel, RemoveUsersFromGroup, LeaveChat, LeaveGroup, LeaveChannel, DeleteChat, ChangeGroupSettings, ChangeGroupSettingsWithOptions, ChangeGroupProfile, GetChatMembers, GetChatMembersPage, FetchChats, FetchChatsFromMarker, GetChats, GetChat, GetChatInfo, GetJoinRequests, ConfirmJoinRequests, ConfirmJoinRequest, DeclineJoinRequests, DeclineJoinRequest, AddAdmin, ReworkInviteLink, ReworkInviteLinkChat, PublicSearch.

Подробности: [чаты](chats.md).

## client.Users

GetUser, GetCachedUser, FetchUsers, GetUsers, SearchUsers, SearchByPhone, GetUserByPhone, GetContacts, GetSelf, GetActiveSessions, GetSessions, CloseSession, Set2FA, AddContact, AddContactByID, UpdateContact, RemoveContact, ImportContacts, GetChatID.

Подробности: [пользователи](users.md).

## client.Uploads

UploadPhoto, UploadPhotoWithOptions, UploadVideo, UploadVoice, UploadFile, NotifyReady, NotifyVideoReady, NotifyVoiceReady, NotifyFileReady, DecodeThumbhash.

Подробности: [загрузки](uploads.md).

## client.Self

GetSelf, ChangeProfile, RequestProfilePhotoUploadURL, SetPresence, ChangeProfileSettings, GetFolders, CreateFolder, UpdateFolder, UpdateFolderWithOptions, DeleteFolder, CloseAllSessions, Logout.

Подробности: [профиль](self.md).

## client.Auth

RequestCode, SendCode, CheckPassword, RequestQR, CheckQR, ConfirmQR, ApproveQR, CreateAuthTrack, SetPassword, SetHint, RequestEmailCode, VerifyEmailCode, CommitTwoFactor, ConfirmRegistration.

Подробности: [низкоуровневая авторизация](auth.md).

## client.Bots

GetInitData.

Подробности: [Bot Web App](bots.md).

## Auth providers

Пакет gomax/pkg/auth предоставляет ConsoleCodeProvider, ConsolePasswordProvider, ConsoleQrHandler, NewSmsAuthFlow, NewQrAuthFlow, SmsAuthFlow.Authenticate и QrAuthFlow.Authenticate.

## Session stores

Пакет gomax/pkg/session предоставляет NewFileStore, NewInMemoryStore и NewSqliteStore. Их назначение описано в разделе [сессий](../session/file.md).

## Что намеренно не описывается отдельно

Непубличные helper-функции (parse..., handle..., внутренние waiter-и и преобразователи) не являются API приложения и могут меняться без обратной совместимости.


## Как читать сигнатуры

Каждый сетевой метод принимает первым аргументом `ctx context.Context`. Он нужен для отмены запроса и таймаута:

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    chat, err := client.Chats.GetChat(ctx, chatID)
    if err != nil {
        return err
    }

Если метод возвращает `error`, её нужно проверять. Если возвращается указатель, например `*gomax.Chat`, результат может быть `nil` при ошибке. Пустые `[]int64`, `map[string]bool` и необязательные строки передавайте только если метод допускает их по смыслу.

## Точные сигнатуры корневого пакета

    func gomax.NewClient(cfg *gomax.Config) *gomax.Client
    func gomax.NewWebClient(cfg *gomax.Config) *gomax.WebClient
    func gomax.DefaultConfig() *gomax.Config
    func gomax.NewSmsAuthFlow(codeProvider gomax.CodeProvider, passwordProvider gomax.PasswordProvider) *gomax.SmsAuthFlow
    func gomax.NewQrAuthFlow(handler gomax.QrHandler, passwordProvider gomax.PasswordProvider) *gomax.QrAuthFlow

Передайте `nil` в `NewClient` или `NewWebClient`, чтобы использовать конфигурацию по умолчанию. Для рабочего приложения обычно достаточно изменить `Phone` и `SessionName`.

## Client и WebClient

Методы жизненного цикла:

    func (c *gomax.Client) Start(ctx context.Context) error
    func (c *gomax.Client) Close() error
    func (c *gomax.Client) SetInteractive(online bool)
    func (c *gomax.Client) Invoke(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error)

У `WebClient` те же методы с получателем `*gomax.WebClient`. `Invoke` — низкоуровневый вызов протокола; для обычного кода используйте сервисы `Messages`, `Chats`, `Users`, `Uploads`, `Self`, `Auth` и `Bots`.

Методы обработчиков у обоих клиентов:

    func (c *gomax.Client) OnStart(func(context.Context) error)
    func (c *gomax.Client) OnMessage(func(context.Context, *gomax.Message) error)
    func (c *gomax.Client) OnMessageEdit(func(context.Context, *gomax.Message) error)
    func (c *gomax.Client) OnMessageDelete(func(context.Context, int64, int64) error)
    func (c *gomax.Client) OnMessageRead(func(context.Context, *types.MessageReadEvent) error)
    func (c *gomax.Client) OnUserUpdate(func(context.Context, *types.UserUpdateEvent) error)
    func (c *gomax.Client) OnReaction(func(context.Context, *gomax.ReactionEvent) error)
    func (c *gomax.Client) OnChatUpdate(func(context.Context, *gomax.Chat) error)
    func (c *gomax.Client) OnPresence(func(context.Context, *gomax.PresenceEvent) error)
    func (c *gomax.Client) OnTyping(func(context.Context, *gomax.TypingEvent) error)
    func (c *gomax.Client) OnDisconnect(func(context.Context, error))
    func (c *gomax.Client) OnRaw(func(context.Context, *types.RawEvent) error)

У `WebClient` сигнатуры те же, только получатель `*gomax.WebClient`. У мобильного `Client` дополнительно:

    func (c *gomax.Client) CallsSeed() int64
    func (c *gomax.Client) GetCallsSeed() int64
    func (c *gomax.Client) GetDeviceID() string

Пример обычного старта:

    cfg := gomax.DefaultConfig()
    cfg.Phone = "+79990000000"
    cfg.SessionName = "max-session.json"

    client := gomax.NewClient(cfg)
    client.OnStart(func(ctx context.Context) error {
        fmt.Println("Клиент авторизован")
        return nil
    })

    if err := client.Start(context.Background()); err != nil {
        log.Fatal(err)
    }

## client.Messages

    func (s *messages.MessageService) SendMessage(ctx context.Context, chatID int64, text string, replyToMsgID int64, attaches []types.Attachment) (*types.Message, error)
    func (s *messages.MessageService) GetChatHistory(ctx context.Context, chatID int64, fromTime int64, count int) ([]types.Message, error)
    func (s *messages.MessageService) GetHistory(ctx context.Context, chatID int64, fromTime int64, count int) ([]types.Message, error)
    func (s *messages.MessageService) GetMessages(ctx context.Context, chatID int64, messageIDs []int64) ([]types.Message, error)
    func (s *messages.MessageService) GetMessage(ctx context.Context, chatID, messageID int64) (*types.Message, error)
    func (s *messages.MessageService) EditMessage(ctx context.Context, chatID int64, messageID int64, newText string) error
    func (s *messages.MessageService) DeleteMessage(ctx context.Context, chatID int64, messageID int64, forAll bool) error
    func (s *messages.MessageService) ForwardMessage(ctx context.Context, chatID int64, messageID int64, sourceChatID int64, notify bool) (*types.Message, error)
    func (s *messages.MessageService) ForwardMessages(ctx context.Context, toChatID int64, fromChatID int64, messageIDs []int64) error
    func (s *messages.MessageService) PinMessage(ctx context.Context, chatID int64, messageID int64) error
    func (s *messages.MessageService) AddReaction(ctx context.Context, chatID int64, messageID int64, reaction string) error
    func (s *messages.MessageService) RemoveReaction(ctx context.Context, chatID int64, messageID int64, reaction string) error
    func (s *messages.MessageService) GetReactions(ctx context.Context, chatID int64, messageIDs []int64) (map[int64][]types.ReactionInfo, error)
    func (s *messages.MessageService) ReadMessage(ctx context.Context, messageID int64, chatID int64) error
    func (s *messages.MessageService) ReadMessages(ctx context.Context, chatID int64, messageIDs []int64) error
    func (s *messages.MessageService) ReadChat(ctx context.Context, chatID int64, markID int64) error
    func (s *messages.MessageService) VotePoll(ctx context.Context, chatID int64, messageID int64, pollID int64, optionIDs []int) error
    func (s *messages.MessageService) GetVideoByID(ctx context.Context, chatID, messageID, videoID int64) (types.Attachment, error)
    func (s *messages.MessageService) GetFileByID(ctx context.Context, chatID, messageID, fileID int64) (types.Attachment, error)

Здесь `replyToMsgID == 0` означает обычное сообщение. `forAll == true` удаляет сообщение для всех, если сервер и права это позволяют. `GetHistory` — совместимое имя для того же сценария, что и `GetChatHistory`.

## client.Chats

    func (s *chats.ChatService) JoinChat(ctx context.Context, link string) (*types.Chat, error)
    func (s *chats.ChatService) JoinGroup(ctx context.Context, link string) (*types.Chat, error)
    func (s *chats.ChatService) JoinChannel(ctx context.Context, link string) (*types.Chat, error)
    func (s *chats.ChatService) ResolveGroupByLink(ctx context.Context, link string) (*types.Chat, error)
    func (s *chats.ChatService) CreateGroup(ctx context.Context, name string, participantIDs []int64, notify bool) (*types.Chat, error)
    func (s *chats.ChatService) InviteUsersToGroup(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error
    func (s *chats.ChatService) InviteUsersToChannel(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error
    func (s *chats.ChatService) RemoveUsersFromGroup(ctx context.Context, chatID int64, userIDs []int64, cleanMsgPeriod int) error
    func (s *chats.ChatService) LeaveChat(ctx context.Context, chatID int64) error
    func (s *chats.ChatService) LeaveGroup(ctx context.Context, chatID int64) error
    func (s *chats.ChatService) LeaveChannel(ctx context.Context, chatID int64) error
    func (s *chats.ChatService) DeleteChat(ctx context.Context, chatID int64) error
    func (s *chats.ChatService) ChangeGroupSettings(ctx context.Context, chatID int64, allCanPin bool, onlyAdminCanAdd bool) error
    func (s *chats.ChatService) ChangeGroupSettingsWithOptions(ctx context.Context, chatID int64, options map[string]bool) error
    func (s *chats.ChatService) ChangeGroupProfile(ctx context.Context, chatID int64, name string, description string, photoToken string) error
    func (s *chats.ChatService) GetChatMembers(ctx context.Context, chatID int64, count int, marker string) ([]types.Member, string, error)
    func (s *chats.ChatService) GetChatMembersPage(ctx context.Context, chatID int64, marker int, count int) ([]types.Member, int, error)
    func (s *chats.ChatService) FetchChats(ctx context.Context, count int, marker string) ([]types.Chat, string, error)
    func (s *chats.ChatService) FetchChatsFromMarker(ctx context.Context, marker int64) ([]types.Chat, error)
    func (s *chats.ChatService) GetChats(ctx context.Context, chatIDs []int64) ([]types.Chat, error)
    func (s *chats.ChatService) GetChat(ctx context.Context, chatID int64) (*types.Chat, error)
    func (s *chats.ChatService) GetChatInfo(ctx context.Context, chatID int64) (*types.Chat, error)
    func (s *chats.ChatService) GetJoinRequests(ctx context.Context, chatID int64, count int) ([]types.Member, error)
    func (s *chats.ChatService) ConfirmJoinRequests(ctx context.Context, chatID int64, userIDs []int64, showHistory bool) error
    func (s *chats.ChatService) ConfirmJoinRequest(ctx context.Context, chatID int64, userID int64, showHistory bool) error
    func (s *chats.ChatService) DeclineJoinRequests(ctx context.Context, chatID int64, userIDs []int64) error
    func (s *chats.ChatService) DeclineJoinRequest(ctx context.Context, chatID int64, userID int64) error
    func (s *chats.ChatService) AddAdmin(ctx context.Context, chatID int64, userID int64, permissions int) error
    func (s *chats.ChatService) ReworkInviteLink(ctx context.Context, chatID int64) (string, error)
    func (s *chats.ChatService) ReworkInviteLinkChat(ctx context.Context, chatID int64) (*types.Chat, error)
    func (s *chats.ChatService) PublicSearch(ctx context.Context, query string, count int) ([]types.Chat, error)

`GetChatMembers` и `FetchChats` используют строковый marker. `GetChatMembersPage` и `FetchChatsFromMarker` — совместимые варианты с числовым marker. Передавайте marker из предыдущего ответа, а не придумывайте его вручную.

## client.Users

    func (s *users.UserService) GetUser(ctx context.Context, userID int64) (*types.User, error)
    func (s *users.UserService) GetCachedUser(ctx context.Context, userID int64) (*types.User, error)
    func (s *users.UserService) FetchUsers(ctx context.Context, userIDs []int64) ([]types.User, error)
    func (s *users.UserService) GetUsers(ctx context.Context, userIDs []int64) ([]types.User, error)
    func (s *users.UserService) SearchUsers(ctx context.Context, query string) ([]types.User, error)
    func (s *users.UserService) SearchByPhone(ctx context.Context, phone string) (*types.User, error)
    func (s *users.UserService) GetUserByPhone(ctx context.Context, phone string) (*types.User, error)
    func (s *users.UserService) GetContacts(ctx context.Context) ([]types.User, error)
    func (s *users.UserService) GetSelf(ctx context.Context) (*types.User, error)
    func (s *users.UserService) GetActiveSessions(ctx context.Context) ([]users.SessionItem, error)
    func (s *users.UserService) GetSessions(ctx context.Context) ([]users.SessionItem, error)
    func (s *users.UserService) CloseSession(ctx context.Context, sessionID int64) error
    func (s *users.UserService) Set2FA(ctx context.Context, password string, hint string, email string) error
    func (s *users.UserService) AddContact(ctx context.Context, userID int64, firstName string, lastName string, phone string) error
    func (s *users.UserService) AddContactByID(ctx context.Context, contactID int64) (*types.User, error)
    func (s *users.UserService) UpdateContact(ctx context.Context, userID int64, firstName string, lastName string) error
    func (s *users.UserService) RemoveContact(ctx context.Context, contactID int64) error
    func (s *users.UserService) ImportContacts(ctx context.Context, contacts map[string]string) ([]types.User, error)
    func (s *users.UserService) GetChatID(ctx context.Context, firstUserID int64, secondUserID int64) int64

`GetCachedUser`, `FetchUsers` и `SearchByPhone` сохранены как совместимые имена PyMax. `GetChatID` локальный и не выполняет сетевой запрос.

## client.Uploads

    func (s *uploads.UploadService) UploadPhoto(ctx context.Context, data []byte, fileName string) (*types.Attachment, error)
    func (s *uploads.UploadService) UploadPhotoWithOptions(ctx context.Context, data []byte, fileName string, profile bool) (*types.Attachment, error)
    func (s *uploads.UploadService) UploadVideo(ctx context.Context, data []byte, fileName string, duration int) (*types.Attachment, error)
    func (s *uploads.UploadService) UploadVoice(ctx context.Context, data []byte, duration int) (*types.Attachment, error)
    func (s *uploads.UploadService) UploadFile(ctx context.Context, data []byte, fileName string) (*types.Attachment, error)
    func (s *uploads.UploadService) NotifyReady(payload map[string]interface{})
    func (s *uploads.UploadService) NotifyVideoReady(id int64)
    func (s *uploads.UploadService) NotifyVoiceReady(id int64)
    func (s *uploads.UploadService) NotifyFileReady(id int64)
    func uploads.DecodeThumbhash(value string) ([]byte, error)

`NotifyReady`/`Notify*Ready` — внутренние точки доставки push-событий для завершения обработки медиа. В обычном приложении вызывайте `UploadPhoto`, `UploadVideo`, `UploadVoice` или `UploadFile`, а не notify-методы. Для `UploadPhotoWithOptions` параметр `profile == true` означает загрузку для аватара профиля.

## client.Self

    func (s *selfapi.SelfService) GetSelf(ctx context.Context) (*types.User, error)
    func (s *selfapi.SelfService) ChangeProfile(ctx context.Context, firstName string, lastName string, description string, photoToken string) error
    func (s *selfapi.SelfService) RequestProfilePhotoUploadURL(ctx context.Context) (string, error)
    func (s *selfapi.SelfService) SetPresence(online bool)
    func (s *selfapi.SelfService) ChangeProfileSettings(ctx context.Context, settings map[string]interface{}) error
    func (s *selfapi.SelfService) GetFolders(ctx context.Context) (*types.FolderList, error)
    func (s *selfapi.SelfService) CreateFolder(ctx context.Context, title string, chatInclude []int64) (*types.Folder, error)
    func (s *selfapi.SelfService) UpdateFolder(ctx context.Context, folderID string, title string, chatInclude []int64) (*types.Folder, error)
    func (s *selfapi.SelfService) UpdateFolderWithOptions(ctx context.Context, folderID string, title string, chatInclude []int64, filters []interface{}, options []interface{}) (*types.Folder, error)
    func (s *selfapi.SelfService) DeleteFolder(ctx context.Context, folderID string) error
    func (s *selfapi.SelfService) CloseAllSessions(ctx context.Context) error
    func (s *selfapi.SelfService) Logout(ctx context.Context) error

`SetPresence` не возвращает ошибку, потому что только меняет флаг для следующего сеанса или переподключения. Сетевые методы профиля и папок возвращают ошибку.

## client.Auth

Это низкоуровневый сервис. Для стандартной авторизации используйте `NewClient`/`NewWebClient` и готовые flow из [SMS](../authentication/sms.md) или [QR](../authentication/qr.md).

    func (s *auth.AuthService) RequestCode(ctx context.Context, phone string, mode []byte) (map[string]interface{}, error)
    func (s *auth.AuthService) SendCode(ctx context.Context, token string, code string) (map[string]interface{}, error)
    func (s *auth.AuthService) CheckPassword(ctx context.Context, trackID string, password string) (map[string]interface{}, error)
    func (s *auth.AuthService) RequestQR(ctx context.Context) (map[string]interface{}, error)
    func (s *auth.AuthService) CheckQR(ctx context.Context, trackID string) (map[string]interface{}, error)
    func (s *auth.AuthService) ConfirmQR(ctx context.Context, trackID string) (map[string]interface{}, error)
    func (s *auth.AuthService) ApproveQR(ctx context.Context, qrLink string) error
    func (s *auth.AuthService) CreateAuthTrack(ctx context.Context) (string, error)
    func (s *auth.AuthService) SetPassword(ctx context.Context, trackID string, password string) error
    func (s *auth.AuthService) SetHint(ctx context.Context, trackID string, hint string) error
    func (s *auth.AuthService) RequestEmailCode(ctx context.Context, trackID string, email string) error
    func (s *auth.AuthService) VerifyEmailCode(ctx context.Context, trackID string, code string) error
    func (s *auth.AuthService) CommitTwoFactor(ctx context.Context, trackID string, password string, hint string, capabilities []string) error
    func (s *auth.AuthService) ConfirmRegistration(ctx context.Context, firstName string, lastName string, token string) (map[string]interface{}, error)

Вызовы возвращают сырые `map[string]interface{}`, потому что это слой совместимости с протоколом PyMax. Для прикладного кода предпочтительнее готовый flow.

## client.Bots

    func (s *bots.BotsService) GetInitData(ctx context.Context, botID int64, chatID int64, startParam string) (*types.InitData, error)

## Auth providers и готовые flow

    type gomax.CodeProvider interface {
        GetCode(context.Context) (string, error)
    }

    type gomax.PasswordProvider interface {
        GetPassword(context.Context) (string, error)
    }

    type gomax.PasswordProviderHint interface {
        GetPasswordWithHint(context.Context, string) (string, error)
    }

    type gomax.QrHandler interface {
        HandleQr(context.Context, string) error
    }

    func auth.NewSmsAuthFlow(codeProvider auth.CodeProvider, passwordProvider auth.PasswordProvider) *auth.SmsAuthFlow
    func (f *auth.SmsAuthFlow) Authenticate(ctx context.Context, invoker api.Invoker, phone string) (*auth.AuthResult, error)

    func auth.NewQrAuthFlow(handler auth.QrHandler, passwordProvider auth.PasswordProvider) *auth.QrAuthFlow
    func (f *auth.QrAuthFlow) Authenticate(ctx context.Context, invoker api.Invoker) (*auth.AuthResult, error)

Готовые консольные реализации:

    var codeProvider auth.ConsoleCodeProvider
    var passwordProvider auth.ConsolePasswordProvider
    var qrHandler auth.ConsoleQrHandler

Обычно их вручную создавать не нужно: передайте `nil` в `NewSmsAuthFlow` или `NewQrAuthFlow`, и будут включены консольные провайдеры по умолчанию.

Точные методы консольных реализаций:

    func (p *auth.ConsoleCodeProvider) GetCode(ctx context.Context) (string, error)
    func (p *auth.ConsolePasswordProvider) GetPassword(ctx context.Context) (string, error)
    func (p *auth.ConsolePasswordProvider) GetPasswordWithHint(ctx context.Context, hint string) (string, error)
    func (h *auth.ConsoleQrHandler) HandleQr(ctx context.Context, qrURL string) error

Для собственного ввода реализуйте соответствующий интерфейс. QR-handler получает готовую ссылку и сам решает, показать её в терминале, сохранить в файл или отдать в UI.

## Session stores

    func session.NewInMemoryStore() *session.InMemoryStore
    func session.NewFileStore(workDir string, sessionName string) *session.FileStore
    func session.NewSqliteStore(db *sql.DB) (*session.SqliteStore, error)

Минимальный интерфейс для собственной реализации:

    type session.Store interface {
        SaveSession(*session.SessionInfo) error
        LoadSession() (*session.SessionInfo, error)
        UpdateToken(phone string, newToken string) error
    }

Расширенный встроенный интерфейс также содержит:

    LoadSessionByDeviceID(string) (*session.SessionInfo, error)
    LoadSessionByPhone(string) (*session.SessionInfo, error)
    DeleteSession(string) error
    DeleteAllSessions() error
    Close() error

Подключение своего хранилища:

    cfg := gomax.DefaultConfig()
    cfg.Store = session.NewInMemoryStore()
    client := gomax.NewClient(cfg)

Файловое хранилище создаётся автоматически при `PersistSession == true`. Подробнее: [файловая сессия](../session/file.md), [память](../session/memory.md), [SQLite](../session/sqlite.md).

Методы встроенных store:

    func (s *session.InMemoryStore) SaveSession(info *session.SessionInfo) error
    func (s *session.InMemoryStore) LoadSession() (*session.SessionInfo, error)
    func (s *session.InMemoryStore) UpdateToken(phone string, newToken string) error
    func (s *session.InMemoryStore) LoadSessionByDeviceID(deviceID string) (*session.SessionInfo, error)
    func (s *session.InMemoryStore) LoadSessionByPhone(phone string) (*session.SessionInfo, error)
    func (s *session.InMemoryStore) DeleteSession(token string) error
    func (s *session.InMemoryStore) DeleteAllSessions() error
    func (s *session.InMemoryStore) Close() error

    func (s *session.FileStore) SaveSession(info *session.SessionInfo) error
    func (s *session.FileStore) LoadSession() (*session.SessionInfo, error)
    func (s *session.FileStore) UpdateToken(phone string, newToken string) error
    func (s *session.FileStore) LoadSessionByDeviceID(deviceID string) (*session.SessionInfo, error)
    func (s *session.FileStore) LoadSessionByPhone(phone string) (*session.SessionInfo, error)
    func (s *session.FileStore) DeleteSession(token string) error
    func (s *session.FileStore) DeleteAllSessions() error
    func (s *session.FileStore) Close() error

    func (s *session.SqliteStore) SaveSession(info *session.SessionInfo) error
    func (s *session.SqliteStore) LoadSession() (*session.SessionInfo, error)
    func (s *session.SqliteStore) UpdateToken(phone string, newToken string) error
    func (s *session.SqliteStore) LoadSessionByDeviceID(deviceID string) (*session.SessionInfo, error)
    func (s *session.SqliteStore) LoadSessionByPhone(phone string) (*session.SessionInfo, error)
    func (s *session.SqliteStore) DeleteSession(token string) error
    func (s *session.SqliteStore) DeleteAllSessions() error
    func (s *session.SqliteStore) Close() error

## Низкоуровневые конструкторы пакетов

Эти функции нужны в основном для прямой работы с сервисами и тестовыми invoker-ами. В обычном приложении сервисы уже созданы внутри клиента:

    func auth.NewAuthService(invoker api.Invoker) *auth.AuthService
    func bots.NewBotsService(invoker api.Invoker) *bots.BotsService
    func chats.NewChatService(invoker api.Invoker) *chats.ChatService
    func messages.NewMessageService(invoker api.Invoker) *messages.MessageService
    func selfapi.NewSelfService(invoker api.Invoker) *selfapi.SelfService
    func uploads.NewUploadService(invoker api.Invoker) *uploads.UploadService
    func users.NewUserService(invoker api.Invoker) *users.UserService

## Что входит в полный справочник

Здесь перечислены все экспортируемые функции и методы, предназначенные для использования приложением: корневые конструкторы, методы `Client`/`WebClient`, сервисы, авторизация, загрузки, события и session stores.

Непубличные helper-функции (`parse...`, `handle...`, внутренние waiter-и и преобразователи) намеренно не включены: они не являются API приложения и могут меняться без обратной совместимости.

Типы и поля структур собраны отдельно: [типы и данные](types.md).


## Расширенный низкоуровневый API

Следующие функции экспортированы для расширения и сборки собственного транспорта и протокола. Для обычного бота или автоматизации они не нужны: gomax.NewClient и gomax.NewWebClient создают всё сами.

Дополнительные экспортируемые типы:

    type gomax.NonRecoverableError struct { Err error }
    func (e *gomax.NonRecoverableError) Error() string
    func (e *gomax.NonRecoverableError) Unwrap() error

    type api.Invoker interface {
        Invoke(context.Context, protocol.Opcode, interface{}) (map[string]interface{}, error)
    }

Client реализует api.Invoker. NonRecoverableError используется самим клиентом для ошибок, после которых бессмысленно автоматически переподключаться.

### dispatch.Router

    func dispatch.NewRouter() *dispatch.Router
    func (r *dispatch.Router) OnMessage(handler dispatch.MessageHandler, filters ...dispatch.MessagePredicate)
    func (r *dispatch.Router) OnMessageEdit(handler dispatch.MessageEditHandler)
    func (r *dispatch.Router) OnMessageDelete(handler dispatch.MessageDeleteHandler)
    func (r *dispatch.Router) OnMessageRead(handler dispatch.MessageReadHandler)
    func (r *dispatch.Router) OnUserUpdate(handler dispatch.UserUpdateHandler)
    func (r *dispatch.Router) OnReaction(handler dispatch.ReactionHandler)
    func (r *dispatch.Router) OnChatUpdate(handler dispatch.ChatUpdateHandler)
    func (r *dispatch.Router) OnPresence(handler dispatch.PresenceHandler)
    func (r *dispatch.Router) OnTyping(handler dispatch.TypingHandler)
    func (r *dispatch.Router) OnDisconnect(handler dispatch.DisconnectHandler)
    func (r *dispatch.Router) OnStart(handler dispatch.StartHandler)
    func (r *dispatch.Router) OnEvent(handler dispatch.EventHandler)
    func (r *dispatch.Router) DispatchMessage(ctx context.Context, msg *types.Message)
    func (r *dispatch.Router) DispatchMessageEdit(ctx context.Context, msg *types.Message)
    func (r *dispatch.Router) DispatchMessageDelete(ctx context.Context, chatID int64, messageID int64)
    func (r *dispatch.Router) DispatchMessageRead(ctx context.Context, event *types.MessageReadEvent)
    func (r *dispatch.Router) DispatchUserUpdate(ctx context.Context, event *types.UserUpdateEvent)
    func (r *dispatch.Router) DispatchReaction(ctx context.Context, event *types.ReactionEvent)
    func (r *dispatch.Router) DispatchChatUpdate(ctx context.Context, chat *types.Chat)
    func (r *dispatch.Router) DispatchPresence(ctx context.Context, event *types.PresenceEvent)
    func (r *dispatch.Router) DispatchTyping(ctx context.Context, event *types.TypingEvent)
    func (r *dispatch.Router) DispatchDisconnect(ctx context.Context, err error)
    func (r *dispatch.Router) DispatchStart(ctx context.Context)
    func (r *dispatch.Router) DispatchEvent(ctx context.Context, event *types.RawEvent)

Методы Dispatch* предназначены для собственного event loop. В прикладном коде используйте client.On*.

### protocol

    func protocol.NewTcpProtocol() (*protocol.TcpProtocol, error)
    func (p *protocol.TcpProtocol) Version() uint8
    func (p *protocol.TcpProtocol) Encode(frame *protocol.OutboundFrame) ([]byte, error)
    func (p *protocol.TcpProtocol) Decode(raw []byte) (*protocol.InboundFrame, error)

    func protocol.NewWsProtocol(binary bool) (*protocol.WsProtocol, error)
    func (p *protocol.WsProtocol) Version() uint8
    func (p *protocol.WsProtocol) Encode(frame *protocol.OutboundFrame) ([]byte, error)
    func (p *protocol.WsProtocol) Decode(raw []byte) (*protocol.InboundFrame, error)

    func protocol.NewMsgpackCodec() *protocol.MsgpackCodec
    func (c *protocol.MsgpackCodec) Encode(payload any) ([]byte, error)
    func (c *protocol.MsgpackCodec) Decode(data []byte) (any, error)
    func (c *protocol.MsgpackCodec) Normalize(value any) any
    func (c *protocol.MsgpackCodec) NormalizeKey(key any) string
    func (e *protocol.Ext) EncodeMsgpack(enc *msgpack.Encoder) error

    func protocol.NewPayloadDecoder(codec *protocol.MsgpackCodec) (*protocol.PayloadDecoder, error)
    func (d *protocol.PayloadDecoder) Decode(data []byte, flags uint8) (map[string]any, error)

    func protocol.DecodeHeader(data []byte) (*protocol.Header, error)
    func protocol.ExtractPayloadLen(headerData []byte) (uint32, error)
    func (h *protocol.Header) Encode() []byte
    func (h *protocol.Header) EncodeTo(dst []byte) error

    func protocol.NewRequest(opcode protocol.Opcode, seq uint16, payload any) *protocol.OutboundFrame
    func protocol.NewEvent(opcode protocol.Opcode, seq uint16, payload any) *protocol.OutboundFrame
    func (f *protocol.InboundFrame) IsResponse() bool
    func (f *protocol.InboundFrame) IsEvent() bool
    func (f *protocol.InboundFrame) IsError() bool
    func (f *protocol.InboundFrame) ErrorString() string
    func (e protocol.Command) String() string
    func (e protocol.Command) IsValid() bool
    func (o protocol.Opcode) String() string
    func (o protocol.Opcode) IsNotification() bool
    func protocol.CountOpcodes() int
    func protocol.NewApiError(frame *protocol.InboundFrame) *protocol.ApiError
    func (e *protocol.ApiError) Error() string

Сжатие payload:

    func protocol.NewLZ4BlockDecompressor() *protocol.LZ4BlockDecompressor
    func (d *protocol.LZ4BlockDecompressor) Decompress(src []byte, maxOutput int) ([]byte, error)
    func protocol.NewZstdDecompressor() (*protocol.ZstdDecompressor, error)
    func (d *protocol.ZstdDecompressor) Decompress(src []byte, maxOutput int) ([]byte, error)

Функции протокола нужны только при работе с wire-уровнем. Формат кадров и опкоды описаны в протоколе: ../protocol/wire.md.

### connection

    func connection.DefaultConfig() connection.Config
    func connection.NewConnectionManager(reader connection.Reader, transport transport.Transport, protocol protocol.Protocol, cfg *connection.Config, onClose func(error), onEvent func(*protocol.InboundFrame)) *connection.ConnectionManager
    func (m *connection.ConnectionManager) Start(ctx context.Context) error
    func (m *connection.ConnectionManager) Close() error
    func (m *connection.ConnectionManager) Fail(err error)
    func (m *connection.ConnectionManager) WaitClosed() error
    func (m *connection.ConnectionManager) IsOpen() bool
    func (m *connection.ConnectionManager) Events() <-chan *protocol.InboundFrame
    func (m *connection.ConnectionManager) Send(ctx context.Context, frame *protocol.OutboundFrame) error
    func (m *connection.ConnectionManager) SendEvent(ctx context.Context, opcode protocol.Opcode, payload any) error
    func (m *connection.ConnectionManager) SendRequest(ctx context.Context, opcode protocol.Opcode, payload any) (*protocol.InboundFrame, error)
    func (m *connection.ConnectionManager) Handshake(ctx context.Context, payload any) (*protocol.InboundFrame, error)

    func connection.NewSeqGenerator() *connection.SeqGenerator
    func connection.NewSeqGeneratorWithStart(start uint16) *connection.SeqGenerator
    func (s *connection.SeqGenerator) Next() uint16
    func (s *connection.SeqGenerator) Current() uint16
    func (s *connection.SeqGenerator) Reset(value uint16)

    func connection.NewPendingTracker() *connection.PendingTracker
    func (p *connection.PendingTracker) Create(seq uint16) chan *protocol.InboundFrame
    func (p *connection.PendingTracker) Resolve(seq uint16, frame *protocol.InboundFrame) bool
    func (p *connection.PendingTracker) Discard(seq uint16)
    func (p *connection.PendingTracker) CancelAll()
    func (p *connection.PendingTracker) Count() int
    func (p *connection.PendingTracker) Has(seq uint16) bool

    func connection.NewTCPReader(t transport.Transport) *connection.TCPReader
    func (r *connection.TCPReader) ReadFrame() ([]byte, error)
    func connection.NewWSReader(t transport.Transport) *connection.WSReader
    func (r *connection.WSReader) ReadFrame() ([]byte, error)

### transport

    func transport.DefaultTCPOptions() transport.TCPOptions
    func transport.NewTCPTransport(opts transport.TCPOptions) *transport.TCPTransport
    func (t *transport.TCPTransport) Connect(ctx context.Context) error
    func (t *transport.TCPTransport) Send(data []byte) error
    func (t *transport.TCPTransport) Recv(n int) ([]byte, error)
    func (t *transport.TCPTransport) Connected() bool
    func (t *transport.TCPTransport) Close() error

    func transport.DefaultWSOptions(wsURL string) transport.WSOptions
    func transport.NewWebSocketTransport(opts transport.WSOptions) *transport.WebSocketTransport
    func (w *transport.WebSocketTransport) Connect(ctx context.Context) error
    func (w *transport.WebSocketTransport) Send(data []byte) error
    func (w *transport.WebSocketTransport) Recv(n int) ([]byte, error)
    func (w *transport.WebSocketTransport) Connected() bool
    func (w *transport.WebSocketTransport) Close() error

    func transport.GetEmbeddedRootCAPEM() []byte
    func transport.GetRootCACertPool() (*x509.CertPool, error)
    func transport.NewTLSConfig(serverName string) (*tls.Config, error)

TCPTransport работает поверх TCP/TLS, WebSocketTransport — поверх WebSocket. Эти типы реализуют общий transport.Transport.

### fingerprint

    func fingerprint.DefaultFingerprint() *fingerprint.ApkBuildFingerprint
    func fingerprint.NewFingerprintGenerator(data *fingerprint.ApkBuildFingerprint) *fingerprint.FingerprintGenerator
    func (g *fingerprint.FingerprintGenerator) GenerateFingerprint(deviceID string, callsSeed int64, arch string) ([]byte, error)

Генерация fingerprint обычно выполняется автоматически во время SMS flow.

## Имена-совместимости с PyMax

В Go API оставлены дополнительные названия, которые встречаются в PyMax: GetHistory, FetchUsers, GetCachedUser, SearchByPhone, GetSessions, GetActiveSessions, JoinGroup, JoinChannel, LeaveGroup, LeaveChannel, UpdateFolderWithOptions и другие. Они перечислены в разделах выше и не требуют отдельного клиента.

## Итог по покрытию

Документация теперь разделена на три уровня:

1. Быстрый старт — что установить и какой минимальный код написать.
2. [Практические разделы](messages.md) — как выполнять конкретные задачи с примерами.
3. Этот файл и типы — полный список функций, точные сигнатуры, аргументы, возвращаемые значения и структуры данных.
