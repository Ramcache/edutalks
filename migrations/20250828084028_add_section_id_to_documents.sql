-- +goose Up
ALTER TABLE documents
    ADD COLUMN section_id INT REFERENCES sections(id) ON DELETE SET NULL;

CREATE INDEX idx_documents_section ON documents(section_id);

-- +goose Down
ALTER TABLE documents DROP COLUMN IF EXISTS section_id;
