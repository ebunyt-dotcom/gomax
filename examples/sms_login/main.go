package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ebunyt-dotcom/gomax"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79990000000"           // обязательно замените на свой номер при первом входе
	cfg.SessionName = "sms-session.json" // необязательно; отдельная сессия

	client := gomax.NewClient(cfg)
	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("Вход выполнен: %s (ID %d)\n", client.Me.FirstName, client.Me.ID)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
