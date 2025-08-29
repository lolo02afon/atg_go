-- Добавление поля для отслеживания последнего пересланного поста
ALTER TABLE channel_duplicate
    ADD COLUMN last_post_id INTEGER DEFAULT NULL;
