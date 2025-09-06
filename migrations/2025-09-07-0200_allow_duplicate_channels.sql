-- Разрешаем дублирование ссылок в таблице каналов.

-- Удаляем первичный ключ с поля url.
ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_pkey;

-- Добавляем суррогатный ключ.
ALTER TABLE channels
    ADD COLUMN IF NOT EXISTS id BIGSERIAL;

-- Устанавливаем id как первичный ключ, если он ещё не задан.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'channels'::regclass
          AND conname = 'channels_pkey'
    ) THEN
        ALTER TABLE channels
            ADD CONSTRAINT channels_pkey PRIMARY KEY (id);
    END IF;
END $$;
