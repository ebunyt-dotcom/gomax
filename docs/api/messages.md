# Сообщения

Сервис: `client.Messages`. Все методы принимают `ctx context.Context` первым
аргументом. В примерах `ctx`, `chatID` и ID сообщений уже объявлены.

## Отправка

### `SendMessage`

Отправляет текст и/или вложения. `replyToMsgID == 0` — без ответа.

```go
msg, err := client.Messages.SendMessage(ctx, chatID, "Привет", 0, nil)
```

### `EditMessage`

Меняет текст существующего сообщения.

```go
err := client.Messages.EditMessage(ctx, chatID, messageID, "Новый текст")
```

### `DeleteMessage`

Удаляет сообщение. При `forAll == true` запрашивается удаление для всех.

```go
err := client.Messages.DeleteMessage(ctx, chatID, messageID, true)
```

### `ForwardMessage`

Пересылает одно сообщение и возвращает созданное сообщение.

```go
copy, err := client.Messages.ForwardMessage(ctx, targetChatID, messageID, sourceChatID, true)
```

### `ForwardMessages`

Пересылает несколько сообщений одним запросом.

```go
err := client.Messages.ForwardMessages(ctx, targetChatID, sourceChatID, []int64{101, 102})
```

### `PinMessage`

Закрепляет сообщение в чате.

```go
err := client.Messages.PinMessage(ctx, chatID, messageID)
```

## Получение

### `GetChatHistory`

Возвращает историю чата. `fromTime == 0` — начиная с текущей точки синхронизации.

```go
items, err := client.Messages.GetChatHistory(ctx, chatID, 0, 50)
```

### `GetHistory`

Совместимое имя `GetChatHistory`.

```go
items, err := client.Messages.GetHistory(ctx, chatID, 0, 50)
```

### `GetMessages`

Возвращает несколько сообщений по ID.

```go
items, err := client.Messages.GetMessages(ctx, chatID, []int64{101, 102})
```

### `GetMessage`

Возвращает одно сообщение по ID.

```go
item, err := client.Messages.GetMessage(ctx, chatID, messageID)
```

### `GetVideoByID`

Получает вложение видео из сообщения.

```go
video, err := client.Messages.GetVideoByID(ctx, chatID, messageID, videoID)
```

### `GetFileByID`

Получает вложение файла из сообщения.

```go
file, err := client.Messages.GetFileByID(ctx, chatID, messageID, fileID)
```

## Реакции и прочтение

### `AddReaction`

Добавляет реакцию.

```go
err := client.Messages.AddReaction(ctx, chatID, messageID, "👍")
```

### `RemoveReaction`

Убирает реакцию.

```go
err := client.Messages.RemoveReaction(ctx, chatID, messageID, "👍")
```

### `GetReactions`

Возвращает реакции для указанных сообщений.

```go
reactions, err := client.Messages.GetReactions(ctx, chatID, []int64{messageID})
```

### `ReadMessage`

Помечает одно сообщение прочитанным. Порядок аргументов: `messageID`, затем `chatID`.

```go
err := client.Messages.ReadMessage(ctx, messageID, chatID)
```

### `ReadMessages`

Помечает несколько сообщений прочитанными.

```go
err := client.Messages.ReadMessages(ctx, chatID, []int64{101, 102})
```

### `ReadChat`

Отправляет read-marker для чата.

```go
err := client.Messages.ReadChat(ctx, chatID, markID)
```

## Опросы

### `VotePoll`

Голосует в опросе. Для одного выбора передайте один ID варианта.

```go
err := client.Messages.VotePoll(ctx, chatID, messageID, pollID, []int{optionID})
```

## Вложения

Вложения сначала создаются через `client.Uploads`, затем передаются в
`SendMessage`. Поля wire-протокола вручную заполнять не нужно.

```go
photo, err := client.Uploads.UploadPhoto(ctx, imageBytes, "photo.jpg")
if err != nil {
    return err
}
_, err = client.Messages.SendMessage(ctx, chatID, "Фото", 0,
    []gomax.Attachment{*photo})
```

Все методы возвращают `error`. При ошибке не используйте возвращённый указатель
или результат.
