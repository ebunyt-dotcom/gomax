# Сервис сообщений (`client.Messages`)

Сервис `MessageService` отвечает за отправку, редактирование, удаление, пересылку и закрепление сообщений, простановку реакций, участие в опросах и отметку сообщений прочитанными (масслукинг / накрутка просмотров).

Доступ к сервису осуществляется через поле `client.Messages` (или `webClient.Messages`).

---

## 📋 Список методов

1. [`SendMessage`](#1-sendmessage) — отправка текстового сообщения с вложениями и ответом
2. [`AddReaction`](#2-addreaction) — установка эмодзи-реакции на сообщение
3. [`RemoveReaction`](#3-removereaction) — снятие ранее установленной реакции
4. [`ReadMessages`](#4-readmessages) — отметка конкретных сообщений прочитанными (масслукинг)
5. [`ReadChat`](#5-readchat) — прочтение диалога/канала до указанного сообщения
6. [`GetChatHistory`](#6-getchathistory) — получение истории сообщений
7. [`GetHistory`](#7-gethistory) — псевдоним `GetChatHistory` для обратной совместимости с PyMax
8. [`EditMessage`](#8-editmessage) — редактирование текста отправленного сообщения
9. [`DeleteMessage`](#9-deletemessage) — удаление сообщения (для себя или для всех)
10. [`ForwardMessages`](#10-forwardmessages) — пересылка сообщений в другой чат
11. [`PinMessage`](#11-pinmessage) — закрепление сообщения вверху чата
12. [`VotePoll`](#12-votepoll) — голосование в опросе

---

### 1. `SendMessage`

Отправляет текстовое сообщение в указанный чат, диалог или канал с возможностью ответа (reply) и прикрепления медиавложений.

```go
func (s *MessageService) SendMessage(
    ctx context.Context,
    chatID int64,
    text string,
    replyToMsgID int64,
    attaches []types.Attachment,
) (*types.Message, error)
```

* **Опкод протокола**: `OpMsgSend` (64).
* **Параметры**:
  * `ctx` (`context.Context`): контекст выполнения запроса.
  * `chatID` (`int64`): уникальный идентификатор целевого чата или пользователя.
  * `text` (`string`): текст сообщения (поддерживает UTF-8 и эмодзи).
  * `replyToMsgID` (`int64`): идентификатор сообщения, на которое создается ответ. Передайте `0`, если ответ не требуется.
  * `attaches` (`[]types.Attachment`): слайс загруженных вложений (фото, видео, файлы, голосовые). Передайте `nil` или пустой слайс для чисто текстового сообщения.
* **Возвращаемое значение**: `(*types.Message, error)` — объект отправленного сообщения с присвоенным сервером `ID` и локальным клиентским `CID`.

#### Пример вызова:
```go
// Отправка обычного сообщения
msg, err := client.Messages.SendMessage(ctx, 123456789, "Привет из GoMax!", 0, nil)
if err != nil {
    log.Printf("Ошибка отправки: %v", err)
}
fmt.Printf("Отправлено сообщение ID: %d\n", msg.ID)

// Отправка ответа на другое сообщение
replyMsg, err := client.Messages.SendMessage(ctx, 123456789, "Ответ на вопрос", msg.ID, nil)
```

---

### 2. `AddReaction`

Ставит эмодзи-реакцию на указанное сообщение в чате или посте канала.

```go
func (s *MessageService) AddReaction(
    ctx context.Context,
    chatID int64,
    messageID int64,
    reaction string,
) error
```

* **Опкод протокола**: `OpMsgReaction` (178).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `messageID` (`int64`): идентификатор целевого сообщения.
  * `reaction` (`string`): строковый идентификатор реакции или эмодзи (например `"👍"`, `"🔥"`, `"❤️"`, `"like"`).
* **Возвращаемое значение**: `error` — `nil` при успешной установке.

#### Пример вызова:
```go
err := client.Messages.AddReaction(ctx, chatID, messageID, "🔥")
if err != nil {
    log.Fatalf("Не удалось поставить реакцию: %v", err)
}
```

---

### 3. `RemoveReaction`

Снимает ранее установленную реакцию с сообщения.

```go
func (s *MessageService) RemoveReaction(
    ctx context.Context,
    chatID int64,
    messageID int64,
    reaction string,
) error
```

* **Опкод протокола**: `OpMsgCancelReaction` (179).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `messageID` (`int64`): идентификатор целевого сообщения.
  * `reaction` (`string`): идентификатор снимаемой реакции.
* **Возвращаемое значение**: `error` — `nil` при успешном снятии.

#### Пример вызова:
```go
err := client.Messages.RemoveReaction(ctx, chatID, messageID, "🔥")
```

---

### 4. `ReadMessages`

Помечает список указанных сообщений как прочитанные. Данный метод является фундаментальной основой для алгоритмов **масслукинга (mass-looking)** и автоматической накрутки счетчиков просмотров постов в каналах.

```go
func (s *MessageService) ReadMessages(
    ctx context.Context,
    chatID int64,
    messageIDs []int64,
) error
```

* **Опкод протокола**: `OpChatMark` (50).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата или канала.
  * `messageIDs` (`[]int64`): массив идентификаторов сообщений для отметки о прочтении.
* **Возвращаемое значение**: `error` — `nil` при успехе.

#### Пример вызова:
```go
ids := []int64{101, 102, 103, 104, 105}
err := client.Messages.ReadMessages(ctx, channelID, ids)
if err != nil {
    log.Printf("Ошибка накрутки просмотров: %v", err)
}
```

---

### 5. `ReadChat`

Помечает весь чат прочитанным до указанного идентификатора сообщения.

```go
func (s *MessageService) ReadChat(
    ctx context.Context,
    chatID int64,
    markID int64,
) error
```

* **Опкод протокола**: `OpChatMark` (50).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `markID` (`int64`): идентификатор крайнего прочитанного сообщения (все сообщения до него будут считаться прочитанными).
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
err := client.Messages.ReadChat(ctx, chatID, lastMsgID)
```

---

### 6. `GetChatHistory`

Запрашивает историю сообщений в чате с пагинацией по времени.

```go
func (s *MessageService) GetChatHistory(
    ctx context.Context,
    chatID int64,
    fromTime int64,
    count int,
) ([]types.Message, error)
```

* **Опкод протокола**: `OpChatHistory` (49).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `fromTime` (`int64`): таймстемп в миллисекундах, от которого запрашивается история (передайте `0` для получения самых свежих сообщений).
  * `count` (`int`): количество запрашиваемых сообщений (по умолчанию 50, если передано `<= 0`).
* **Возвращаемое значение**: `([]types.Message, error)` — срез полученных сообщений.

#### Пример вызова:
```go
messages, err := client.Messages.GetChatHistory(ctx, chatID, 0, 100)
if err != nil {
    log.Fatalf("Ошибка получения истории: %v", err)
}

for _, m := range messages {
    fmt.Printf("[%d] %d: %s\n", m.ID, m.SenderID, m.Text)
}
```

---

### 7. `GetHistory`

Псевдоним для метода `GetChatHistory`. Создан для 100% сигнатурной совместимости с библиотекой PyMax (`client.messages.get_history(...)`).

```go
func (s *MessageService) GetHistory(
    ctx context.Context,
    chatID int64,
    fromTime int64,
    count int,
) ([]types.Message, error)
```

Параметры и логика полностью идентичны [`GetChatHistory`](#6-getchathistory).

---

### 8. `EditMessage`

Редактирует текст ранее отправленного сообщения.

```go
func (s *MessageService) EditMessage(
    ctx context.Context,
    chatID int64,
    messageID int64,
    newText string,
) error
```

* **Опкод протокола**: `OpMsgEdit` (67).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `messageID` (`int64`): идентификатор редактируемого сообщения.
  * `newText` (`string`): новый текст сообщения.
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
err := client.Messages.EditMessage(ctx, chatID, msgID, "Обновленный текст сообщения")
```

---

### 9. `DeleteMessage`

Удаляет сообщение из переписки.

```go
func (s *MessageService) DeleteMessage(
    ctx context.Context,
    chatID int64,
    messageID int64,
    forAll bool,
) error
```

* **Опкод протокола**: `OpMsgDelete` (66).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `messageID` (`int64`): идентификатор удаляемого сообщения.
  * `forAll` (`bool`): `true` — удалить для всех участников диалога/чата; `false` — удалить только для себя.
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
// Удаление сообщения у всех участников
err := client.Messages.DeleteMessage(ctx, chatID, msgID, true)
```

---

### 10. `ForwardMessages`

Пересылает одно или несколько сообщений из одного чата в другой.

```go
func (s *MessageService) ForwardMessages(
    ctx context.Context,
    toChatID int64,
    fromChatID int64,
    messageIDs []int64,
) error
```

* **Опкод протокола**: `OpMsgSend` (64) с флагом пересылки.
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `toChatID` (`int64`): идентификатор чата-получателя.
  * `fromChatID` (`int64`): идентификатор исходного чата-источника.
  * `messageIDs` (`[]int64`): слайс идентификаторов пересылаемых сообщений.
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
err := client.Messages.ForwardMessages(ctx, targetChatID, sourceChatID, []int64{101, 102})
```

---

### 11. `PinMessage`

Закрепляет сообщение в верхней панели чата или канала.

```go
func (s *MessageService) PinMessage(
    ctx context.Context,
    chatID int64,
    messageID int64,
) error
```

* **Опкод протокола**: `OpChatUpdate` (55).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `messageID` (`int64`): идентификатор закрепляемого сообщения.
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
err := client.Messages.PinMessage(ctx, chatID, importantMsgID)
```

---

### 12. `VotePoll`

Отправляет голос за один или несколько вариантов ответа в опросе.

```go
func (s *MessageService) VotePoll(
    ctx context.Context,
    chatID int64,
    messageID int64,
    pollID int64,
    optionIDs []int,
) error
```

* **Опкод протокола**: `OpSendVote` (304).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `chatID` (`int64`): идентификатор чата.
  * `messageID` (`int64`): идентификатор сообщения с опросом.
  * `pollID` (`int64`): уникальный идентификатор самого опроса.
  * `optionIDs` (`[]int`): массив индексов выбранных вариантов ответов (0-indexed).
* **Возвращаемое значение**: `error`.

#### Пример вызова:
```go
// Голосование за первый и третий вариант ответа
err := client.Messages.VotePoll(ctx, chatID, msgID, pollID, []int{0, 2})
```
