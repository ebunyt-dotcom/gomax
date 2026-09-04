# Хранилище в оперативной памяти (InMemoryStore)

`InMemoryStore` сохраняет сессионные данные исключительно в оперативной памяти (RAM) запущенного процесса. Данные не записываются на диск и уничтожаются при остановке приложения.

---

## 🎯 Сценарии применения

* **Serverless и микросервисы**: когда токен авторизации получается динамически из защищенного внешнего хранилища (HashiCorp Vault, AWS Secrets Manager, Redis).
* **Одноразовые скрипты и чекеры**: проверка валидности номеров, однократная рассылка или парсинг без сохранения следов на сервере.
* **Безопасность (No-Disk Policy)**: работа на серверах с жесткими требованиями к отсутствию персистентных конфиденциальных данных на накопителе.

---

## 🔧 Сигнатура конструктора

```go
func NewInMemoryStore() *InMemoryStore
```

Хранилище полностью потокобезопасно и защищено `sync.RWMutex`.

---

## 💻 Пример использования с предварительно известным токеном

Если у вас уже есть токен авторизации (например, сохраненный в переменной окружения или полученный из базы данных), вы можете подключить клиент без повторного прохождения SMS-авторизации:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gomax"
	"gomax/pkg/session"
)

func main() {
	token := os.Getenv("MAX_AUTH_TOKEN")
	if token == "" {
		token = "a8f3d1e2b4c5...your_token"
	}

	cfg := gomax.DefaultConfig()
	cfg.Token = token
	cfg.PersistSession = false
	cfg.Store = session.NewInMemoryStore()

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("✅ Авторизован в RAM без записи на диск! User: %s (ID: %d)\n", client.Me.FirstName, client.Me.ID)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```
