# События

Обработчики регистрируются до `Start`. Они доступны и у `Client`, и у
`WebClient`. Ошибка обработчика не останавливает клиента.

## Методы регистрации

### `OnStart`

Вызывается после успешного входа.

```go
client.OnStart(func(ctx context.Context) error {
    fmt.Println("Вход выполнен")
    return nil
})
```

### `OnMessage`

Вызывается для нового сообщения и сообщений, полученных при синхронизации.

```go
client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
    fmt.Println(msg.ChatID, msg.Text)
    return nil
})
```

### `OnMessageEdit`

Вызывается после изменения сообщения.

```go
client.OnMessageEdit(func(ctx context.Context, msg *gomax.Message) error {
    fmt.Println("Изменено:", msg.ID, msg.Text)
    return nil
})
```

### `OnMessageDelete`

Вызывается после удаления сообщения.

```go
client.OnMessageDelete(func(ctx context.Context, chatID, messageID int64) error {
    fmt.Println("Удалено:", chatID, messageID)
    return nil
})
```

### `OnMessageRead`

Получает `MessageReadEvent` с chat ID, message ID и marker.

```go
client.OnMessageRead(func(ctx context.Context, event *types.MessageReadEvent) error {
    fmt.Println(event.ChatID, event.Mark)
    return nil
})
```

### `OnUserUpdate`

Вызывается после изменения контакта или профиля пользователя.

```go
client.OnUserUpdate(func(ctx context.Context, event *types.UserUpdateEvent) error {
    fmt.Println(event.User.ID, event.User.FirstName)
    return nil
})
```

### `OnReaction`

Получает добавление или удаление реакции.

```go
client.OnReaction(func(ctx context.Context, event *gomax.ReactionEvent) error {
    fmt.Println(event.MessageID, event.Reaction, event.Removed)
    return nil
})
```

### `OnChatUpdate`

Вызывается после изменения данных чата.

```go
client.OnChatUpdate(func(ctx context.Context, chat *gomax.Chat) error {
    fmt.Println(chat.ID, chat.Title)
    return nil
})
```

### `OnPresence`

Получает изменение online-статуса.

```go
client.OnPresence(func(ctx context.Context, event *gomax.PresenceEvent) error {
    fmt.Println(event.UserID, event.Online)
    return nil
})
```

### `OnTyping`

Получает индикатор печати.

```go
client.OnTyping(func(ctx context.Context, event *gomax.TypingEvent) error {
    fmt.Println(event.ChatID, event.UserID)
    return nil
})
```

### `OnDisconnect`

Вызывается при закрытии или обрыве соединения. Этот handler не возвращает ошибку.

```go
client.OnDisconnect(func(ctx context.Context, err error) {
    log.Println("Соединение закрыто:", err)
})
```

### `OnRaw`

Получает каждый серверный event в исходном виде. Для известных opcode сначала
вызывается typed handler, затем тот же frame передаётся в `OnRaw`.

```go
client.OnRaw(func(ctx context.Context, event *types.RawEvent) error {
    fmt.Println(event.Opcode, event.Payload)
    return nil
})
```

## Поведение

- handlers сообщений и typed events вызываются в отдельных goroutine;
- `OnDisconnect` вызывается при завершении текущего соединения;
- для тяжёлой работы передавайте данные в свою очередь;
- `OnRaw` намеренно дублирует typed events и подходит для трассировки протокола;
- все typed-регистрации принимают необязательные predicate-фильтры.
- `IncludeRouter` подключает живое дерево: handlers, добавленные в дочерний
  router после подключения, тоже работают;
- `OnError` имеет глобальную область, а `OnErrorScoped(ErrorScopeLocal, ...)`
  ограничивает обработчик ошибками его собственного router-а.
