-- Перевод поля category в массив, чтобы один заказ мог относиться к нескольким категориям
ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_category_fkey,
    ALTER COLUMN category TYPE TEXT[] USING ARRAY[category], -- существующие значения превращаем в массив из одного элемента
    ALTER COLUMN category SET DEFAULT NULL; -- избегаем значения по умолчанию, чтобы отсутствие категорий обозначалось NULL

