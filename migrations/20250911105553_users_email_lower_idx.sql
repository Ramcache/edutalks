-- +goose Up
-- +goose NO TRANSACTION
UPDATE users
SET email = lower(email)
WHERE email <> lower(email);

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS users_email_lower_idx
    ON users (lower(email));

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX CONCURRENTLY IF EXISTS users_email_lower_idx;
