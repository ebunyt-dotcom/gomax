# Начало работы

## Требования

- Go 1.26+;
- доступ к `api2.oneme.ru:443` для TCP или к WebSocket endpoint;
- номер телефона для первого SMS-входа либо приложение Max для QR-входа.

## Создание проекта

```powershell
mkdir my-max-app
cd my-max-app
go mod init my-max-app
go get github.com/ebunyt-dotcom/gomax
```

## SMS-вход

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79990000000"
cfg.SessionName = "session.json"

client := gomax.NewClient(cfg)
client.OnStart(func(ctx context.Context) error {
    fmt.Printf("Готово: %s\n", client.Me.FirstName)
    return nil
})

err := client.Start(context.Background())
```

`Phone` нужен для получения первого SMS-кода. После успешной авторизации токен и параметры устройства сохраняются в `WorkDir/SessionName`.

## QR-вход

QR работает через WebSocket-клиент:

```go
cfg := gomax.DefaultConfig()
cfg.SessionName = "qr-session.json"
cfg.QrAuthFlow = gomax.NewQrAuthFlow(nil, nil)

client := gomax.NewWebClient(cfg)
if err := client.Start(context.Background()); err != nil {
    log.Fatal(err)
}
```

`NewClient` QR не показывает: это отдельный TCP/SMS-сценарий.

## Использование сервисов

После создания клиента доступны:

```go
client.Messages // сообщения и реакции
client.Chats    // группы, каналы и участники
client.Users    // пользователи и контакты
client.Uploads  // фото, видео, voice и файлы
client.Self     // профиль, папки, presence
client.Auth     // низкоуровневые auth-операции
client.Bots     // данные bot web app
```

Все сетевые методы принимают `context.Context` и возвращают `error`.

## Корректное завершение

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := client.Start(ctx); err != nil {
    log.Print(err)
}
```

При отмене контекста клиент закрывает соединение и завершает фоновые циклы.

## Если что-то не работает

- QR не появляется — проверьте, что используется `NewWebClient`, а не `NewClient`.
- SMS не приходит — это ответ сервера/оператора; не повторяйте запрос много раз подряд.
- Старая версия после `go get` — проверьте `go list -m -json github.com/ebunyt-dotcom/gomax`.
