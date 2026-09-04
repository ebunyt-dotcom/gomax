# Готовые примеры

Все примеры находятся в каталоге examples/. В каждом каталоге есть отдельный main.go.

## Быстрый выбор

| Нужно сделать | Пример |
|---|---|
| Авторизоваться по SMS | [`sms_login/main.go`](../examples/sms_login/main.go) |
| Авторизоваться по QR | [`qr_login/main.go`](../examples/qr_login/main.go) |
| Получать события и сообщения | [`events/main.go`](../examples/events/main.go) |
| Загрузить и отправить фото | [`send_media/main.go`](../examples/send_media/main.go) |
| Получить свой профиль и чаты | [`profile_and_chats/main.go`](../examples/profile_and_chats/main.go) |
| Запустить простой echo | [`quickstart/main.go`](../examples/quickstart/main.go) |

## Как запустить

Из корня репозитория:

    go mod download
    go run ./examples/sms_login

Для другого сценария замените последний путь на нужный каталог. Перед запуском измените отмеченные в коде значения Phone, chatID и имена файлов.

## SMS: что происходит

В sms_login:

1. создаётся конфигурация через DefaultConfig();
2. указывается Phone;
3. создаётся NewClient;
4. вызывается Start;
5. код из SMS вводится в консоли;
6. сессия сохраняется в JSON, и повторный SMS обычно не требуется.

## QR: что происходит

В qr_login номер телефона не нужен:

1. создаётся NewWebClient;
2. QrAuthFlow запрашивает QR;
3. QR печатается в терминале;
4. код сканируется приложением Max;
5. WebSocket-клиент сохраняет сессию.

Важно: NewClient — SMS/TCP, NewWebClient — QR/WebSocket.
