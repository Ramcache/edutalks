-- +goose Up
CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       username VARCHAR(50) UNIQUE NOT NULL,
                       full_name TEXT NOT NULL,
                       phone VARCHAR(20),
                       email TEXT UNIQUE NOT NULL,
                       address TEXT,
                       password_hash TEXT NOT NULL,
                       role VARCHAR(20) DEFAULT 'user',
                       created_at TIMESTAMP DEFAULT now(),
                       updated_at TIMESTAMP DEFAULT now()
);

-- +goose Down
DROP TABLE users;
