-- +goose Up
CREATE TABLE refresh_tokens (
                                id SERIAL PRIMARY KEY,
                                user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                token TEXT NOT NULL,
                                created_at TIMESTAMP DEFAULT now()
);

-- +goose Down
DROP TABLE refresh_tokens;
