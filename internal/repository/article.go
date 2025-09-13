package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"edutalks/internal/logger"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type ArticleRepo interface {
	Create(ctx context.Context, a *models.Article) (*models.Article, error)
	GetAll(ctx context.Context, limit, offset int, tag string, onlyPublished bool) ([]*models.Article, error)
	GetByID(ctx context.Context, id int64) (*models.Article, error)
	Update(ctx context.Context, a *models.Article) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
	UpdatePublish(ctx context.Context, id int64, publish bool) error
}

type articleRepo struct{ db *pgxpool.Pool }

func NewArticleRepo(db *pgxpool.Pool) ArticleRepo { return &articleRepo{db: db} }

func (r *articleRepo) Create(ctx context.Context, a *models.Article) (*models.Article, error) {
	log := logger.WithCtx(ctx)

	tagsJSON, _ := json.Marshal(a.Tags)
	const q = `
		INSERT INTO articles (author_id, title, summary, body_html, tags, is_published, published_at)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6, CASE WHEN $6 THEN NOW() ELSE NULL END)
		RETURNING id, author_id, title, summary, body_html, is_published, published_at, created_at, updated_at, tags
	`

	var out models.Article
	var tagsRaw []byte
	err := r.db.QueryRow(ctx, q,
		a.AuthorID,
		a.Title,
		a.Summary,
		a.BodyHTML,
		tagsJSON,
		a.IsPublished,
	).Scan(
		&out.ID,
		&out.AuthorID,
		&out.Title,
		&out.Summary,
		&out.BodyHTML,
		&out.IsPublished,
		&out.PublishedAt,
		&out.CreatedAt,
		&out.UpdatedAt,
		&tagsRaw,
	)
	if err != nil {
		log.Error("article repo: create failed", zap.Error(err))
		return nil, err
	}
	if err := json.Unmarshal(tagsRaw, &out.Tags); err != nil {
		log.Warn("article repo: failed to unmarshal tags after create", zap.Error(err))
	}
	log.Info("article repo: created",
		zap.Int64("id", out.ID),
		zap.Bool("published", out.IsPublished),
	)
	return &out, nil
}

func (r *articleRepo) GetAll(ctx context.Context, limit, offset int, tag string, onlyPublished bool) ([]*models.Article, error) {
	log := logger.WithCtx(ctx)

	const qBase = `
		SELECT id, author_id, title, summary, body_html, is_published, published_at, created_at, updated_at, tags
		FROM articles
	`
	where := []string{}
	args := []any{}
	i := 1

	if onlyPublished {
		where = append(where, fmt.Sprintf("is_published = $%d", i))
		args = append(args, true)
		i++
	}
	if tag != "" {
		where = append(where, fmt.Sprintf(`
			EXISTS (
				SELECT 1
				FROM jsonb_array_elements_text(tags) AS t(val)
				WHERE t.val = $%d
			)`, i))
		args = append(args, tag)
		i++
	}

	sql := qBase
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		log.Error("article repo: get all query failed", zap.Error(err),
			zap.Int("limit", limit), zap.Int("offset", offset), zap.String("tag", tag), zap.Bool("only_published", onlyPublished))
		return nil, err
	}
	defer rows.Close()

	var list []*models.Article
	for rows.Next() {
		var a models.Article
		var tagsRaw []byte
		if err := rows.Scan(
			&a.ID, &a.AuthorID, &a.Title, &a.Summary, &a.BodyHTML,
			&a.IsPublished, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt, &tagsRaw,
		); err != nil {
			log.Error("article repo: scan in get all failed", zap.Error(err))
			return nil, err
		}
		if err := json.Unmarshal(tagsRaw, &a.Tags); err != nil {
			log.Warn("article repo: failed to unmarshal tags in get all", zap.Error(err), zap.Int64("id", a.ID))
		}
		list = append(list, &a)
	}
	if err := rows.Err(); err != nil {
		log.Error("article repo: rows error in get all", zap.Error(err))
		return nil, err
	}

	log.Debug("article repo: get all done",
		zap.Int("returned", len(list)),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("tag", tag),
		zap.Bool("only_published", onlyPublished),
	)
	return list, nil
}

func (r *articleRepo) GetByID(ctx context.Context, id int64) (*models.Article, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, author_id, title, summary, body_html, is_published, published_at, created_at, updated_at, tags
		FROM articles WHERE id=$1
	`
	var a models.Article
	var tagsRaw []byte
	if err := r.db.QueryRow(ctx, q, id).Scan(
		&a.ID, &a.AuthorID, &a.Title, &a.Summary, &a.BodyHTML,
		&a.IsPublished, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt, &tagsRaw,
	); err != nil {
		log.Warn("article repo: get by id failed", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}
	if err := json.Unmarshal(tagsRaw, &a.Tags); err != nil {
		log.Warn("article repo: failed to unmarshal tags in get by id", zap.Error(err), zap.Int64("id", id))
	}
	log.Debug("article repo: got by id", zap.Int64("id", id), zap.Bool("published", a.IsPublished))
	return &a, nil
}

func (r *articleRepo) Update(ctx context.Context, a *models.Article) error {
	log := logger.WithCtx(ctx)

	tagsJSON, _ := json.Marshal(a.Tags)
	const q = `
		UPDATE articles
		SET title=$1,
		    summary=$2,
		    body_html=$3,
		    tags=$4::jsonb,
		    is_published=$5,
		    published_at = CASE WHEN $5 THEN COALESCE(published_at, NOW()) ELSE NULL END,
		    updated_at=NOW()
		WHERE id=$6
	`
	_, err := r.db.Exec(ctx, q, a.Title, a.Summary, a.BodyHTML, tagsJSON, a.IsPublished, a.ID)
	if err != nil {
		log.Error("article repo: update failed", zap.Error(err), zap.Int64("id", a.ID))
		return err
	}
	log.Info("article repo: updated", zap.Int64("id", a.ID), zap.Bool("published", a.IsPublished))
	return nil
}

func (r *articleRepo) Delete(ctx context.Context, id int64) error {
	log := logger.WithCtx(ctx)

	_, err := r.db.Exec(ctx, "DELETE FROM articles WHERE id=$1", id)
	if err != nil {
		log.Error("article repo: delete failed", zap.Error(err), zap.Int64("id", id))
		return err
	}
	log.Info("article repo: deleted", zap.Int64("id", id))
	return nil
}

func (r *articleRepo) Exists(ctx context.Context, id int64) (bool, error) {
	log := logger.WithCtx(ctx)

	const q = `SELECT EXISTS(SELECT 1 FROM articles WHERE id = $1)`
	var ok bool
	if err := r.db.QueryRow(ctx, q, id).Scan(&ok); err != nil {
		log.Error("article repo: exists query failed", zap.Error(err), zap.Int64("id", id))
		return false, err
	}
	log.Debug("article repo: exists check", zap.Int64("id", id), zap.Bool("exists", ok))
	return ok, nil
}

func (r *articleRepo) UpdatePublish(ctx context.Context, id int64, publish bool) error {
	log := logger.WithCtx(ctx)

	const q = `
		UPDATE articles
		SET is_published = $2,
		    published_at = CASE WHEN $2 THEN COALESCE(published_at, NOW()) ELSE NULL END,
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, id, publish)
	if err != nil {
		log.Error("article repo: update publish failed", zap.Error(err), zap.Int64("id", id), zap.Bool("publish", publish))
		return err
	}
	log.Info("article repo: publish updated", zap.Int64("id", id), zap.Bool("publish", publish))
	return nil
}
