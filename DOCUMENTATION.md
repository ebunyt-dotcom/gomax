# Полное руководство разработчика по библиотеке GoMax

**GoMax** — это идиоматичная, высокопроизводительная библиотека на языке Go для взаимодействия с внутренним API мессенджера Max. Библиотека является полным портом Python-библиотеки [PyMax](https://github.com/MaxApiTeam/PyMax), сохраняя все названия методов, структуру протокола и логику взаимодействия, но предоставляя преимущества компилируемого языка: минимальное потребление RAM, отсутствие GIL, высокую скорость обработки сетевых пакетов и параллельную работу с тысячами аккаунтов на горутинах.

---

## Содержание

1. [Архитектура библиотеки](#1-архитектура-библиотеки)
2. [Установка и подключение к проекту](#2-установка-и-подключение-к-проекту)
3. [Конфигурация клиента](#3-конфигурация-клиента)
4. [Авторизация](#4-авторизация)
   - [SMS-авторизация (телефон)](#sms-авторизация)
   - [QR-авторизация (веб)](#qr-авторизация)
   - [Двухфакторная аутентификация (2FA)](#двухфакторная-аутентификация-2fa)
5. [Хранилище сессий (Persistence)](#5-хранилище-сессий-persistence)
   - [SqliteStore](#sqlitestore)
   - [FileStore](#filestore)
   - [InMemoryStore](#inmemorystore)
6. [Цифровые отпечатки устройств (Fingerprints)](#6-цифровые-отпечатки-устройств-fingerprints)
7. [API Сервисы и методы](#7-api-сервисы-и-методы)
   - [Сообщения и реакции (`client.Messages`)](#сообщения-и-реакции-clientmessages)
   - [Чаты, группы и каналы (`client.Chats`)](#чаты-группы-и-каналы-clientchats)
   - [Пользователи, сессии и безопасность (`client.Users`)](#пользователи-сессии-и-безопасность-clientusers)
   - [Загрузка медиа и вложений (`client.Uploads`)](#загрузка-медиа-и-вложений-clientuploads)
8. [Обработка событий и фильтры](#8-обработка-событий-и-фильтры)
9. [Отказоустойчивость и автопереподключение](#9-отказоустойчивость-и-автопереподключение)
10. [Практические рецепты для массовых действий](#10-практические-рецепты-для-массовых-действий)
    - [Масс-реакции](#масс-реакции)
    - [Вступления в группы и каналы](#вступления-в-группы-и-каналы)
    - [Инвайтинг участников](#инвайтинг-участников)
    - [Масслукинг (просмотры постов)](#масслукинг-просмотры-постов)
    - [Масштабирование на сотни аккаунтов через прокси](#масштабирование-на-сотни-аккаунтов-через-прокси)

---

## 1. Архитектура библиотеки

Проект организован по модульному принципу:

```
gomax/
├── gomax.go                     # Корневой фасад верхнего уровня (алиасы и конструкторы)
├── pkg/
│   ├── protocol/                # 114 опкодов, 10-байтный бинарный заголовок, MsgPack, сжатие
│   ├── transport/               # Сетевые адаптеры TCP (TLS) и WebSocket
│   ├── connection/              # Менеджер сессий, reader'ы фреймов, seq-генератор, keepalive
│   ├── types/                   # Доменные структуры (Message, Chat, User, Attachment, Poll)
│   ├── session/                 # Хранение сессий (SQLite, JSON File, Memory)
│   ├── fingerprint/             # Эмуляция отпечатков реальных Android-устройств
│   ├── auth/                    # SMS flow, QR flow, 2FA, консольные и кастомные провайдеры
│   ├── dispatch/                # Роутер событий с цепочками фильтров
│   ├── client/                  # Высокоуровневые Client (TCP) и WebClient (WS)
│   └── api/
│       ├── chats/               # Управление чатами, участниками, правами
│       ├── messages/            # Сообщения, реакции, просмотры, опросы
│       ├── users/               # Профили, контакты, активные сессии, 2FA
│       └── uploads/             # HTTP multipart загрузка фото, видео, аудио, документов
└── examples/
    ├── quickstart/              # Базовый эхо-бот
    └── mass_actions/            # Скрипт массовых реакций, инвайтов и просмотров
```

---

## 2. Установка и подключение к проекту

### Вариант A. Локальное использование (без публикации в интернет)
Если библиотека лежит в папке `gomax` рядом с вашим проектом, добавьте директиву `replace` в ваш `go.mod`:

```text
module my_bot

go 1.26

require gomax v0.0.0
replace gomax => ../gomax
```

В коде импортируйте пакет:
```go
import "gomax"
```

### Вариант B. Через GitHub
Если вы загрузите содержимое папки `gomax` в свой репозиторий GitHub (например, `github.com/username/gomax`):
```bash
go get github.com/username/gomax
```

---

## 3. Конфигурация клиента

Для настройки клиента используется структура `gomax.Config`:

```go
cfg := gomax.DefaultConfig()

// Основные параметры
cfg.Phone = "+79991234567"          // Номер телефона для SMS авторизации
cfg.WorkDir = "cache"               // Папка для кэша и сессий
cfg.SessionName = "bot1.json"       // Имя файла сессии

// Сетевые настройки (по умолчанию используются официальные адреса Max)
cfg.Host = "api2.oneme.ru"          // TCP хост
cfg.Port = 443                      // TCP порт
cfg.URL = "wss://api.oneme.ru/websocket" // WebSocket URL
cfg.UseSSL = true                   // Использование TLS
cfg.Proxy = "socks5://127.0.0.1:9050" // Поддержка SOCKS5 / HTTP прокси

// Надежность и переподключение
cfg.Reconnect = true                // Автоматическое переподключение при разрыве связи
cfg.ReconnectDelay = 3 * time.Second // Задержка перед повторным подключением

// Готовый токен (если есть)
cfg.Token = ""                      // При наличии токена SMS/QR не запрашиваются
```

---

## 4. Авторизация

### SMS-авторизация
Используется нативным TCP-клиентом `Client`:
1. Клиент отправляет запрос SMS на номер `cfg.Phone`.
2. Провайдер запрашивает код у пользователя (по умолчанию `ConsoleCodeProvider` читает терминал).
3. При необходимости запрашивается пароль 2FA.
4. Полученный `token` сохраняется в выбранное хранилище сессий.

```go
client := gomax.NewClient(cfg)
ctx := context.Background()
if err := client.Start(ctx); err != nil {
    log.Fatal(err)
}
```

#### Кастомный поставщик SMS-кода
Если вы автоматизируете регистрацию через сервис приёма SMS (SMS-Activate, Vak-SMS и др.), реализуйте интерфейс `auth.CodeProvider`:

```go
type MySmsProvider struct {
    ActivationID string
}

func (p *MySmsProvider) GetCode(ctx context.Context) (string, error) {
    // Опрос API SMS-сервиса
    code := pollSmsApi(p.ActivationID)
    return code, nil
}
```

### QR-авторизация
Используется клиентом `WebClient`:
1. Клиент запрашивает ссылку QR-кода.
2. Ссылка выводится пользователю (в консоль или отдается в веб-интерфейс).
3. Клиент опрашивает сервер до подтверждения входа через мобильное приложение Max.

```go
webClient := gomax.NewWebClient(cfg)
if err := webClient.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### Двухфакторная аутентификация (2FA)
Если на аккаунте включен облачный пароль, библиотека автоматически перехватывает требование пароля и вызывает `PasswordProvider`:
- `auth.ConsolePasswordProvider` — ввод пароля в терминале.
- Вы можете передать свой провайдер пароля для автоматической подстановки из базы данных.

---

## 5. Хранилище сессий (Persistence)

GoMax поддерживает три типа хранилищ для токенов и состояния синхронизации:

### SqliteStore
Рекомендуется для промышленного мультиаккаунтинга (сотни и тысячи аккаунтов в единой или раздельных базах SQLite):
```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "gomax/pkg/session"
)

db, err := sql.Open("sqlite3", "sessions.db")
sqliteStore, err := session.NewSqliteStore(db)

cfg.Store = sqliteStore
```

### FileStore
Сохраняет данные сессии в JSON-файл в директории `WorkDir`:
```go
cfg.PersistSession = true
cfg.SessionName = "acc_1.json"
// Создаст файл cache/acc_1.json с токеном и device ID
```

### InMemoryStore
Сессия хранится исключительно в памяти (после перезапуска процесс заново проходит авторизацию):
```go
cfg.PersistSession = false
```

---

## 6. Цифровые отпечатки устройств (Fingerprints)

Мессенджер Max проверяет подлинность мобильного клиента с помощью 96-байтного SHA-256 отпечатка:
- **Хэш 1:** SHA256(Сертификат APK + `callsSeed` + `deviceID`)
- **Хэш 2:** SHA256(Classes.dex + `callsSeed` + `deviceID`)
- **Хэш 3:** SHA256(Библиотека `libcalls.so` архитектуры arm64/x86 + `callsSeed` + `deviceID`)

В GoMax встроен генератор `pkg/fingerprint/fingerprint.go`, который вычисляет корректный отпечаток на каждом входе в аккаунт, предотвращая подозрения со стороны антифрод-системы.

---

## 7. API Сервисы и методы

### Сообщения и реакции (`client.Messages`)

| Метод | Описание |
|---|---|
| `SendMessage(ctx, chatID, text, replyTo, attaches)` | Отправка текста с опциональным ответом и вложениями |
| `AddReaction(ctx, chatID, messageID, reaction)` | Установка эмодзи-реакции на сообщение |
| `RemoveReaction(ctx, chatID, messageID, reaction)` | Удаление реакции |
| `ReadMessages(ctx, chatID, messageIDs)` | Отметка списка сообщений прочитанными (масслукинг) |
| `ReadChat(ctx, chatID, markID)` | Прочтение чата до указанной метки сообщения |
| `GetChatHistory(ctx, chatID, fromTime, count)` | Получение списка сообщений истории чата |
| `GetHistory(ctx, chatID, fromTime, count)` | Алиас для `GetChatHistory` |
| `EditMessage(ctx, chatID, messageID, newText)` | Редактирование текста отправленного сообщения |
| `DeleteMessage(ctx, chatID, messageID, forAll)` | Удаление сообщения (для себя или для всех) |
| `ForwardMessages(ctx, toChatID, fromChatID, msgIDs)` | Пересылка сообщений |
| `PinMessage(ctx, chatID, messageID)` | Закрепление сообщения в чате |
| `VotePoll(ctx, chatID, messageID, pollID, optionIDs)`| Голосование в опросе |

#### Пример отправки сообщения:
```go
msg, err := client.Messages.SendMessage(ctx, chatID, "Привет из Go!", 0, nil)
if err != nil {
    log.Printf("Ошибка: %v", err)
} else {
    fmt.Printf("Сообщение отправлено, ID: %d\n", msg.ID)
}
```

---

### Чаты, группы и каналы (`client.Chats`)

| Метод | Описание |
|---|---|
| `JoinChat(ctx, link)` | Вступление в группу или канал по ссылке-приглашению |
| `InviteUsersToGroup(ctx, chatID, userIDs, showHistory)` | Добавление пользователей в группу |
| `InviteUsersToChannel(ctx, chatID, userIDs, showHistory)` | Добавление пользователей в канал |
| `RemoveUsersFromGroup(ctx, chatID, userIDs, cleanPeriod)` | Исключение пользователей из группы |
| `CreateGroup(ctx, name, participantIDs, notify)` | Создание новой группы |
| `LeaveChat(ctx, chatID)` | Выход из группы или канала |
| `DeleteChat(ctx, chatID)` | Удаление диалога |
| `ChangeGroupSettings(ctx, chatID, allCanPin, onlyAdminAdd)` | Изменение прав и настроек группы |
| `GetChatMembers(ctx, chatID, count, marker)` | Получение списка участников чата |
| `FetchChats(ctx, count, marker)` | Получение списка диалогов аккаунта |
| `ReworkInviteLink(ctx, chatID)` | Создание новой ссылки-приглашения взамен старой |

#### Пример вступления и инвайта:
```go
// Вступление по ссылке
chat, err := client.Chats.JoinChat(ctx, "https://max.mail.ru/join/aBcDeFgHiJ")

// Добавление пользователей в чат
err = client.Chats.InviteUsersToGroup(ctx, chat.ID, []int64{123456, 789012}, true)
```

---

### Пользователи, сессии и безопасность (`client.Users`)

| Метод | Описание |
|---|---|
| `GetUser(ctx, userID)` | Получение профиля пользователя по ID |
| `GetUsers(ctx, userIDs)` | Пакетное получение информации о пользователях |
| `SearchUsers(ctx, query)` | Глобальный поиск пользователей по имени/нику |
| `GetContacts(ctx)` | Список контактов аккаунта |
| `GetSelf(ctx)` | Получение профиля текущего пользователя (также в `client.Me`) |
| `GetActiveSessions(ctx)` | Список всех активных устройств и сессий аккаунта |
| `CloseSession(ctx, sessionID)` | Удаленное завершение другой сессии |
| `Set2FA(ctx, password, hint, email)` | Установка пароля двухфакторной аутентификации |

---

### Загрузка медиа и вложений (`client.Uploads`)

Сервис `UploadService` берет на себя получение слота загрузки и отправку файла на сервер:

| Метод | Возвращаемый тип | Описание |
|---|---|---|
| `UploadPhoto(ctx, data, fileName)` | `*types.Attachment` | Загрузка изображения |
| `UploadVideo(ctx, data, fileName, duration)` | `*types.Attachment` | Загрузка видео |
| `UploadFile(ctx, data, fileName)` | `*types.Attachment` | Загрузка документа любого формата |
| `UploadVoice(ctx, data, duration)` | `*types.Attachment` | Загрузка голосового сообщения (.ogg) |

#### Пример отправки фото:
```go
photoData, _ := os.ReadFile("avatar.jpg")
attach, err := client.Uploads.UploadPhoto(ctx, photoData, "avatar.jpg")
if err == nil {
    // Отправляем сообщение с прикрепленным фото
    _, _ = client.Messages.SendMessage(ctx, chatID, "Вот фото:", 0, []types.Attachment{*attach})
}
```

---

## 8. Обработка событий и фильтры

В GoMax встроен реактивный диспетчер событий с поддержкой предикатных фильтров:

```go
// 1. Хук запуска (вызывается сразу после входа в аккаунт)
client.OnStart(func(ctx context.Context) error {
    fmt.Println("Клиент готов к приему сообщений!")
    return nil
})

// 2. Обработчик входящих сообщений с фильтрацией
client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
    fmt.Printf("Получено сообщение: %s\n", msg.Text)
    return nil
}, 
// Фильтры (сообщение обработается только если все предикаты вернут true):
func(m *gomax.Message) bool {
    return !m.IsOutgoing // Игнорировать свои сообщения
},
func(m *gomax.Message) bool {
    return m.Text != "" // Только текстовые
})
```

---

## 9. Отказоустойчивость и автопереподключение

В реальных сценариях (мобильные сети, прокси, рестарты серверов) соединение может прерываться. В GoMax встроен бесконечный цикл восстановления:

- При сетевой ошибке клиент получает уведомление в канал `disconnectCh`.
- Менеджер соединений безопасно закрывает сокет.
- Клиент выдерживает паузу `cfg.ReconnectDelay` (по умолчанию 3 секунды).
- Выполняется повторное подключение, сессионный хэндшейк и авторизация по сохраненному токену без вмешательства оператора.

---

## 10. Практические рецепты для массовых действий

### Масс-реакции
Ставит случайные эмодзи на последние посты в канале с защитной задержкой:

```go
emojis := []string{"👍", "🔥", "❤️", "🎉", "👏"}
messages, err := client.Messages.GetHistory(ctx, targetChannelID, 0, 20)
if err == nil {
    for _, msg := range messages {
        emoji := emojis[rand.Intn(len(emojis))]
        _ = client.Messages.AddReaction(ctx, targetChannelID, msg.ID, emoji)
        
        // Задержка 1-2 секунды между запросами для защиты от антифрода
        time.Sleep(time.Duration(1000+rand.Intn(1000)) * time.Millisecond)
    }
}
```

### Вступления в группы и каналы
```go
links := []string{
    "https://max.mail.ru/join/link1",
    "https://max.mail.ru/join/link2",
}

for _, link := range links {
    chat, err := client.Chats.JoinChat(ctx, link)
    if err != nil {
        log.Printf("Не удалось вступить по ссылке %s: %v", link, err)
    } else {
        log.Printf("Вступили в чат: %s (ID: %d)", chat.Title, chat.ID)
    }
    time.Sleep(5 * time.Second)
}
```

### Инвайтинг участников
```go
// Добавление пачки пользователей в группу
targetUsers := []int64{10001, 10002, 10003, 10004}
err := client.Chats.InviteUsersToGroup(ctx, targetGroupID, targetUsers, true)
if err != nil {
    log.Printf("Ошибка инвайтинга: %v", err)
}
```

### Масслукинг (просмотры постов)
Отправка серверу подтверждения прочтения сообщений (засчитывает просмотры постов):

```go
messages, err := client.Messages.GetHistory(ctx, channelID, 0, 50)
if err == nil {
    var ids []int64
    for _, msg := range messages {
        ids = append(ids, msg.ID)
    }
    // Отправляем пакетное подтверждение прочтения
    _ = client.Messages.ReadMessages(ctx, channelID, ids)
}
```

### Масштабирование на сотни аккаунтов через прокси
Благодаря легковесности горутин Go, вы можете параллельно запустить сотни аккаунтов, выделив каждому свой изолированный SOCKS5/HTTP прокси:

```go
package main

import (
	"context"
	"fmt"
	"gomax"
	"sync"
)

type Account struct {
	Phone string
	Proxy string
}

func runBot(ctx context.Context, acc Account, wg *sync.WaitGroup) {
	defer wg.Done()

	cfg := gomax.DefaultConfig()
	cfg.Phone = acc.Phone
	cfg.Proxy = acc.Proxy
	cfg.SessionName = fmt.Sprintf("session_%s.json", acc.Phone)

	client := gomax.NewClient(cfg)
	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("Аккаунт %s подключен через %s\n", acc.Phone, acc.Proxy)
		return nil
	})

	_ = client.Start(ctx)
}

func main() {
	accounts := []Account{
		{Phone: "+79991111111", Proxy: "socks5://proxy1.example.com:1080"},
		{Phone: "+79992222222", Proxy: "socks5://proxy2.example.com:1080"},
		{Phone: "+79993333333", Proxy: "socks5://proxy3.example.com:1080"},
	}

	var wg sync.WaitGroup
	ctx := context.Background()

	for _, acc := range accounts {
		wg.Add(1)
		go runBot(ctx, acc, &wg)
	}

	wg.Wait()
}
```
