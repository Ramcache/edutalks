-- +goose Up
CREATE TABLE IF NOT EXISTS articles (
                                        id           BIGSERIAL PRIMARY KEY,
                                        author_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
                                        title        VARCHAR(255) NOT NULL,
                                        summary      TEXT,
                                        body_html    TEXT NOT NULL,
                                        tags         JSONB NOT NULL DEFAULT '[]'::jsonb,
                                        is_published BOOLEAN NOT NULL DEFAULT TRUE,
                                        published_at TIMESTAMPTZ,
                                        created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
                                        updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_articles_tags_gin ON articles USING GIN (tags);

-- +goose Down
DROP TABLE IF EXISTS articles;
