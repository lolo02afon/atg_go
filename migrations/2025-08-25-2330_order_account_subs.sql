-- Таблица подписок аккаунтов на каналы из заказов
CREATE TABLE IF NOT EXISTS order_account_subs (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE, -- ID заказа, на канал которого подписался аккаунт
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE -- ID аккаунта, подписанного на канал заказа
);
