-- Основная таблица аккаунтов Telegram
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,                           -- Уникальный идентификатор аккаунта
    phone TEXT NOT NULL UNIQUE,                      -- Номер телефона в формате +79991112233 (уникальный)
    api_id INTEGER NOT NULL,                         -- API ID из my.telegram.org
    api_hash TEXT NOT NULL,                          -- API Hash из my.telegram.org
    is_authorized BOOLEAN DEFAULT false,             -- Флаг успешной авторизации
    phone_code_hash TEXT,                            -- Хэш кода подтверждения из Telegram
    floodwait_until TIMESTAMP NULL                  -- Время окончания флуд-бана (NULL если нет блокировки)
);

-- Таблица со списокм каналов в определенной тематике 
CREATE TABLE IF NOT EXISTS channels (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,              -- Произвольное название группы каналов
    urls JSONB NOT NULL              -- Массив URL в формате ["https://t.me/channel1", "https://t.me/channel2"]
);

-- Таблица активности аккаунтов
CREATE TABLE IF NOT EXISTS activity (
    id SERIAL PRIMARY KEY,
    id_account INTEGER NOT NULL,
    id_channel VARCHAR(20) NOT NULL, -- ID канала как строка до 20 символов
    id_message VARCHAR(20) NOT NULL, -- ID сообщения как строка до 20 символов
    activity_type TEXT NOT NULL,
    date_time TIMESTAMP NOT NULL DEFAULT NOW()
);
