# События

Обработчики регистрируются на `Client` или `WebClient` до вызова `Start`.

Полный пример: [`examples/events/main.go`](../../examples/events/main.go).

## Обработчики

| Метод | Когда вызывается | Данные |
|---|---|---|
| `OnStart` | Клиент авторизован и готов | `context.Context` |
| `OnMessage` | Новое или синхронизированное сообщение | `*types.Message` |
| `OnMessageEdit` | Сообщение изменено | `*types.Message` |
| `OnMessageDelete` | Сообщение удалено | `chatID`, `messageID` |
| `OnMessageRead` | Пришёл read-marker | `*types.MessageReadEvent` |
| `OnUserUpdate` | Изменился контакт/профиль | `*types.UserUpdateEvent` |
| `OnReaction` | Реакция добавлена или убрана | `*types.ReactionEvent` |
| `OnChatUpdate` | Изменились данные чата | `*types.Chat` |
| `OnPresence` | Изменился online-статус | `*types.PresenceEvent` |
| `OnTyping` | Пользователь печатает | `*types.TypingEvent` |
| `OnRaw` | Нераспознанное событие | `*types.RawEvent` |
| `OnDisconnect` | Соединение закрыто/оборвалось | `error` |

## Пример

```go
client.OnStart(func(ctx context.Context) error {
    fmt.Println("Клиент готов")
    return nil
})

client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
    fmt.Printf("%d: %s\n", msg.ChatID, msg.Text)
    return nil
})

client.OnDisconnect(func(ctx context.Context, err error) {
    log.Println("Соединение закрыто:", err)
})
```

## Важное поведение

- Обработчики сообщений и typed events вызываются в отдельных goroutine.
- Ошибка из обработчика не останавливает клиент автоматически.
- Не блокируйте обработчик надолго; для тяжёлой работы отправьте данные в свою очередь.
- `OnRaw` получает только события, которые не были разобраны в typed event.

## Фильтрация

Публичный метод `OnMessage` принимает обычный handler. Если нужна фильтрация, проверяйте поля сообщения внутри handler или используйте `dispatch.Router` напрямую в расширенном приложении.
