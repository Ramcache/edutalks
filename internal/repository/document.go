package repository

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DocumentRepository struct {
	db *pgxpool.Pool
}

func NewDocumentRepository(db *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{db: db}
}

type DocumentRepo interface {
	SaveDocument(ctx context.Context, doc *models.Document) error
	GetPublicDocumentsPaginated(ctx context.Context, limit, offset int) ([]*models.Document, int, error)
	GetDocumentByID(ctx context.Context, id int) (*models.Document, error)
	DeleteDocument(ctx context.Context, id int) error
	GetAllDocuments(ctx context.Context) ([]*models.Document, error)
}

func (r *DocumentRepository) SaveDocument(ctx context.Context, doc *models.Document) error {
	logger.Log.Info("Репозиторий: сохранение документа", zap.String("filename", doc.Filename), zap.Int("user_id", doc.UserID))
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
	if err != nil {
		logger.Log.Error("Ошибка сохранения документа (repo)", zap.Error(err))
	}
	return err
}

func (r *DocumentRepository) GetPublicDocumentsPaginated(ctx context.Context, limit, offset int) ([]*models.Document, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, filename, filepath, description, is_public, uploaded_at
		FROM documents
		WHERE is_public = true
		ORDER BY uploaded_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		logger.Log.Error("Ошибка получения публичных документов (repo)", zap.Error(err))
		return nil, 0, err
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
			logger.Log.Error("Ошибка сканирования документа (repo)", zap.Error(err))
			return nil, 0, err
		}
		docs = append(docs, &d)
	}

	// Общее число документов (total)
	var total int
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM documents WHERE is_public = true
	`).Scan(&total)
	if err != nil {
		logger.Log.Error("Ошибка подсчёта документов (repo)", zap.Error(err))
		return nil, 0, err
	}

	return docs, total, nil
}

func (r *DocumentRepository) GetDocumentByID(ctx context.Context, id int) (*models.Document, error) {
	logger.Log.Info("Репозиторий: получение документа по ID", zap.Int("doc_id", id))
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
		logger.Log.Error("Ошибка получения документа по ID (repo)", zap.Int("doc_id", id), zap.Error(err))
		return nil, err
	}
	return &d, nil
}

func (r *DocumentRepository) DeleteDocument(ctx context.Context, id int) error {
	logger.Log.Info("Репозиторий: удаление документа", zap.Int("doc_id", id))
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		logger.Log.Error("Ошибка удаления документа (repo)", zap.Int("doc_id", id), zap.Error(err))
	}
	return err
}

func (r *DocumentRepository) GetAllDocuments(ctx context.Context) ([]*models.Document, error) {
	query := `SELECT id, user_id, filename, filepath, is_public, description, uploaded_at FROM documents ORDER BY uploaded_at DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		logger.Log.Error("Ошибка получения всех документов (repo)", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(&d.ID, &d.UserID, &d.Filename, &d.Filepath, &d.IsPublic, &d.Description, &d.UploadedAt); err != nil {
			logger.Log.Error("Ошибка сканирования документа (repo)", zap.Error(err))
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, nil
}
