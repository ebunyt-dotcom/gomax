# Сессия в JSON

`FileStore` — стандартное хранилище GoMax. Оно сохраняет токен, device ID, sync-маркеры и параметры аккаунта в JSON.

## Включение

```go
cfg := gomax.DefaultConfig()
cfg.WorkDir = "./cache"
cfg.SessionName = "account.json"
cfg.PersistSession = true

client := gomax.NewClient(cfg)
```

Файл будет создан по пути `WorkDir/SessionName`. Каталог создаётся автоматически, права файла — `0600`.

## Что обязательно

Ничего, если используется `DefaultConfig()`. `Phone` нужен только для первого SMS-входа. После сохранения сессии библиотека использует токен.

## Свой store

```go
cfg.Store = session.NewFileStore("./cache", "account.json")
```

Если указан `Store`, он имеет приоритет над `PersistSession`.

## Методы `Store`

Эти методы одинаковы у `FileStore`, `InMemoryStore` и `SqliteStore`.

| Метод | Назначение | Пример |
|---|---|---|
| `SaveSession` | Сохранить token, телефон, device ID и sync-состояние. | `err := store.SaveSession(info)` |
| `LoadSession` | Загрузить основную сессию. | `info, err := store.LoadSession()` |
| `UpdateToken` | Обновить token для телефона. | `err := store.UpdateToken(phone, token)` |
| `LoadSessionByDeviceID` | Найти сессию по устройству. | `info, err := store.LoadSessionByDeviceID(deviceID)` |
| `LoadSessionByPhone` | Найти сессию по номеру. | `info, err := store.LoadSessionByPhone(phone)` |
| `DeleteSession` | Удалить одну сессию по token. | `err := store.DeleteSession(token)` |
| `DeleteAllSessions` | Удалить все сессии. | `err := store.DeleteAllSessions()` |
| `Close` | Закрыть store. | `err := store.Close()` |

Пример записи сессии вручную:

```go
info := &session.SessionInfo{
    Phone: "+79990000000",
    Token: token,
    DeviceID: deviceID,
}
err := store.SaveSession(info)
```

## Безопасность

Добавьте каталог сессий в `.gitignore`:

```gitignore
cache/
*.json
```

Файл сессии равносилен ключу доступа к аккаунту.
