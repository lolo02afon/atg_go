-- Таблица категорий
CREATE TABLE IF NOT EXISTS category (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    name VARCHAR(64) NOT NULL UNIQUE                    -- Короткое уникальное название категории
);
