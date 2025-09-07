-- +goose NO TRANSACTION

-- +goose Up
-- Индекс для быстрого поиска по последним 10 цифрам телефона
CREATE INDEX CONCURRENTLY IF NOT EXISTS users_phone_last10_idx
    ON users (right(regexp_replace(phone, '\D', '', 'g'), 10));

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS users_phone_last10_idx;
