-- Разделяем требования по активной аудитории на отдельные метрики
ALTER TABLE channel_post
    DROP COLUMN subs_active_all,
    ADD COLUMN subs_active_view INTEGER,
    ADD COLUMN subs_active_reaction INTEGER,
    ADD COLUMN subs_active_repost INTEGER;
