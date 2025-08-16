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
