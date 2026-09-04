# GoMax (Go-клиент для Max API)

**GoMax** — высокопроизводительная библиотека на языке Go для работы с внутренним API мессенджера Max, полностью перенесённая с Python-библиотеки [PyMax](https://github.com/MaxApiTeam/PyMax).

Библиотека сохраняет 100% сигнатур и наименований методов оригинального API, реализуя при этом максимальную скорость, низкое потребление памяти и удобную работу на горутинах.

---

## Возможности и методы API

### 1. Сетевые клиенты
- `gomax.NewClient(cfg)` — полнофункциональный TCP-клиент (`Client`) для работы по нативному бинарному протоколу с авторизацией по номеру/SMS.
- `gomax.NewWebClient(cfg)` — WebSocket-клиент (`WebClient`) с поддержкой авторизации через QR-код.
- `OnStart(handler)` — хук успешного подключения и готовности сессии.
- `OnMessage(handler)` — перехват и диспетчеризация входящих сообщений в реальном времени.

### 2. Сообщения и реакции (`client.Messages`)
- `SendMessage(ctx, chatID, text, replyTo, attaches)` — отправка текстовых и мультимедиа сообщений.
- `AddReaction(ctx, chatID, messageID, reaction)` — установка эмодзи-реакций (**масс-реакции**).
- `RemoveReaction(ctx, chatID, messageID, reaction)` — снятие реакции.
- `ReadMessages(ctx, chatID, messageIDs)` — отметка списка сообщений прочитанными (**масслукинг / накрутка просмотров**).
- `ReadChat(ctx, chatID, markID)` — прочтение чата до указанного сообщения.
- `GetChatHistory(ctx, chatID, fromTime, count)` — получение истории сообщений диалога или канала.
- `EditMessage(ctx, chatID, messageID, newText)` — редактирование отправленного сообщения.
- `DeleteMessage(ctx, chatID, messageID, forAll)` — удаление сообщения.
- `ForwardMessages(ctx, toChatID, fromChatID, messageIDs)` — пересылка сообщений.
- `PinMessage(ctx, chatID, messageID)` — закрепление сообщения.
- `VotePoll(ctx, chatID, messageID, pollID, optionIDs)` — участие в опросах.

### 3. Чаты, группы и каналы (`client.Chats`)
- `JoinChat(ctx, link)` — вступление в группу или канал по ссылке-приглашению (**вступления**).
- `InviteUsersToGroup(ctx, chatID, userIDs, showHistory)` — добавление участников в группу (**инвайтинг**).
- `InviteUsersToChannel(ctx, chatID, userIDs, showHistory)` — добавление участников в канал.
- `RemoveUsersFromGroup(ctx, chatID, userIDs, cleanMsgPeriod)` — исключение участников.
- `CreateGroup(ctx, name, participantIDs, notify)` — создание новой группы.
- `LeaveChat(ctx, chatID)` — выход из чата или канала.
- `DeleteChat(ctx, chatID)` — удаление диалога.
- `ChangeGroupSettings(ctx, chatID, allCanPin, onlyAdminCanAdd)` — настройка прав группы.
- `GetChatMembers(ctx, chatID, count, marker)` — получение списка участников.
- `FetchChats(ctx, count, marker)` — список активных чатов и диалогов.
- `ReworkInviteLink(ctx, chatID)` — перегенерация ссылки-приглашения.

### 4. Пользователи и контакты (`client.Users`)
- `GetUser(ctx, userID)` — профиль пользователя по ID.
- `GetUsers(ctx, userIDs)` — пакетный запрос пользователей.
- `SearchUsers(ctx, query)` — поиск пользователей по никнейму или имени.
- `GetContacts(ctx)` — список контактов аккаунта.
- `GetSelf(ctx)` — профиль текущего авторизованного пользователя (`client.Me`).

### 5. Авторизация и сессии (`pkg/auth`, `pkg/session`)
- `SmsAuthFlow` — автоматический запрос SMS-кода и поддержка двухфакторной аутентификации (2FA).
- `QrAuthFlow` — получение ссылки на QR-код и опрос статуса подтверждения с телефона.
- `FileStore` — локальное сохранение токена и состояния сессии в формате JSON.
- `InMemoryStore` — хранение сессии только в оперативной памяти (без записи на диск).
- `FingerprintGenerator` — эмуляция цифрового отпечатка Android-устройств (SHA-256 хэши APK, cert, dex и системных библиотек).

---

## Быстрый старт

```go
package main

import (
	"context"
	"fmt"
	"log"

	"gomax"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79990000000"
	cfg.SessionName = "my_session.json"

	client := gomax.NewClient(cfg)

	// Хук при запуске
	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("Успешный вход! Мой ID: %d\n", client.Me.ID)
		return nil
	})

	// Эхо-ответчик на входящие сообщения
	client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
		if msg.Text != "" {
			_, _ = client.Messages.SendMessage(ctx, msg.ChatID, "Привет из GoMax!", msg.ID, nil)
		}
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
}
```

---

## Пример: Масс-реакции, инвайтинг, вступления и масслукинг

Полный пример находится в `examples/mass_actions/main.go`:

```go
// 1. Вступление в чат по ссылке
chat, err := client.Chats.JoinChat(ctx, "https://max.mail.ru/join/example")

// 2. Инвайтинг пользователей
err = client.Chats.InviteUsersToGroup(ctx, targetChatID, []int64{1001, 1002, 1003}, true)

// 3. Масс-реакции
err = client.Messages.AddReaction(ctx, targetChatID, messageID, "🔥")

// 4. Масслукинг (отметка о прочтении / просмотр)
err = client.Messages.ReadMessages(ctx, targetChatID, []int64{messageID})
```

---

## Как подключить библиотеку в свой проект

### Вариант 1: Локально (без загрузки на GitHub)
В файле `go.mod` вашего проекта укажите `replace`:
```text
module my_bot

go 1.26

require gomax v0.0.0
replace gomax => ../gomax
```

### Вариант 2: Через GitHub
1. Создайте репозиторий на GitHub, например `github.com/ваше-имя/gomax`.
2. Загрузите файлы из папки `gomax`.
3. В любом проекте выполните:
   ```bash
   go get github.com/ваше-имя/gomax
   ```
