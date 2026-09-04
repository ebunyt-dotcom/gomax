# Руководство по быстрому старту (Getting Started)

В этом руководстве описывается установка Go-библиотеки **GoMax**, требования к окружению, базовое подключение к серверам Max через бинарный TCP-транспорт или WebSocket, а также создание первого рабочего бота (Echo Bot).

---

## 📋 Требования

* **Go**: версия 1.20 или выше.
* **Операционная система**: Linux, macOS, Windows (архитектуры `amd64`, `arm64`).
* **Зависимости**:
  * `github.com/vmihailenco/msgpack/v5` — сериализация фреймов.
  * `github.com/klauspost/compress` — сжатие Zstd.
  * `github.com/pierrec/lz4/v4` — сжатие LZ4.
  * `github.com/gorilla/websocket` — транспорт WebSocket для веб-клиента.
  * `github.com/mattn/go-sqlite3` — SQLite хранилище сессий (опционально).

---

## 📦 Установка

### Подключение через Go Modules

Если библиотека опубликована в Git-репозитории:

```bash
go get -u github.com/ebunyt-dotcom/gomax
```

### Локальная разработка (Local Replace)

Если вы работаете с библиотекой локально в мультимодульном проекте, добавьте директиву `replace` в ваш `go.mod`:

```go
module mybot

go 1.22

require (
    gomax v0.0.0
)

replace gomax => ../gomax
```

---

## 🚀 Первое подключение

Библиотека предоставляет два типа клиентов:
1. `Client` (`gomax.NewClient`) — бинарный TCP-клиент с поддержкой TLS. Обеспечивает максимальную скорость и минимальные накладные расходы. Используется для SMS-авторизации, автоматизации и массовых действий.
2. `WebClient` (`gomax.NewWebClient`) — клиент на основе WebSocket. Поддерживает быстрый вход через сканирование QR-кода в мобильном приложении.

### Пример 1: Быстрый старт с SMS-авторизацией (TCP)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gomax"
)

func main() {
	// 1. Создаем конфигурацию по умолчанию
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79991112233" // Укажите ваш номер телефона
	cfg.WorkDir = "./sessions"
	cfg.SessionName = "bot_account.json"
	cfg.PersistSession = true // Сохранять токен в файл для повторного входа

	// 2. Создаем экземпляр клиента
	client := gomax.NewClient(cfg)

	// 3. Регистрируем обработчик успешного запуска
	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("✅ Успешная авторизация! ID: %d, Имя: %s\n", client.Me.ID, client.Me.FirstName)

		// Отправка тестового сообщения себе или в чат
		// client.Messages.SendMessage(ctx, client.Me.ID, "Бот успешно запущен!", 0, nil)
		return nil
	})

	// 4. Регистрируем обработчик входящих сообщений
	client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
		// Игнорируем собственные исходящие сообщения
		if msg.IsOutgoing {
			return nil
		}

		fmt.Printf("📩 Новое сообщение от %d в чате %d: %s\n", msg.SenderID, msg.ChatID, msg.Text)

		// Простое эхо: отвечаем на любое текстовое сообщение
		if msg.Text != "" {
			_, err := client.Messages.SendMessage(ctx, msg.ChatID, "Эхо: "+msg.Text, msg.ID, nil)
			if err != nil {
				log.Printf("Ошибка отправки ответа: %v", err)
			}
		}
		return nil
	})

	// 5. Контекст с перехватом сигналов остановки (Ctrl+C, SIGTERM)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println("⏳ Запуск клиента Max...")
	// Метод Start блокирует выполнение до завершения контекста или вызова client.Close()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Критическая ошибка работы клиента: %v", err)
	}

	fmt.Println("🛑 Работа клиента завершена корректно.")
}
```

---

## 📱 Пример 2: Авторизация через QR-код (WebSocket)

Для мгновенного входа без SMS-кода используется `WebClient`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"gomax"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.SessionName = "web_session.json"

	webClient := gomax.NewWebClient(cfg)

	webClient.OnStart(func(ctx context.Context) error {
		fmt.Printf("🚀 Web-клиент готов! Авторизован как %s (%d)\n", webClient.Me.FirstName, webClient.Me.ID)
		return nil
	})

	// При запуске в консоль будет выведена ссылка или QR-код для сканирования приложением Max
	if err := webClient.Start(context.Background()); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
}
```

---

## 🛠 Архитектурные особенности

* **Конкурентность и потокобезопасность**: Все вызовы сервисов (`client.Messages`, `client.Chats`, `client.Users`, `client.Uploads`) безопасны для вызова из сотен одновременных горутин.
* **Автоматический Reconnect**: При разрыве TCP или WS соединения клиент автоматически выполняет повторные попытки переподключения с сохранением зарегистрированных обработчиков.
* **Отсутствие фоновых утечек**: Завершение переданного `context.Context` гарантированно закрывает сетевые сокеты, каналы и вспомогательные горутины.
