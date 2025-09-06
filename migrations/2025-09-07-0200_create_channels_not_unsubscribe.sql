-- Таблица с каналами и группами, отписка от которых запрещена.
CREATE TABLE channels_not_unsubscribe (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- Уникальный идентификатор записи
    url_channel TEXT NOT NULL UNIQUE                     -- Ссылка на канал или группу
);
