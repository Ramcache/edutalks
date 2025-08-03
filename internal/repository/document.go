package repository

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5"
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
	GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error)
	GetDocumentByID(ctx context.Context, id int) (*models.Document, error)
	DeleteDocument(ctx context.Context, id int) error
	GetAllDocuments(ctx context.Context) ([]*models.Document, error)
	Search(ctx context.Context, query string) ([]models.Document, error)
}

// Сохранение документа
func (r *DocumentRepository) SaveDocument(ctx context.Context, doc *models.Document) error {
	logger.Log.Info("Репозиторий: сохранение документа", zap.String("filename", doc.Filename), zap.Int("user_id", doc.UserID))
	query := `
		INSERT INTO documents (user_id, filename, filepath, description, is_public, category, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query,
		doc.UserID,
		doc.Filename,
		doc.Filepath,
		doc.Description,
		doc.IsPublic,
		doc.Category,
		doc.UploadedAt,
	)
	if err != nil {
		logger.Log.Error("Ошибка сохранения документа (repo)", zap.Error(err))
	}
	return err
}

// Публичные документы с фильтром по категории (если передана)
func (r *DocumentRepository) GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error) {
	var (
		rows  pgx.Rows
		err   error
		docs  []*models.Document
		query string
		args  []interface{}
		total int
	)

	if category != "" {
		query = `
			SELECT id, user_id, filename, filepath, description, is_public, category, uploaded_at
			FROM documents
			WHERE is_public = true AND category = $1
			ORDER BY uploaded_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{category, limit, offset}
		rows, err = r.db.Query(ctx, query, args...)
	} else {
		query = `
			SELECT id, user_id, filename, filepath, description, is_public, category, uploaded_at
			FROM documents
			WHERE is_public = true
			ORDER BY uploaded_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
		rows, err = r.db.Query(ctx, query, args...)
	}
	if err != nil {
		logger.Log.Error("Ошибка получения публичных документов (repo)", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.Document
		err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.Filename,
			&d.Filepath,
			&d.Description,
			&d.IsPublic,
			&d.Category,
			&d.UploadedAt,
		)
		if err != nil {
			logger.Log.Error("Ошибка сканирования документа (repo)", zap.Error(err))
			return nil, 0, err
		}
		docs = append(docs, &d)
	}

	// total (фильтр по категории, если задана)
	if category != "" {
		err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM documents WHERE is_public = true AND category = $1`, category).Scan(&total)
	} else {
		err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM documents WHERE is_public = true`).Scan(&total)
	}
	if err != nil {
		logger.Log.Error("Ошибка подсчёта документов (repo)", zap.Error(err))
		return nil, 0, err
	}

	return docs, total, nil
}

// Получение по ID
func (r *DocumentRepository) GetDocumentByID(ctx context.Context, id int) (*models.Document, error) {
	logger.Log.Info("Репозиторий: получение документа по ID", zap.Int("doc_id", id))
	query := `
		SELECT id, user_id, filename, filepath, description, is_public, category, uploaded_at
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
		&d.Category,
		&d.UploadedAt,
	)
	if err != nil {
		logger.Log.Error("Ошибка получения документа по ID (repo)", zap.Int("doc_id", id), zap.Error(err))
		return nil, err
	}
	return &d, nil
}

// Удаление
func (r *DocumentRepository) DeleteDocument(ctx context.Context, id int) error {
	logger.Log.Info("Репозиторий: удаление документа", zap.Int("doc_id", id))
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		logger.Log.Error("Ошибка удаления документа (repo)", zap.Int("doc_id", id), zap.Error(err))
	}
	return err
}

// Для админки — все документы
func (r *DocumentRepository) GetAllDocuments(ctx context.Context) ([]*models.Document, error) {
	query := `SELECT id, user_id, filename, filepath, description, is_public, category, uploaded_at FROM documents ORDER BY uploaded_at DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		logger.Log.Error("Ошибка получения всех документов (repo)", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(&d.ID, &d.UserID, &d.Filename, &d.Filepath, &d.Description, &d.IsPublic, &d.Category, &d.UploadedAt); err != nil {
			logger.Log.Error("Ошибка сканирования документа (repo)", zap.Error(err))
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, nil
}

func (r *DocumentRepository) Search(ctx context.Context, query string) ([]models.Document, error) {
	q := "%" + query + "%"
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, filename, description, is_public, category, uploaded_at
		FROM documents
		WHERE filename ILIKE $1 OR description ILIKE $1 OR category ILIKE $1
	`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(
			&doc.ID, &doc.UserID, &doc.Filename, &doc.Description,
			&doc.IsPublic, &doc.Category, &doc.UploadedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
