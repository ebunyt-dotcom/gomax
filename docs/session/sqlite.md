# Сессия в SQLite

`SqliteStore` оборачивает готовое подключение `*sql.DB`. Он подходит, когда несколько аккаунтов должны храниться в одной базе.

## Инициализация

```go
db, err := sql.Open("sqlite3", "./sessions.db")
if err != nil { log.Fatal(err) }
defer db.Close()

store, err := session.NewSqliteStore(db)
if err != nil { log.Fatal(err) }

cfg := gomax.DefaultConfig()
cfg.Phone = "+79990000000"
cfg.Store = store
client := gomax.NewClient(cfg)
```

GoMax создаёт таблицу `max_sessions` автоматически. SQLite-драйвер нужно добавить в приложение самостоятельно; библиотека намеренно не выбирает драйвер.

## Поля

Хранятся `phone`, `token`, `device_id`, `mt_instance_id`, четыре sync-маркера и `config_hash`.

## Важно

- Один `SqliteStore` может обслуживать несколько клиентов, если драйвер и база поддерживают параллельные запросы.
- Закрывайте `*sql.DB` после завершения работы.
- Не публикуйте файл базы: в нём находятся токены.
