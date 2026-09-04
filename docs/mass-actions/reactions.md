# Массовая расстановка реакций (Mass Reactions)

Массовая расстановка реакций (эмодзи) используется для продвижения каналов, создания активности на публикациях и поднятия вовлеченности аудитории.

---

## 🛡 Защита от блокировок (Антифрод)

При расстановке реакций с одного или группы аккаунтов критически важно соблюдать следующие правила:
1. **Рандомизация пауз (Jitter)**: никогда не отправляйте запросы с фиксированным интервалом. Используйте псевдослучайную задержку (например, от 800 до 2500 мс).
2. **Разнообразие эмодзи**: распределяйте различные эмодзи (`👍`, `🔥`, `❤️`, `🎉`, `👏`) среди ботов, чтобы распределение выглядело органично.
3. **Изоляция прокси**: при запуске десятков аккаунтов каждый бот должен работать через индивидуальный приватный мобильный или резидентский прокси.

---

## 💻 Пример 1: Расстановка реакций на последние посты канала одним аккаунтом

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
	cfg.SessionName = "bot1.json"

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		channelID := int64(987654321) // ID целевого канала
		emojis := []string{"🔥", "👍", "❤️", "🚀", "👏"}

		// 1. Получаем последние 20 сообщений канала
		messages, err := client.Messages.GetChatHistory(ctx, channelID, 0, 20)
		if err != nil {
			return fmt.Errorf("ошибка получения истории: %w", err)
		}

		fmt.Printf("Найдено %d сообщений. Начинаем расстановку реакций...\n", len(messages))

		for _, msg := range messages {
			// Выбираем случайный эмодзи
			selectedEmoji := emojis[rand.Intn(len(emojis))]

			err := client.Messages.AddReaction(ctx, channelID, msg.ID, selectedEmoji)
			if err != nil {
				log.Printf("Не удалось поставить реакцию на пост %d: %v", msg.ID, err)
			} else {
				fmt.Printf("✅ Поставлена реакция %s на сообщение %d\n", selectedEmoji, msg.ID)
			}

			// Рандомная задержка от 1.0 до 3.0 секунд
			sleepMs := 1000 + rand.Intn(2000)
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
		}

		fmt.Println("🎉 Массовая расстановка реакций завершена!")
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

---

## 💻 Пример 2: Накрутка 50 реакций на один целевой пост пулом аккаунтов

Смотрите подробный паттерн оркестрации группы клиентов в разделе [Мультиаккаунтинг 100+ аккаунтов](multi-account.md).
