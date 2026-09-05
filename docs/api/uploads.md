# Загрузки

Сервис: `client.Uploads`. Методы принимают содержимое файла как `[]byte`.
После загрузки результат передаётся в `client.Messages.SendMessage`.

Полные сигнатуры находятся в [полном справочнике](reference.md). Каждый
пример ниже — отдельный фрагмент; `ctx`, данные файла и `client` уже созданы.

## Методы

### `UploadPhoto`

Загружает изображение и возвращает `*gomax.Attachment` типа `PHOTO`.

```go
photo, err := client.Uploads.UploadPhoto(ctx, imageBytes, "photo.jpg")
```

### `UploadPhotoWithOptions`

Загружает изображение. При `profile == true` оно предназначено для аватара.

```go
avatar, err := client.Uploads.UploadPhotoWithOptions(ctx, imageBytes, "avatar.jpg", true)
```

### `UploadVideo`

Загружает видео. `duration` указывается в секундах.

```go
video, err := client.Uploads.UploadVideo(ctx, videoBytes, "clip.mp4", 12)
```

### `UploadVoice`

Загружает голосовое сообщение. `duration` указывается в секундах.

```go
voice, err := client.Uploads.UploadVoice(ctx, audioBytes, 4)
```

### `UploadFile`

Загружает произвольный файл.

```go
file, err := client.Uploads.UploadFile(ctx, documentBytes, "report.pdf")
```

### `DecodeThumbhash`

Декодирует base64 thumbhash в байты. Это функция пакета `uploads`, а не метод
сервиса.

```go
thumb, err := uploads.DecodeThumbhash(value)
```

## Отправка результата

```go
file, err := client.Uploads.UploadFile(ctx, data, "report.pdf")
if err != nil {
    return err
}
_, err = client.Messages.SendMessage(ctx, chatID, "Документ", 0,
    []gomax.Attachment{*file})
```

## Методы уведомлений

Эти методы вызываются клиентом при получении `NOTIF_ATTACH`. В обычном коде их
вызывать не нужно.

### `NotifyReady`

Передаёт сырое push-уведомление обработчику ожидающей загрузки.

```go
client.Uploads.NotifyReady(payload)
```

### `NotifyVideoReady`

Завершает ожидание обработки видео по ID.

```go
client.Uploads.NotifyVideoReady(videoID)
```

### `NotifyVoiceReady`

Завершает ожидание обработки voice по ID аудио.

```go
client.Uploads.NotifyVoiceReady(audioID)
```

### `NotifyFileReady`

Завершает ожидание обработки файла по ID.

```go
client.Uploads.NotifyFileReady(fileID)
```

Для видео, voice и файлов сервер может завершить обработку отдельным событием;
клиент ждёт его автоматически.
