# Загрузки

Сервис доступен как `client.Uploads`. Загрузка состоит из запроса upload-slot к API и HTTP-загрузки файла.

Готовый пример загрузки и отправки фото: [`examples/send_media/main.go`](https://github.com/ebunyt-dotcom/gomax/blob/main/examples/send_media/main.go).

## Методы

| Метод | Что принимает | Что возвращает |
|---|---|---|
| `UploadPhoto` | bytes, имя файла | `*Attachment` типа `PHOTO` |
| `UploadPhotoWithOptions` | bytes, имя, `profile` | Фото или аватар |
| `UploadVideo` | bytes, имя, duration | `VIDEO` после обработки |
| `UploadVoice` | bytes, duration | `VOICE` после обработки |
| `UploadFile` | bytes, имя | `FILE` после обработки |
| `DecodeThumbhash` | base64 thumbhash | bytes |

## Фото

```go
photo, err := client.Uploads.UploadPhoto(ctx, imageBytes, "avatar.jpg")
if err != nil { return err }
_, err = client.Messages.SendMessage(ctx, chatID, "", 0, []gomax.Attachment{*photo})
```

Для профиля:

```go
avatar, err := client.Uploads.UploadPhotoWithOptions(ctx, imageBytes, "avatar.jpg", true)
```

## Видео, voice и файлы

```go
video, err := client.Uploads.UploadVideo(ctx, videoBytes, "clip.mp4", 12)
voice, err := client.Uploads.UploadVoice(ctx, audioBytes, 4)
file, err := client.Uploads.UploadFile(ctx, documentBytes, "report.pdf")
```

Для видео, voice и файлов сервер может сначала принять bytes, а затем прислать `NOTIF_ATTACH`. GoMax ждёт это событие до 60 секунд.

## Отправка результата

```go
attachment, err := client.Uploads.UploadFile(ctx, data, "report.pdf")
if err != nil { return err }
_, err = client.Messages.SendMessage(ctx, chatID, "Документ", 0, []gomax.Attachment{*attachment})
```

Не подставляйте вручную `_type`, `videoId`, `audioId` или `fileId`: они формируются из `Attachment` при отправке.
