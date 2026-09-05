# Несколько аккаунтов

Для каждого аккаунта создавайте отдельный `Config` и отдельное хранилище сессии.

## Пример

```go
for _, account := range accounts {
    cfg := gomax.DefaultConfig()
    cfg.Phone = account.Phone
    cfg.WorkDir = "./sessions"
    cfg.SessionName = account.Name + ".json"
    cfg.Proxy = account.Proxy

    client := gomax.NewClient(cfg)
    go func() {
        if err := client.Start(ctx); err != nil {
            log.Println(account.Name, err)
        }
    }()
}
```

## Ограничения и рекомендации

Для большого числа аккаунтов используйте собственный `session.Store` или
`SqliteStore`, ограничивайте количество одновременных запросов и соблюдайте
правила Max.

Не храните все токены в исходном коде и не запускайте много авторизаций одновременно.
