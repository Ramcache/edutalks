-- +goose Up
ALTER TABLE users ADD COLUMN email_subscription BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN email_subscription;
