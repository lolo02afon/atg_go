-- Аккаунты с мониторингом не должны выполнять заказы.
-- Сначала снимаем их с текущих заказов, затем добавляем ограничение.
UPDATE accounts
SET order_id = NULL
WHERE account_monitoring = TRUE AND order_id IS NOT NULL;

ALTER TABLE accounts
    ADD CONSTRAINT account_monitoring_no_order
    CHECK (NOT account_monitoring OR order_id IS NULL);
