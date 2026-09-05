# Документация GoMax

GoMax — Go-клиент Max с двумя способами подключения:

- `NewClient` — TCP + SMS, подходит для серверных задач;
- `NewWebClient` — WebSocket + QR, подходит для входа через приложение Max.

API пакетов доступен в [Go Reference на pkg.go.dev](https://pkg.go.dev/github.com/ebunyt-dotcom/gomax).

## С чего начать

1. Установите библиотеку: `go get github.com/ebunyt-dotcom/gomax`.
2. Выберите клиент.
3. Начните с [быстрого старта](getting-started.md).
4. После первого входа изучите [сессии](session/file.md), чтобы не вводить код повторно.

## Разделы

| Раздел | Для чего |
|---|---|
| [Начало работы](getting-started.md) | Первый проект, SMS, QR, остановка клиента |
| [Готовые примеры](examples.md) | Готовый `main.go` под каждую типовую задачу |
| [Задачи](tasks/index.md) | Готовые сценарии: история, реакции, инвайты, чаты и файлы |
| [Конфигурация](configuration.md) | Все основные поля `Config` |
| [SMS](authentication/sms.md) | Вход по номеру и обработчики кода |
| [QR](authentication/qr.md) | QR-вход через WebSocket |
| [2FA](authentication/2fa.md) | Пароль и настройка двухфакторной защиты |
| [Сообщения](api/messages.md) | Отправка, история, реакции, пересылка |
| [Чаты](api/chats.md) | Группы, каналы, участники и заявки |
| [Пользователи](api/users.md) | Поиск, контакты и сессии |
| [Загрузки](api/uploads.md) | Фото, видео, voice и файлы |
| [Профиль](api/self.md) | Свой профиль, папки, presence и logout |
| [Auth](api/auth.md) | Низкоуровневые SMS/QR/2FA операции |
| [Bot Web App](api/bots.md) | Получение данных bot web app |
| [Полный API](api/reference.md) | Все публичные функции и методы |
| [Типы и данные](api/types.md) | Поля сообщений, чатов, пользователей, вложений и событий |
| [События](dispatch/events.md) | `OnStart`, `OnMessage`, `OnRaw` и другие обработчики |
| [JSON-сессия](session/file.md) | Сохранение одного аккаунта в файл |
| [RAM-сессия](session/memory.md) | Сессия без записи на диск |
| [SQLite-сессия](session/sqlite.md) | Пользовательское хранилище для нескольких аккаунтов |
| [Пакетные операции](mass-actions/inviting.md) | Повторяющиеся действия и ограничения |
| [Протокол](protocol/wire.md) | TCP/WS framing и основные опкоды |

## Минимальный пример

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79990000000"
client := gomax.NewClient(cfg)
if err := client.Start(context.Background()); err != nil {
    log.Fatal(err)
}
```

Полные примеры находятся в каталоге [`examples/`](https://github.com/ebunyt-dotcom/gomax/tree/main/examples).
