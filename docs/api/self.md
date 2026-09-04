# Профиль и настройки аккаунта

Сервис: `client.Self`. В примерах `ctx` и `chatID` уже объявлены.

## Профиль

### `GetSelf`

Возвращает свой профиль.

```go
me, err := client.Self.GetSelf(ctx)
```

### `ChangeProfile`

Меняет имя, фамилию, описание и token фотографии.

```go
err := client.Self.ChangeProfile(ctx, "Ivan", "Ivanov", "Описание", "")
```

### `RequestProfilePhotoUploadURL`

Возвращает URL для загрузки фотографии профиля.

```go
url, err := client.Self.RequestProfilePhotoUploadURL(ctx)
```

### `SetPresence`

Меняет признак активности клиента. Это локальная настройка для следующего
login/reconnect.

```go
client.Self.SetPresence(false)
```

### `ChangeProfileSettings`

Меняет настройки приватности и профиля. Имена ключей задаёт сервер.

```go
err := client.Self.ChangeProfileSettings(ctx, map[string]interface{}{
    "SEARCH_BY_PHONE": false,
})
```

## Папки чатов

### `GetFolders`

Возвращает папки и marker синхронизации.

```go
folders, err := client.Self.GetFolders(ctx)
```

### `CreateFolder`

Создаёт папку с указанными чатами.

```go
folder, err := client.Self.CreateFolder(ctx, "Работа", []int64{chatID})
```

### `UpdateFolder`

Меняет название и список чатов папки.

```go
folder, err := client.Self.UpdateFolder(ctx, folderID, "Проекты", []int64{chatID})
```

### `UpdateFolderWithOptions`

Меняет папку и дополнительно передаёт фильтры и options протокола.

```go
folder, err := client.Self.UpdateFolderWithOptions(ctx, folderID, "Проекты",
    []int64{chatID}, filters, options)
```

### `DeleteFolder`

Удаляет папку.

```go
err := client.Self.DeleteFolder(ctx, folderID)
```

## Сессия

### `CloseAllSessions`

Закрывает все другие устройства, оставляя текущую сессию.

```go
err := client.Self.CloseAllSessions(ctx)
```

### `Logout`

Завершает текущую сессию. После этого потребуется новый вход.

```go
err := client.Self.Logout(ctx)
```
