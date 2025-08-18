-- Преобразует поле gender в перечислимый массив
DO $$
BEGIN
    -- Создаём перечислимый тип, если он ещё не создан
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'gender_type') THEN
        CREATE TYPE gender_type AS ENUM ('male', 'female', 'neutral');
    END IF;

    -- Преобразуем столбец gender в массив перечислений, если он текстовый
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'accounts' AND column_name = 'gender' AND data_type = 'text'
    ) THEN
        ALTER TABLE accounts
            DROP CONSTRAINT IF EXISTS accounts_gender_check,
            ALTER COLUMN gender DROP DEFAULT,
            ALTER COLUMN gender TYPE gender_type[] USING ARRAY[gender]::gender_type[],
            ALTER COLUMN gender SET DEFAULT ARRAY['neutral']::gender_type[];
    END IF;
END$$;
