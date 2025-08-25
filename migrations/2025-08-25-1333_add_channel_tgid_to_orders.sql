-- Добавляем поле channel_tgid для хранения ID канала, извлечённого из url_default
ALTER TABLE orders
    ADD COLUMN channel_tgid TEXT DEFAULT NULL;

-- Заполняем channel_tgid для существующих записей, извлекая ID из url_default
UPDATE orders
SET channel_tgid = substring(url_default from 'https://t\\.me/c/([0-9]+)')
WHERE url_default LIKE 'https://t.me/c/%';
