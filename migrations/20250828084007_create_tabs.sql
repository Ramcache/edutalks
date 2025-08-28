-- +goose Up
CREATE TABLE tabs (
                      id         SERIAL PRIMARY KEY,
                      slug       TEXT NOT NULL UNIQUE,
                      title      TEXT NOT NULL,
                      position   INT  NOT NULL DEFAULT 0,
                      is_active  BOOLEAN NOT NULL DEFAULT TRUE,
                      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                      updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_tabs_active ON tabs(is_active);

-- +goose Down
DROP TABLE IF EXISTS tabs;
