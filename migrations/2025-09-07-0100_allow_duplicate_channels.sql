-- Убираем ограничение уникальности ссылок каналов.
ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_pkey;

-- Добавляем суррогатный первичный ключ.
ALTER TABLE channels ADD COLUMN id BIGSERIAL PRIMARY KEY;
