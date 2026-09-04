# Вход через QR

QR-вход работает через `WebClient` и не требует номера телефона в конфигурации.

## Минимальный вариант

```go
cfg := gomax.DefaultConfig()
cfg.SessionName = "qr-session.json"
cfg.QrAuthFlow = gomax.NewQrAuthFlow(nil, nil)

client := gomax.NewWebClient(cfg)
client.OnStart(func(ctx context.Context) error {
    fmt.Println("Вход выполнен")
    return nil
})

log.Fatal(client.Start(context.Background()))
```

Стандартный обработчик печатает QR-код и ссылку в терминал. Отсканируйте код в приложении Max.

Готовый пример: [`examples/qr_login/main.go`](../../examples/qr_login/main.go).

## Как работает flow

1. WebSocket-клиент выполняет handshake.
2. `OpGetQr` (`288`) создаёт QR-сессию.
3. `OpGetQrStatus` (`289`) проверяет `trackId` с интервалом сервера.
4. После подтверждения вызывается `OpLoginByQr` (`291`).
5. Токен сохраняется в файл сессии.

## Собственный обработчик QR

```go
type MyQrHandler struct{}

func (MyQrHandler) HandleQr(ctx context.Context, qrURL string) error {
    fmt.Println("Откройте или покажите эту ссылку:", qrURL)
    return nil
}

cfg.QrAuthFlow = gomax.NewQrAuthFlow(MyQrHandler{}, nil)
```

Интерфейс `QrHandler` содержит один метод: `HandleQr(context.Context, string) error`.

## Если нужен пароль 2FA

Передайте второй аргумент в `NewQrAuthFlow`. Если он `nil`, используется консольный ввод.

```go
cfg.QrAuthFlow = gomax.NewQrAuthFlow(nil, myPasswordProvider)
```

`NewClient` QR не показывает: он предназначен для TCP/SMS-входа.
