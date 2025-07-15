package repository

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"

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
	Create(ctx context.Context, news *models.News) error
	ListPaginated(ctx context.Context, limit, offset int) ([]*models.News, int, error)
	GetByID(ctx context.Context, id int) (*models.News, error)
	Update(ctx context.Context, id int, title, content, imageURL string) error
	Delete(ctx context.Context, id int) error
}

func (r *NewsRepository) Create(ctx context.Context, news *models.News) error {
	logger.Log.Info("Репозиторий: создание новости", zap.String("title", news.Title))
	query := `INSERT INTO news (title, content, image_url) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(ctx, query, news.Title, news.Content, news.ImageURL)
	if err != nil {
		logger.Log.Error("Ошибка создания новости (repo)", zap.Error(err))
	}
	return err
}

func (r *NewsRepository) ListPaginated(ctx context.Context, limit, offset int) ([]*models.News, int, error) {
	// Получаем сами новости
	rows, err := r.db.Query(ctx, `
		SELECT id, title, content, created_at, image_url
		FROM news
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		logger.Log.Error("Ошибка получения списка новостей (repo)", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var newsList []*models.News
	for rows.Next() {
		var n models.News
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.ImageURL); err != nil {
			logger.Log.Error("Ошибка сканирования новости (repo)", zap.Error(err))
			return nil, 0, err
		}
		newsList = append(newsList, &n)
	}

	// Получаем общее количество новостей (для total)
	var total int
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM news").Scan(&total)
	if err != nil {
		logger.Log.Error("Ошибка подсчёта новостей (repo)", zap.Error(err))
		return nil, 0, err
	}

	return newsList, total, nil
}

func (r *NewsRepository) GetByID(ctx context.Context, id int) (*models.News, error) {
	logger.Log.Info("Репозиторий: получение новости по ID", zap.Int("news_id", id))
	query := `SELECT id, title, content, created_at, image_url FROM news WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)

	var n models.News
	if err := row.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.ImageURL); err != nil {
		logger.Log.Error("Ошибка получения новости по ID (repo)", zap.Int("news_id", id), zap.Error(err))
		return nil, err
	}

	return &n, nil
}

func (r *NewsRepository) Update(ctx context.Context, id int, title, content, imageURL string) error {
	logger.Log.Info("Репозиторий: обновление новости", zap.Int("news_id", id))
	query := `UPDATE news SET title = $1, content = $2, image_url = $3 WHERE id = $4`
	_, err := r.db.Exec(ctx, query, title, content, imageURL, id)
	if err != nil {
		logger.Log.Error("Ошибка обновления новости (repo)", zap.Int("news_id", id), zap.Error(err))
	}
	return err
}

func (r *NewsRepository) Delete(ctx context.Context, id int) error {
	logger.Log.Info("Репозиторий: удаление новости", zap.Int("news_id", id))
	_, err := r.db.Exec(ctx, `DELETE FROM news WHERE id = $1`, id)
	if err != nil {
		logger.Log.Error("Ошибка удаления новости (repo)", zap.Int("news_id", id), zap.Error(err))
	}
	return err
}
