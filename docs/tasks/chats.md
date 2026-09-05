# Чаты

Сценарий для поиска, списка, вступления и изменения чатов. Все примеры —
фрагменты: `client`, `ctx` и идентификаторы уже созданы. Полные методы:
[Chats API](../api/chats.md).

## Найти чат по ID

```go
chat, err := client.Chats.GetChat(ctx, chatID)
```

Для нескольких чатов:

```go
chats, err := client.Chats.GetChats(ctx, []int64{chatID, anotherChatID})
```

## Получить список диалогов

```go
chats, marker, err := client.Chats.FetchChats(ctx, 50, "")
for marker != "" {
    next, newMarker, err := client.Chats.FetchChats(ctx, 50, marker)
    if err != nil {
        return err
    }
    chats = append(chats, next...)
    marker = newMarker
}
```

## Вступить по ссылке

```go
chat, err := client.Chats.JoinChat(ctx, "https://max.ru/join/abc123")
```

Проверить ссылку без вступления можно через `ResolveGroupByLink`.

## Создать группу

```go
chat, err := client.Chats.CreateGroup(ctx, "Команда", []int64{userID}, true)
```

## Изменить настройки и профиль

```go
err := client.Chats.ChangeGroupSettings(ctx, chatID, false, true)
err = client.Chats.ChangeGroupProfile(ctx, chatID, "Новое имя", "Описание", "")
```

## Получить новую invite-ссылку

```go
link, err := client.Chats.ReworkInviteLink(ctx, chatID)
```

## Выйти или удалить чат

```go
err := client.Chats.LeaveChat(ctx, chatID)
err = client.Chats.DeleteChat(ctx, chatID)
```
