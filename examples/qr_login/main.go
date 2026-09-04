package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ebunyt-dotcom/gomax"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.SessionName = "qr-session.json"
	cfg.QrAuthFlow = gomax.NewQrAuthFlow(nil, nil) // стандартный QR печатается в терминал

	client := gomax.NewWebClient(cfg)
	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("QR-вход выполнен: %s (ID %d)\n", client.Me.FirstName, client.Me.ID)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
