# Документация GoMax

Актуальная документация публикуется на GitHub Pages:

https://ebunyt-dotcom.github.io/gomax/

## Быстрые ссылки

- [Начало работы](docs/getting-started.md)
- [SMS-авторизация](docs/authentication/sms.md)
- [QR-авторизация](docs/authentication/qr.md)
- [Примеры](docs/examples.md)
- [Полный API](docs/api/reference.md)
- [Типы и данные](docs/api/types.md)
- [FAQ](docs/faq.md)
- [Диагностика](docs/troubleshooting.md)

## Онлайн API Go

Описание пакетов и экспортируемых символов Go автоматически формируется сервисом pkg.go.dev:

https://pkg.go.dev/github.com/ebunyt-dotcom/gomax

Для локального просмотра API используйте:

    go doc github.com/ebunyt-dotcom/gomax
    go doc github.com/ebunyt-dotcom/gomax/pkg/types

Сайт документации собирается GitHub Actions из mkdocs.yml. Исходники страниц находятся в каталоге docs.
