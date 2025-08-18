-- Создаём тип gender_type, если он ещё не создан
DO $$
BEGIN
    CREATE TYPE gender_type AS ENUM ('male', 'female', 'neutral');
EXCEPTION
    WHEN duplicate_object THEN NULL; -- Тип уже существует
END$$;

-- Обеспечиваем уникальность названий каналов, чтобы использовать их в качестве категории
ALTER TABLE channels
    ADD CONSTRAINT IF NOT EXISTS channels_name_key UNIQUE (name);

-- Таблица заказов на размещение ссылок
CREATE TABLE IF NOT EXISTS orders (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    name TEXT NOT NULL,
    category TEXT REFERENCES channels(name) ON DELETE SET NULL, -- Название категории (берём из channels.name, может отсутствовать)
    url_description TEXT NOT NULL, -- ссылка для описания аккаунта
    url_default TEXT NOT NULL, -- ссылка по умолчанию
    accounts_number_theory INTEGER NOT NULL,
    accounts_number_fact INTEGER NOT NULL DEFAULT 0,
    gender gender_type[] NOT NULL DEFAULT ARRAY['neutral']::gender_type[] CHECK (array_length(gender, 1) >= 1), -- Целевая аудитория по полу
    date_time TIMESTAMPTZ NOT NULL DEFAULT NOW() -- сохраняем время с учётом часового пояса
);

-- Добавляем колонку category с внешним ключом, если таблица уже создана
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS category TEXT REFERENCES channels(name) ON DELETE SET NULL;

ALTER TABLE orders
    ADD CONSTRAINT IF NOT EXISTS orders_category_fkey FOREIGN KEY (category) REFERENCES channels(name) ON DELETE SET NULL;

-- Добавляем колонку gender, если таблица уже создана
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS gender gender_type[] NOT NULL DEFAULT ARRAY['neutral']::gender_type[];

ALTER TABLE orders
    ADD CONSTRAINT IF NOT EXISTS orders_gender_not_empty CHECK (array_length(gender, 1) >= 1);

-- Добавление поля order_id в таблицу accounts
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS order_id INTEGER; -- Поле для связи с заказом

-- Функция для автоматического обновления количества фактических аккаунтов
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

-- Триггеры на таблицу accounts
CREATE TRIGGER accounts_order_insert AFTER INSERT ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_order_accounts_number();

CREATE TRIGGER accounts_order_update AFTER UPDATE OF order_id ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_order_accounts_number();

CREATE TRIGGER accounts_order_delete AFTER DELETE ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_order_accounts_number();

-- Пересоздание внешнего ключа с опцией ON DELETE SET NULL
ALTER TABLE accounts
    DROP CONSTRAINT IF EXISTS accounts_order_id_fkey; -- Удаляем старый внешний ключ, если он был

ALTER TABLE accounts
    ADD CONSTRAINT accounts_order_id_fkey
        FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE SET NULL; -- При удалении заказа поле обнуляется

