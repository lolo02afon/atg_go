-- Добавляет колонку для фиксации лимита подписок на каналы
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS channels_limit_until TIMESTAMPTZ;
