-- Скрипт создания таблицы статистики и функций для её заполнения
-- Таблица "statistics" хранит агрегированные значения за сутки по московскому времени

-- Создание таблицы со статистикой
CREATE TABLE IF NOT EXISTS statistics (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    stat_date DATE NOT NULL UNIQUE,          -- Дата, за которую рассчитана статистика (МСК)
    comment_mean DOUBLE PRECISION NOT NULL,  -- Среднее число комментариев на аккаунт
    reaction_mean DOUBLE PRECISION NOT NULL, -- Среднее число реакций на аккаунт
    account_floodban INTEGER NOT NULL,       -- Количество аккаунтов во флуд-бане
    account_all INTEGER NOT NULL             -- Число авторизованных аккаунтов
);

-- Функция собирает статистику за текущие сутки (по времени Москвы)
CREATE OR REPLACE FUNCTION collect_statistics() RETURNS VOID AS $$
DECLARE
    moscow_now   TIMESTAMPTZ := timezone('Europe/Moscow', now()); -- Текущее время по Мск
    day_start    TIMESTAMPTZ;                                     -- Начало текущих суток
    day_end      TIMESTAMPTZ;                                     -- Конец текущих суток
    total_acc    INTEGER;                                         -- Количество авторизованных аккаунтов
    comment_cnt  INTEGER;                                         -- Количество комментариев за сутки
    reaction_cnt INTEGER;                                         -- Количество реакций за сутки
    flood_cnt    INTEGER;                                         -- Количество уникальных аккаунтов во флуд-бане
BEGIN
    -- Определяем границы суток по московскому времени
    day_start := date_trunc('day', moscow_now);
    day_end   := day_start + interval '1 day';

    -- Считаем авторизованные аккаунты
    SELECT COUNT(*) INTO total_acc
    FROM accounts
    WHERE is_authorized;

    -- Подсчёт комментариев и реакций за текущие сутки
    SELECT
        COUNT(*) FILTER (WHERE activity_type = 'comment') AS comments,
        COUNT(*) FILTER (WHERE activity_type = 'reaction') AS reactions
    INTO comment_cnt, reaction_cnt
    FROM activity
    WHERE date_time >= day_start AT TIME ZONE 'UTC'
      AND date_time <  day_end   AT TIME ZONE 'UTC';

    -- Подсчёт уникальных аккаунтов во флуд-бане
    SELECT COUNT(DISTINCT id) INTO flood_cnt
    FROM accounts
    WHERE floodwait_until > now();

    -- Запись или обновление статистики за текущие сутки
    INSERT INTO statistics (stat_date, comment_mean, reaction_mean, account_floodban, account_all)
    VALUES (
        moscow_now::date,
        COALESCE(comment_cnt::DOUBLE PRECISION / NULLIF(total_acc, 0), 0),
        COALESCE(reaction_cnt::DOUBLE PRECISION / NULLIF(total_acc, 0), 0),
        flood_cnt,
        total_acc
    )
    ON CONFLICT (stat_date) DO UPDATE
        SET comment_mean   = EXCLUDED.comment_mean,
            reaction_mean  = EXCLUDED.reaction_mean,
            account_floodban = EXCLUDED.account_floodban,
            account_all      = EXCLUDED.account_all;
END;
$$ LANGUAGE plpgsql;

-- Функция для фиксации времени окончания флуд-бана у конкретного аккаунта
CREATE OR REPLACE FUNCTION mark_flood_ban(p_account_id INT, p_until TIMESTAMPTZ) RETURNS VOID AS $$
BEGIN
    UPDATE accounts
    SET floodwait_until = p_until
    WHERE id = p_account_id;
END;
$$ LANGUAGE plpgsql;

