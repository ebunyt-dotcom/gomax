# SMS-Авторизация (Вход по номеру телефона)

Модуль `gomax/pkg/auth` реализует защищенный процесс аутентификации в Max по номеру телефона с подтверждением одноразовым кодом из SMS (или звонка-сброса), а также автоматическим запросом 2FA-пароля в случае включенной двухфакторной защиты.

---

## 🔄 Жизненный цикл SMS-авторизации

1. **Инициализация сессии**: клиент подключается к серверу и отправляет `OpSessionInit` (опкод 6).
2. **Запрос кода (`OpAuthRequest`, опкод 17)**: на указанный номер отправляется SMS-код.
3. **Получение кода**: вызывается метод `CodeProvider.GetCode(ctx)`.
4. **Отправка кода (`OpAuth`, опкод 18)**: код отправляется на проверку.
5. **Проверка 2FA**: если аккаунт защищен облачным паролем, сервер возвращает требование пароля, и вызывается `PasswordProvider.GetPassword(ctx)`.
6. **Получение токена**: сервер выдает постоянный сессионный токен (`token`) и ID пользователя (`userId`), которые сохраняются в хранилище сессий.

---

## 🧩 Интерфейс `CodeProvider`

По умолчанию используется `ConsoleCodeProvider`, ожидающий ввода кода из стандартного потока ввода консоли (`os.Stdin`).

```go
type CodeProvider interface {
    GetCode(ctx context.Context) (string, error)
}
```

---

## 🛠 Реализация кастомного провайдера SMS (API сервисов приема SMS)

В автоматизированных системах (масс-регистраторы, фермы аккаунтов) код необходимо получать из API сервисов (SMS-Activate, Vak-SMS, OnlineSIM и др.). Для этого достаточно реализовать интерфейс `CodeProvider`:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gomax"
	"gomax/pkg/auth"
)

// SmsActivateProvider реализует auth.CodeProvider для сервиса активаций
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
			// Пример запроса статуса к API sms-activate
			url := fmt.Sprintf("https://api.sms-activate.org/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", p.ApiKey, p.ActivationID)
			resp, err := p.Client.Get(url)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			var status string
			fmt.Fscanf(resp.Body, &status)

			// Если код получен: STATUS_OK:123456
			if len(status) > 10 && status[:9] == "STATUS_OK" {
				code := status[10:]
				return code, nil
			}
		}
	}
}
```

---

## 💻 Полный пример настройки клиента с кастомным `SmsAuthFlow`

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
	cfg.SessionName = "sms_account.json"

	// Подключаем наш SMS-провайдер и стандартный парольный провайдер
	customCodeProvider := &auth.ConsoleCodeProvider{}
	customPwdProvider := &auth.ConsolePasswordProvider{}

	cfg.AuthFlow = *auth.NewSmsAuthFlow(customCodeProvider, customPwdProvider)

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("✅ Бот авторизован! Добро пожаловать, %s!\n", client.Me.FirstName)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatalf("Ошибка входа: %v", err)
	}
}
```

---

## ⚠️ Возможные ошибки

* `submit auth code failed: INVALID_CODE`: указан неверный или просроченный код подтверждения.
* `request auth code failed: FLOOD_WAIT`: превышен лимит частых запросов SMS на данный номер. Рекомендуется ожидать от 5 до 15 минут.
* `phone number required for initial authentication`: передан пустой `cfg.Phone` при отсутствии сохраненной сессии в хранилище.
