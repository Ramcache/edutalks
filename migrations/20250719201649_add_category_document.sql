-- +goose Up
ALTER TABLE documents ADD COLUMN category VARCHAR(64) DEFAULT '';

-- +goose Down
ALTER TABLE documents DROP COLUMN category;