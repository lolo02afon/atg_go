-- Добавляет поле post_count_day в таблицу channel_duplicate
-- Поле хранит число постов в день; значение по умолчанию NULL
ALTER TABLE channel_duplicate
    ADD COLUMN post_count_day integer;
