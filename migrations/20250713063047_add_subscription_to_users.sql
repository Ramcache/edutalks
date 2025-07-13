-- +goose Up
ALTER TABLE users
    ADD COLUMN has_subscription BOOLEAN DEFAULT false;

-- +goose Down
ALTER TABLE users
    DROP COLUMN has_subscription;
