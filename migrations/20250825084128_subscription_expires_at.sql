-- +goose Up
ALTER TABLE users
    ADD COLUMN subscription_expires_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN subscription_expires_at;