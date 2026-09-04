# Диагностика

## Программа сразу завершается

Проверьте, что ошибка client.Start не игнорируется:

    if err := client.Start(ctx); err != nil {
        log.Fatal(err)
    }

Если используется context.WithTimeout, не задавайте слишком короткий timeout для долгоживущего клиента. Для фонового клиента используйте context.WithCancel и вызывайте cancel при остановке.

## В PowerShell не активируется Python venv

Для GoMax виртуальное окружение Python не нужно. Установите Go-зависимость в Go-проект:

    go get github.com/ebunyt-dotcom/gomax
    go mod tidy

## go run сообщает missing go.sum entry

Запустите команды в каталоге проекта, где находится go.mod:

    go get github.com/ebunyt-dotcom/gomax
    go mod tidy

Если библиотека подключена локально, проверьте replace в go.mod и наличие всех зависимостей.

## Не находится пакет gomax

Импорт должен быть таким:

    import "github.com/ebunyt-dotcom/gomax"

После изменения go.mod повторите:

    go mod tidy

## Клиент подключается, но авторизация не продолжается

Смотрите последнюю строку лога:

- SMS: ожидается ввод кода в той же консоли;
- QR: код должен вывести ConsoleQrHandler, после чего его нужно отсканировать в приложении;
- 2FA: потребуется пароль аккаунта;
- registration: новому аккаунту могут понадобиться FirstName и LastName в cfg.Registration.

## Нет сообщений в OnMessage

OnMessage вызывается только после успешного Start и получения серверного события. Проверьте:

- обработчик зарегистрирован до Start;
- возвращается nil, если обработка успешна;
- используется тот же client, который был запущен;
- соединение не закрывается сразу после Start.

Для диагностики временно добавьте OnRaw:

    client.OnRaw(func(ctx context.Context, event *types.RawEvent) error {
        log.Printf("raw event: type=%s opcode=%d", event.Type, event.Opcode)
        return nil
    })

## Сессия сломалась или нужно войти заново

Остановите клиент, удалите только файл конкретной сессии из WorkDir и запустите снова. Не удаляйте весь каталог проекта. Токен из удалённой сессии придётся получить заново.

## Ошибка при загрузке

Проверьте:

- файл прочитан целиком;
- data не nil;
- для видео и voice указана длительность;
- UploadTimeout достаточно большой;
- после загрузки отправляется именно полученный Attachment.

## Безопасность

Не публикуйте в issue, логах или Git:

- Token;
- JSON-файл сессии;
- QR-ссылку;
- SMS-коды и пароль 2FA.

Для полного списка методов см. [API reference](api/reference.md).

