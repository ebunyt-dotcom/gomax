package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ebunyt-dotcom/gomax"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79990000000" // обязательно замените на свой номер
	cfg.SessionName = "profile-session.json"

	client := gomax.NewClient(cfg)
	client.OnStart(func(ctx context.Context) error {
		profile, err := client.Self.GetSelf(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Профиль: %s %s (ID %d)\n", profile.FirstName, profile.LastName, profile.ID)

		chats, _, err := client.Chats.FetchChats(ctx, 40, "")
		if err != nil {
			return err
		}
		for _, chat := range chats {
			fmt.Printf("Чат: %s (ID %d)\n", chat.Title, chat.ID)
		}
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
