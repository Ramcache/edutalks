-- +goose Up
ALTER TABLE email_verification_tokens ADD COLUMN created_at TIMESTAMP DEFAULT now();

-- +goose Down
ALTER TABLE email_verification_tokens DROP COLUMN created_at;