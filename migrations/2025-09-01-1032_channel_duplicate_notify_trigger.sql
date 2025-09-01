-- Создаём функцию и триггер для уведомления об изменениях в channel_duplicate
CREATE OR REPLACE FUNCTION notify_channel_duplicate() RETURNS trigger AS $$
BEGIN
    -- Отправляем ID записи в канал уведомлений
    PERFORM pg_notify('channel_duplicate_changed', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER channel_duplicate_notify_trg
AFTER INSERT OR UPDATE ON channel_duplicate
FOR EACH ROW EXECUTE FUNCTION notify_channel_duplicate();
