# atg_go

Основные модули Telegram:

- `accounts_auth` — авторизация аккаунтов и проверка сессий.
- `generation_category_channels` — подбор похожих каналов.
- `invite_activities` — комментарии и реакции в чужих обсуждениях.
- `invite_activities_statistics` — статистика инвайтинга.
- `accounts_sessions_disconnect` — отключение подозрительных сессий.
- `subs_active` — поддержание активности аккаунтов.

В каталоге `internal` и `pkg/telegram` дополнительно находятся:

- `technical` — вспомогательные пакеты (middleware, httputil, mutex и др.).
- `base` — общие сущности и утилиты (например, `order`, `channel_duplicate`).

База данных: PostgreSQL 17.5.

Миграции хранятся в каталоге `migrations` и именуются с префиксом даты и времени.
