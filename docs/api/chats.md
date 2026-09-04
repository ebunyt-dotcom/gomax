# Чаты и каналы

Сервис доступен как `client.Chats`.

## Методы

| Метод | Назначение |
|---|---|
| `GetChat`, `GetChatInfo`, `GetChats` | Получить информацию о чатах |
| `FetchChats`, `FetchChatsFromMarker` | Получить список диалогов |
| `JoinChat`, `JoinGroup`, `JoinChannel` | Вступить по ссылке |
| `ResolveGroupByLink` | Проверить ссылку без вступления |
| `CreateGroup` | Создать группу |
| `InviteUsersToGroup/Channel` | Добавить участников |
| `RemoveUsersFromGroup` | Удалить участников |
| `GetChatMembers`, `GetChatMembersPage` | Получить участников с пагинацией |
| `GetJoinRequests` | Получить заявки на вступление |
| `ConfirmJoinRequest(s)` | Одобрить заявки |
| `DeclineJoinRequest(s)` | Отклонить заявки |
| `ChangeGroupSettings` | Изменить основные права |
| `ChangeGroupSettingsWithOptions` | Передать набор options |
| `ChangeGroupProfile` | Изменить название, описание, аватар |
| `ReworkInviteLink` | Создать новую invite-ссылку |
| `LeaveChat`, `LeaveGroup`, `LeaveChannel` | Выйти |
| `DeleteChat` | Удалить диалог |
| `AddAdmin` | Выдать права администратора |
| `PublicSearch` | Искать публичные чаты |

## Получение чата

```go
chat, err := client.Chats.GetChat(ctx, chatID)
if err != nil { return err }
fmt.Printf("%s (%d)\n", chat.Title, chat.ID)
```

## Вступление по ссылке

```go
chat, err := client.Chats.JoinChat(ctx, "https://max.ru/join/abc123")
```

`JoinChat` принимает полную ссылку или её часть `join/...`. `ResolveGroupByLink` только проверяет ссылку.

## Участники

```go
members, next, err := client.Chats.GetChatMembers(ctx, chatID, 50, "")
if err != nil { return err }
_ = members
_ = next
```

Передайте возвращённый `next` в следующий вызов. Для API-варианта с числовым marker используйте `GetChatMembersPage`.

## Заявки на вступление

```go
requests, err := client.Chats.GetJoinRequests(ctx, chatID, 100)
err = client.Chats.ConfirmJoinRequest(ctx, chatID, requests[0].UserID, true)
err = client.Chats.DeclineJoinRequest(ctx, chatID, requests[0].UserID)
```

Вызовы требуют прав администратора и зависят от настроек конкретного чата.

## Настройки и профиль

```go
err := client.Chats.ChangeGroupSettings(ctx, chatID, false, true)
err = client.Chats.ChangeGroupProfile(ctx, chatID, "Новое имя", "Описание", "photo-token")
```

Для частичного набора параметров используйте `ChangeGroupSettingsWithOptions`.
