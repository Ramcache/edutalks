-- +goose Up
CREATE TABLE IF NOT EXISTS password_reset_tokens (
                                                     id BIGSERIAL PRIMARY KEY,
                                                     user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                                     token_hash TEXT NOT NULL,              -- хранится только хэш
                                                     expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
                                                     used_at TIMESTAMP WITH TIME ZONE,
                                                     created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);

-- На всякий случай (если нет поля с хэшем пароля)
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- +goose Down
DROP TABLE IF EXISTS password_reset_tokens;
