-- Обновляет допустимые значения причины удаления канала
ALTER TABLE category_channels_delete
    DROP CONSTRAINT IF EXISTS category_channels_delete_reason_check,
    ADD CONSTRAINT category_channels_delete_reason_check CHECK (reason IN (
        'не существует канала по ссылке',
        'канал закрыт',
        'недоступно обсуждение'
    ));

