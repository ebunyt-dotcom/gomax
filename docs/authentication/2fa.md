# Двухфакторная аутентификация

GoMax поддерживает пароль 2FA во время входа и отдельные операции настройки через `Auth`/`Users`.

## 2FA во время входа

При ответе сервера с password challenge библиотека сама вызовет переданный `PasswordProvider`. Если provider не задан, пароль будет запрошен в консоли.

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79990000000"
cfg.AuthFlow = gomax.NewSmsAuthFlow(myCodeProvider, myPasswordProvider)
client := gomax.NewClient(cfg)
```

## Низкоуровневая проверка пароля

```go
result, err := client.Auth.CheckPassword(ctx, trackID, password)
```

## Настройка 2FA

Высокоуровневый вызов:

```go
err := client.Users.Set2FA(ctx, password, hint, email)
```

Для пошагового сценария:

```go
trackID, err := client.Auth.CreateAuthTrack(ctx)
err = client.Auth.SetPassword(ctx, trackID, password)
err = client.Auth.SetHint(ctx, trackID, hint)
err = client.Auth.RequestEmailCode(ctx, trackID, email)
err = client.Auth.VerifyEmailCode(ctx, trackID, code)
err = client.Auth.CommitTwoFactor(ctx, trackID, password, hint, capabilities)
```

Поля `email`, `hint` и `capabilities` зависят от сценария и требований сервера; передавайте только нужные значения.

## Безопасность

Не записывайте пароль и токен в логи. Не храните файлы сессии в репозитории.
