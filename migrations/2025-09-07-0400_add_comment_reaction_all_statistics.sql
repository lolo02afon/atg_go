-- Добавление полей comment_all и reaction_all в таблицу statistics
ALTER TABLE statistics
    ADD COLUMN comment_all INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN reaction_all INTEGER NOT NULL DEFAULT 0;

-- Функция collect_statistics больше не используется
DROP FUNCTION IF EXISTS collect_statistics();
