-- Добавление JSONB-поля для пропуска постов
ALTER TABLE channel_duplicate
    ADD COLUMN post_skip JSONB DEFAULT NULL;
