package main

import (
	"context"
	"fmt"
	"log"

	"gomax"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79990000000"
	cfg.WorkDir = "cache"
	cfg.SessionName = "main.json"

	client := gomax.NewClient(cfg)

	// on_start listener
	client.OnStart(func(ctx context.Context) error {
		fmt.Println("Клиент успешно запущен!")
		if client.Me != nil {
			fmt.Printf("Ваш ID: %d, Имя: %s\n", client.Me.ID, client.Me.FirstName)
		}
		return nil
	})

	// on_message listener
	client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
		fmt.Printf("Получено сообщение [%d] в чате %d: %s\n", msg.SenderID, msg.ChatID, msg.Text)

		if msg.Text != "" {
			_, err := client.Messages.SendMessage(ctx, msg.ChatID, "Привет от GoMax!", msg.ID, nil)
			if err != nil {
				log.Printf("Ошибка отправки ответа: %v", err)
			}
		}
		return nil
	})

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Ошибка работы клиента: %v", err)
	}
}
