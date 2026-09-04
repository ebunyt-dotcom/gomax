# Сервис пользователей и безопасности (`client.Users`)

Сервис `UserService` предоставляет функционал для работы с профилями пользователей, глобальным поиском людей, телефонной книгой (контактами), а также мониторингом и управлением активными сессиями (устройствами) и двухфакторной защитой (2FA).

Доступ к сервису осуществляется через поле `client.Users` (или `webClient.Users`).

---

## 📋 Список методов

1. [`GetUser`](#1-getuser) — получение профиля пользователя по ID
2. [`GetUsers`](#2-getusers) — пакетное получение списка профилей по массиву ID
3. [`SearchUsers`](#3-searchusers) — глобальный поиск пользователей по имени/юзернейму
4. [`GetContacts`](#4-getcontacts) — получение списка сохраненных контактов аккаунта
5. [`GetSelf`](#5-getself) — получение собственного профиля текущего пользователя
6. [`GetActiveSessions`](#6-getactivesessions) — просмотр всех активных устройств и авторизованных сессий
7. [`CloseSession`](#7-closesession) — завершение (выброс) удаленной активной сессии
8. [`Set2FA`](#8-set2fa) — установка или изменение облачного пароля двухфакторной защиты

---

### 1. `GetUser`

Получает публичные данные профиля конкретного пользователя по его уникальному ID.

```go
func (s *UserService) GetUser(
    ctx context.Context,
    userID int64,
) (*types.User, error)
```

* **Опкод протокола**: `OpContactInfo` (32).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `userID` (`int64`): идентификатор целевого пользователя.
* **Возвращаемое значение**: `(*types.User, error)` — структура пользователя (`ID`, `FirstName`, `LastName`, `Phone` и др.).

#### Пример вызова:
```go
user, err := client.Users.GetUser(ctx, 12345678)
if err != nil {
    log.Fatalf("Ошибка получения пользователя: %v", err)
}
fmt.Printf("Имя: %s %s, Телефон: %s\n", user.FirstName, user.LastName, user.Phone)
```

---

### 2. `GetUsers`

Пакетно запрашивает информацию о группе пользователей за один сетевой RPC-запрос. Рекомендуется использовать вместо цикла вызовов `GetUser` для снижения нагрузки на сокет.

```go
func (s *UserService) GetUsers(
    ctx context.Context,
    userIDs []int64,
) ([]types.User, error)
```

* **Опкод протокола**: `OpContactInfo` (32).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `userIDs` (`[]int64`): массив идентификаторов пользователей.
* **Возвращаемое значение**: `([]types.User, error)` — список профилей.

#### Пример вызова:
```go
ids := []int64{1001, 1002, 1003}
users, err := client.Users.GetUsers(ctx, ids)
if err != nil {
    log.Fatal(err)
}
for _, u := range users {
    fmt.Printf("Пользователь [%d]: %s %s\n", u.ID, u.FirstName, u.LastName)
}
```

---

### 3. `SearchUsers`

Выполняет поиск пользователей в базе Max по имени, фамилии или никнейму.

```go
func (s *UserService) SearchUsers(
    ctx context.Context,
    query string,
) ([]types.User, error)
```

* **Опкод протокола**: `OpContactSearch` (37).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `query` (`string`): поисковый запрос (минимум 3 символа).
* **Возвращаемое значение**: `([]types.User, error)` — найденные пользователи.

#### Пример вызова:
```go
results, err := client.Users.SearchUsers(ctx, "Иван Иванов")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Найдено совпадений: %d\n", len(results))
```

---

### 4. `GetContacts`

Возвращает полный список контактов из телефонной книги аккаунта.

```go
func (s *UserService) GetContacts(
    ctx context.Context,
) ([]types.User, error)
```

* **Опкод протокола**: `OpContactList` (36).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
* **Возвращаемое значение**: `([]types.User, error)` — список сохраненных контактов.

#### Пример вызова:
```go
contacts, err := client.Users.GetContacts(ctx)
if err != nil {
    log.Fatal(err)
}
for _, c := range contacts {
    fmt.Printf("Контакт: %s (%s)\n", c.FirstName, c.Phone)
}
```

---

### 5. `GetSelf`

Возвращает актуальный профиль текущего авторизованного пользователя.

```go
func (s *UserService) GetSelf(
    ctx context.Context,
) (*types.User, error)
```

* **Опкод протокола**: `OpProfile` (16).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
* **Возвращаемое значение**: `(*types.User, error)` — данные своего профиля.

#### Пример вызова:
```go
me, err := client.Users.GetSelf(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Мой аккаунт: %s (ID: %d, Phone: %s)\n", me.FirstName, me.ID, me.Phone)
```

---

### 6. `GetActiveSessions`

Запрашивает список всех активных устройств, подключенных к данной учетной записи (веб-версии, мобильные клиенты, десктоп).

```go
func (s *UserService) GetActiveSessions(
    ctx context.Context,
) ([]SessionItem, error)
```

* **Опкод протокола**: `OpSessionsInfo` (96).
* **Возвращаемая структура `SessionItem`**:
  ```go
  type SessionItem struct {
      ID         int64  `json:"id"`          // Идентификатор сессии
      Device     string `json:"device"`      // Модель устройства (напр. "iPhone 15", "Google Pixel")
      Location   string `json:"location"`    // Геолокация/страна
      Client     string `json:"client"`      // Версия клиента
      IP         string `json:"ip"`          // IP-адрес подключения
      LastActive int64  `json:"last_active"` // Таймстемп последней активности
  }
  ```

#### Пример вызова:
```go
sessions, err := client.Users.GetActiveSessions(ctx)
if err != nil {
    log.Fatal(err)
}

for _, s := range sessions {
    fmt.Printf("Сессия [%d]: Устройство: %s, IP: %s\n", s.ID, s.Device, s.IP)
}
```

---

### 7. `CloseSession`

Принудительно закрывает и разлогинивает указанную удаленную сессию устройства.

```go
func (s *UserService) CloseSession(
    ctx context.Context,
    sessionID int64,
) error
```

* **Опкод протокола**: `OpSessionsClose` (97).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `sessionID` (`int64`): идентификатор сессии, полученный из `GetActiveSessions`.
* **Возвращаемое значение**: `error`.

#### Пример вызова (закрытие всех сторонних сессий):
```go
sessions, _ := client.Users.GetActiveSessions(ctx)
for _, s := range sessions {
    // Завершаем все сессии, кроме текущей
    if s.Device != "MyGoMaxClient" {
        _ = client.Users.CloseSession(ctx, s.ID)
        fmt.Printf("Сессия %d успешно завершена\n", s.ID)
    }
}
```

---

### 8. `Set2FA`

Устанавливает или изменяет облачный мастер-пароль двухфакторной аутентификации на аккаунте.

```go
func (s *UserService) Set2FA(
    ctx context.Context,
    password, hint, email string,
) error
```

* **Опкод протокола**: `OpAuthSet2Fa` (111).
* **Параметры**:
  * `ctx` (`context.Context`): контекст запроса.
  * `password` (`string`): новый пароль.
  * `hint` (`string`): подсказка для пароля.
  * `email` (`string`): e-mail для сброса.
* **Возвращаемое значение**: `error`.
