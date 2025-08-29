package repository

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"strconv"
	"strings"

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
	SaveDocument(ctx context.Context, doc *models.Document) (int, error)
	GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error)
	GetDocumentByID(ctx context.Context, id int) (*models.Document, error)
	DeleteDocument(ctx context.Context, id int) error
	GetAllDocuments(ctx context.Context) ([]*models.Document, error)
	Search(ctx context.Context, query string) ([]models.Document, error)
	GetPublicDocumentsByFilterPaginated(
		ctx context.Context,
		limit, offset int,
		sectionID *int,
		category string,
	) ([]*models.Document, int, error)
	UpdateDocumentSection(ctx context.Context, id int, sectionID *int) error
}

// Сохранение документа и возврат ID
func (r *DocumentRepository) SaveDocument(ctx context.Context, doc *models.Document) (int, error) {
	logger.Log.Info("Репозиторий: сохранение документа", zap.String("filename", doc.Filename), zap.Int("user_id", doc.UserID))
	query := `
    INSERT INTO documents (user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING id`
	var id int

	err := r.db.QueryRow(ctx, query,
		doc.UserID,
		doc.Title,
		doc.Filename,
		doc.Filepath,
		doc.Description,
		doc.IsPublic,
		doc.Category,
		doc.SectionID,
		doc.UploadedAt,
	).Scan(&id)

	if err != nil {
		logger.Log.Error("Ошибка сохранения документа (repo)", zap.Error(err))
		return 0, err
	}
	return id, nil
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
			SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at
			FROM documents
			WHERE is_public = true AND category = $1
			ORDER BY uploaded_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{category, limit, offset}
		rows, err = r.db.Query(ctx, query, args...)
	} else {
		query = `
			SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at
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
			&d.Title,
			&d.Filename,
			&d.Filepath,
			&d.Description,
			&d.IsPublic,
			&d.Category,
			&d.SectionID,
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
		SELECT id, user_id,title, filename, filepath, description, is_public, category, section_id, uploaded_at
		FROM documents WHERE id = $1
	`
	var d models.Document
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.UserID,
		&d.Title,
		&d.Filename,
		&d.Filepath,
		&d.Description,
		&d.IsPublic,
		&d.Category,
		&d.SectionID,
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
	query := `
		SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at
		FROM documents
		ORDER BY uploaded_at DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		logger.Log.Error("Ошибка получения всех документов (repo)", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.Title,
			&d.Filename,
			&d.Filepath,
			&d.Description,
			&d.IsPublic,
			&d.Category,
			&d.SectionID,
			&d.UploadedAt,
		); err != nil {
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
		SELECT id, user_id, title, filename, description, is_public, category, section_id, uploaded_at
		FROM documents
		WHERE title ILIKE $1 OR filename ILIKE $1 OR description ILIKE $1 OR category ILIKE $1
	`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(
			&doc.ID,
			&doc.UserID,
			&doc.Title,
			&doc.Filename,
			&doc.Description,
			&doc.IsPublic,
			&doc.Category,
			&doc.SectionID,
			&doc.UploadedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// Публичные документы с фильтрами (section_id и/или category)
func (r *DocumentRepository) GetPublicDocumentsByFilterPaginated(
	ctx context.Context,
	limit, offset int,
	sectionID *int,
	category string,
) ([]*models.Document, int, error) {

	var (
		rows  pgx.Rows
		err   error
		docs  []*models.Document
		args  []interface{}
		cond  []string
		total int
	)

	queryBase := `SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at
	              FROM documents WHERE is_public = true`

	if sectionID != nil {
		cond = append(cond, "section_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, *sectionID)
	}
	if category != "" {
		cond = append(cond, "category = $"+strconv.Itoa(len(args)+1))
		args = append(args, category)
	}
	if len(cond) > 0 {
		queryBase += " AND " + strings.Join(cond, " AND ")
	}

	query := queryBase + " ORDER BY uploaded_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err = r.db.Query(ctx, query, args...)
	if err != nil {
		logger.Log.Error("Ошибка выборки документов (repo, filter)", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.Document
		if err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.Title,
			&d.Filename,
			&d.Filepath,
			&d.Description,
			&d.IsPublic,
			&d.Category,
			&d.SectionID,
			&d.UploadedAt,
		); err != nil {
			return nil, 0, err
		}
		docs = append(docs, &d)
	}

	// total
	countQuery := `SELECT COUNT(*) FROM documents WHERE is_public = true`
	argsCnt := []interface{}{}
	if len(cond) > 0 {
		countQuery += " AND " + strings.Join(cond, " AND ")
		argsCnt = append(argsCnt, args[:len(args)-2]...) // без limit/offset
	}
	if err := r.db.QueryRow(ctx, countQuery, argsCnt...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return docs, total, nil
}

func (r *DocumentRepository) UpdateDocumentSection(ctx context.Context, id int, sectionID *int) error {
	_, err := r.db.Exec(ctx, `UPDATE documents SET section_id=$1, uploaded_at=uploaded_at WHERE id=$2`, sectionID, id)
	return err
}
