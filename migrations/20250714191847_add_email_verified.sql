-- +goose Up
ALTER TABLE users ADD COLUMN email_verified BOOLEAN DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROP COLUMN email_verified;