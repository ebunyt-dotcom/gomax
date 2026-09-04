# Конфигурация

Начинайте с `gomax.DefaultConfig()`. Так вы получите рабочие значения endpoint, таймаутов, устройства, сессии и переподключения.

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79990000000"
cfg.SessionName = "my-session.json"
client := gomax.NewClient(cfg)
```

## Основные поля

| Поле | По умолчанию | Обязательно | За что отвечает |
|---|---|---:|---|
| `Phone` | пусто | Для первого SMS | Номер в международном формате |
| `Token` | пусто | Нет | Готовый токен сессии |
| `Host` | `api2.oneme.ru` | Нет | TCP-сервер |
| `Port` | `443` | Нет | TCP-порт |
| `UseSSL` | `true` | Нет | TLS для TCP |
| `URL` | `wss://api.oneme.ru/websocket` | Нет | WebSocket endpoint |
| `Proxy` | пусто | Нет | Прокси-соединение |
| `WorkDir` | `cache` | Нет | Каталог сессии |
| `SessionName` | `main.json` | Нет | Имя файла сессии |
| `PersistSession` | `true` | Нет | Сохранять ли сессию на диск |
| `Store` | JSON store | Нет | Собственное хранилище |
| `Reconnect` | `true` | Нет | Переподключаться после разрыва |
| `ReconnectDelay` | `1s` | Нет | Пауза между попытками |
| `RequestTimeout` | `30s` | Нет | Таймаут RPC-запроса |
| `UploadTimeout` | `15m` | Нет | Рекомендуемый лимит загрузок |
| `Interactive` | `true` | Нет | Признак активного клиента/presence |

## SMS и QR

```go
cfg.AuthFlow = gomax.NewSmsAuthFlow(myCodeProvider, myPasswordProvider)
cfg.QrAuthFlow = gomax.NewQrAuthFlow(nil, nil)
```

`AuthFlow` используется `NewClient`, `QrAuthFlow` — `NewWebClient`.

## Сессия

```go
cfg.WorkDir = "./cache"
cfg.SessionName = "account.json"
cfg.PersistSession = true
```

Если токен хранится самостоятельно:

```go
cfg.Token = os.Getenv("MAX_TOKEN")
cfg.PersistSession = false
cfg.Store = session.NewInMemoryStore()
```

Подробнее: [JSON](session/file.md), [RAM](session/memory.md), [SQLite](session/sqlite.md).

## Профиль устройства

Поля `DeviceType`, `AppVersion`, `BuildNumber`, `OSVersion`, `Timezone`, `Screen`, `Locale`, `DeviceLocale`, `DeviceName`, `Arch`, `PushDeviceType` и `UserAgent` нужны редко. Они предназначены для совместимости с PyMax и настройки handshake.

Если вы не эмулируете конкретное устройство, оставьте значения `DefaultConfig()`.
