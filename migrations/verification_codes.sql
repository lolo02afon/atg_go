CREATE TABLE verification_codes (
    id SERIAL PRIMARY KEY,
    account_id INTEGER UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    send BOOLEAN DEFAULT FALSE
);