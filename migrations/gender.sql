-- Обновляет поле gender в таблице accounts на перечисляемый тип с возможностью выбора нескольких значений
DO $$
BEGIN
    CREATE TYPE gender_type AS ENUM ('male', 'female', 'neutral');
EXCEPTION
    WHEN duplicate_object THEN NULL; -- Игнорируем, если тип уже создан
END$$;

-- Переопределяем столбец gender как массив перечисляемых значений
ALTER TABLE accounts
    DROP COLUMN IF EXISTS gender;

ALTER TABLE accounts
    ADD COLUMN gender gender_type[] NOT NULL DEFAULT ARRAY['neutral']::gender_type[];

-- Убедимся, что в массиве есть хотя бы одно значение
ALTER TABLE accounts
    ADD CONSTRAINT accounts_gender_not_empty CHECK (array_length(gender, 1) >= 1);
