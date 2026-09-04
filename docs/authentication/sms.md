# SMS-Авторизация и Авторегистрация (Вход по номеру телефона)

Модуль `gomax/pkg/auth` реализует защищенный процесс аутентификации в Max по номеру телефона с подтверждением одноразовым кодом из SMS (или звонка-сброса), автоматическим созданием новых профилей (авторегистрация), а также обработкой 2FA облачных паролей.

---

## 🔄 Жизненный цикл SMS-авторизации и регистрации

В зависимости от того, зарегистрирован ли номер в Max ранее или это новая SIM-карта, сервер Max выбирает одну из веток:

```
                  [Клиент]
                     │
         1. OpAuthRequest (опкод 17)
                     │
                     ▼
             [Сервер Max] ──► SMS-код на телефон
                     │
         2. OpAuth (опкод 18, код из SMS)
                     │
         ┌───────────┴───────────┐
         ▼                       ▼
 [Существующий аккаунт]   [Новый номер (не зарегистрирован)]
         │                       │
   Возвращает `token`      Возвращает `registerToken`
         │                       │
         │                3. OpAuthConfirm (опкод 23)
         │                   Имя, Фамилия, registerToken
         │                       │
         │                 Возвращает финальный `token`
         ▼                       ▼
   Авторизован!             Зарегистрирован и авторизован!
```

1. **Инициализация сессии**: клиент подключается к серверу и отправляет `OpSessionInit` (опкод 6).
2. **Запрос кода (`OpAuthRequest`, опкод 17)**: на указанный номер отправляется SMS-код.
3. **Получение кода**: вызывается метод `CodeProvider.GetCode(ctx)`.
4. **Отправка кода (`OpAuth`, опкод 18)**: код отправляется на проверку.
5. **Проверка статуса ответа**:
   * **Если номер уже зарегистрирован**: сервер возвращает постоянный `token`.
   * **Если аккаунт защищен 2FA**: сервер запрашивает пароль, вызывается `PasswordProvider.GetPassword(ctx)`.
   * **Если номер НОВЫЙ (не зарегистрирован)**: сервер возвращает временный `registerToken` (тип `REGISTER`). В этом случае GoMax автоматически запускает этап подтверждения регистрации (`OpAuthConfirm`, опкод 23), передавая имя и фамилию пользователя, и получает постоянный сессионный `token`.

---

## 🆕 Авторегистрация новых аккаунтов (Auto-Registration)

В GoMax поддержка авторегистрации встроена нативно и **включена по умолчанию** (`AutoRegister: true`).

### Способы указания данных нового профиля:

### Вариант 1: Автоматическая генерация имен (Zero-Config)
Если вы не задали имя вручную, GoMax автоматически сгенерирует реалистичное славянское имя и фамилию (например, «Александр Смирнов», «Дмитрий Кузнецов» и др.) через `RandomRegistrationProvider`.

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79991234567"
cfg.AutoRegister = true // Включено по умолчанию

client := gomax.NewClient(cfg)
```

### Вариант 2: Явное задание имени через `RegistrationConfig`
Если для нового аккаунта требуются конкретные имя и фамилия:

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79991234567"
cfg.Registration = &gomax.RegistrationConfig{
    FirstName: "Иван",
    LastName:  "Петров", // Необязательно
}

client := gomax.NewClient(cfg)
```

> [!NOTE]
> Для уже существующих аккаунтов параметр `Registration` просто игнорируется, и вход происходит по стандартному токену без изменения профиля.

### Вариант 3: Кастомный провайдер имен `RegistrationProvider`
Для интеграции с генераторами персонажей, базами данных или спарсенными профилями:

```go
type CustomRegistrationProvider struct{}

func (p *CustomRegistrationProvider) GetRegistrationNames(ctx context.Context, phone string) (firstName, lastName string, err error) {
    // Получение имени из БД или генератора
    return "Сергей", "Волков", nil
}

// Подключение:
smsFlow := auth.NewSmsAuthFlow(codeProvider, pwdProvider)
smsFlow.RegistrationProvider = &CustomRegistrationProvider{}
cfg.AuthFlow = *smsFlow
```

---

## 🧩 Интерфейс `CodeProvider`

Для получения SMS-кода используется интерфейс `CodeProvider`. По умолчанию активен `ConsoleCodeProvider`, запрашивающий код в консоли:

```go
type CodeProvider interface {
    GetCode(ctx context.Context) (string, error)
}
```

---

## 🛠 Реализация для SMS-активаторов (SMS-Activate, Vak-SMS, OnlineSIM)

В промышленных фермах и масс-регистраторах код берется из API смс-сервиса:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gomax"
	"gomax/pkg/auth"
)

// SmsActivateProvider опрашивает API sms-activate до получения кода
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
			// Ответ формата: STATUS_OK:123456
			if strings.HasPrefix(status, "STATUS_OK:") {
				return strings.TrimPrefix(status, "STATUS_OK:"), nil
			}
		}
	}
}
```

---

## 💻 Полный пример: Полностью автоматическая регистрация нового номера

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"gomax"
	"gomax/pkg/auth"
)

func main() {
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79998887766" // Новый номер для регистрации
	cfg.SessionName = "new_registered_bot.json"

	// 1. Настройка имени для регистрации
	cfg.Registration = &gomax.RegistrationConfig{
		FirstName: "Максим",
		LastName:  "Разработчик",
	}

	// 2. Подключение провайдера кодов (например консольный или API)
	smsProvider := &auth.ConsoleCodeProvider{}
	pwdProvider := &auth.ConsolePasswordProvider{}

	cfg.AuthFlow = *auth.NewSmsAuthFlow(smsProvider, pwdProvider)

	client := gomax.NewClient(cfg)

	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("🎉 Аккаунт успешно создан и авторизован!\n")
		fmt.Printf("ID: %d, Имя: %s %s\n", client.Me.ID, client.Me.FirstName, client.Me.LastName)
		return nil
	})

	if err := client.Start(context.Background()); err != nil {
		log.Fatalf("Ошибка регистрации: %v", err)
	}
}
```

---

## ⚠️ Возможные ошибки

* `confirm registration failed`: отклонение имени сервером (содержит спецсимволы или мат) либо истек `registerToken`.
* `registration token received: RegistrationConfig or AutoRegister is required`: флаг `AutoRegister` был отключен, а `Registration` не передан.
* `submit auth code failed: INVALID_CODE`: введен неправильный SMS-код.
* `request auth code failed: FLOOD_WAIT`: слишком частые запросы кодов на один номер. Рекомендуется сменить номер или подождать 15 минут.
