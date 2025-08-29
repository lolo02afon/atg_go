-- Таблица для дублирования контента из другого канала
CREATE TABLE channel_duplicate (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE, -- Связь с заказом; при удалении заказа запись удаляется
    url_channel_duplicate TEXT NOT NULL, -- Ссылка на канал, откуда будет дублироваться контент
    channel_duplicate_tgid TEXT DEFAULT NULL, -- ID телеграм-канала источника
    post_text_remove TEXT DEFAULT NULL, -- Текст, который нужно удалять из каждого поста
    post_text_add TEXT DEFAULT NULL -- Текст, который нужно добавлять в конец каждого поста
);
