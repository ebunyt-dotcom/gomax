# Пакетная работа с чатами

Повторяющиеся действия выполняются обычным циклом Go.

## Пример

```go
links := []string{"https://max.ru/join/one", "https://max.ru/join/two"}
for _, link := range links {
    chat, err := client.Chats.JoinChat(ctx, link)
    if err != nil {
        log.Printf("%s: %v", link, err)
        continue
    }
    log.Println("Вступили:", chat.ID)
}
```

## Ограничения и рекомендации

Проверяйте права доступа и лимиты. Между запросами при необходимости добавляйте
задержку.
