-- Запрещаем дублирование комбинации order_id и account_id
ALTER TABLE order_account_subs
    ADD CONSTRAINT order_account_subs_order_id_account_id_unique UNIQUE (order_id, account_id);
