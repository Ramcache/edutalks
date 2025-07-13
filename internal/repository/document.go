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
	query := `
		INSERT INTO documents (user_id, filename, filepath, description, is_public, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query,
		doc.UserID,
		doc.Filename,
		doc.Filepath,
		doc.Description,
		doc.IsPublic,
		doc.UploadedAt,
	)
	return err
}

func (r *DocumentRepository) GetPublicDocuments(ctx context.Context) ([]*models.Document, error) {
	query := `
		SELECT id, user_id, filename, filepath, description, is_public, uploaded_at
		FROM documents
		WHERE is_public = true
		ORDER BY uploaded_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var d models.Document
		err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.Filename,
			&d.Filepath,
			&d.Description,
			&d.IsPublic,
			&d.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}

	return docs, nil
}

func (r *DocumentRepository) GetDocumentByID(ctx context.Context, id int) (*models.Document, error) {
	query := `
		SELECT id, user_id, filename, filepath, description, is_public, uploaded_at
		FROM documents WHERE id = $1
	`
	var d models.Document
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.UserID,
		&d.Filename,
		&d.Filepath,
		&d.Description,
		&d.IsPublic,
		&d.UploadedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DocumentRepository) DeleteDocument(ctx context.Context, id int) error {
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
