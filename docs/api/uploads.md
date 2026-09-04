# Сервис загрузки медиафайлов (`client.Uploads`)

Сервис `UploadService` обеспечивает двухфазную загрузку медиафайлов (фотографий, видеороликов, документов и голосовых сообщений) в инфраструктуру Max с последующим прикреплением их к сообщениям через структуру `types.Attachment`.

Доступ к сервису осуществляется через поле `client.Uploads` (или `webClient.Uploads`).

---

## ⚙️ Двухфазная архитектура загрузки в Max

Загрузка любых медиафайлов в Max происходит по следующему протоколу:
1. **Фаза RPC-согласования**: клиент отправляет запрос на выделение слота загрузки (`OpPhotoUpload`, `OpVideoUpload` или `OpFileUpload`). Сервер возвращает временный преподписанный HTTP URL (`uploadURL`) и постоянный токен вложения (`token`).
2. **Фаза HTTP-транспорта**: клиент выполняет HTTP POST запрос методом `multipart/form-data` на полученный URL, передавая бинарные байты файла.
3. **Фаза прикрепления**: полученный токен встраивается в объект `types.Attachment` и передается в метод `client.Messages.SendMessage`.

---

## 📋 Список методов

1. [`UploadPhoto`](#1-uploadphoto) — загрузка изображения (JPEG, PNG, WebP)
2. [`UploadVideo`](#2-uploadvideo) — загрузка видеофайла (MP4, MKV) с указанием длительности
3. [`UploadFile`](#3-uploadfile) — загрузка произвольного документа или файла
4. [`UploadVoice`](#4-uploadvoice) — загрузка голосового сообщения в формате OGG/Opus

---

### 1. `UploadPhoto`

Загружает статическое изображение и подготавливает объект `AttachmentPhoto`.

```go
func (s *UploadService) UploadPhoto(
    ctx context.Context,
    data []byte,
    fileName string,
) (*types.Attachment, error)
```

* **Опкод протокола**: `OpPhotoUpload` (78).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `data` (`[]byte`): бинарное содержимое изображения.
  * `fileName` (`string`): имя файла (например `"avatar.png"` или `"screenshot.jpg"`).
* **Возвращаемое значение**: `(*types.Attachment, error)`.

#### Пример вызова (загрузка и отправка фото в чат):
```go
photoBytes, err := os.ReadFile("picture.jpg")
if err != nil {
    log.Fatal(err)
}

// 1. Загружаем фото на сервер
att, err := client.Uploads.UploadPhoto(ctx, photoBytes, "picture.jpg")
if err != nil {
    log.Fatalf("Ошибка загрузки фото: %v", err)
}

// 2. Отправляем сообщение с прикрепленным фото
_, err = client.Messages.SendMessage(ctx, chatID, "Вот ваше фото!", 0, []gomax.Attachment{*att})
```

---

### 2. `UploadVideo`

Запрашивает слот под видеоматериал и передает бинарные видеоданные.

```go
func (s *UploadService) UploadVideo(
    ctx context.Context,
    data []byte,
    fileName string,
    duration int,
) (*types.Attachment, error)
```

* **Опкод протокола**: `OpVideoUpload` (80).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `data` (`[]byte`): бинарные данные видео.
  * `fileName` (`string`): имя файла (например `"clip.mp4"`).
  * `duration` (`int`): продолжительность видеоролика в секундах.
* **Возвращаемое значение**: `(*types.Attachment, error)`.

#### Пример вызова:
```go
videoBytes, _ := os.ReadFile("trailer.mp4")
att, err := client.Uploads.UploadVideo(ctx, videoBytes, "trailer.mp4", 45)
if err != nil {
    log.Fatal(err)
}

_, err = client.Messages.SendMessage(ctx, chatID, "Посмотрите трейлер:", 0, []gomax.Attachment{*att})
```

---

### 3. `UploadFile`

Загружает произвольный документ (PDF, ZIP, TXT, DOCX и т.д.).

```go
func (s *UploadService) UploadFile(
    ctx context.Context,
    data []byte,
    fileName string,
) (*types.Attachment, error)
```

* **Опкод протокола**: `OpFileUpload` (82).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `data` (`[]byte`): содержимое файла.
  * `fileName` (`string`): исходное имя документа (например `"document.pdf"`).
* **Возвращаемое значение**: `(*types.Attachment, error)`.

#### Пример вызова:
```go
pdfBytes, _ := os.ReadFile("invoice.pdf")
att, err := client.Uploads.UploadFile(ctx, pdfBytes, "invoice.pdf")
if err != nil {
    log.Fatal(err)
}

_, _ = client.Messages.SendMessage(ctx, chatID, "Счет на оплату", 0, []gomax.Attachment{*att})
```

---

### 4. `UploadVoice`

Загружает аудиофайл голосового сообщения (обычно в контейнере OGG с кодеком Opus).

```go
func (s *UploadService) UploadVoice(
    ctx context.Context,
    data []byte,
    duration int,
) (*types.Attachment, error)
```

* **Опкод протокола**: `OpFileUpload` (82) со специфичным типом `AttachmentVoice`.
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `data` (`[]byte`): байты аудиофайла.
  * `duration` (`int`): длительность записи в секундах.
* **Возвращаемое значение**: `(*types.Attachment, error)`.

#### Пример вызова:
```go
voiceBytes, _ := os.ReadFile("greeting.ogg")
att, err := client.Uploads.UploadVoice(ctx, voiceBytes, 7)
if err != nil {
    log.Fatal(err)
}

_, _ = client.Messages.SendMessage(ctx, chatID, "", 0, []gomax.Attachment{*att})
```
