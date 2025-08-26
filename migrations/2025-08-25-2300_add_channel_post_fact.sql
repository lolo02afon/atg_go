-- Таблица фактических просмотров по группам часов для поста канала
CREATE TABLE IF NOT EXISTS channel_post_fact (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    channel_post_theory_id INTEGER NOT NULL REFERENCES channel_post_theory(id) ON DELETE CASCADE, -- связь с прогнозом поста
    view_4group_fact DOUBLE PRECISION NOT NULL DEFAULT 0, -- Часы: 7–24 → 0.5–3.2%
    view_3group_fact DOUBLE PRECISION NOT NULL DEFAULT 0, -- Часы: 4–6 → 3.7–6.3%
    view_2group_fact DOUBLE PRECISION NOT NULL DEFAULT 0, -- Часы: 2–3 → 6.7–11.0%
    view_1group_fact DOUBLE PRECISION NOT NULL DEFAULT 0  -- Часы: 1 → 20.6–25.7%
);
