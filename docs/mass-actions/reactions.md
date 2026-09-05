# Пакетные реакции

`AddReaction` добавляет emoji к одному сообщению. Для нескольких сообщений используйте цикл с контролем ошибок.

## Пример

```go
for _, messageID := range messageIDs {
    if err := client.Messages.AddReaction(ctx, chatID, messageID, "👍"); err != nil {
        log.Println(messageID, err)
    }
}
```

## Ограничения и рекомендации

Для удаления используйте `RemoveReaction`. Не отправляйте запросы без
ограничений: сервер может применить rate limit.
