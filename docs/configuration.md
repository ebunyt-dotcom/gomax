# Конфигурация

Конфигурация создаётся через `gomax.DefaultConfig()`. Обязательное поле для
первого SMS-входа — `Phone` или готовый `Token`.

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79990000000"
cfg.SessionName = "my-session.json"
client := gomax.NewClient(cfg)
```

## Подключение

| Поле | По умолчанию | Назначение |
|---|---|---|
| `Phone` | `""` | Номер для первого SMS-входа. |
| `Token` | `""` | Готовый token; заменяет SMS. |
| `Host` | `api2.oneme.ru` | TCP-сервер для `NewClient`. |
| `Port` | `443` | TCP-порт. |
| `UseSSL` | `true` | TLS для TCP. |
| `URL` | `wss://api.oneme.ru/websocket` | WebSocket URL для `NewWebClient`. |
| `Proxy` | `""` | `http://...` или `socks5://...`. |

## Сессия

| Поле | По умолчанию | Назначение |
|---|---|---|
| `WorkDir` | `cache` | Каталог файловой сессии. |
| `SessionName` | `main.json` | Имя файла сессии. |
| `PersistSession` | `true` | Сохранять token и device ID. |
| `Store` | файловый store | Собственное `session.Store`. |

```go
cfg.WorkDir = "data"
cfg.SessionName = "account.json"
cfg.PersistSession = true
```

Если `Store` задан, он используется вместо автоматического store. При
`PersistSession=false` пользовательский `Store` намеренно игнорируется и
используется изолированный `InMemoryStore`, как в PyMax.

## Переподключение и таймауты

| Поле | По умолчанию | Назначение |
|---|---|---|
| `Reconnect` | `true` | Повторять подключение после разрыва. |
| `ReconnectDelay` | `1s` | Пауза между попытками. |
| `RequestTimeout` | `30s` | Таймаут RPC-запроса. |
| `UploadTimeout` | `15m` | Максимальное ожидание загрузки. |
| `DisableUploadTimeout` | `false` | Отключить общий HTTP upload timeout. |
| `Interactive` | `true` | Передавать серверу активный статус. |
| `Relogin` | `true` | Сбросить отозванный token и заново авторизоваться. |
| `PasswordMaxAttempts` | `nil` | Лимит 2FA; `nil` — без лимита, `0`/отрицательное значение — сразу отказать. |
| `LogLevel` | `INFO` | Уровень внутренних логов. |
| `Telemetry` | `true` | Отправлять Max фоновые telemetry events. |

## Профиль устройства

Эти поля нужны только для совместимости с мобильным профилем PyMax. В обычном
коде оставьте значения `DefaultConfig`.

| Поле | Назначение |
|---|---|
| `DeviceID` | Стабильный ID устройства. |
| `MtInstanceID` | ID экземпляра клиента. |
| `DeviceType` | Тип устройства, обычно `ANDROID`. |
| `AppVersion` | Версия приложения. |
| `BuildNumber` | Номер сборки. |
| `OSVersion` | Версия ОС. |
| `Timezone` | Часовой пояс. |
| `Screen` | Разрешение и плотность экрана. |
| `Locale` | Язык приложения. |
| `DeviceLocale` | Локаль устройства. |
| `DeviceName` | Имя устройства. |
| `Arch` | Архитектура, например `arm64-v8a`. |
| `PushDeviceType` | Тип push-сервиса. |
| `UserAgent` | Полная ручная замена user-agent map. |
| `ProtocolVersion` | Версия TCP frame, по умолчанию `10`. |
| `ClientSessionID` | Явный session ID; `1..70` генерируется автоматически. |
| `RestoreUserAgentFromSession` | Восстанавливать характеристики устройства из сессии. |
| `DisableUserAgentRestore` | Явно отключить восстановление user-agent. |
| `Sync` | Точечные overrides sync-маркеров и config hash. |

## Авторизация и регистрация

| Поле | Назначение |
|---|---|
| `AuthFlow` | Свой SMS/2FA flow для `NewClient`. |
| `QrAuthFlow` | Свой QR/2FA flow для `NewWebClient`. |
| `Registration` | Имя и фамилия при регистрации нового аккаунта. |

```go
cfg.Registration = &gomax.RegistrationConfig{
    FirstName: "Ivan",
    LastName:  "Ivanov",
}
```

`Registration` нужна только если сервер после SMS вернул registration token.

## Разные клиенты

```go
// SMS/TCP
tcpClient := gomax.NewClient(gomax.DefaultConfig())

// QR/WebSocket
qrCfg := gomax.DefaultConfig()
qrCfg.QrAuthFlow = gomax.NewQrAuthFlow(nil, nil)
webClient := gomax.NewWebClient(qrCfg)
```
