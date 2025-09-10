-- Создание таблицы accounts_sessions_disconnect для учёта отключённых сессий
CREATE TABLE accounts_sessions_disconnect (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    date_time TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Дата и время события
    accounts_check INTEGER NOT NULL DEFAULT 0, -- Количество проверенных авторизованных аккаунтов
    accounts_sessions_disconnect INTEGER NOT NULL DEFAULT 0 -- Количество отключённых сессий
);
