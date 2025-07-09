-- +goose Up
CREATE TABLE documents (
                           id SERIAL PRIMARY KEY,
                           user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
                           filename TEXT NOT NULL,
                           filepath TEXT NOT NULL,
                           uploaded_at TIMESTAMP DEFAULT now()
);

-- +goose Down
DROP TABLE documents;
