-- +goose Up
ALTER TABLE documents
    ADD COLUMN allow_free_download BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE documents DROP COLUMN allow_free_download;
