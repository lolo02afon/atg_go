-- Переименование столбцов распределения просмотров по часам
-- Обновляем теоретические данные
ALTER TABLE channel_post_theory RENAME COLUMN view_4group_theory TO view_7_24hour_theory;
ALTER TABLE channel_post_theory RENAME COLUMN view_3group_theory TO view_4_6hour_theory;
ALTER TABLE channel_post_theory RENAME COLUMN view_2group_theory TO view_2_3hour_theory;
ALTER TABLE channel_post_theory RENAME COLUMN view_1group_theory TO view_1hour_theory;

-- Обновляем фактические данные
ALTER TABLE channel_post_fact RENAME COLUMN view_4group_fact TO view_7_24hour_fact;
ALTER TABLE channel_post_fact RENAME COLUMN view_3group_fact TO view_4_6hour_fact;
ALTER TABLE channel_post_fact RENAME COLUMN view_2group_fact TO view_2_3hour_fact;
ALTER TABLE channel_post_fact RENAME COLUMN view_1group_fact TO view_1hour_fact;
