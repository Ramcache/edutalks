-- +goose Up
-- 1) Исправим частые кириллические «похожие» буквы
UPDATE users
SET email = translate(email, 'АВЕКМНОРСТХаеорсух', 'ABEKMHOPCTXaeopcux')
WHERE email ~ '[А-Яа-яЁё]';

-- 2) Приведём к нижнему регистру
UPDATE users
SET email = lower(email)
WHERE email <> lower(email);

-- 3) Запретим кириллицу в email
ALTER TABLE users
    ADD CONSTRAINT email_no_cyrillic_chk
        CHECK (email !~ '[А-Яа-яЁё]');

-- +goose Down
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS email_no_cyrillic_chk;
