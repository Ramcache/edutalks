-- +goose Up
ALTER TABLE documents
    ADD COLUMN description TEXT,
    ADD COLUMN is_public BOOLEAN DEFAULT false;

-- +goose Down
ALTER TABLE documents
    DROP COLUMN description,
    DROP COLUMN is_public;
