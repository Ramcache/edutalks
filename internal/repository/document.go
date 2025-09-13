package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	SaveDocument(ctx context.Context, doc *models.Document) (int, error)
	GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error)
	GetDocumentByID(ctx context.Context, id int) (*models.Document, error)
	DeleteDocument(ctx context.Context, id int) error
	GetAllDocuments(ctx context.Context, limit int) ([]*models.Document, error)
	Search(ctx context.Context, query string) ([]models.Document, error)
	GetPublicDocumentsByFilterPaginated(
		ctx context.Context,
		limit, offset int,
		sectionID *int,
		category string,
	) ([]*models.Document, int, error)
	UpdateDocumentSection(ctx context.Context, id int, sectionID *int) error
	GetPublicDocuments(
		ctx context.Context,
		sectionID *int,
		category string,
	) ([]*models.Document, error)
}

// SaveDocument — сохранить документ и вернуть его ID
func (r *DocumentRepository) SaveDocument(ctx context.Context, doc *models.Document) (int, error) {
	log := logger.WithCtx(ctx)

	const query = `
		INSERT INTO documents (
			user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at, allow_free_download
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id
	`

	var id int
	if err := r.db.QueryRow(ctx, query,
		doc.UserID,
		doc.Title,
		doc.Filename,
		doc.Filepath,
		doc.Description,
		doc.IsPublic,
		doc.Category,
		doc.SectionID,
		doc.UploadedAt,
		doc.AllowFreeDownload,
	).Scan(&id); err != nil {
		log.Error("document repo: save failed", zap.Error(err),
			zap.String("filename", doc.Filename), zap.Int("user_id", doc.UserID))
		return 0, err
	}

	log.Info("document repo: saved", zap.Int("id", id), zap.String("filename", doc.Filename))
	return id, nil
}

// GetPublicDocumentsPaginated — публичные документы (опц. фильтр по категории) с пагинацией + total
func (r *DocumentRepository) GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error) {
	log := logger.WithCtx(ctx)

	var (
		rows  pgx.Rows
		err   error
		docs  []*models.Document
		query string
		args  []any
		total int
	)

	if strings.TrimSpace(category) != "" {
		query = `
			SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at, allow_free_download
			FROM documents
			WHERE is_public = true AND category = $1
			ORDER BY uploaded_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []any{category, limit, offset}
		rows, err = r.db.Query(ctx, query, args...)
	} else {
		query = `
			SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at, allow_free_download
			FROM documents
			WHERE is_public = true
			ORDER BY uploaded_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []any{limit, offset}
		rows, err = r.db.Query(ctx, query, args...)
	}
	if err != nil {
		log.Error("document repo: get public paginated query failed", zap.Error(err),
			zap.String("category", category), zap.Int("limit", limit), zap.Int("offset", offset))
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
			&d.AllowFreeDownload,
		); err != nil {
			log.Error("document repo: scan public paginated failed", zap.Error(err))
			return nil, 0, err
		}
		docs = append(docs, &d)
	}
	if err := rows.Err(); err != nil {
		log.Error("document repo: rows error public paginated", zap.Error(err))
		return nil, 0, err
	}

	// total
	if strings.TrimSpace(category) != "" {
		if err := r.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM documents WHERE is_public = true AND category = $1`, category,
		).Scan(&total); err != nil {
			log.Error("document repo: count public paginated with category failed", zap.Error(err))
			return nil, 0, err
		}
	} else {
		if err := r.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM documents WHERE is_public = true`,
		).Scan(&total); err != nil {
			log.Error("document repo: count public paginated failed", zap.Error(err))
			return nil, 0, err
		}
	}

	log.Debug("document repo: public paginated done",
		zap.Int("returned", len(docs)), zap.Int("total", total),
		zap.String("category", category), zap.Int("limit", limit), zap.Int("offset", offset))
	return docs, total, nil
}

// GetDocumentByID — получить документ по ID
func (r *DocumentRepository) GetDocumentByID(ctx context.Context, id int) (*models.Document, error) {
	log := logger.WithCtx(ctx)

	const query = `
		SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at, allow_free_download
		FROM documents WHERE id = $1
	`

	var d models.Document
	if err := r.db.QueryRow(ctx, query, id).Scan(
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
		&d.AllowFreeDownload,
	); err != nil {
		log.Warn("document repo: get by id failed", zap.Int("doc_id", id), zap.Error(err))
		return nil, err
	}

	log.Debug("document repo: got by id", zap.Int("doc_id", id), zap.Bool("is_public", d.IsPublic))
	return &d, nil
}

// DeleteDocument — удалить документ
func (r *DocumentRepository) DeleteDocument(ctx context.Context, id int) error {
	log := logger.WithCtx(ctx)

	const query = `DELETE FROM documents WHERE id = $1`
	if _, err := r.db.Exec(ctx, query, id); err != nil {
		log.Error("document repo: delete failed", zap.Int("doc_id", id), zap.Error(err))
		return err
	}

	log.Info("document repo: deleted", zap.Int("doc_id", id))
	return nil
}

// GetAllDocuments — все документы (для админки), опционально ограничить количеством
func (r *DocumentRepository) GetAllDocuments(ctx context.Context, limit int) ([]*models.Document, error) {
	log := logger.WithCtx(ctx)

	query := `
		SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at, allow_free_download
		FROM documents
		ORDER BY uploaded_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		log.Error("document repo: get all query failed", zap.Error(err), zap.Int("limit", limit))
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
			&d.AllowFreeDownload,
		); err != nil {
			log.Error("document repo: scan get all failed", zap.Error(err))
			return nil, err
		}
		docs = append(docs, &d)
	}
	if err := rows.Err(); err != nil {
		log.Error("document repo: rows error get all", zap.Error(err))
		return nil, err
	}

	log.Debug("document repo: get all done", zap.Int("returned", len(docs)), zap.Int("limit", limit))
	return docs, nil
}

// Search — поиск по нескольким полям (без filepath)
func (r *DocumentRepository) Search(ctx context.Context, query string) ([]models.Document, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, user_id, title, filename, description, is_public, category, section_id, uploaded_at, allow_free_download
		FROM documents
		WHERE title ILIKE $1 OR filename ILIKE $1 OR description ILIKE $1 OR category ILIKE $1
	`
	pattern := "%" + query + "%"

	rows, err := r.db.Query(ctx, q, pattern)
	if err != nil {
		log.Error("document repo: search query failed", zap.Error(err), zap.String("query", query))
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.Title,
			&d.Filename,
			&d.Description,
			&d.IsPublic,
			&d.Category,
			&d.SectionID,
			&d.UploadedAt,
			&d.AllowFreeDownload,
		); err != nil {
			log.Error("document repo: scan search failed", zap.Error(err))
			return nil, err
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		log.Error("document repo: rows error search", zap.Error(err))
		return nil, err
	}

	log.Debug("document repo: search done", zap.String("query", query), zap.Int("returned", len(docs)))
	return docs, nil
}

// GetPublicDocumentsByFilterPaginated — публичные документы c фильтрами (section_id/category) + пагинация + total
func (r *DocumentRepository) GetPublicDocumentsByFilterPaginated(
	ctx context.Context,
	limit, offset int,
	sectionID *int,
	category string,
) ([]*models.Document, int, error) {

	log := logger.WithCtx(ctx)

	var (
		rows  pgx.Rows
		err   error
		docs  []*models.Document
		args  []any
		cond  []string
		total int
	)

	queryBase := `
		SELECT id, user_id, title, filename, filepath, description, is_public, category, section_id, uploaded_at, allow_free_download
		FROM documents
		WHERE is_public = true
	`

	if sectionID != nil {
		cond = append(cond, "section_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, *sectionID)
	}
	if strings.TrimSpace(category) != "" {
		cond = append(cond, "category = $"+strconv.Itoa(len(args)+1))
		args = append(args, category)
	}
	if len(cond) > 0 {
		queryBase += " AND " + strings.Join(cond, " AND ")
	}

	query := queryBase +
		" ORDER BY uploaded_at DESC" +
		" LIMIT $" + strconv.Itoa(len(args)+1) +
		" OFFSET $" + strconv.Itoa(len(args)+2)

	args = append(args, limit, offset)

	rows, err = r.db.Query(ctx, query, args...)
	if err != nil {
		log.Error("document repo: get public filtered paginated query failed", zap.Error(err),
			zap.Any("section_id", sectionID), zap.String("category", category),
			zap.Int("limit", limit), zap.Int("offset", offset))
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
			&d.AllowFreeDownload,
		); err != nil {
			log.Error("document repo: scan public filtered paginated failed", zap.Error(err))
			return nil, 0, err
		}
		docs = append(docs, &d)
	}
	if err := rows.Err(); err != nil {
		log.Error("document repo: rows error public filtered paginated", zap.Error(err))
		return nil, 0, err
	}

	// total
	countQuery := `SELECT COUNT(*) FROM documents WHERE is_public = true`
	var argsCnt []any
	if len(cond) > 0 {
		countQuery += " AND " + strings.Join(cond, " AND ")
		argsCnt = append(argsCnt, args[:len(args)-2]...) // отбросить limit/offset
	}
	if err := r.db.QueryRow(ctx, countQuery, argsCnt...).Scan(&total); err != nil {
		log.Error("document repo: count public filtered paginated failed", zap.Error(err))
		return nil, 0, err
	}

	log.Debug("document repo: public filtered paginated done",
		zap.Int("returned", len(docs)), zap.Int("total", total),
		zap.Any("section_id", sectionID), zap.String("category", category),
		zap.Int("limit", limit), zap.Int("offset", offset))
	return docs, total, nil
}

// UpdateDocumentSection — сменить раздел у документа
func (r *DocumentRepository) UpdateDocumentSection(ctx context.Context, id int, sectionID *int) error {
	log := logger.WithCtx(ctx)

	if _, err := r.db.Exec(ctx,
		`UPDATE documents SET section_id=$1, uploaded_at=uploaded_at WHERE id=$2`, sectionID, id,
	); err != nil {
		log.Error("document repo: update section failed", zap.Error(err), zap.Int("doc_id", id), zap.Any("section_id", sectionID))
		return err
	}

	log.Info("document repo: section updated", zap.Int("doc_id", id), zap.Any("section_id", sectionID))
	return nil
}

// GetPublicDocuments — публичные документы по фильтрам (без пагинации)
func (r *DocumentRepository) GetPublicDocuments(
	ctx context.Context,
	sectionID *int,
	category string,
) ([]*models.Document, error) {
	log := logger.WithCtx(ctx)

	query := `
		SELECT id, user_id, COALESCE(title, '') AS title, filename, filepath, description, is_public,
		       category, section_id, uploaded_at, allow_free_download
		FROM documents
		WHERE is_public = true
	`
	args := []any{}
	idx := 1

	if sectionID != nil {
		query += fmt.Sprintf(" AND section_id = $%d", idx)
		args = append(args, *sectionID)
		idx++
	}
	if strings.TrimSpace(category) != "" {
		query += fmt.Sprintf(" AND category = $%d", idx)
		args = append(args, category)
		idx++
	}

	query += " ORDER BY uploaded_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		log.Error("document repo: get public query failed", zap.Error(err),
			zap.Any("section_id", sectionID), zap.String("category", category))
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
			&d.AllowFreeDownload,
		); err != nil {
			log.Error("document repo: scan get public failed", zap.Error(err))
			return nil, err
		}
		docs = append(docs, &d)
	}
	if err := rows.Err(); err != nil {
		log.Error("document repo: rows error get public", zap.Error(err))
		return nil, err
	}

	log.Debug("document repo: get public done",
		zap.Int("returned", len(docs)),
		zap.Any("section_id", sectionID),
		zap.String("category", category),
	)
	return docs, nil
}
