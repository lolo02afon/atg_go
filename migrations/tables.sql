-- Таблица прокси-серверов
CREATE TABLE IF NOT EXISTS proxy (
    id SERIAL PRIMARY KEY,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    login TEXT,
    password TEXT,
    ipv6 TEXT,
    account_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NULL
);

-- Таблица заказов на размещение ссылок
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    url_default TEXT,
    accounts_number_theory INTEGER NOT NULL,
    accounts_number_fact INTEGER NOT NULL DEFAULT 0,
    date_time TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Основная таблица аккаунтов Telegram
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,                           -- Уникальный идентификатор аккаунта
    phone TEXT NOT NULL UNIQUE,                      -- Номер телефона в формате +79991112233 (уникальный)
    api_id INTEGER NOT NULL,                         -- API ID из my.telegram.org
    api_hash TEXT NOT NULL,                          -- API Hash из my.telegram.org
    is_authorized BOOLEAN DEFAULT false,             -- Флаг успешной авторизации
    phone_code_hash TEXT,                            -- Хэш кода подтверждения из Telegram
    floodwait_until TIMESTAMP NULL,                 -- Время окончания флуд-бана (NULL если нет блокировки)
    proxy_id INTEGER REFERENCES proxy(id)          -- Привязка к прокси
);

-- Триггер для автоматического обновления account_count
CREATE OR REPLACE FUNCTION update_proxy_account_count() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.proxy_id IS NOT NULL THEN
            UPDATE proxy SET account_count = account_count + 1 WHERE id = NEW.proxy_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.proxy_id IS NOT NULL THEN
            UPDATE proxy SET account_count = account_count - 1 WHERE id = OLD.proxy_id;
        END IF;
        IF NEW.proxy_id IS NOT NULL THEN
            UPDATE proxy SET account_count = account_count + 1 WHERE id = NEW.proxy_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.proxy_id IS NOT NULL THEN
            UPDATE proxy SET account_count = account_count - 1 WHERE id = OLD.proxy_id;
        END IF;
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER accounts_proxy_insert AFTER INSERT ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_proxy_account_count();

CREATE TRIGGER accounts_proxy_update AFTER UPDATE OF proxy_id ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_proxy_account_count();

CREATE TRIGGER accounts_proxy_delete AFTER DELETE ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_proxy_account_count();

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
    id_message VARCHAR(20) NOT NULL, -- ID сообщения (для реакций из обсуждения, для комментариев ID поста)
    activity_type TEXT NOT NULL,
    date_time TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Таблица для хранения сессий аккаунтов Telegram
CREATE TABLE IF NOT EXISTS account_session (
    id SERIAL PRIMARY KEY,
    date_time TIMESTAMP NOT NULL DEFAULT NOW(),
    account INTEGER NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    data_json TEXT NOT NULL
);

