-- +goose Up
CREATE TABLE news (
                      id SERIAL PRIMARY KEY,
                      title TEXT NOT NULL,
                      content TEXT NOT NULL,
                      created_at TIMESTAMP DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS news;
