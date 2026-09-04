# Публикация документации

Сайт собирается из `docs/` через MkDocs Material и публикуется GitHub Actions.

## Один раз включить Pages

В репозитории откройте **Settings → Pages** и выберите:

```text
Build and deployment → Source → GitHub Actions
```

После этого сайт доступен по адресу:

```text
https://ebunyt-dotcom.github.io/gomax/
```

## Что запускает публикацию

Workflow `.github/workflows/docs.yml` запускается при изменении:

- `docs/**`;
- `mkdocs.yml`;
- самого workflow;
- ручным запуском из **Actions → Deploy documentation → Run workflow**.

Он выполняет четыре шага: устанавливает MkDocs Material, собирает сайт,
загружает Pages artifact и публикует его.

## Где менять сайт

| Файл | Что менять |
|---|---|
| `mkdocs.yml` | Меню, название, тема, поиск и адрес сайта. |
| `docs/index.md` | Главная страница. |
| `docs/api/*.md` | Методы сервисов и примеры. |
| `docs/stylesheets/extra.css` | Внешний вид. |
| `.github/workflows/docs.yml` | Сборка и публикация. |

Локальный запуск для публикации не нужен. Если workflow красный, откройте
**Actions → Deploy documentation → build** и смотрите шаг, завершившийся
ошибкой.
