# История сообщений

Сценарий для получения истории и чтения сообщений. Все примеры — фрагменты:
`client`, `ctx` и идентификаторы уже созданы. Полные методы: [Messages API](../api/messages.md).

## Получить последние сообщения

`GetChatHistory` возвращает до `count` сообщений. `fromTime == 0` означает
текущую точку синхронизации.

```go
history, err := client.Messages.GetChatHistory(ctx, chatID, 0, 50)
if err != nil {
    return err
}
for _, msg := range history {
    fmt.Printf("%d %d: %s\n", msg.ID, msg.SenderID, msg.Text)
}
```

## Получить историю от времени

`fromTime` — Unix-время в миллисекундах.

```go
fromTime := time.Now().Add(-24 * time.Hour).UnixMilli()
history, err := client.Messages.GetChatHistory(ctx, chatID, fromTime, 100)
```

## Получить сообщения по ID

```go
messages, err := client.Messages.GetMessages(ctx, chatID,
    []int64{101, 102, 103})
one, err := client.Messages.GetMessage(ctx, chatID, 101)
```

## Пометить историю прочитанной

```go
ids := make([]int64, 0, len(history))
for _, msg := range history {
    ids = append(ids, msg.ID)
}
err := client.Messages.ReadMessages(ctx, chatID, ids)
```

Для marker всего чата используйте `ReadChat`:

```go
err := client.Messages.ReadChat(ctx, chatID, markID)
```

`GetHistory` — совместимое имя `GetChatHistory`.
