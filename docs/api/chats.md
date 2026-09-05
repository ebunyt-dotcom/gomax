# Чаты и каналы

Сервис: `client.Chats`. В примерах `ctx`, `chatID`, `userID` и ссылки уже
объявлены. Методы, которые меняют чат, требуют соответствующих прав.

Полные сигнатуры находятся в [полном справочнике](reference.md). Каждый
пример ниже — отдельный фрагмент; импорты и создание клиента не повторяются.

## Получение чатов

### `GetChat`

Возвращает чат по ID.

```go
chat, err := client.Chats.GetChat(ctx, chatID)
```

### `GetChatInfo`

Возвращает расширенную информацию о чате.

```go
chat, err := client.Chats.GetChatInfo(ctx, chatID)
```

### `GetChats`

Возвращает несколько чатов по ID.

```go
chats, err := client.Chats.GetChats(ctx, []int64{chatID, anotherChatID})
```

### `FetchChats`

Получает страницу диалогов. Передайте `marker` из предыдущего ответа для
следующей страницы.

```go
chats, nextMarker, err := client.Chats.FetchChats(ctx, 50, "")
```

### `FetchChatsFromMarker`

Вариант пагинации с числовым marker.

```go
chats, err := client.Chats.FetchChatsFromMarker(ctx, marker)
```

### `PublicSearch`

Ищет публичные чаты и каналы.

```go
chats, err := client.Chats.PublicSearch(ctx, "golang", 20)
```

## Вступление

### `JoinChat`

Вступает по полной ссылке или по части `join/...`.

```go
chat, err := client.Chats.JoinChat(ctx, "https://max.ru/join/abc123")
```

### `JoinGroup`

Вступает в группу по ссылке.

```go
chat, err := client.Chats.JoinGroup(ctx, "join/abc123")
```

### `JoinChannel`

Вступает в канал по ссылке.

```go
chat, err := client.Chats.JoinChannel(ctx, "join/abc123")
```

### `ResolveGroupByLink`

Проверяет ссылку и возвращает группу без вступления.

```go
chat, err := client.Chats.ResolveGroupByLink(ctx, "join/abc123")
```

## Создание и участники

### `CreateGroup`

Создаёт группу и добавляет начальных участников.

```go
chat, err := client.Chats.CreateGroup(ctx, "Команда", []int64{userID}, true)
```

### `InviteUsersToGroup`

Приглашает пользователей в группу. `showHistory` определяет доступ к истории.

```go
err := client.Chats.InviteUsersToGroup(ctx, chatID, []int64{userID}, true)
```

### `InviteUsersToChannel`

Приглашает пользователей в канал.

```go
err := client.Chats.InviteUsersToChannel(ctx, chatID, []int64{userID}, false)
```

### `RemoveUsersFromGroup`

Удаляет пользователей из группы. `cleanMsgPeriod` задаёт очистку сообщений.

```go
err := client.Chats.RemoveUsersFromGroup(ctx, chatID, []int64{userID}, 0)
```

### `GetChatMembers`

Возвращает участников и строковый marker следующей страницы.

```go
members, nextMarker, err := client.Chats.GetChatMembers(ctx, chatID, 50, "")
```

### `GetChatMembersPage`

Вариант получения участников с числовым marker.

```go
members, nextMarker, err := client.Chats.GetChatMembersPage(ctx, chatID, 0, 50)
```

### `GetJoinRequests`

Возвращает заявки на вступление в группу.

```go
requests, err := client.Chats.GetJoinRequests(ctx, chatID, 100)
```

### `ConfirmJoinRequests`

Одобряет несколько заявок.

```go
err := client.Chats.ConfirmJoinRequests(ctx, chatID, []int64{userID}, true)
```

### `ConfirmJoinRequest`

Одобряет одну заявку.

```go
err := client.Chats.ConfirmJoinRequest(ctx, chatID, userID, true)
```

### `DeclineJoinRequests`

Отклоняет несколько заявок.

```go
err := client.Chats.DeclineJoinRequests(ctx, chatID, []int64{userID})
```

### `DeclineJoinRequest`

Отклоняет одну заявку.

```go
err := client.Chats.DeclineJoinRequest(ctx, chatID, userID)
```

### `AddAdmin`

Назначает администратора. `permissions` — битовая маска прав сервера.

```go
err := client.Chats.AddAdmin(ctx, chatID, userID, permissions)
```

## Настройки и завершение

### `ChangeGroupSettings`

Изменяет базовые права группы.

```go
err := client.Chats.ChangeGroupSettings(ctx, chatID, false, true)
```

### `ChangeGroupSettingsWithOptions`

Изменяет только переданные настройки.

```go
err := client.Chats.ChangeGroupSettingsWithOptions(ctx, chatID,
    map[string]bool{"allCanPin": false})
```

### `ChangeGroupProfile`

Меняет название, описание и token аватара.

```go
err := client.Chats.ChangeGroupProfile(ctx, chatID, "Новое имя", "Описание", "")
```

### `ReworkInviteLink`

Создаёт новую invite-ссылку и возвращает её строкой.

```go
link, err := client.Chats.ReworkInviteLink(ctx, chatID)
```

### `ReworkInviteLinkChat`

Создаёт новую invite-ссылку и возвращает обновлённый чат.

```go
chat, err := client.Chats.ReworkInviteLinkChat(ctx, chatID)
```

### `LeaveChat`

Выходит из чата универсальным методом.

```go
err := client.Chats.LeaveChat(ctx, chatID)
```

### `LeaveGroup`

Выходит из группы.

```go
err := client.Chats.LeaveGroup(ctx, chatID)
```

### `LeaveChannel`

Выходит из канала.

```go
err := client.Chats.LeaveChannel(ctx, chatID)
```

### `DeleteChat`

Удаляет диалог.

```go
err := client.Chats.DeleteChat(ctx, chatID)
```
