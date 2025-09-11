-- +goose Up
DELETE FROM email_verification_tokens a
    USING email_verification_tokens b
WHERE a.user_id = b.user_id
  AND a.created_at < b.created_at;

ALTER TABLE email_verification_tokens
    ADD CONSTRAINT uniq_user_token UNIQUE (user_id);

-- +goose Down
ALTER TABLE email_verification_tokens
    DROP CONSTRAINT IF EXISTS uniq_user_token;
