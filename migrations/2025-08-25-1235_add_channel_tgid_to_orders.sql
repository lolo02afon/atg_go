-- Добавляем поле channel_tgid в таблицу orders для хранения ID канала Telegram.
-- Поле будет заполняться при обновлении ссылок, если оно пустое.
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS channel_tgid BIGINT;
