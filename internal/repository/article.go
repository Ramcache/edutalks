package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"edutalks/internal/models"
)

type ArticleRepo interface {
	Create(ctx context.Context, a *models.Article) (*models.Article, error)
}

type articleRepo struct{ db *pgxpool.Pool }

func NewArticleRepo(db *pgxpool.Pool) ArticleRepo { return &articleRepo{db: db} }

func (r *articleRepo) Create(ctx context.Context, a *models.Article) (*models.Article, error) {
	tagsJSON, _ := json.Marshal(a.Tags) // []string -> jsonb

	const q = `
		INSERT INTO articles (author_id, title, summary, body_html, tags, is_published, published_at)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6, CASE WHEN $6 THEN now() ELSE NULL END)
		RETURNING id, author_id, title, summary, body_html, is_published, published_at, created_at, updated_at
	`

	var out models.Article
	err := r.db.
		QueryRow(ctx, q,
			a.AuthorID,    // *int64 (nullable)
			a.Title,       // string
			a.Summary,     // *string (nullable)
			a.BodyHTML,    // string
			tagsJSON,      // []byte -> jsonb
			a.IsPublished, // bool
		).
		Scan(
			&out.ID,
			&out.AuthorID, // *int64
			&out.Title,
			&out.Summary, // *string
			&out.BodyHTML,
			&out.IsPublished,
			&out.PublishedAt, // *time.Time
			&out.CreatedAt,
			&out.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}

	out.Tags = a.Tags
	return &out, nil
}
