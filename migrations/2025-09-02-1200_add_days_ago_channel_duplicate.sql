-- Добавляет поле days_ago в таблицу channel_duplicate
-- Поле хранит число дней; значение по умолчанию NULL
ALTER TABLE channel_duplicate
    ADD COLUMN days_ago integer;
