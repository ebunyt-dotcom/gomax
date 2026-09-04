# Свой профиль и настройки

Сервис доступен как `client.Self`.

Пример профиля и списка чатов: [`examples/profile_and_chats/main.go`](https://github.com/ebunyt-dotcom/gomax/blob/main/examples/profile_and_chats/main.go).

## Методы

| Метод | Назначение |
|---|---|
| `GetSelf` | Получить свой профиль |
| `ChangeProfile` | Изменить имя, описание и photo token |
| `RequestProfilePhotoUploadURL` | Получить upload URL для аватара |
| `SetPresence` | Изменить `Interactive` клиента |
| `ChangeProfileSettings` | Изменить настройки приватности |
| `GetFolders` | Получить папки чатов |
| `CreateFolder` | Создать папку |
| `UpdateFolder` / `UpdateFolderWithOptions` | Изменить папку |
| `DeleteFolder` | Удалить папку |
| `CloseAllSessions` | Закрыть другие устройства |
| `Logout` | Завершить текущую сессию |

## Профиль

```go
profile, err := client.Self.GetSelf(ctx)
err = client.Self.ChangeProfile(ctx, "Ivan", "Ivanov", "Описание", "")
```

Для нового аватара сначала используйте `Uploads.UploadPhotoWithOptions(..., true)`, затем передайте полученный token в `ChangeProfile`.

## Presence

```go
client.Self.SetPresence(false)
```

Это локально меняет `Interactive`, который будет отправлен при следующем login/reconnect.

## Папки

```go
folder, err := client.Self.CreateFolder(ctx, "Работа", []int64{chatID})
folder, err = client.Self.UpdateFolder(ctx, folder.ID, "Проекты", []int64{chatID, anotherChatID})
err = client.Self.DeleteFolder(ctx, folder.ID)
```

## Завершение сессий

`CloseAllSessions` закрывает другие устройства, а `Logout` завершает текущий сеанс. Не вызывайте их случайно: после logout потребуется новая авторизация.
