package repository

import (
	"context"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentRepository struct {
	db *pgxpool.Pool
}

func NewDocumentRepository(db *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) SaveDocument(ctx context.Context, doc *models.Document) error {
	query := `INSERT INTO documents (user_id, filename, filepath) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(ctx, query, doc.UserID, doc.Filename, doc.Filepath)
	return err
}
