# GoMax

Go-библиотека для работы с Max через TCP и WebSocket. API и формат обмена адаптированы из PyMax, а публичный интерфейс сделан привычным для Go.

## Установка

```powershell
go get github.com/ebunyt-dotcom/gomax
```

Требуется Go 1.26 или новее.

## Что выбрать

| Задача | Клиент |
|---|---|
| Вход по телефону и SMS, обычная автоматизация | `gomax.NewClient` |
| Вход через QR в приложении Max | `gomax.NewWebClient` |

Оба клиента имеют одинаковые сервисы: `Messages`, `Chats`, `Users`, `Uploads`, `Self`, `Auth`, `Bots`.

## Быстрый старт: SMS

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ebunyt-dotcom/gomax"
)

func main() {
    cfg := gomax.DefaultConfig()
    cfg.Phone = "+79990000000" // обязательно только при первом входе
    cfg.SessionName = "main.json"

    client := gomax.NewClient(cfg)
    client.OnStart(func(ctx context.Context) error {
        fmt.Printf("Вход выполнен: %s (ID %d)\n", client.Me.FirstName, client.Me.ID)
        return nil
    })

    if err := client.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

При первом запуске библиотека запросит SMS-код в консоли и сохранит сессию. При следующих запусках `Phone` и SMS обычно уже не нужны.

## Быстрый старт: QR

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ebunyt-dotcom/gomax"
)

func main() {
    cfg := gomax.DefaultConfig()
    cfg.SessionName = "qr.json"
    cfg.QrAuthFlow = gomax.NewQrAuthFlow(nil, nil)

    client := gomax.NewWebClient(cfg)
    client.OnStart(func(ctx context.Context) error {
        fmt.Println("QR-вход выполнен")
        return nil
    })

    if err := client.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

В консоли появится QR-код. Отсканируйте его в приложении Max. Для QR нужен именно `NewWebClient`, а не `NewClient`.

## Настройки: обязательно и необязательно

| Поле | Обязательно | Назначение |
|---|---:|---|
| `Phone` | Да для первого SMS-входа | Номер аккаунта в международном формате |
| `SessionName` | Нет | Имя файла сессии; по умолчанию `main.json` |
| `WorkDir` | Нет | Каталог сессии; по умолчанию `cache` |
| `Token` | Нет | Уже полученный токен вместо SMS/QR |
| `Store` | Нет | Собственный store вместо JSON-файла |
| `QrAuthFlow` | Нет для `NewWebClient` | Собственный QR/password provider |
| `Proxy` | Нет | HTTP/SOCKS-прокси |
| `Reconnect` | Нет | Переподключение после разрыва |

Обычно достаточно начать с `gomax.DefaultConfig()` и изменить только `Phone`, `SessionName` или `QrAuthFlow`.

## Основные операции

```go
// сообщение
_, err := client.Messages.SendMessage(ctx, chatID, "Привет", 0, nil)

// история
history, err := client.Messages.GetChatHistory(ctx, chatID, 0, 50)

// чат
chat, err := client.Chats.GetChat(ctx, chatID)

// пользователь
user, err := client.Users.GetUser(ctx, userID)

// файл
file, err := client.Uploads.UploadFile(ctx, data, "report.pdf")
_, err = client.Messages.SendMessage(ctx, chatID, "", 0, []gomax.Attachment{*file})
```

## Документация

- [Готовые примеры](docs/examples.md)
- [Начало работы](docs/getting-started.md)
- [Конфигурация](docs/configuration.md)
- [SMS](docs/authentication/sms.md) · [QR](docs/authentication/qr.md) · [2FA](docs/authentication/2fa.md)
- [Сообщения](docs/api/messages.md) · [Чаты](docs/api/chats.md) · [Пользователи](docs/api/users.md) · [Загрузки](docs/api/uploads.md)
- [Профиль и настройки](docs/api/self.md)
- [Auth](docs/api/auth.md) · [Bot Web App](docs/api/bots.md)
- [События](docs/dispatch/events.md)
- [Сессии: JSON](docs/session/file.md) · [RAM](docs/session/memory.md) · [SQLite](docs/session/sqlite.md)
- [Протокол](docs/protocol/wire.md)

## Важные замечания

- Не публикуйте токены и файлы сессий в Git.
- Соблюдайте лимиты и правила Max: массовые операции могут привести к ограничениям аккаунта.
- `context.Context` передаётся во все сетевые вызовы и должен отменяться при завершении приложения.

## Лицензия

Проект распространяется по лицензии MIT. Библиотека не является официальным SDK Max.
