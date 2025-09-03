-- Добавляет post_reactions для выбора реакции к посту
-- Хранит список эмодзи или NULL
ALTER TABLE channel_duplicate
    ADD COLUMN post_reactions TEXT[];
