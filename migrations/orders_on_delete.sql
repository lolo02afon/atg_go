-- Обновление внешнего ключа order_id для установки NULL при удалении заказа
ALTER TABLE accounts
    DROP CONSTRAINT IF EXISTS accounts_order_id_fkey;

ALTER TABLE accounts
    ADD CONSTRAINT accounts_order_id_fkey
        FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE SET NULL;
