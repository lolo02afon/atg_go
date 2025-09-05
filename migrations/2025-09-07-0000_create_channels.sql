-- Таблица каналов для хранения уникальных ссылок.
CREATE TABLE IF NOT EXISTS channels (
    url TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
