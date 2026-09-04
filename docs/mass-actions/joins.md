# Массовые вступления в каналы и чаты (Mass Joins)

Автоматическое вступление в каналы и чаты по ссылкам-приглашениям используется при прогреве свежезарегистрированных аккаунтов, подписке бот-ферм на целевые каналы или автоматическом добавлении аккаунтов в рабочие чаты.

---

## 🔗 Обработка ссылок методом `JoinChat`

Метод `client.Chats.JoinChat` принимает ссылки в любом удобном формате:
* Полный HTTPS URL: `https://max.ru/join/k7LmN9Pq`
* Короткий путь: `join/k7LmN9Pq`
* Голый идентификатор/хэш инвайта.

Библиотека GoMax автоматически нормализует строку и выполняет RPC-вызов `OpChatJoin` (опкод 57).

---

## 🛡 Рекомендации по прогреву и безопасности

* **Интервалы**: не вступайте в десятки каналов подряд без пауз. Делайте задержку от 10 до 45 секунд между подписками.
* **Лимиты**: для «свежих» (недавно зарегистрированных) аккаунтов безопасно вступать не более чем в 10-15 каналов в сутки. Для отлежавшихся аккаунтов — до 50 каналов в сутки.

---

## 💻 Пример: Последовательное вступление в список каналов

```go
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"gomax"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.SessionName = "subscriber.json"
	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		channelLinks := []string{
			"https://max.ru/join/dev_community",
			"https://max.ru/join/golang_news",
			"https://max.ru/join/tech_digest",
		}

		for i, link := range channelLinks {
			fmt.Printf("[%d/%d] Вступаем в канал по ссылке: %s\n", i+1, len(channelLinks), link)

			chat, err := client.Chats.JoinChat(ctx, link)
			if err != nil {
				log.Printf("⚠️ Ошибка вступления: %v", err)
			} else {
				fmt.Printf("✅ Успешно вступили! Название: %s (ID: %d)\n", chat.Title, chat.ID)
			}

			// Рандомизированная задержка между вступлениями (15..30 секунд)
			delay := 15 + rand.Intn(15)
			fmt.Printf("Пауза %d секунд...\n", delay)
			time.Sleep(time.Duration(delay) * time.Second)
		}

		fmt.Println("Все подписки оформлены!")
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```
