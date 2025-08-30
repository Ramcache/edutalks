-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS ux_tabs_slug ON tabs (slug);
CREATE UNIQUE INDEX IF NOT EXISTS ux_sections_tab_slug ON sections (tab_id, slug);

-- +goose Down
DROP INDEX IF EXISTS ux_sections_tab_slug;
DROP INDEX IF EXISTS ux_tabs_slug;
