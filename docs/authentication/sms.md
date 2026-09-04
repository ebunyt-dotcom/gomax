# Вход по SMS

SMS-вход используется клиентом `NewClient`.

## Минимальный вариант

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79990000000"

client := gomax.NewClient(cfg)
log.Fatal(client.Start(context.Background()))
```

При первом запуске GoMax запросит код в консоли. После успешного входа токен сохранится в сессии.

Готовый пример: [`examples/sms_login/main.go`](https://github.com/ebunyt-dotcom/gomax/blob/main/examples/sms_login/main.go).

## Что обязательно

| Настройка | Нужна |
|---|---:|
| `Phone` | Да, если нет сохранённого `Token` |
| `Token` | Альтернатива `Phone` и SMS |
| `SessionName`/`WorkDir` | Нет, есть значения по умолчанию |

Если SMS не пришло, сначала проверьте доступность номера и лимиты сервера. Не отправляйте много повторных запросов.

## Свой провайдер кода

```go
type CodeProvider interface {
    GetCode(context.Context) (string, error)
}

cfg.AuthFlow = gomax.NewSmsAuthFlow(myCodeProvider, nil)
```

Второй аргумент — провайдер пароля 2FA. `nil` включает консольный ввод.

## Регистрация нового аккаунта

Если сервер вернул регистрацию вместо обычного входа, задайте профиль:

```go
cfg.Registration = &gomax.RegistrationConfig{
    FirstName: "Ivan",
    LastName:  "Ivanov",
}
```

## Низкоуровневые методы

Для ручного сценария доступны `client.Auth.RequestCode`, `SendCode`, `CheckPassword` и `ConfirmRegistration`. В обычном приложении используйте `client.Start`: он сам выполняет порядок операций.
