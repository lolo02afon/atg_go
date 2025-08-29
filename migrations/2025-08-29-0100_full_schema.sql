-- Все структуры и функции объединены в одну миграцию.

-- Перечисление полов аккаунтов
CREATE TYPE gender_enum AS ENUM ('male', 'female', 'neutral');

-- Таблица прокси
CREATE TABLE proxy (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    login TEXT,
    password TEXT,
    ipv6 TEXT,
    account_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NULL
);

-- Таблица заказов
CREATE TABLE orders (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT[] DEFAULT NULL,
    url_description TEXT NOT NULL,
    url_default TEXT NOT NULL,
    accounts_number_theory INTEGER NOT NULL,
    accounts_number_fact INTEGER NOT NULL DEFAULT 0,
    gender gender_enum[] NOT NULL DEFAULT ARRAY['neutral']::gender_enum[] CHECK (array_length(gender,1) > 0),
    date_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    channel_tgid TEXT DEFAULT NULL,
    subs_active_count INTEGER DEFAULT 0,
    CONSTRAINT orders_url_default_unique UNIQUE (url_default)
);

-- Таблица аккаунтов
CREATE TABLE accounts (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    phone TEXT NOT NULL UNIQUE,
    api_id INTEGER NOT NULL,
    api_hash TEXT NOT NULL,
    is_authorized BOOLEAN DEFAULT false,
    gender gender_enum[] NOT NULL DEFAULT ARRAY['neutral']::gender_enum[] CHECK (array_length(gender,1) > 0),
    phone_code_hash TEXT,
    floodwait_until TIMESTAMPTZ NULL,
    channels_limit_until TIMESTAMPTZ NULL,
    proxy_id INTEGER REFERENCES proxy(id),
    order_id INTEGER REFERENCES orders(id) ON DELETE SET NULL,
    account_monitoring BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT account_monitoring_no_order CHECK (NOT account_monitoring OR order_id IS NULL)
);

-- Таблица категорий каналов
CREATE TABLE categories (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    urls JSONB NOT NULL
);

-- Таблица активности (простая без внешних ключей)
CREATE TABLE activity (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    id_account INTEGER NOT NULL,
    id_channel VARCHAR(20) NOT NULL,
    id_message VARCHAR(20) NOT NULL,
    activity_type TEXT NOT NULL,
    date_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Таблица сессий аккаунтов
CREATE TABLE account_session (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    date_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    account INTEGER NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    data_json TEXT NOT NULL
);

-- Таблица постов канала
CREATE TABLE channel_post (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    post_date_time TIMESTAMPTZ NOT NULL,
    post_url TEXT NOT NULL,
    subs_active_view INTEGER,
    subs_active_reaction INTEGER,
    subs_active_repost INTEGER
);

-- Теоретические просмотры
CREATE TABLE channel_post_theory (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    channel_post_id INTEGER NOT NULL REFERENCES channel_post(id) ON DELETE CASCADE,
    view_7_24hour_theory DOUBLE PRECISION NOT NULL DEFAULT 0,
    view_4_6hour_theory DOUBLE PRECISION NOT NULL DEFAULT 0,
    view_2_3hour_theory DOUBLE PRECISION NOT NULL DEFAULT 0,
    view_1hour_theory DOUBLE PRECISION NOT NULL DEFAULT 0,
    reaction_24hour_theory INTEGER NOT NULL DEFAULT 0,
    repost_24hour_theory INTEGER NOT NULL DEFAULT 0
);

-- Фактические просмотры
CREATE TABLE channel_post_fact (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    channel_post_theory_id INTEGER NOT NULL REFERENCES channel_post_theory(id) ON DELETE CASCADE,
    view_7_24hour_fact DOUBLE PRECISION NOT NULL DEFAULT 0,
    view_4_6hour_fact DOUBLE PRECISION NOT NULL DEFAULT 0,
    view_2_3hour_fact DOUBLE PRECISION NOT NULL DEFAULT 0,
    view_1hour_fact DOUBLE PRECISION NOT NULL DEFAULT 0,
    reaction_24hour_fact INTEGER NOT NULL DEFAULT 0,
    repost_24hour_fact INTEGER NOT NULL DEFAULT 0
);

-- Подписки аккаунтов на каналы заказов
CREATE TABLE order_account_subs (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    CONSTRAINT order_account_subs_order_id_account_id_unique UNIQUE (order_id, account_id)
);

-- Таблица статистики
CREATE TABLE statistics (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    stat_date DATE NOT NULL UNIQUE,
    comment_mean DOUBLE PRECISION NOT NULL,
    reaction_mean DOUBLE PRECISION NOT NULL,
    account_floodban INTEGER NOT NULL,
    account_all INTEGER NOT NULL
);

-- Функция сбора статистики
CREATE OR REPLACE FUNCTION collect_statistics() RETURNS VOID AS $$
DECLARE
    moscow_now   TIMESTAMPTZ := timezone('Europe/Moscow', now());
    day_start    TIMESTAMPTZ;
    day_end      TIMESTAMPTZ;
    total_acc    INTEGER;
    comment_cnt  INTEGER;
    reaction_cnt INTEGER;
    flood_cnt    INTEGER;
BEGIN
    day_start := date_trunc('day', moscow_now);
    day_end   := day_start + interval '1 day';

    SELECT COUNT(*) INTO total_acc
    FROM accounts
    WHERE is_authorized;

    SELECT
        COUNT(*) FILTER (WHERE activity_type = 'comment') AS comments,
        COUNT(*) FILTER (WHERE activity_type = 'reaction') AS reactions
    INTO comment_cnt, reaction_cnt
    FROM activity
    WHERE date_time >= day_start AT TIME ZONE 'UTC'
      AND date_time <  day_end   AT TIME ZONE 'UTC';

    SELECT COUNT(DISTINCT id) INTO flood_cnt
    FROM accounts
    WHERE floodwait_until > now();

    INSERT INTO statistics (stat_date, comment_mean, reaction_mean, account_floodban, account_all)
    VALUES (
        moscow_now::date,
        COALESCE(comment_cnt::DOUBLE PRECISION / NULLIF(total_acc, 0), 0),
        COALESCE(reaction_cnt::DOUBLE PRECISION / NULLIF(total_acc, 0), 0),
        flood_cnt,
        total_acc
    )
    ON CONFLICT (stat_date) DO UPDATE
        SET comment_mean   = EXCLUDED.comment_mean,
            reaction_mean  = EXCLUDED.reaction_mean,
            account_floodban = EXCLUDED.account_floodban,
            account_all      = EXCLUDED.account_all;
END;
$$ LANGUAGE plpgsql;

-- Функция фиксации флуд-бана
CREATE OR REPLACE FUNCTION mark_flood_ban(p_account_id INT, p_until TIMESTAMPTZ) RETURNS VOID AS $$
BEGIN
    UPDATE accounts
    SET floodwait_until = p_until
    WHERE id = p_account_id;
END;
$$ LANGUAGE plpgsql;

-- Таблица критичных обращений
CREATE TABLE "Sos" (
    date_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    msg TEXT
);

-- Функция обновления количества аккаунтов на прокси
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

-- Функция и триггеры обновления счётчика аккаунтов в заказе
CREATE OR REPLACE FUNCTION update_order_accounts_number() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.order_id IS NOT NULL THEN
            UPDATE orders SET accounts_number_fact = accounts_number_fact + 1 WHERE id = NEW.order_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.order_id IS NOT DISTINCT FROM NEW.order_id THEN
            RETURN NEW;
        END IF;
        IF OLD.order_id IS NOT NULL THEN
            UPDATE orders SET accounts_number_fact = accounts_number_fact - 1 WHERE id = OLD.order_id;
        END IF;
        IF NEW.order_id IS NOT NULL THEN
            UPDATE orders SET accounts_number_fact = accounts_number_fact + 1 WHERE id = NEW.order_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.order_id IS NOT NULL THEN
            UPDATE orders SET accounts_number_fact = accounts_number_fact - 1 WHERE id = OLD.order_id;
        END IF;
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER accounts_order_insert AFTER INSERT ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_order_accounts_number();

CREATE TRIGGER accounts_order_update AFTER UPDATE OF order_id ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_order_accounts_number();

CREATE TRIGGER accounts_order_delete AFTER DELETE ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_order_accounts_number();
