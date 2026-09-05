# Реакции

Сценарий для добавления, удаления и чтения реакций. Все примеры — фрагменты:
`client`, `ctx` и идентификаторы уже созданы. Полные методы: [Messages API](../api/messages.md).

## Поставить реакцию

```go
err := client.Messages.AddReaction(ctx, chatID, messageID, "👍")
```

## Снять реакцию

```go
err := client.Messages.RemoveReaction(ctx, chatID, messageID, "👍")
```

## Посмотреть реакции

```go
reactions, err := client.Messages.GetReactions(ctx, chatID,
    []int64{messageID})
if err != nil {
    return err
}
for id, items := range reactions {
    fmt.Println(id, items)
}
```

## Поставить реакцию нескольким сообщениям

```go
for _, id := range messageIDs {
    if err := client.Messages.AddReaction(ctx, chatID, id, "🔥"); err != nil {
        log.Println(id, err)
    }
}
```

Ограничивайте частоту запросов и обрабатывайте каждую ошибку отдельно.
