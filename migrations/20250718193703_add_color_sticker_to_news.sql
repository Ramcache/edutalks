-- +goose Up
ALTER TABLE news
    ADD COLUMN color   VARCHAR(32) DEFAULT '',
    ADD COLUMN sticker VARCHAR(128) DEFAULT '';

-- +goose Down
ALTER TABLE news
    DROP COLUMN color,
    DROP COLUMN sticker;