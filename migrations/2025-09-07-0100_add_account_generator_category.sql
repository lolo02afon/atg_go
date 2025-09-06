-- Добавляет поле account_generator_category и необходимые ограничения.
ALTER TABLE accounts ADD COLUMN account_generator_category BOOLEAN NOT NULL DEFAULT FALSE;

-- Запрещаем назначение на заказы аккаунтов генерации категорий.
ALTER TABLE accounts ADD CONSTRAINT account_generator_category_no_order
    CHECK (NOT account_generator_category OR order_id IS NULL);

-- Одновременно account_monitoring и account_generator_category не могут быть TRUE.
ALTER TABLE accounts ADD CONSTRAINT account_monitoring_generator_excl
    CHECK (NOT (account_monitoring AND account_generator_category));
