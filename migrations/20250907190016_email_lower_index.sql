-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS users_email_lower_idx
    ON users (lower(email));

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS users_email_lower_idx;
