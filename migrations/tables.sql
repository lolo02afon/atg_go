-- Основная таблица аккаунтов Telegram
CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    phone TEXT NOT NULL UNIQUE,                      -- Номер телефона в формате +79991112233 (уникальный)
    api_id INTEGER NOT NULL,                         -- API ID из my.telegram.org
    api_hash TEXT NOT NULL,                          -- API Hash из my.telegram.org
    is_authorized BOOLEAN DEFAULT false,             -- Флаг успешной авторизации
    phone_code_hash TEXT,                            -- Хэш кода подтверждения из Telegram
    floodwait_until TIMESTAMPTZ NULL,                -- Время окончания флуд-бана с учётом часового пояса
    proxy_id INTEGER REFERENCES proxy(id)          -- Привязка к прокси
);

-- Таблица со списокм каналов в определенной тематике 
CREATE TABLE IF NOT EXISTS channels (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    name TEXT NOT NULL,              -- Произвольное название группы каналов
    urls JSONB NOT NULL              -- Массив URL в формате ["https://t.me/channel1", "https://t.me/channel2"]
);

-- Таблица активности аккаунтов
CREATE TABLE IF NOT EXISTS activity (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    id_account INTEGER NOT NULL,
    id_channel VARCHAR(20) NOT NULL, -- ID канала как строка до 20 символов
    id_message VARCHAR(20) NOT NULL, -- ID сообщения (для реакций из обсуждения, для комментариев ID поста)
    activity_type TEXT NOT NULL,
    date_time TIMESTAMPTZ NOT NULL DEFAULT NOW() -- Время события с учётом часового пояса
);

-- Таблица для хранения сессий аккаунтов Telegram
CREATE TABLE IF NOT EXISTS account_session (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    date_time TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Время сохранения сессии с часовым поясом
    account INTEGER NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    data_json TEXT NOT NULL
);

