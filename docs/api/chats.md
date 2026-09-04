# Сервис чатов и каналов (`client.Chats`)

Сервис `ChatService` управляет групповыми чатами, каналами и личными беседами: вступлением по инвайт-ссылкам, массовым инвайтингом пользователей, созданием групп, исключением участников, управлением правами и пагинацией списков диалогов.

Доступ к сервису осуществляется через поле `client.Chats` (или `webClient.Chats`).

---

## 📋 Список методов

1. [`JoinChat`](#1-joinchat) — вступление в чат или канал по ссылке-приглашению
2. [`InviteUsersToGroup`](#2-inviteuserstogroup) — добавление пользователей в группу (инвайтинг)
3. [`InviteUsersToChannel`](#3-inviteuserstochannel) — добавление пользователей в канал
4. [`RemoveUsersFromGroup`](#4-removeusersfromgroup) — исключение пользователей из группы (кик)
5. [`CreateGroup`](#5-creategroup) — создание нового группового чата
6. [`LeaveChat`](#6-leavechat) — выход из чата или канала
7. [`DeleteChat`](#7-deletechat) — полное удаление беседы / диалога
8. [`ChangeGroupSettings`](#8-changegroupsettings) — изменение прав и настроек группы
9. [`GetChatMembers`](#9-getchatmembers) — постраничное получение участников чата
10. [`FetchChats`](#10-fetchchats) — получение списка активных диалогов и чатов
11. [`ReworkInviteLink`](#11-reworkinvitelink) — перегенерация (сброс) ссылки-приглашения

---

### 1. `JoinChat`

Выполняет вступление аккаунта в публичный или приватный чат/канал по ссылке-приглашению или хэшу ссылки.

```go
func (s *ChatService) JoinChat(
    ctx context.Context,
    link string,
) (*types.Chat, error)
```

* **Опкод протокола**: `OpChatJoin` (57).
* **Параметры**:
  * `ctx` (`context.Context`): контекст выполнения запроса.
  * `link` (`string`): ссылка-приглашение (например `"https://max.ru/join/aBcDeFg123"` или `"join/aBcDeFg123"`). Метод автоматически нормализует префикс ссылки.
* **Возвращаемое значение**: `(*types.Chat, error)` — объект чата, содержащий присвоенный `ID`, `Title` и параметры.

#### Пример вызова:
```go
chat, err := client.Chats.JoinChat(ctx, "https://max.ru/join/xK9L2pQ7")
if err != nil {
    log.Fatalf("Не удалось вступить в чат: %v", err)
}
fmt.Printf("✅ Успешно вступили в чат: %s (ID: %d)\n", chat.Title, chat.ID)
```

---

### 2. `InviteUsersToGroup`

Добавляет список пользователей в группу по их `userID`. Данный метод используется в сценариях пакетного инвайтинга (привлечения аудитории).

```go
func (s *ChatService) InviteUsersToGroup(
    ctx context.Context,
    chatID int64,
    userIDs []int64,
    showHistory bool,
) error
```

* **Опкод протокола**: `OpChatMembers` (59) с операцией `"ADD"`.
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор группы.
  * `userIDs` (`[]int64`): массив идентификаторов пользователей для добавления.
  * `showHistory` (`bool`): открывать ли новым участникам историю предыдущих сообщений чата.
* **Возвращаемое значение**: `error` — `nil` при успешном добавлении.

#### Пример вызова:
```go
targets := []int64{55123401, 55123402, 55123403}
err := client.Chats.InviteUsersToGroup(ctx, groupID, targets, true)
if err != nil {
    log.Printf("Ошибка инвайтинга: %v", err)
}
```

---

### 3. `InviteUsersToChannel`

Добавляет пользователей в канал. Псевдоним метода `InviteUsersToGroup` с аналогичной сигнатурой для соответствия API PyMax.

```go
func (s *ChatService) InviteUsersToChannel(
    ctx context.Context,
    chatID int64,
    userIDs []int64,
    showHistory bool,
) error
```

---

### 4. `RemoveUsersFromGroup`

Удаляет (кикает) участников из группы с возможностью зачистки их сообщений за определенный период.

```go
func (s *ChatService) RemoveUsersFromGroup(
    ctx context.Context,
    chatID int64,
    userIDs []int64,
    cleanMsgPeriod int,
) error
```

* **Опкод протокола**: `OpChatMembers` (59) с операцией `"REMOVE"`.
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `userIDs` (`[]int64`): массив исключаемых пользователей.
  * `cleanMsgPeriod` (`int`): период в секундах, за который необходимо удалить сообщения удаляемых участников (например `86400` — за сутки; `0` — не удалять сообщения).
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
// Исключить нарушителя и стереть его сообщения за последние 24 часа
err := client.Chats.RemoveUsersFromGroup(ctx, chatID, []int64{spammerID}, 86400)
```

---

### 5. `CreateGroup`

Создает новую групповую беседу с указанным названием и стартовым составом участников.

```go
func (s *ChatService) CreateGroup(
    ctx context.Context,
    name string,
    participantIDs []int64,
    notify bool,
) (*types.Chat, error)
```

* **Опкод протокола**: `OpChatCreate` (63).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `name` (`string`): название создаваемой группы.
  * `participantIDs` (`[]int64`): список ID пользователей, приглашаемых при создании.
  * `notify` (`bool`): отправлять ли уведомления приглашенным пользователям.
* **Возвращаемое значение**: `(*types.Chat, error)` — созданный объект чата с заполненным `ID`.

#### Пример вызова:
```go
chat, err := client.Chats.CreateGroup(ctx, "VIP Клуб GoMax", []int64{user1, user2}, true)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Создан чат: %s (ID: %d)\n", chat.Title, chat.ID)
```

---

### 6. `LeaveChat`

Осуществляет добровольный выход текущего аккаунта из группы или канала.

```go
func (s *ChatService) LeaveChat(
    ctx context.Context,
    chatID int64,
) error
```

* **Опкод протокола**: `OpChatLeave` (58).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
err := client.Chats.LeaveChat(ctx, oldChatID)
```

---

### 7. `DeleteChat`

Полностью удаляет диалог или канал (для создателя) либо очищает историю диалога в списке чатов.

```go
func (s *ChatService) DeleteChat(
    ctx context.Context,
    chatID int64,
) error
```

* **Опкод протокола**: `OpChatDelete` (52).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
err := client.Chats.DeleteChat(ctx, tempChatID)
```

---

### 8. `ChangeGroupSettings`

Обновляет глобальные права и настройки поведения группы.

```go
func (s *ChatService) ChangeGroupSettings(
    ctx context.Context,
    chatID int64,
    allCanPin bool,
    onlyAdminCanAdd bool,
) error
```

* **Опкод протокола**: `OpChatUpdate` (55).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `allCanPin` (`bool`): разрешить ли обычным участникам закреплять сообщения (`true` — все, `false` — только администраторы).
  * `onlyAdminCanAdd` (`bool`): ограничить ли добавление участников только администраторами (`true` — только админы, `false` — все участники).
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
// Закреплять могут только админы, инвайтить могут все
err := client.Chats.ChangeGroupSettings(ctx, chatID, false, false)
```

---

### 9. `GetChatMembers`

Возвращает список участников группы или канала с поддержкой курсорной постраничной пагинации.

```go
func (s *ChatService) GetChatMembers(
    ctx context.Context,
    chatID int64,
    count int,
    marker string,
) ([]types.Member, string, error)
```

* **Опкод протокола**: `OpChatMembers` (59).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `count` (`int`): количество участников на одну страницу (по умолчанию 50, если передано `<= 0`).
  * `marker` (`string`): курсор пагинации для получения следующей страницы (передайте `""` для первой страницы).
* **Возвращаемое значение**:
  * `[]types.Member`: список объектов участников (содержит `UserID`, `Role` и др.).
  * `string`: маркер для следующей страницы (`""`, если все участники получены).
  * `error`: ошибка выполнения запроса.

#### Пример вызова (полный обход участников):
```go
var marker string
var allMembers []gomax.Member

for {
    members, nextMarker, err := client.Chats.GetChatMembers(ctx, chatID, 100, marker)
    if err != nil {
        log.Fatal(err)
    }
    allMembers = append(allMembers, members...)
    if nextMarker == "" {
        break // Достигнут конец списка
    }
    marker = nextMarker
}
fmt.Printf("Всего участников в чате: %d\n", len(allMembers))
```

---

### 10. `FetchChats`

Получает постраничный список активных диалогов, групп и каналов пользователя.

```go
func (s *ChatService) FetchChats(
    ctx context.Context,
    count int,
    marker string,
) ([]types.Chat, string, error)
```

* **Опкод протокола**: `OpChatsList` (53).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `count` (`int`): размер порции чатов (по умолчанию 40).
  * `marker` (`string`): курсор пагинации (для первой страницы `""`).
* **Возвращаемое значение**: `([]types.Chat, string, error)` — список чатов, следующий маркер и ошибка.

#### Пример вызова:
```go
chats, nextMarker, err := client.Chats.FetchChats(ctx, 50, "")
if err != nil {
    log.Fatal(err)
}
for _, c := range chats {
    fmt.Printf("Чат: %s [ID: %d]\n", c.Title, c.ID)
}
```

---

### 11. `ReworkInviteLink`

Аннулирует текущую инвайт-ссылку и генерирует новую постоянную ссылку-приглашение в чат.

```go
func (s *ChatService) ReworkInviteLink(
    ctx context.Context,
    chatID int64,
) (string, error)
```

* **Опкод протокола**: `OpChatCheckLink` (56).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
* **Возвращаемое значение**: `(string, error)` — обновленная ссылка-приглашение.

#### Пример вызова:
```go
newLink, err := client.Chats.ReworkInviteLink(ctx, chatID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Новая ссылка на чат: %s\n", newLink)
```

---

### 12. `JoinGroup` и `JoinChannel`

Специализированные псевдонимы для `JoinChat` с аналогичным поведением и автоматической нормализацией ссылок.

```go
func (s *ChatService) JoinGroup(ctx context.Context, link string) (*types.Chat, error)
func (s *ChatService) JoinChannel(ctx context.Context, link string) (*types.Chat, error)
```

---

### 13. `ResolveGroupByLink`

Позволяет проверить валидность ссылки-приглашения и получить информацию о группе/канале (название, аватар) **без фактического вступления** в чат.

```go
func (s *ChatService) ResolveGroupByLink(ctx context.Context, link string) (*types.Chat, error)
```

* **Опкод протокола**: `OpChatCheckLink` (56).

---

### 14. `GetChats` и `GetChat`

Получает полную информацию о чатах по их идентификаторам.

```go
func (s *ChatService) GetChats(ctx context.Context, chatIDs []int64) ([]types.Chat, error)
func (s *ChatService) GetChat(ctx context.Context, chatID int64) (*types.Chat, error)
```

* **Опкод протокола**: `OpChatInfo` (48).

---

### 15. `ChangeGroupProfile`

Обновляет название, описание и аватар группы.

```go
func (s *ChatService) ChangeGroupProfile(ctx context.Context, chatID int64, name, description, photoToken string) error
```

* **Опкод протокола**: `OpChatUpdate` (55).

---

### 16. Заявки на вступление (`JoinRequests`)

Управление очередью участников в закрытых каналах и чатах:

```go
// Получить список ожидающих заявок
func (s *ChatService) GetJoinRequests(ctx context.Context, chatID int64, count int) ([]types.Member, error)

// Одобрить заявки
func (s *ChatService) ConfirmJoinRequests(ctx context.Context, chatID int64, userIDs []int64) error

// Отклонить заявки
func (s *ChatService) DeclineJoinRequests(ctx context.Context, chatID int64, userIDs []int64) error
```

* **Опкод протокола**: `OpChatMembers` (59).
