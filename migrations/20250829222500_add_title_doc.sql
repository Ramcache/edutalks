-- +goose Up
ALTER TABLE documents ADD COLUMN title TEXT;

-- +goose Down
ALTER TABLE documents DROP COLUMN IF EXISTS title;