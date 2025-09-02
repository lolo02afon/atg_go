-- Изменяет тип post_count_day на массив TIME
-- Хранит расписание публикаций в формате HH:MM
ALTER TABLE channel_duplicate
    ALTER COLUMN post_count_day TYPE time[] USING NULL;
