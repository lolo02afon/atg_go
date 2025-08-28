-- Добавление столбцов реакций и репостов за первые 24 часа
ALTER TABLE channel_post_theory
    ADD COLUMN reaction_24hour_theory INTEGER NOT NULL DEFAULT 0, -- Реакции за первые 24 часа
    ADD COLUMN repost_24hour_theory INTEGER NOT NULL DEFAULT 0;   -- Репосты за первые 24 часа

ALTER TABLE channel_post_fact
    ADD COLUMN reaction_24hour_fact INTEGER NOT NULL DEFAULT 0, -- Реакции за первые 24 часа
    ADD COLUMN repost_24hour_fact INTEGER NOT NULL DEFAULT 0;  -- Репосты за первые 24 часа
