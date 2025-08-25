-- Делаем поле url_default уникальным, чтобы один и тот же URL нельзя было использовать в разных заказах
ALTER TABLE orders
    ADD CONSTRAINT orders_url_default_unique UNIQUE (url_default);
