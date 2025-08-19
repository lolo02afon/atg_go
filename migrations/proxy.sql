-- Функция и триггеры поддерживают актуальное количество аккаунтов на каждом прокси
CREATE OR REPLACE FUNCTION update_proxy_account_count() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.proxy_id IS NOT NULL THEN
            UPDATE proxy SET account_count = account_count + 1 WHERE id = NEW.proxy_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.proxy_id IS NOT NULL THEN
            UPDATE proxy SET account_count = account_count - 1 WHERE id = OLD.proxy_id;
        END IF;
        IF NEW.proxy_id IS NOT NULL THEN
            UPDATE proxy SET account_count = account_count + 1 WHERE id = NEW.proxy_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.proxy_id IS NOT NULL THEN
            UPDATE proxy SET account_count = account_count - 1 WHERE id = OLD.proxy_id;
        END IF;
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER accounts_proxy_insert AFTER INSERT ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_proxy_account_count();

CREATE TRIGGER accounts_proxy_update AFTER UPDATE OF proxy_id ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_proxy_account_count();

CREATE TRIGGER accounts_proxy_delete AFTER DELETE ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_proxy_account_count();
