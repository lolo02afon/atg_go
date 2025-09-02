-- Удаляет поле days_ago из таблицы channel_duplicate
ALTER TABLE channel_duplicate
    DROP COLUMN days_ago;
