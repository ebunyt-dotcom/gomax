# Bot Web App

Сервис: `client.Bots`. Примеры ниже — фрагменты: `client`, `ctx`, `botID` и
`chatID` уже созданы.

## Получение данных

### `GetInitData`

Получает URL и query ID для открытия bot web app.

```go
data, err := client.Bots.GetInitData(ctx, botID, chatID, "start-param")
if err != nil {
    return err
}
fmt.Println(data.URL, data.QueryID)
```

Параметры:

- `botID` — обязательный ID бота;
- `chatID` — ID чата, можно передать `0`;
- `startParam` — необязательный start-параметр.
