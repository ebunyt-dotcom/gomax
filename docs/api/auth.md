# Низкоуровневая авторизация

Сервис `client.Auth` нужен, если вы хотите управлять отдельными шагами auth вручную. Для обычного приложения достаточно `client.Start`.

## Методы

| Метод | Назначение |
|---|---|
| `RequestCode` | Запросить SMS-код |
| `SendCode` | Проверить SMS-код |
| `CheckPassword` | Завершить password challenge |
| `RequestQR` / `CheckQR` / `ConfirmQR` | Управлять QR flow вручную |
| `ApproveQR` | Подтвердить QR из авторизованного клиента |
| `CreateAuthTrack` | Создать track для настройки 2FA |
| `SetPassword`, `SetHint` | Подготовить 2FA |
| `RequestEmailCode`, `VerifyEmailCode` | Проверить email-код |
| `CommitTwoFactor` | Завершить настройку 2FA |
| `ConfirmRegistration` | Завершить регистрацию |

Методы возвращают `map[string]interface{}` для сохранения полной структуры ответа сервера. Встроенные `SmsAuthFlow` и `QrAuthFlow` уже выполняют правильный порядок вызовов.

```go
result, err := client.Auth.RequestQR(ctx)
status, err := client.Auth.CheckQR(ctx, trackID)
result, err = client.Auth.ConfirmQR(ctx, trackID)
```

Не логируйте содержимое ответов auth: в них могут находиться токены и challenge-данные.
