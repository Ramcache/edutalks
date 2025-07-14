-- +goose Up
CREATE TABLE email_verification_tokens (
                                           id SERIAL PRIMARY KEY,
                                           user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
                                           token TEXT UNIQUE NOT NULL,
                                           expires_at TIMESTAMP NOT NULL,
                                           confirmed BOOLEAN DEFAULT FALSE
);

-- +goose Down
DROP TABLE email_verification_tokens;