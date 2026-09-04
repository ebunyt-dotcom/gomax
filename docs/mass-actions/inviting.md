# Массовый инвайтинг пользователей (Mass Inviting)

Массовый инвайтинг (добавление целевой аудитории в чаты и группы) осуществляется с помощью метода `client.Chats.InviteUsersToGroup` (или `InviteUsersToChannel`).

---

## ⚠️ Лимиты и правила безопасности

1. **Размер пачки (Batch size)**: не пытайтесь добавить 100 человек за один запрос. Оптимальный размер одной пачки — от 3 до 10 пользователей (`userIDs`).
2. **Интервалы между пачками**: выдерживайте паузу от 10 до 30 секунд между пачками инвайтов с одного аккаунта.
3. **Лимит на аккаунт**: в Max действуют суточные лимиты на инвайты неконтактов (обычно до 30-50 успешных добавлений в сутки на один аккаунт). Для больших объемов используйте пул из десятков аккаунтов.
4. **Приватность пользователей**: если у целевого пользователя в настройках приватности запрещено добавление в группы посторонними лицами, сервер вернет ошибку `USER_PRIVACY_RESTRICTED`. Такие ошибки следует отлавливать и продолжать цикл.

---

## 💻 Пример пакетного инвайтинга с контролем ошибок

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gomax"
)

func chunkSlice(slice []int64, chunkSize int) [][]int64 {
	var chunks [][]int64
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

func main() {
	cfg := gomax.DefaultConfig()
	cfg.SessionName = "inviter.json"
	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		targetChatID := int64(123456789)

		// Список собранных (спарсенных) ID пользователей
		targetUserIDs := []int64{
			5500101, 5500102, 5500103, 5500104, 5500105,
			5500106, 5500107, 5500108, 5500109, 5500110,
		}

		// Разбиваем на пачки по 5 пользователей
		batches := chunkSlice(targetUserIDs, 5)

		for i, batch := range batches {
			fmt.Printf("Добавление пачки %d/%d (пользователей: %d)...\n", i+1, len(batches), len(batch))

			// showHistory: true — открывать ли историю чата новым участникам
			err := client.Chats.InviteUsersToGroup(ctx, targetChatID, batch, true)
			if err != nil {
				log.Printf("⚠️ Ошибка добавления пачки: %v", err)
			} else {
				fmt.Printf("✅ Пачка %d успешно добавлена!\n", i+1)
			}

			// Ожидание перед следующей пачкой
			time.Sleep(15 * time.Second)
		}

		fmt.Println("Инвайтинг завершен!")
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```
