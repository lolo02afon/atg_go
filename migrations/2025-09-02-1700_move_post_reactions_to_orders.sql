-- Переносит post_reactions из channel_duplicate в orders
-- Добавляет поле post_reactions в таблицу orders и удаляет его из channel_duplicate
ALTER TABLE orders
    ADD COLUMN post_reactions TEXT[];

-- Копируем существующие данные реакций из channel_duplicate в orders
UPDATE orders o
SET post_reactions = cd.post_reactions
FROM (
    SELECT DISTINCT ON (order_id) order_id, post_reactions
    FROM channel_duplicate
    WHERE post_reactions IS NOT NULL
    ORDER BY order_id
) cd
WHERE o.id = cd.order_id;

-- Удаляем поле из channel_duplicate
ALTER TABLE channel_duplicate
    DROP COLUMN post_reactions;
