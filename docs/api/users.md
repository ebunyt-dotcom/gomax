# Пользователи и контакты

Сервис: `client.Users`. В примерах `ctx`, `userID` и номера телефонов уже
объявлены.

Полные сигнатуры находятся в [полном справочнике](reference.md). Каждый
пример ниже — отдельный фрагмент; импорты и создание клиента не повторяются.

## Получение пользователей

### `GetUser`

Возвращает пользователя по ID.

```go
user, err := client.Users.GetUser(ctx, userID)
```

### `GetUsers`

Возвращает пользователей по нескольким ID.

```go
users, err := client.Users.GetUsers(ctx, []int64{userID, anotherUserID})
```

### `FetchUsers`

Пакетно получает пользователей. Это имя сохранено для совместимости с PyMax.

```go
users, err := client.Users.FetchUsers(ctx, []int64{userID})
```

### `GetCachedUser`

Совместимый alias получения одного пользователя.

```go
user, err := client.Users.GetCachedUser(ctx, userID)
```

### `SearchUsers`

Ищет пользователей по имени или строке поиска.

```go
users, err := client.Users.SearchUsers(ctx, "Иван")
```

### `SearchByPhone`

Ищет пользователя по номеру телефона.

```go
user, err := client.Users.SearchByPhone(ctx, "+79990000000")
```

### `GetUserByPhone`

То же назначение, отдельное имя для явного сценария поиска по номеру.

```go
user, err := client.Users.GetUserByPhone(ctx, "+79990000000")
```

### `GetSelf`

Возвращает профиль текущего аккаунта.

```go
me, err := client.Users.GetSelf(ctx)
```

### `GetContacts`

Возвращает контакты аккаунта.

```go
contacts, err := client.Users.GetContacts(ctx)
```

### `GetChatID`

Вычисляет ID личного диалога для двух пользователей. Сетевой запрос не делает.

```go
chatID := client.Users.GetChatID(ctx, firstUserID, secondUserID)
```

## Контакты

### `AddContact`

Добавляет пользователя в контакты и задаёт имя.

```go
err := client.Users.AddContact(ctx, userID, "Ivan", "Ivanov", "+79990000000")
```

### `AddContactByID`

Добавляет контакт по ID и возвращает созданного пользователя.

```go
user, err := client.Users.AddContactByID(ctx, userID)
```

### `UpdateContact`

Меняет имя существующего контакта.

```go
err := client.Users.UpdateContact(ctx, userID, "Новое имя", "")
```

### `RemoveContact`

Удаляет контакт.

```go
err := client.Users.RemoveContact(ctx, userID)
```

### `ImportContacts`

Импортирует контакты. Ключ — номер, значение — имя.

```go
users, err := client.Users.ImportContacts(ctx, map[string]string{
    "+79990000000": "Ivan Ivanov",
})
```

## Сессии и 2FA

### `GetSessions`

Возвращает устройства и активные сессии аккаунта.

```go
sessions, err := client.Users.GetSessions(ctx)
```

### `GetActiveSessions`

Возвращает только активные сессии.

```go
sessions, err := client.Users.GetActiveSessions(ctx)
```

### `CloseSession`

Закрывает удалённую сессию по ID. Не передавайте ID текущего устройства.

```go
err := client.Users.CloseSession(ctx, sessionID)
```

### `Set2FA`

Настраивает пароль 2FA и связанные параметры.

```go
err := client.Users.Set2FA(ctx, "пароль", "подсказка", "mail@example.com")
```
