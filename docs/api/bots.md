# Bot Web App

Сервис доступен как `client.Bots`.

```go
data, err := client.Bots.GetInitData(ctx, botID, chatID, "start-param")
if err != nil { return err }
fmt.Println(data.URL, data.QueryID)
```

Параметры:

- `botID` — обязательный ID бота;
- `chatID` — ID чата, можно передать `0`;
- `startParam` — необязательный start-параметр.

Результат `InitData` содержит URL и query ID, которые нужны приложению бота.
