# Масслукинг и накрутка просмотров (Views & Mass-Looking)

Масслукинг (автоматическая отметка постов и сообщений прочитанными) используется для продвижения каналов, поднятия счетчиков просмотров («глазиков») и имитации органической пользовательской активности ботов.

---

## ⚙️ Механика протокола Max

В протоколе Max за отметку о прочтении отвечает RPC-операция `OpChatMark` (опкод 50).

Когда клиент вызывает метод `client.Messages.ReadMessages(ctx, chatID, messageIDs)`:
1. Формируется фрейм с полезной нагрузкой:
   ```json
   {
     "chatId": 987654321,
     "messageIds": [101, 102, 103, 104],
     "type": "READ"
   }
   ```
2. Сервер фиксирует факт прочтения указанных постов текущим пользователем и инкрементирует глобальный счетчик просмотров каждого сообщения.

---

## 🎯 Преимущества `ReadMessages` над `ReadChat`

* `ReadChat(chatID, markID)` помечает прочитанными **все подряд** сообщения вплоть до `markID`.
* `ReadMessages(chatID, messageIDs)` позволяет передавать **точечный список конкретных ID постов** (например, только последние 5 постов), экономя трафик и исключая подозрительную активность.

---

## 💻 Пример 1: Автоматический просмотр последних публикаций канала

```go
package main

import (
	"context"
	"fmt"
	"log"

	"gomax"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.SessionName = "viewer.json"
	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		channelID := int64(123456789)

		// 1. Получаем последние 15 постов канала
		history, err := client.Messages.GetChatHistory(ctx, channelID, 0, 15)
		if err != nil {
			return fmt.Errorf("ошибка получения постов: %w", err)
		}

		// 2. Собираем массив ID сообщений
		var msgIDs []int64
		for _, msg := range history {
			msgIDs = append(msgIDs, msg.ID)
		}

		// 3. Отправляем пакетную отметку о прочтении
		err = client.Messages.ReadMessages(ctx, channelID, msgIDs)
		if err != nil {
			return fmt.Errorf("ошибка отметки о прочтении: %w", err)
		}

		fmt.Printf("✅ Успешно просмотрено %d постов в канале %d!\n", len(msgIDs), channelID)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

---

## 💻 Пример 2: Накрутка просмотров пулом из десятков аккаунтов

Каждый аккаунт из вашего пула запускается со своим приватным прокси и отправляет `ReadMessages` на один и тот же список ID сообщений. Сервер засчитает уникальный просмотр от каждого аккаунта.

Подробную реализацию пула смотрите в главе [Мультиаккаунтинг 100+ аккаунтов](multi-account.md).
