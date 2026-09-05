# Приглашения и заявки

Сценарий для приглашений, списка участников и заявок на вступление. Все
примеры — фрагменты: `client`, `ctx`, `chatID` и идентификаторы уже созданы.
Полные методы: [Chats API](../api/chats.md).

## Пригласить в группу

`showHistory` определяет, получат ли новые участники доступ к старой истории.

```go
userIDs := []int64{1001, 1002}
err := client.Chats.InviteUsersToGroup(ctx, chatID, userIDs, true)
```

## Пригласить в канал

```go
err := client.Chats.InviteUsersToChannel(ctx, channelID, userIDs, false)
```

## Получить участников

```go
members, nextMarker, err := client.Chats.GetChatMembers(ctx, chatID, 50, "")
if err != nil {
    return err
}
_ = members
if nextMarker != "" {
    nextPage, _, err := client.Chats.GetChatMembers(ctx, chatID, 50, nextMarker)
    _ = nextPage
    _ = err
}
```

## Получить заявки

```go
requests, err := client.Chats.GetJoinRequests(ctx, chatID, 100)
```

## Одобрить или отклонить заявку

```go
err := client.Chats.ConfirmJoinRequest(ctx, chatID, userID, true)
err = client.Chats.DeclineJoinRequest(ctx, chatID, anotherUserID)
```

Для нескольких пользователей используйте `ConfirmJoinRequests` и
`DeclineJoinRequests`:

```go
err := client.Chats.ConfirmJoinRequests(ctx, chatID, userIDs, true)
err = client.Chats.DeclineJoinRequests(ctx, chatID, userIDs)
```

Операции требуют прав администратора и зависят от настроек чата.
