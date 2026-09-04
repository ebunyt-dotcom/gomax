# Сессия в RAM

`InMemoryStore` хранит состояние только в памяти процесса. После остановки приложения токен и sync-состояние исчезают.

## Включение

```go
cfg := gomax.DefaultConfig()
cfg.Token = os.Getenv("MAX_TOKEN")
cfg.PersistSession = false
cfg.Store = session.NewInMemoryStore()

client := gomax.NewClient(cfg)
```

## Когда использовать

- одноразовый скрипт;
- контейнер без постоянного диска;
- токен уже хранится во внешнем секрет-хранилище.

Если `Token` не задан и store пуст, клиент попросит SMS или QR в зависимости от выбранного клиента.
