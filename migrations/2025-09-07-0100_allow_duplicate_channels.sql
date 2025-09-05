-- Разрешаем дублирование ссылок каналов, заменяя первичный ключ.
DO $$
DECLARE
    pk_name TEXT;
BEGIN
    -- Ищем имя текущего первичного ключа.
    SELECT conname INTO pk_name
    FROM pg_constraint
    WHERE conrelid = 'channels'::regclass
      AND contype = 'p';

    -- Удаляем найденный первичный ключ, если он существует.
    IF pk_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE channels DROP CONSTRAINT %I', pk_name);
    END IF;
END $$;

-- Добавляем суррогатный идентификатор.
ALTER TABLE channels
    ADD COLUMN IF NOT EXISTS id BIGSERIAL;

-- Устанавливаем `id` в качестве первичного ключа.
ALTER TABLE channels
    ADD CONSTRAINT IF NOT EXISTS channels_pkey PRIMARY KEY (id);

