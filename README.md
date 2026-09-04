# GoMax 👋 (Max API Go Client)

[![Go Reference](https://pkg.go.dev/badge/github.com/ebunyt-dotcom/gomax.svg)](https://pkg.go.dev/github.com/ebunyt-dotcom/gomax)
[![Go Report Card](https://goreportcard.com/badge/github.com/ebunyt-dotcom/gomax)](https://goreportcard.com/report/github.com/ebunyt-dotcom/gomax)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**GoMax** — быстрая, легкая и дружелюбная библиотека на языке Go для взаимодействия с внутренним API мессенджера Max. 

Библиотека является прямым портом популярной Python-библиотеки [PyMax](https://github.com/MaxApiTeam/PyMax) на Go. Она сохраняет привычные названия методов, протокол и логику работы, но при этом работает значительно быстрее, потребляет минимум оперативной памяти и отлично масштабируется благодаря горутинам.

---

## ✨ Возможности

- 🚀 **Высокая скорость и лёгкость**: чистый Go, минимум сторонних зависимостей, работа через родной TCP бинарный протокол или WebSocket.
- 🔐 **Простая авторизация**:
  - Вход по номеру телефона и SMS (с вводом кода прямо в консоли или через свой callback).
  - Вход через QR-код с экрана телефона (`WebClient`).
  - Поддержка двухфакторной аутентификации (2FA).
  - Автоматическое сохранение сессии в JSON-файл — при повторном запуске SMS больше не требуется!
- 💬 **Сообщения и реакции**:
  - Отправка, редактирование, удаление, ответ (reply) и пересылка сообщений.
  - Полноценная поддержка реакций (**масс-реакции**).
  - Отправка и участие в опросах (голосования).
- 👥 **Группы и каналы**:
  - Создание групп и каналов.
  - Вступление по ссылке-приглашению (`JoinChat`).
  - Добавление участников (`InviteUsersToGroup` / **инвайтинг**).
  - Настройка прав, исключение участников, получение списков пользователей.
- 👀 **Масслукинг**:
  - Отметка сообщений прочитанными (`ReadMessages`).
- 📁 **Мультимедиа**:
  - Загрузка фотографий, видео, голосовых заметок и документов через нативный HTTP Upload API.
- 🛡️ **Безопасность**:
  - Встроенный генератор цифровых отпечатков (fingerprint) Android-клиентов для предотвращения детекта.

---

## 📦 Установка

Для подключения библиотеки к вашему проекту на Go выполните:

```bash
go get github.com/ebunyt-dotcom/gomax
```

### Первый проект с нуля за 1 минуту:

1. Создайте новую папку и инициализируйте модуль:
   ```bash
   mkdir my_max_bot
   cd my_max_bot
   go mod init my_max_bot
   ```

2. Скачайте GoMax:
   ```bash
   go get github.com/ebunyt-dotcom/gomax
   ```

3. Создайте файл `main.go` и запустите бота:
   ```bash
   go run main.go
   ```

---

## 🚀 Быстрый старт: Простой эхо-бот

Создайте файл `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ebunyt-dotcom/gomax"
)

func main() {
	// 1. Создаем конфигурацию
	cfg := gomax.DefaultConfig()
	cfg.Phone = "+79991234567"          // Ваш номер телефона
	cfg.SessionName = "my_session.json" // Имя файла для сохранения сессии

	// 2. Создаем клиент
	client := gomax.NewClient(cfg)

	// 3. Хук OnStart: вызывается сразу после успешного входа
	client.OnStart(func(ctx context.Context) error {
		fmt.Printf("🎉 Успешный вход! Мой ID: %d, Имя: %s\n", client.Me.ID, client.Me.FirstName)
		return nil
	})

	// 4. Хук OnMessage: реагируем на входящие сообщения
	client.OnMessage(func(ctx context.Context, msg *gomax.Message) error {
		// Игнорируем свои же сообщения
		if msg.SenderID == client.Me.ID {
			return nil
		}

		if msg.Text != "" {
			fmt.Printf("Новое сообщение от %d: %s\n", msg.SenderID, msg.Text)
			// Отвечаем эхом
			_, err := client.Messages.SendMessage(ctx, msg.ChatID, "Вы написали: "+msg.Text, msg.ID, nil)
			if err != nil {
				log.Printf("Ошибка отправки: %v", err)
			}
		}
		return nil
	})

	// 5. Запуск клиента
	ctx := context.Background()
	fmt.Println("Подключение к Max API...")
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Ошибка работы клиента: %v", err)
	}
}
```

### 💡 Как работает ввод кода при первом запуске:
1. При самом первом запуске библиотека автоматически отправляет запрос на получение SMS-кода на указанный номер.
2. В консоли появится запрос:
   ```text
   Enter SMS verification code: 
   ```
   Просто введите полученный код из SMS и нажмите Enter.
3. Библиотека авторизуется и сохранит сессию в файл `my_session.json`.
4. При всех последующих запусках код вводить **не потребуется** — GoMax мгновенно восстановит сессию из файла!

> 💡 **Совет:** Если вам нужно получать код не из консоли (например, через Telegram-бота, админку или веб-интерфейс), вы можете передать свой обработчик через `cfg.AuthFlow = auth.NewSmsAuthFlow(myCodeProvider, nil)`. Подробнее в [руководстве по SMS-авторизации](docs/authentication/sms.md).

---

## ⚡ Примеры: Масс-действия

Библиотека оптимизирована для параллельной автоматизации:

```go
// 1. Вступление в группу или канал по ссылке-приглашению
chat, err := client.Chats.JoinChat(ctx, "https://max.mail.ru/join/aBcDeFgHiJ")

// 2. Инвайтинг пользователей в группу по их ID
err = client.Chats.InviteUsersToGroup(ctx, targetChatID, []int64{10001, 10002, 10003}, true)

// 3. Установка быстрой реакции (масс-реакция)
err = client.Messages.AddReaction(ctx, targetChatID, messageID, "🔥")

// 4. Масслукинг: отметка сообщений прочитанными (просмотры)
err = client.Messages.ReadMessages(ctx, targetChatID, []int64{messageID})
```

Больше готовых примеров смотрите в папке [`examples/`](examples/):
- [`examples/echo_bot/`](examples/echo_bot/main.go) — классический эхо-бот
- [`examples/mass_actions/`](examples/mass_actions/main.go) — инвайтинг, реакции, вступления
- [`examples/qr_login/`](examples/qr_login/main.go) — вход по QR-коду через WebSocket

---

## 📚 Документация

В репозитории подготовлена подробная структурированная документация по каждому разделу:

| Раздел | Ссылка на руководство | Описание |
| :--- | :--- | :--- |
| 🚀 **Быстрый старт** | [docs/getting-started/quickstart.md](docs/getting-started/quickstart.md) | Первый запуск, структура проекта, базовые концепции |
| ⚙️ **Конфигурация** | [docs/getting-started/configuration.md](docs/getting-started/configuration.md) | Настройки прокси, реконнекта, SSL, рабочих директорий |
| 🏗️ **Архитектура** | [docs/getting-started/architecture.md](docs/getting-started/architecture.md) | Устройство TCP/WS транспорта, протокола и сервисов |
| 📱 **SMS Авторизация** | [docs/authentication/sms.md](docs/authentication/sms.md) | Вход по номеру телефона, консольный ввод, кастомные провайдеры |
| 📷 **QR Авторизация** | [docs/authentication/qr.md](docs/authentication/qr.md) | Вход по QR-коду через WebClient |
| 🔑 **2FA (Двухфакторка)** | [docs/authentication/2fa.md](docs/authentication/2fa.md) | Поддержка паролей двухфакторной аутентификации |
| 💾 **Сессии** | [docs/authentication/sessions.md](docs/authentication/sessions.md) | Хранение сессий (FileStore, InMemoryStore, кастомные хранилища) |
| ✉️ **Сообщения** | [docs/services/messages.md](docs/services/messages.md) | Отправка текста, пересылка, редактирование, удаление, опросы |
| 👥 **Чаты и Каналы** | [docs/services/chats.md](docs/services/chats.md) | Создание, настройки, выход, участники, управление правами |
| ➕ **Инвайтинг** | [docs/services/inviting.md](docs/services/inviting.md) | Добавление пользователей в группы и каналы |
| ⚡ **Масс-действия** | [docs/services/mass_actions.md](docs/services/mass_actions.md) | Масс-реакции, масслукинг, параллельное выполнение |
| 📁 **Загрузка файлов** | [docs/services/uploads.md](docs/services/uploads.md) | Загрузка фото, видео, аудио и документов |
| 👤 **Пользователи** | [docs/services/users.md](docs/services/users.md) | Поиск пользователей, профили, контакты |
| 🔔 **События** | [docs/events/handlers.md](docs/events/handlers.md) | Регистрация хендлеров, события OnStart, OnMessage |
| 🎯 **Фильтры** | [docs/events/filters.md](docs/events/filters.md) | Фильтрация входящих сообщений (текст, команды, чаты) |

---

## 🤝 Вклад в проект (Contributing)

Мы рады любым предложениям и улучшениям! Если вы нашли ошибку, хотите добавить новый функционал или улучшить документацию:
1. Создайте форк репозитория.
2. Создайте ветку для ваших изменений (`git checkout -b feature/cool-feature`).
3. Зафиксируйте изменения (`git commit -m 'feat: add cool feature'`).
4. Отправьте ветку (`git push origin feature/cool-feature`).
5. Откройте **Pull Request**.

---

## ⚖️ Отказ от ответственности / Disclaimer

> [!IMPORTANT]
> Данный программный проект (**GoMax**) создан исключительно в **образовательных, исследовательских и ознакомительных целях** (reverse-engineering, изучение сетевых протоколов и обеспечение интероперабельности программного обеспечения).

- **Ни на что не претендуем**: Автор проекта не является официальным представителем, аффилированным лицом, сотрудником или партнером платформы Max, Mail.ru, VK или любых связанных с ними компаний. Все товарные знаки, названия и права на платформу принадлежат их законным владельцам.
- **Ничего не пропагандируем и не желаем плохого**: Данный проект не пропагандирует спам, накрутки, взлом, массовые рассылки, нарушение правил сообщества или любые деструктивные действия. Автор искренне уважает чужой труд, разработчиков сервиса и пользователей платформы.
- **Снятие ответственности**: Автор и контрибьюторы библиотеки полностью снимают с себя любую ответственность за то, как именно, кем и в каких целях используется данный код. Вся ответственность за любые действия, безопасность учетных записей, возможные временные или постоянные блокировки (бан аккаунтов), а также за соблюдение пользовательского соглашения сторонних сервисов целиком и полностью лежит на конечном пользователе, использующем данное ПО.
- Используя данную библиотеку, вы подтверждаете, что делаете это на свой собственный страх и риск.
