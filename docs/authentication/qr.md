# QR-Авторизация (WebClient)

Авторизация через QR-код позволяет мгновенно привязать клиент без использования номера телефона и SMS-кодов, используя официальное мобильное приложение Max на смартфоне (раздел «Связанные устройства» / «QR-код»).

Реализована в модуле `gomax/pkg/auth` и нативно используется в `gomax.WebClient`.

---

## 🔄 Механика работы QR-входа

1. Клиент подключается к WebSocket шлюзу `wss://api.oneme.ru/websocket`.
2. Выполняется запрос создания сессии QR-входа: `OpAuthRequest` (опкод 17) с параметром `{"type": "QR"}`.
3. Сервер возвращает глубокую ссылку (Deep Link) вида `max://login?token=...` или HTTPS-ссылку.
4. Вызывается метод `QrHandler.HandleQr(ctx, qrURL)`.
5. Запускается фоновый поллинг статуса: каждые 2 секунды отправляется `OpAuth` (опкод 18) с `{"type": "POLL_QR"}`.
6. Как только пользователь подтверждает вход на смартфоне, сервер возвращает сессионный `token` и `userId`.

---

## 🧩 Интерфейс `QrHandler`

```go
type QrHandler interface {
    HandleQr(ctx context.Context, qrURL string) error
}
```

Стандартная реализация `ConsoleQrHandler` печатает ссылку в консоль.

---

## 🎨 Реализация отрисовки QR-кода прямо в терминале

Для удобства пользователя ссылку можно превратить в графический QR-код прямо в ANSI-терминале с помощью сторонней библиотеки (например, `github.com/skip2/go-qrcode`):

```go
package main

import (
	"context"
	"fmt"

	"github.com/skip2/go-qrcode"
	"gomax"
	"gomax/pkg/auth"
)

// TerminalQrHandler рисует ASCII QR-код в консоли
type TerminalQrHandler struct{}

func (h *TerminalQrHandler) HandleQr(ctx context.Context, qrURL string) error {
	fmt.Println("👉 Отсканируйте этот QR-код в приложении Max:")
	
	// Генерация ASCII-символов QR-кода
	qr, err := qrcode.New(qrURL, qrcode.Medium)
	if err != nil {
		return err
	}
	
	fmt.Println(qr.ToSmallString(false))
	fmt.Printf("Или перейдите по ссылке: %s\n\n", qrURL)
	return nil
}
```

---

## 💻 Полный пример запуска WebClient с QR-входом

```go
package main

import (
	"context"
	"fmt"
	"log"

	"gomax"
	"gomax/pkg/auth"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.SessionName = "qr_session.json"
	cfg.PersistSession = true

	// Создаем QR flow с собственным обработчиком
	qrFlow := auth.NewQrAuthFlow(&auth.ConsoleQrHandler{}, &auth.ConsolePasswordProvider{})

	webClient := gomax.NewWebClient(cfg)

	webClient.OnStart(func(ctx context.Context) error {
		fmt.Printf("🎉 Успешный вход через QR! ID: %d, Имя: %s\n", webClient.Me.ID, webClient.Me.FirstName)
		return nil
	})

	webClient.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
		if !msg.IsOutgoing {
			fmt.Printf("Получено сообщение: %s\n", msg.Text)
		}
		return nil
	})

	fmt.Println("🚀 Инициализация Web-клиента...")
	if err := webClient.Start(context.Background()); err != nil {
		log.Fatalf("Ошибка сессии WebClient: %v", err)
	}
}
```

---

## ⏱ Время жизни QR-кода

QR-код действителен в течение 120 секунд. Если за это время вход не подтвержден, цикл поллинга завершится ошибкой таймаута, и потребуется повторный запуск клиента.
