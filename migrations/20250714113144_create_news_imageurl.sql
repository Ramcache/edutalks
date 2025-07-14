-- +goose Up
ALTER TABLE news
    ADD COLUMN image_url TEXT DEFAULT '';

-- +goose Down
ALTER TABLE news
    DROP COLUMN image_url;