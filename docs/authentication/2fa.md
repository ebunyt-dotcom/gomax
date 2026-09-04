# Двухфакторная аутентификация (2FA / Облачный пароль)

Двухфакторная защита (2FA) в Max представляет собой дополнительный пароль учетной записи, который запрашивается сервером после успешного ввода SMS-кода или при авторизации на новых устройствах.

---

## 🔒 Перехват запроса 2FA

В процессе выполнения `SmsAuthFlow` при передаче SMS-кода сервер может вернуть ответ с кодом ошибки или статусом необходимости ввода пароля (`NEED_PASSWORD` / `2FA`).

Библиотека GoMax автоматически перехватывает этот статус и обращается к интерфейсу `PasswordProvider`:

```go
type PasswordProvider interface {
    GetPassword(ctx context.Context) (string, error)
}
```

---

## 🛡 Провайдеры пароля

### 1. Стандартный ввод в консоли (`ConsolePasswordProvider`)
Запрашивает ввод пароля через терминал:
```go
pwdProvider := &auth.ConsolePasswordProvider{}
```

### 2. Статический пароль (`StaticPasswordProvider`)
Используется в скриптах автоматизации, когда пароль известен заранее:

```go
package main

import (
	"context"
	"gomax/pkg/auth"
)

type StaticPasswordProvider struct {
	Password string
}

func (p *StaticPasswordProvider) GetPassword(ctx context.Context) (string, error) {
	return p.Password, nil
}
```

---

## ⚙️ Установка и изменение 2FA на аккаунте

Вы можете программно установить или обновить двухфакторный пароль на текущем аккаунте с помощью метода сервиса пользователей `client.Users.Set2FA`.

### Сигнатура метода
```go
func (s *UserService) Set2FA(ctx context.Context, password, hint, email string) error
```

* **Опкод**: `OpAuthSet2Fa` (111).
* **Параметры**:
  * `password` — новый облачный пароль (не пустая строка).
  * `hint` — подсказка для восстановления пароля (необязательно, можно `""`).
  * `email` — резервный адрес электронной почты для сброса пароля (необязательно, можно `""`).
* **Возвращает**: `error` в случае ошибки или отклонения сервером.

### Пример установки 2FA пароля

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
	cfg.SessionName = "my_account.json"

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		fmt.Println("Установка облачного пароля...")
		err := client.Users.Set2FA(ctx, "SuperSecurePass2026!", "Детство на Марсе", "backup@example.com")
		if err != nil {
			return fmt.Errorf("ошибка установки 2FA: %w", err)
		}

		fmt.Println("✅ Двухфакторная защита успешно активирована!")
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```
