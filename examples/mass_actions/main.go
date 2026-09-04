package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"gomax"
)

// Демонстрация массовых действий: масс-реакции, масслукинг, вступления и инвайтинг
func main() {
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79990000000"
	cfg.SessionName = "bot1.json"

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		fmt.Println("Аккаунт авторизован и готов к работе.")

		// 1. Вступление по ссылке (JoinChat)
		inviteLink := "https://max.mail.ru/join/abc123xyz"
		chat, err := client.Chats.JoinChat(ctx, inviteLink)
		if err != nil {
			log.Printf("Не удалось вступить в чат: %v", err)
		} else {
			fmt.Printf("Успешно вступили в чат: %s (ID: %d)\n", chat.Title, chat.ID)
		}

		targetChatID := int64(12345678)

		// 2. Инвайтинг пользователей (InviteUsersToGroup)
		usersToInvite := []int64{1001, 1002, 1003}
		if err := client.Chats.InviteUsersToGroup(ctx, targetChatID, usersToInvite, true); err != nil {
			log.Printf("Ошибка инвайта: %v", err)
		} else {
			fmt.Printf("Инвайты отправлены для пользователей: %v\n", usersToInvite)
		}

		// 3. Получение постов и Масс-реакции (AddReaction)
		history, err := client.Messages.GetChatHistory(ctx, targetChatID, 0, 10)
		if err == nil {
			emojis := []string{"👍", "🔥", "❤️", "👏"}
			for _, msg := range history {
				reaction := emojis[rand.Intn(len(emojis))]
				if err := client.Messages.AddReaction(ctx, targetChatID, msg.ID, reaction); err != nil {
					log.Printf("Не удалось поставить реакцию на %d: %v", msg.ID, err)
				} else {
					fmt.Printf("Поставлена реакция %s на сообщение %d\n", reaction, msg.ID)
				}

				// 4. Масслукинг / просмотр (ReadMessages)
				if err := client.Messages.ReadMessages(ctx, targetChatID, []int64{msg.ID}); err != nil {
					log.Printf("Не удалось отметить прочитанным: %v", err)
				} else {
					fmt.Printf("Сообщение %d прочитано (просмотр учтен)\n", msg.ID)
				}

				// Антифрод задержка между действиями
				time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)
			}
		}

		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
}
