package repository

import (
	"context"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NewsRepository struct {
	db *pgxpool.Pool
}

func NewNewsRepository(db *pgxpool.Pool) *NewsRepository {
	return &NewsRepository{db: db}
}

func (r *NewsRepository) Create(ctx context.Context, news *models.News) error {
	query := `INSERT INTO news (title, content) VALUES ($1, $2)`
	_, err := r.db.Exec(ctx, query, news.Title, news.Content)
	return err
}

func (r *NewsRepository) List(ctx context.Context) ([]*models.News, error) {
	rows, err := r.db.Query(ctx, `SELECT id, title, content, created_at FROM news ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var newsList []*models.News
	for rows.Next() {
		var n models.News
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt); err != nil {
			return nil, err
		}
		newsList = append(newsList, &n)
	}

	return newsList, nil
}

func (r *NewsRepository) GetByID(ctx context.Context, id int) (*models.News, error) {
	query := `SELECT id, title, content, created_at FROM news WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)

	var n models.News
	if err := row.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NewsRepository) Update(ctx context.Context, id int, title, content string) error {
	query := `UPDATE news SET title = $1, content = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, title, content, id)
	return err
}

func (r *NewsRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM news WHERE id = $1`, id)
	return err
}
