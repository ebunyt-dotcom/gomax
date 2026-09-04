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

## Безопасность

Добавьте каталог сессий в `.gitignore`:

```gitignore
cache/
*.json
```

Файл сессии равносилен ключу доступа к аккаунту.
