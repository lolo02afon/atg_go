-- Фиксируем активных подписчиков в момент публикации
-- Отказались от ограничения сверху, чтобы не обрезать реальные значения
ALTER TABLE channel_post
    ADD COLUMN subs_active_all INTEGER;
