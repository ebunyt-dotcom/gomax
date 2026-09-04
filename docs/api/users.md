# Пользователи и контакты

Сервис доступен как `client.Users`.

## Методы

| Метод | Назначение |
|---|---|
| `GetUser`, `GetUsers` | Получить один или несколько профилей |
| `FetchUsers` | Пакетное получение без локального cache-контракта |
| `GetCachedUser` | Совместимый alias получения пользователя |
| `SearchUsers` | Поиск по имени или запросу |
| `SearchByPhone`, `GetUserByPhone` | Поиск по номеру |
| `GetContacts` | Получить контакты |
| `GetSelf` | Получить свой профиль |
| `AddContact`, `AddContactByID` | Добавить контакт |
| `UpdateContact` | Изменить имя контакта |
| `RemoveContact` | Удалить контакт |
| `ImportContacts` | Импортировать номера и имена |
| `GetChatID` | Получить детерминированный ID диалога |
| `GetSessions`, `GetActiveSessions` | Получить устройства аккаунта |
| `CloseSession` | Закрыть удалённую сессию |
| `Set2FA` | Настроить 2FA |

## Поиск пользователя

```go
user, err := client.Users.SearchByPhone(ctx, "+79990000000")
users, err := client.Users.SearchUsers(ctx, "Иван")
```

## Контакты

```go
err := client.Users.AddContact(ctx, userID, "Ivan", "Ivanov", phone)
err = client.Users.UpdateContact(ctx, userID, "Новое имя", "")
err = client.Users.RemoveContact(ctx, userID)
```

`AddContactByID` возвращает созданный `User`; `AddContact` возвращает только ошибку.

## Сессии устройств

```go
sessions, err := client.Users.GetSessions(ctx)
for _, item := range sessions {
    fmt.Println(item.ID, item.Device, item.IP)
}
```

Закрывайте только те устройства, которыми вы управляете. Текущая сессия может быть отключена сервером после этой операции.
