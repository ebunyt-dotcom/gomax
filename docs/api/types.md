# Типы и данные

В GoMax почти все методы работают с типами из пакета `github.com/ebunyt-dotcom/gomax/pkg/types`. В корневом пакете есть короткие имена-алиасы, поэтому можно писать `gomax.Message`, `gomax.Chat`, `gomax.User` и так далее.

## Основные идентификаторы

- `chatID` — ID диалога, группы или канала.
- `messageID` — ID сообщения внутри чата.
- `userID` — ID пользователя.
- `replyToMsgID` — ID сообщения, на которое отвечаем. Значение `0` означает «без ответа».
- `fromTime` — Unix-время в миллисекундах, от которого загружать историю.
- `marker` — курсор пагинации, возвращённый предыдущим запросом.

ID имеют тип `int64`. Не подставляйте имя чата вместо ID: сначала получите объект через `GetChat`, `GetChats` или `JoinChat`.

## Message

Сообщение имеет поля:

| Поле | Тип | Что означает |
|---|---|---|
| ID | int64 | ID сообщения |
| CID | int64 | внутренний client message ID |
| ChatID | int64 | чат сообщения |
| SenderID | int64 | отправитель |
| Text | string | текст |
| Time | int64 | время сообщения, Unix milliseconds |
| EditedAt | int64 | время изменения |
| ReplyToMsgID | int64 | исходное сообщение для ответа |
| Attachments | []Attachment | фото, видео, файл и т.д. |
| Reactions | []ReactionInfo | реакции |
| IsOutgoing | bool | сообщение отправлено текущим аккаунтом |
| IsPinned | bool | сообщение закреплено |
| IsDeleted | bool | сообщение удалено |

Пример:

```go
msg, err := client.Messages.GetMessage(ctx, chatID, messageID)
if err != nil {
    return err
}
fmt.Println(msg.Text, msg.SenderID)
```

## Attachment

Вложение передаётся в `SendMessage` через поле `Attachments`.

| Поле | Тип | Что означает |
|---|---|---|
| Type | AttachmentType | тип: PHOTO, VIDEO, AUDIO, FILE, VOICE, VIDEO_NOTE, POLL, STICKER |
| ID | string | ID файла или медиа |
| URL | string | URL загрузки или готового ресурса |
| FileName | string | имя файла |
| FileSize | int64 | размер |
| Duration | int | длительность аудио/видео |
| Width | int | ширина |
| Height | int | высота |
| Token | string | токен медиа |

Для нового медиафайла обычно используется такой порядок:

```go
data, err := os.ReadFile("photo.jpg")
if err != nil {
    return err
}

attachment, err := client.Uploads.UploadPhoto(ctx, data, "photo.jpg")
if err != nil {
    return err
}

_, err = client.Messages.SendMessage(ctx, chatID, "Фото", 0,
    []gomax.Attachment{*attachment})
```

Константы типов доступны и как `gomax.AttachmentPhoto`, и как `types.AttachmentPhoto`.

## Chat

| Поле | Тип | Что означает |
|---|---|---|
| ID | int64 | ID чата |
| Type | ChatType | DIALOG, CHAT или CHANNEL |
| Title | string | название |
| Description | string | описание |
| Icon | string | идентификатор/URL иконки |
| MembersCount | int | количество участников |
| OwnerID | int64 | владелец |
| CreatedAt | time.Time | время создания |
| PinnedMsgID | int64 | закреплённое сообщение |
| IsChannel | bool | это канал |
| IsPublic | bool | публичный чат |
| InviteLink | string | ссылка-приглашение |

Типы чатов:

```go
gomax.ChatTypeDialog
gomax.ChatTypeChat
gomax.ChatTypeChannel
```

## User и Member

Поля `User`:

| Поле | Тип | Что означает |
|---|---|---|
| `ID` | `int64` | ID пользователя. |
| `Phone` | `string` | Номер телефона, если доступен. |
| `FirstName` / `LastName` | `string` | Имя и фамилия. |
| `Nickname` | `string` | Никнейм. |
| `AvatarURL` | `string` | URL аватара. |
| `Bio` | `string` | Описание профиля. |
| `IsBot` | `bool` | Пользователь — бот. |
| `IsContact` | `bool` | Пользователь есть в контактах. |
| `IsMutual` | `bool` | Контакт взаимный. |
| `IsVerified` | `bool` | Профиль подтверждён. |

`Member` содержит:

| Поле | Тип | Что означает |
|---|---|---|
| UserID | int64 | пользователь |
| Role | string | ADMIN, MEMBER или OWNER |
| JoinedAt | int64 | время вступления |
| InvitedBy | int64 | кто пригласил |

## Реакции и опросы

`ReactionInfo`:

- `Reaction string` — идентификатор реакции;
- `Count int` — количество;
- `Self bool` — поставил ли её текущий аккаунт.

`PollOption` содержит `ID`, `Text`, `Votes`. `Poll` содержит `ID`, `Question`, `Options`, флаги `Multiple`, `Anonymous`, `Closed` и выбранные варианты `SelectedOpts`.

## События

Типичные обработчики:

```go
client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
    fmt.Println(msg.ChatID, msg.Text)
    return nil
})

client.OnReaction(func(ctx context.Context, ev *gomax.ReactionEvent) error {
    fmt.Println(ev.MessageID, ev.Reaction, ev.Removed)
    return nil
})

client.OnPresence(func(ctx context.Context, ev *gomax.PresenceEvent) error {
    fmt.Println(ev.UserID, ev.Online)
    return nil
})
```

Доступные структуры событий:

- `MessageReadEvent`: `ChatID`, `MessageID`, `Mark`;
- `UserUpdateEvent`: `User`;
- `ReactionEvent`: `ChatID`, `MessageID`, `UserID`, `Reaction`, `Removed`;
- `PresenceEvent`: `UserID`, `Online`;
- `TypingEvent`: `ChatID`, `UserID`;
- `RawEvent`: `Type`, `Opcode`, `Payload`;
- `VideoUploadSignal`, `FileUploadSignal`, `VoiceUploadSignal`: ID готового медиа.

Тип события можно сравнивать с константами `EventType`:

```go
types.EventMessageNew
types.EventMessageUpdate
types.EventMessageDelete
types.EventMessageRead
types.EventTyping
types.EventPresence
types.EventReaction
types.EventChatUpdate
types.EventUserUpdate
types.EventVideoReady
types.EventFileReady
types.EventVoiceReady
types.EventRaw
```

Сигналы готовности вложений содержат один идентификатор:

| Тип | Поле |
|---|---|
| `VideoUploadSignal` | `VideoID int64` |
| `FileUploadSignal` | `FileID int64` |
| `VoiceUploadSignal` | `AudioID int64` |

## Папки и Bot Web App

`Folder` содержит `ID`, `Title`, `Include []int64`. `FolderList` содержит список папок и поле `Sync`.

`InitData` возвращается методом `client.Bots.GetInitData` и содержит:

- `QueryID string`;
- `URL string`.

## Результат авторизации

`AuthResult` возвращается готовыми flow и содержит:

| Поле | Тип | Что означает |
|---|---|---|
| Token | string | токен авторизованной сессии |
| UserID | int64 | ID пользователя, если сервер его вернул |
| IsRegister | bool | нужна ли ещё регистрация профиля |

Токен уже сохраняется клиентом в выбранный session store. Не публикуйте его в логах и репозитории.

Подробные сигнатуры функций: [полный API](reference.md).
