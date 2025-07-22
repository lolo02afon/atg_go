CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    phone TEXT NOT NULL,
    api_id INTEGER NOT NULL,
    api_hash TEXT NOT NULL,
    is_authorized BOOLEAN DEFAULT false
);
