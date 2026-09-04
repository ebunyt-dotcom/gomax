# Файловое хранилище JSON (FileStore)

`FileStore` — это простое и надежное файловое хранилище сессий, сериализующее состояние аккаунта в локальный JSON-файл. Используется по умолчанию в `gomax.DefaultConfig()`.

---

## 📁 Структура сохраняемого JSON-файла

Файл сессии создается по пути `filepath.Join(WorkDir, SessionName)` с правами доступа `0600` (чтение и запись только текущему пользователю ОС):

```json
{
  "token": "a8f3...4bc2",
  "device_id": "4a1f8c9b2e3d5a6f",
  "phone": "+79991112233",
  "mt_instance_id": "mt-core-01",
  "user_agent": {
    "device_type": "android",
    "app_version": "2.4.0",
    "build_number": 2400,
    "os_version": "14",
    "device_name": "Google Pixel 8"
  },
  "sync": {
    "chats_sync": 1725459200000,
    "contacts_sync": 1725459200000,
    "drafts_sync": 0,
    "presence_sync": 1725459200000,
    "config_hash": "e3b0c44298fc1c149afbf4c8996fb924"
  }
}
```

---

## 🔧 Сигнатуры и конструктор

```go
func NewFileStore(workDir, sessionName string) *FileStore
```

* Если `sessionName` передан пустой строкой, по умолчанию используется имя `"session.json"`.
* Директория `workDir` создается автоматически рекурсивно (`os.MkdirAll`) с правами `0755`.

### Реализованные методы интерфейса `session.Store`
* `SaveSession(info *SessionInfo) error`: сериализует структуру с красивым форматированием (`json.MarshalIndent`) и атомарно записывает на диск.
* `LoadSession() (*SessionInfo, error)`: читает файл. Если файл отсутствует, возвращает `nil, nil` без ошибки, что сигнализирует клиенту о необходимости первоначальной авторизации.
* `UpdateToken(phone, newToken string) error`: обновляет токен в существующем файле сессии.

---

## 💻 Пример использования

```go
package main

import (
	"context"
	"fmt"
	"log"

	"gomax"
	"gomax/pkg/session"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79991112233"
	
	// Явное указание директории и имени файла сессии
	cfg.WorkDir = "./my_sessions"
	cfg.SessionName = "bot_79991112233.json"
	
	// Или явное создание FileStore
	cfg.Store = session.NewFileStore("./my_sessions", "bot_79991112233.json")

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		fmt.Println("Клиент успешно подключился, сессия записана в JSON!")
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```
