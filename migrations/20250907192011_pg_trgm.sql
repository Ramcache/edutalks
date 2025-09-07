-- +goose NO TRANSACTION
-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- +goose Down
-- (обычно не откатываем расширение)
