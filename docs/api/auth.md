# Ручная авторизация

Сервис: `client.Auth`. Для обычного входа используйте `client.Start`: он сам
вызывает нужные методы в правильном порядке. Этот раздел нужен для ручного
управления шагами SMS, QR и 2FA.

Методы с `map[string]interface{}` возвращают сырой ответ сервера. Не выводите
такие ответы в лог: внутри могут быть token и challenge-данные.

## SMS

### `RequestCode`

Запрашивает код на номер. `mode` — значение протокола; для обычного входа
используйте `nil`.

```go
result, err := client.Auth.RequestCode(ctx, "+79990000000", nil)
```

### `SendCode`

Отправляет код, используя token из ответа `RequestCode`.

```go
result, err := client.Auth.SendCode(ctx, challengeToken, "1234")
```

### `CheckPassword`

Передаёт пароль 2FA для challenge из SMS/QR flow.

```go
result, err := client.Auth.CheckPassword(ctx, trackID, password)
```

### `ConfirmRegistration`

Завершает регистрацию, если сервер вернул registration token.

```go
result, err := client.Auth.ConfirmRegistration(ctx, "Ivan", "Ivanov", token)
```

## QR

### `RequestQR`

Создаёт QR challenge.

```go
result, err := client.Auth.RequestQR(ctx)
```

### `CheckQR`

Проверяет статус QR challenge по `trackID`.

```go
result, err := client.Auth.CheckQR(ctx, trackID)
```

### `ConfirmQR`

Завершает QR-вход после подтверждения в приложении.

```go
result, err := client.Auth.ConfirmQR(ctx, trackID)
```

### `ApproveQR`

Подтверждает QR-ссылку из уже авторизованного клиента.

```go
err := client.Auth.ApproveQR(ctx, qrLink)
```

## Настройка 2FA

### `CreateAuthTrack`

Создаёт track для настройки 2FA.

```go
trackID, err := client.Auth.CreateAuthTrack(ctx)
```

### `SetPassword`

Устанавливает пароль 2FA для track.

```go
err := client.Auth.SetPassword(ctx, trackID, password)
```

### `SetHint`

Сохраняет подсказку к паролю.

```go
err := client.Auth.SetHint(ctx, trackID, "Название домашнего города")
```

### `RequestEmailCode`

Запрашивает код подтверждения на email.

```go
err := client.Auth.RequestEmailCode(ctx, trackID, "mail@example.com")
```

### `VerifyEmailCode`

Проверяет код из email.

```go
err := client.Auth.VerifyEmailCode(ctx, trackID, "123456")
```

### `CommitTwoFactor`

Завершает настройку 2FA.

```go
err := client.Auth.CommitTwoFactor(ctx, trackID, password, hint,
    []string{"EMAIL"})
```
