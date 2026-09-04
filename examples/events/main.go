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

	client := gomax.NewClient(cfg)
	client.OnStart(func(ctx context.Context) error {
		fmt.Println("Клиент готов")
		return nil
	})

	client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
		fmt.Printf("Сообщение: chat=%d sender=%d text=%q\n", msg.ChatID, msg.SenderID, msg.Text)
		return nil
	})

	client.OnReaction(func(ctx context.Context, event *gomax.ReactionEvent) error {
		fmt.Printf("Реакция: message=%d reaction=%s removed=%v\n", event.MessageID, event.Reaction, event.Removed)
		return nil
	})

	client.OnRaw(func(ctx context.Context, event *gomax.RawEvent) error {
		fmt.Printf("Raw event: opcode=%d payload=%v\n", event.Opcode, event.Payload)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
