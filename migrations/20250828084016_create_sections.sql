-- +goose Up
CREATE TABLE sections (
                          id          SERIAL PRIMARY KEY,
                          tab_id      INT NOT NULL REFERENCES tabs(id) ON DELETE CASCADE,
                          slug        TEXT NOT NULL,
                          title       TEXT NOT NULL,
                          description TEXT NOT NULL DEFAULT '',
                          position    INT NOT NULL DEFAULT 0,
                          is_active   BOOLEAN NOT NULL DEFAULT TRUE,
                          created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
                          updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
                          UNIQUE(tab_id, slug)
);
CREATE INDEX idx_sections_tab ON sections(tab_id);
CREATE INDEX idx_sections_active ON sections(is_active);

-- +goose Down
DROP TABLE IF EXISTS sections;
