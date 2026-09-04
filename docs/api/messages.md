# Сообщения

Сервис доступен как `client.Messages` у `Client` и `WebClient`.

## Методы

| Метод | Что делает | Обязательные данные |
|---|---|---|
| `SendMessage` | Отправляет текст, reply или готовые вложения | `chatID`; текст или вложение |
| `GetChatHistory` / `GetHistory` | Получает историю | `chatID` |
| `GetMessages` / `GetMessage` | Получает сообщения по ID | `chatID`, ID сообщений |
| `EditMessage` | Меняет текст сообщения | `chatID`, `messageID`, непустой текст |
| `DeleteMessage` | Удаляет сообщение | `chatID`, `messageID` |
| `ForwardMessage(s)` | Пересылает одно или несколько сообщений | ID чатов и сообщений |
| `PinMessage` | Закрепляет сообщение | `chatID`, `messageID` |
| `AddReaction` / `RemoveReaction` | Добавляет или убирает реакцию | `chatID`, `messageID`, emoji для добавления |
| `GetReactions` | Получает реакции | `chatID`, ID сообщений |
| `ReadMessage` / `ReadMessages` / `ReadChat` | Помечает прочитанным | `chatID` и ID/mark |
| `VotePoll` | Голосует в опросе | IDs чата, сообщения, опроса и ответов |
| `GetVideoByID` | Получает данные видео | `chatID`, `messageID`, `videoID` |
| `GetFileByID` | Получает данные файла | `chatID`, `messageID`, `fileID` |

## Отправка текста и reply

```go
msg, err := client.Messages.SendMessage(ctx, chatID, "Привет", 0, nil)
if err != nil {
    return err
}
fmt.Println(msg.ID)

_, err = client.Messages.SendMessage(ctx, chatID, "Ответ", originalMessageID, nil)
```

`replyToMsgID == 0` означает обычное сообщение. Пустой текст допустим, если переданы вложения.

## История

```go
messages, err := client.Messages.GetChatHistory(ctx, chatID, 0, 50)
```

`count <= 0` заменяется на значение по умолчанию. `fromTime == 0` означает «начиная с текущей точки синхронизации».

## Реакции и прочтение

```go
err := client.Messages.AddReaction(ctx, chatID, messageID, "👍")
err = client.Messages.RemoveReaction(ctx, chatID, messageID, "👍")
err = client.Messages.ReadMessage(ctx, messageID, chatID)
```

## Пересылка и удаление

```go
_, err := client.Messages.ForwardMessage(ctx, targetChatID, messageID, sourceChatID, true)
err = client.Messages.DeleteMessage(ctx, chatID, messageID, true) // для всех
```

## Вложения

Сначала загрузите данные через `Uploads`, затем передайте `Attachment` в `SendMessage`. Wire-поля `_type`, `photoToken`, `videoId`, `audioId` и `fileId` формируются библиотекой.

```go
photo, err := client.Uploads.UploadPhoto(ctx, imageBytes, "photo.jpg")
if err != nil { return err }
_, err = client.Messages.SendMessage(ctx, chatID, "", 0, []gomax.Attachment{*photo})
```

## Что возвращается

Методы возвращают типы из `gomax/types`: `Message`, `Attachment`, `ReactionInfo`, `Poll`. Ошибки сервера возвращаются как `error`; проверяйте их после каждого сетевого вызова.
