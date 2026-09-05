# Пакетное прочтение

`ReadMessages` помечает переданные сообщения прочитанными, а `ReadChat` — чат до указанного marker.

## Пример

```go
history, err := client.Messages.GetChatHistory(ctx, chatID, 0, 50)
if err != nil {
    return err
}

ids := make([]int64, 0, len(history))
for _, msg := range history {
    ids = append(ids, msg.ID)
}
if err := client.Messages.ReadMessages(ctx, chatID, ids); err != nil {
    return err
}
```

## Ограничения и рекомендации

Используйте это только для обычной синхронизации своего аккаунта и учитывайте
правила сервиса.
