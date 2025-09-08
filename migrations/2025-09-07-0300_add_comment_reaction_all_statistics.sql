-- Добавление полей comment_all и reaction_all в таблицу statistics
ALTER TABLE statistics
    ADD COLUMN comment_all INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN reaction_all INTEGER NOT NULL DEFAULT 0;

-- Обновление функции collect_statistics с учётом новых полей
CREATE OR REPLACE FUNCTION collect_statistics() RETURNS VOID AS $$
DECLARE
    moscow_now   TIMESTAMPTZ := timezone('Europe/Moscow', now());
    day_start    TIMESTAMPTZ;
    day_end      TIMESTAMPTZ;
    total_acc    INTEGER;
    comment_cnt  INTEGER;
    reaction_cnt INTEGER;
    flood_cnt    INTEGER;
BEGIN
    day_start := date_trunc('day', moscow_now);
    day_end   := day_start + interval '1 day';

    SELECT COUNT(*) INTO total_acc
    FROM accounts
    WHERE is_authorized;

    SELECT
        COUNT(*) FILTER (WHERE activity_type = 'comment') AS comments,
        COUNT(*) FILTER (WHERE activity_type = 'reaction') AS reactions
    INTO comment_cnt, reaction_cnt
    FROM activity
    WHERE date_time >= day_start AT TIME ZONE 'UTC'
      AND date_time <  day_end   AT TIME ZONE 'UTC';

    SELECT COUNT(DISTINCT id) INTO flood_cnt
    FROM accounts
    WHERE floodwait_until > now();

    INSERT INTO statistics (stat_date, comment_mean, reaction_mean, comment_all, reaction_all, account_floodban, account_all)
    VALUES (
        moscow_now::date,
        COALESCE(comment_cnt::DOUBLE PRECISION / NULLIF(total_acc, 0), 0),
        COALESCE(reaction_cnt::DOUBLE PRECISION / NULLIF(total_acc, 0), 0),
        comment_cnt,
        reaction_cnt,
        flood_cnt,
        total_acc
    )
    ON CONFLICT (stat_date) DO UPDATE
        SET comment_mean   = EXCLUDED.comment_mean,
            reaction_mean  = EXCLUDED.reaction_mean,
            comment_all    = EXCLUDED.comment_all,
            reaction_all   = EXCLUDED.reaction_all,
            account_floodban = EXCLUDED.account_floodban,
            account_all      = EXCLUDED.account_all;
END;
$$ LANGUAGE plpgsql;
