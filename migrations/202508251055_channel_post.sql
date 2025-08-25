-- Таблица постов из каналов
CREATE TABLE IF NOT EXISTS channel_post (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современный автоинкремент
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE, -- при удалении заказа удаляем и пост
    post_date_time TIMESTAMPTZ NOT NULL, -- сохраняем дату и время публикации с часовой зоной, приложение передаёт МСК
    post_url TEXT NOT NULL -- ссылка на пост нужна для последующего доступа
);
