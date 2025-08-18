-- Добавляет поле для указания пола аккаунта
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS gender TEXT NOT NULL DEFAULT 'neutral'
        CHECK (gender IN ('male', 'female', 'neutral'));
