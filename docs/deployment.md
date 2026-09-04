# Публикация документации

Документация собирается из каталога docs через MkDocs Material и публикуется на GitHub Pages.

## Одноразовая настройка GitHub

Workflow уже находится в репозитории, но GitHub Pages нужно один раз включить в настройках:

1. Откройте репозиторий [ebunyt-dotcom/gomax](https://github.com/ebunyt-dotcom/gomax).
2. Перейдите в Settings → Pages.
3. В разделе Build and deployment выберите Source: GitHub Actions.
4. Откройте Actions → Deploy documentation и нажмите Re-run jobs у последнего запуска.

После этого адрес сайта будет:

    https://ebunyt-dotcom.github.io/gomax/

Если до включения Pages workflow был красным, после этой настройки его можно перезапустить в Actions или дождаться нового push в main.

## Как обновляется сайт

Изменения в docs или mkdocs.yml автоматически запускают workflow:

1. GitHub Actions устанавливает MkDocs Material.
2. Сайт собирается из docs.
3. Готовый сайт загружается как Pages artifact.
4. deploy-pages публикует его в GitHub Pages.

Локальный запуск не требуется для публикации. Если workflow завершился ошибкой, откройте его лог в разделе Actions.

## Где менять меню и внешний вид

- mkdocs.yml — название, меню, тема, поиск и настройки сайта;
- docs/stylesheets/extra.css — небольшие визуальные изменения;
- docs/index.md — главная страница;
- .github/workflows/docs.yml — процесс публикации.

Полная инструкция GitHub по custom workflow: [Using custom workflows with GitHub Pages](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages).
