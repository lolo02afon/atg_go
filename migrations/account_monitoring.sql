-- Флаг мониторинга: добавляем колонку, чтобы управлять наблюдением за аккаунтом.
-- По умолчанию не включаем мониторинг, чтобы не выполнять лишнюю работу.
ALTER TABLE accounts
    ADD COLUMN account_monitoring BOOLEAN NOT NULL DEFAULT false;
