# SQLite Хранилище сессий (SqliteStore)

`SqliteStore` — это высокопроизводительное реляционное хранилище сессий, входящее в пакет `gomax/pkg/session`. Оно предназначено для промышленной эксплуатации, систем с параллельным запуском десятков и сотен аккаунтов, предотвращая фрагментацию файловой системы и обеспечивая атомарность транзакций (ACID).

---

## 🗄 Структура таблицы базы данных

При инициализации `SqliteStore` автоматически создает таблицу `max_sessions`, если она еще не создана:

```sql
CREATE TABLE IF NOT EXISTS max_sessions (
    phone TEXT PRIMARY KEY,
    token TEXT NOT NULL,
    device_id TEXT,
    mt_instance_id TEXT,
    chats_sync INTEGER,
    contacts_sync INTEGER,
    drafts_sync INTEGER,
    presence_sync INTEGER,
    config_hash TEXT
);
```

### Назначение полей

* `phone`: основной ключ, номер телефона аккаунта в международном формате.
* `token`: постоянный сессионный токен авторизации Max API.
* `device_id`: уникальный 16-символьный hex-идентификатор привязанного устройства.
* `mt_instance_id`: идентификатор инстанса транспорта.
* `chats_sync`, `contacts_sync`, `drafts_sync`, `presence_sync`: таймстемпы и маркеры инкрементальной синхронизации данных.
* `config_hash`: хэш актуальной конфигурации клиента.

---

## 🔧 Инициализация и сигнатуры методов

```go
func NewSqliteStore(db *sql.DB) (*SqliteStore, error)
```

Методы интерфейса `session.Store`:
* `SaveSession(info *SessionInfo) error`: сохраняет сессию (выполняет `INSERT ... ON CONFLICT(phone) DO UPDATE`).
* `LoadSession() (*SessionInfo, error)`: извлекает активную сессию.
* `UpdateToken(phone, newToken string) error`: атомарно обновляет токен аккаунта по номеру телефона.

---

## 💻 Пример использования: Единая база данных для сотен аккаунтов

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"gomax"
	"gomax/pkg/session"
)

func main() {
	// 1. Открываем подключение к SQLite базе данных
	db, err := sql.Open("sqlite3", "./max_farm.db?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("Ошибка подключения к SQLite: %v", err)
	}
	defer db.Close()

	// 2. Инициализируем хранилище
	store, err := session.NewSqliteStore(db)
	if err != nil {
		log.Fatalf("Ошибка создания хранилища сессий: %v", err)
	}

	// 3. Конфигурируем аккаунт
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79997778899"
	cfg.Store = store // Передаем единое хранилище

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("✅ Аккаунт %s успешно подключен из базы данных SQLite!\n", client.Me.Phone)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

---

## 💡 Рекомендации по оптимизации для ферм аккаунтов

1. **Режим WAL (Write-Ahead Logging)**: всегда добавляйте флаг `?_journal=WAL` в строку подключения к SQLite. Это позволяет сотням горутин параллельно читать сессии без блокировок.
2. **Busy Timeout**: параметр `_busy_timeout=5000` предотвращает ошибки `database is locked` при одновременной записи нескольких токенов.
