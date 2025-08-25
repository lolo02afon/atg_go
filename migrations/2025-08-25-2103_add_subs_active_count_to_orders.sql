-- Добавляем поле subs_active_count, отражающее количество аккаунтов, которые должны активничать в канале
-- Значение по умолчанию 0, но допускается NULL
ALTER TABLE orders
    ADD COLUMN subs_active_count INTEGER DEFAULT 0;
