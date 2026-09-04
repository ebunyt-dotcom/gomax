# Диспетчеризация событий и фильтры (`dispatch`)

Пакет `gomax/pkg/dispatch` предоставляет реактивный маршрутизатор входящих событий (`Router`), поддерживающий асинхронную обработку сообщений в независимых горутинах и цепочки предикатных фильтров (`MessagePredicate`).

---

## ⚡ Модель конкурентности

1. **Неблокирующий Event Loop**: сетевой ридер (`TCPReader` / `WSReader`) считывает фреймы из сокета и немедленно передает их роутеру.
2. **Изолированные горутины**: каждый зарегистрированный обработчик вызывается в собственной горутине `go handler(ctx, msg)`. Медленный запрос или тяжелые вычисления в одном хэндлере никогда не заблокируют получение других входящих сообщений.
3. **Безопасная регистрация**: добавление обработчиков защищено `sync.RWMutex`.

---

## 📩 Регистрация обработчиков

### 1. Обработчик сообщений (`OnMessage`)
```go
client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
    if msg.IsOutgoing {
        return nil // Пропускаем свои сообщения
    }
    fmt.Printf("Получено: %s от %d\n", msg.Text, msg.SenderID)
    return nil
})
```

### 2. Обработчик готовности клиента (`OnStart`)
Вызывается один раз сразу после успешного прохождения авторизации, рукопожатия и получения профиля пользователя (`client.Me`):
```go
client.OnStart(func(ctx context.Context) error {
    fmt.Printf("🚀 Бот готов к работе! ID: %d\n", client.Me.ID)
    return nil
})
```

---

## 🔍 Предикатные фильтры (`MessagePredicate`)

Предикат — это чистая функция вида:
```go
type MessagePredicate func(msg *types.Message) bool
```

Вы можете передавать произвольное количество предикатов в метод `router.OnMessage`. Обработчик выполнится только в том случае, если **все** переданные предикаты вернут `true`.

### Создание переиспользуемых фильтров

```go
package filters

import (
	"strings"
	"gomax/pkg/types"
)

// OnlyIncoming пропускает только входящие сообщения
func OnlyIncoming(msg *types.Message) bool {
	return !msg.IsOutgoing
}

// IsCommand проверяет, что сообщение начинается с указанной команды
func IsCommand(cmd string) func(msg *types.Message) bool {
	return func(msg *types.Message) bool {
		return strings.HasPrefix(strings.TrimSpace(msg.Text), cmd)
	}
}

// ChatIs пропускает сообщения только из конкретного чата
func ChatIs(targetChatID int64) func(msg *types.Message) bool {
	return func(msg *types.Message) bool {
		return msg.ChatID == targetChatID
	}
}
```

---

## 💻 Пример построения бота с фильтрами

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"gomax"
	"gomax/pkg/types"
)

func main() {
	cfg := gomax.DefaultConfig()
	client := gomax.NewClient(cfg)

	// Фильтр для команды /start
	isStartCmd := func(msg *types.Message) bool {
		return !msg.IsOutgoing && strings.HasPrefix(msg.Text, "/start")
	}

	// Фильтр для команды /help
	isHelpCmd := func(msg *types.Message) bool {
		return !msg.IsOutgoing && strings.HasPrefix(msg.Text, "/help")
	}

	// Регистрация роутов
	client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
		_, err := client.Messages.SendMessage(ctx, msg.ChatID, "Привет! Я бот на GoMax.", msg.ID, nil)
		return err
	}, isStartCmd)

	client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
		_, err := client.Messages.SendMessage(ctx, msg.ChatID, "Список доступных команд: /start, /help", msg.ID, nil)
		return err
	}, isHelpCmd)

	_ = client.Start(context.Background())
}
```
