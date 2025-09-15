-- +goose Up
CREATE TABLE IF NOT EXISTS access_token_blacklist (
                                                      id SERIAL PRIMARY KEY,
                                                      token TEXT NOT NULL,
                                                      expires_at TIMESTAMPTZ NOT NULL,
                                                      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_access_token_blacklist_expires_at
    ON access_token_blacklist (expires_at);

-- +goose Down
DROP TABLE IF EXISTS access_token_blacklist;
