package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ebunyt-dotcom/gomax"
)

const chatID int64 = 123456789 // обязательно замените на ID своего чата

func main() {
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79990000000" // обязательно замените на свой номер
	cfg.SessionName = "media-session.json"

	client := gomax.NewClient(cfg)
	client.OnStart(func(ctx context.Context) error {
		data, err := os.ReadFile("photo.jpg") // файл положите рядом с программой
		if err != nil {
			return err
		}

		photo, err := client.Uploads.UploadPhoto(ctx, data, "photo.jpg")
		if err != nil {
			return err
		}

		_, err = client.Messages.SendMessage(ctx, chatID, "Фото", 0, []gomax.Attachment{*photo})
		if err == nil {
			fmt.Println("Фото отправлено")
		}
		return err
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
