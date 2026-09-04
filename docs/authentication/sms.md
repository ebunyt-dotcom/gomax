# SMS-Авторизация (Вход по номеру телефона)

Модуль `gomax/pkg/auth` реализует процесс аутентификации в Max по номеру телефона с подтверждением одноразовым кодом из SMS (или звонка-сброса), а также обработкой 2FA облачного пароля, если на аккаунте включена двухфакторная защита.

---

## 🔄 Жизненный цикл SMS-авторизации

1. **Инициализация сессии**: клиент подключается к серверу и отправляет `OpSessionInit` (опкод 6).
2. **Запрос кода (`OpAuthRequest`, опкод 17)**: на указанный в конфигурации номер телефона отправляется SMS с проверочным кодом.
3. **Получение кода**: вызывается метод `CodeProvider.GetCode(ctx)`.
4. **Отправка кода (`OpAuth`, опкод 18)**: проверочный код передается на сервер.
5. **Проверка 2FA (если требуется)**: если аккаунт защищен облачным паролем, сервер возвращает требование пароля, и вызывается `PasswordProvider.GetPassword(ctx)`.
6. **Получение токена**: сервер выдает сессионный токен (`token`) и ID пользователя (`userId`), которые сохраняются в хранилище сессий.

---

## 🧩 Интерфейс `CodeProvider`

Для передачи кода подтверждения в библиотеку используется интерфейс `CodeProvider`:

```go
type CodeProvider interface {
    GetCode(ctx context.Context) (string, error)
}
```

### Стандартный консольный ввод (`ConsoleCodeProvider`)
Используется по умолчанию. При вызове выводит в консоль приглашение ввести код с клавиатуры:

```go
provider := &auth.ConsoleCodeProvider{}
```

---

## 🛠 Кастомный провайдер кодов (для сервисов приема SMS)

Если вы используете API сервисов активаций (SMS-Activate, Vak-SMS, OnlineSIM и др.), реализуйте интерфейс `CodeProvider`:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gomax/pkg/auth"
)

// SmsActivateProvider опрашивает API sms-сервиса до получения кода
type SmsActivateProvider struct {
	ActivationID string
	ApiKey       string
	Client       *http.Client
}

func (p *SmsActivateProvider) GetCode(ctx context.Context) (string, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			url := fmt.Sprintf("https://api.sms-activate.org/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", p.ApiKey, p.ActivationID)
			resp, err := p.Client.Get(url)
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			status := string(data)
			// Если сервис вернул код: STATUS_OK:123456
			if strings.HasPrefix(status, "STATUS_OK:") {
				return strings.TrimPrefix(status, "STATUS_OK:"), nil
			}
		}
	}
}
```

---

## 💻 Полный пример использования `SmsAuthFlow`

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
	cfg.Phone = "+79991234567"
	cfg.SessionName = "my_account.json"

	// Указываем провайдеры для SMS-кода и 2FA пароля
	codeProvider := &auth.ConsoleCodeProvider{}
	pwdProvider := &auth.ConsolePasswordProvider{}

	cfg.AuthFlow = *auth.NewSmsAuthFlow(codeProvider, pwdProvider)

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("✅ Авторизация успешна! Добро пожаловать, %s (ID: %d)!\n", client.Me.FirstName, client.Me.ID)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatalf("Ошибка авторизации: %v", err)
	}
}
```

---

## ⚠️ Частые ошибки и их решение

* `submit auth code failed: INVALID_CODE`: указан неверный проверочный код или срок его действия истек.
* `request auth code failed: FLOOD_WAIT`: слишком частые попытки запроса кодов на данный номер. Сервер временно заблокировал отправку SMS (обычно на 10-15 минут).
* `phone number required for initial authentication`: в `cfg.Phone` передан пустой номер телефона, а в хранилище сессий нет сохраненного токена.
