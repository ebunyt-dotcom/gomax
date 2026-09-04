# Справочник конфигурации (Config)

Конфигурация клиента задается структурой `gomax.Config` (псевдоним `client.Config`). Она управляет сетевыми адресами, транспортом, прокси-серверами, механизмами аутентификации и хранилищами сессий.

---

## 📌 Определение структуры `Config`

```go
type Config struct {
	Phone          string            // Номер телефона в международном формате (например, "+79991234567")
	WorkDir        string            // Рабочая директория для кэша и файлов сессий (по умолчанию "cache")
	SessionName    string            // Имя файла сессии (по умолчанию "main.json")
	Host           string            // Хост Max API для TCP (по умолчанию "api2.oneme.ru")
	Port           int               // Порт Max API для TCP (по умолчанию 443)
	URL            string            // WebSocket URL для WebClient (по умолчанию "wss://api.oneme.ru/websocket")
	UseSSL         bool              // Использовать TLS шифрование для TCP (по умолчанию true)
	Proxy          string            // URL прокси-сервера (socks5://, socks5h://, http://, https://)
	DeviceID       string            // 16-символьный hex-идентификатор устройства (генерируется случайно при отсутствии)
	Token          string            // Готовый токен авторизации (если уже получен ранее)
	PersistSession bool              // Сохранять ли сессию на диск (по умолчанию true)
	Reconnect      bool              // Автоматически восстанавливать соединение при разрывах (по умолчанию true)
	ReconnectDelay time.Duration     // Пауза перед попыткой повторного подключения (по умолчанию 3s)

	Store          session.Store     // Кастомная реализация хранилища сессий (SQLite, File, Memory)
	AuthFlow       auth.SmsAuthFlow  // Кастомная логика SMS/2FA авторизации
	Registration   *RegistrationConfig // Данные профиля для авторегистрации нового номера
	AutoRegister   bool              // Автоматически регистрировать новые аккаунты (по умолчанию true)
}
```

---

## ⚙️ Значения по умолчанию (`DefaultConfig`)

Функция `gomax.DefaultConfig()` возвращает базовую конфигурацию для подключения к боевой инфраструктуре Max:

```go
cfg := gomax.DefaultConfig()
```

| Поле | Значение по умолчанию | Описание |
| :--- | :--- | :--- |
| `Host` | `"api2.oneme.ru"` | Основной сервер RPC TCP |
| `Port` | `443` | Порт с защитой TLS |
| `URL` | `"wss://api.oneme.ru/websocket"` | Точка подключения WebSocket |
| `UseSSL` | `true` | Защищенное соединение TLS |
| `WorkDir` | `"cache"` | Папка для хранения сессий |
| `SessionName` | `"main.json"` | Имя файла сессии по умолчанию |
| `PersistSession` | `true` | Сессия сохраняется автоматически |
| `Reconnect` | `true` | Включено авто-переподключение |
| `ReconnectDelay` | `3 * time.Second` | Задержка между реконнектами |
| `AutoRegister` | `true` | Авторегистрация новых номеров |
| `Registration` | `nil` | При `nil` имена генерируются автоматически |

---

## 🌐 Настройка прокси (SOCKS5 / HTTP)

GoMax нативно поддерживает работу через прокси как для бинарного TCP-транспорта, так и для WebSocket. Это критически важно при мультиаккаунтинге для изоляции сетевых отпечатков.

### SOCKS5 Прокси с авторизацией

```go
cfg := gomax.DefaultConfig()
cfg.Phone = "+79991234567"
cfg.Proxy = "socks5://login:password@192.168.1.100:1080"
```

### HTTP/HTTPS Прокси

```go
cfg := gomax.DefaultConfig()
cfg.Proxy = "http://username:secret@proxy.example.com:3128"
```

---

## 🛡 Идентификатор устройства (`DeviceID`) и цифровой отпечаток

Поле `DeviceID` представляет собой 16-значную hex-строку (8 случайных байт). Если поле не задано, GoMax сгенерирует криптографически стойкий идентификатор автоматически.

```go
cfg.DeviceID = "4a1f8c9b2e3d5a6f"
```

На основе `DeviceID` и полученного от сервера `callsSeed` генератор отпечатков (`fingerprint.FingerprintGenerator`) эмулирует реальный 96-байтный Android SHA-256 APK/dex отпечаток приложения Max.

---

## 🔄 Настройка политики повторных подключений (Reconnection)

Если сеть нестабильна или сервер сбрасывает соединение (например, при плановом рестарте), GoMax автоматически переподключается с сохранением всех зарегистрированных обработчиков `OnMessage` и `OnStart`.

```go
cfg := gomax.DefaultConfig()
cfg.Reconnect = true                    // Включить авто-реконнект
cfg.ReconnectDelay = 5 * time.Second    // Задержка между попытками
```

Для отключения авто-реконнекта:
```go
cfg.Reconnect = false // Ошибка вернется из client.Start(ctx) немедленно при первом дисконнекте
```

---

## 💾 Подключение кастомных хранилищ сессий

Вы можете переопределить интерфейс `session.Store`:

### SQLite Хранилище (для сотен аккаунтов)
```go
db, err := sql.Open("sqlite3", "./accounts.db")
sqliteStore, err := session.NewSqliteStore(db)

cfg.Store = sqliteStore
```

### Хранилище в памяти (RAM Only, без записи на диск)
```go
cfg.PersistSession = false
cfg.Store = session.NewInMemoryStore()
```
