-- Создаёт таблицу для хранения ссылок удалённых каналов и причин их удаления
CREATE TABLE category_channels_delete (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- Уникальный идентификатор записи
    channel_url TEXT NOT NULL,                           -- Ссылка на канал
    reason TEXT NOT NULL CHECK (reason IN (
        'не существует канала по username',              -- Канал с таким username не найден
        'закрытый канал',                                -- Закрытый канал
        'недоступно обсуждение'                          -- Включены ограничения на обсуждение
    )) -- Причина удаления с ограничением значений
);
