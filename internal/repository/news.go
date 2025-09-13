package repository

import (
	"context"

	"edutalks/internal/logger"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type NewsRepository struct {
	db *pgxpool.Pool
}

func NewNewsRepository(db *pgxpool.Pool) *NewsRepository {
	return &NewsRepository{db: db}
}

type NewsRepo interface {
	Create(ctx context.Context, news *models.News) (int, error)
	ListPaginated(ctx context.Context, limit, offset int) ([]*models.News, int, error)
	GetByID(ctx context.Context, id int) (*models.News, error)
	Update(ctx context.Context, id int, title, content, imageURL, color, sticker string) error
	Delete(ctx context.Context, id int) error
	Search(ctx context.Context, query string) ([]models.News, error)
}

func (r *NewsRepository) Create(ctx context.Context, news *models.News) (int, error) {
	log := logger.WithCtx(ctx)

	const q = `
		INSERT INTO news (title, content, image_url, color, sticker, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`

	var id int
	if err := r.db.QueryRow(ctx, q,
		news.Title,
		news.Content,
		news.ImageURL,
		news.Color,
		news.Sticker,
	).Scan(&id); err != nil {
		log.Error("news repo: create failed", zap.Error(err), zap.String("title", news.Title))
		return 0, err
	}

	log.Info("news repo: created", zap.Int("id", id), zap.String("title", news.Title))
	return id, nil
}

func (r *NewsRepository) ListPaginated(ctx context.Context, limit, offset int) ([]*models.News, int, error) {
	log := logger.WithCtx(ctx)

	rows, err := r.db.Query(ctx, `
		SELECT id, title, content, created_at, image_url, color, sticker
		FROM news
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		log.Error("news repo: list paginated query failed", zap.Error(err),
			zap.Int("limit", limit), zap.Int("offset", offset))
		return nil, 0, err
	}
	defer rows.Close()

	var newsList []*models.News
	for rows.Next() {
		var n models.News
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.ImageURL, &n.Color, &n.Sticker); err != nil {
			log.Error("news repo: scan list paginated failed", zap.Error(err))
			return nil, 0, err
		}
		newsList = append(newsList, &n)
	}
	if err := rows.Err(); err != nil {
		log.Error("news repo: rows error list paginated", zap.Error(err))
		return nil, 0, err
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM news`).Scan(&total); err != nil {
		log.Error("news repo: count failed", zap.Error(err))
		return nil, 0, err
	}

	log.Debug("news repo: list paginated done",
		zap.Int("returned", len(newsList)), zap.Int("total", total),
		zap.Int("limit", limit), zap.Int("offset", offset))
	return newsList, total, nil
}

func (r *NewsRepository) GetByID(ctx context.Context, id int) (*models.News, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, title, content, created_at, image_url, color, sticker
		FROM news WHERE id = $1
	`
	var n models.News
	if err := r.db.QueryRow(ctx, q, id).Scan(
		&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.ImageURL, &n.Color, &n.Sticker,
	); err != nil {
		if err == pgx.ErrNoRows {
			log.Warn("news repo: not found", zap.Int("id", id))
		} else {
			log.Error("news repo: get by id failed", zap.Error(err), zap.Int("id", id))
		}
		return nil, err
	}

	log.Debug("news repo: got by id", zap.Int("id", id))
	return &n, nil
}

func (r *NewsRepository) Update(ctx context.Context, id int, title, content, imageURL, color, sticker string) error {
	log := logger.WithCtx(ctx)

	const q = `
		UPDATE news
		SET title = $1, content = $2, image_url = $3, color = $4, sticker = $5
		WHERE id = $6
	`
	if _, err := r.db.Exec(ctx, q, title, content, imageURL, color, sticker, id); err != nil {
		log.Error("news repo: update failed", zap.Error(err), zap.Int("id", id))
		return err
	}

	log.Info("news repo: updated", zap.Int("id", id))
	return nil
}

func (r *NewsRepository) Delete(ctx context.Context, id int) error {
	log := logger.WithCtx(ctx)

	if _, err := r.db.Exec(ctx, `DELETE FROM news WHERE id = $1`, id); err != nil {
		log.Error("news repo: delete failed", zap.Error(err), zap.Int("id", id))
		return err
	}

	log.Info("news repo: deleted", zap.Int("id", id))
	return nil
}

func (r *NewsRepository) Search(ctx context.Context, query string) ([]models.News, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, title, content, image_url, color, sticker, created_at
		FROM news
		WHERE title ILIKE $1 OR content ILIKE $1
	`
	pattern := "%" + query + "%"

	rows, err := r.db.Query(ctx, q, pattern)
	if err != nil {
		log.Error("news repo: search query failed", zap.Error(err), zap.String("query", query))
		return nil, err
	}
	defer rows.Close()

	var results []models.News
	for rows.Next() {
		var n models.News
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.ImageURL, &n.Color, &n.Sticker, &n.CreatedAt); err != nil {
			log.Error("news repo: scan search failed", zap.Error(err))
			return nil, err
		}
		results = append(results, n)
	}
	if err := rows.Err(); err != nil {
		log.Error("news repo: rows error search", zap.Error(err))
		return nil, err
	}

	log.Debug("news repo: search done", zap.String("query", query), zap.Int("returned", len(results)))
	return results, nil
}
