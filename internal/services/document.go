package services

import (
	"context"

	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"

	"go.uber.org/zap"
)

type DocumentService struct {
	repo repository.DocumentRepo
}

func NewDocumentService(repo repository.DocumentRepo) *DocumentService {
	return &DocumentService{repo: repo}
}

type DocumentServiceInterface interface {
	Upload(ctx context.Context, doc *models.Document) (int, error)
	GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error)
	GetDocumentByID(ctx context.Context, id int) (*models.Document, error)
	Delete(ctx context.Context, id int) error
	GetAllDocuments(ctx context.Context, limit int) ([]*models.Document, error)
	Search(ctx context.Context, query string) ([]models.Document, error)
	GetPublicDocumentsByFilterPaginated(ctx context.Context, limit, offset int, sectionID *int, category string) ([]*models.Document, int, error)
	GetPublicDocuments(ctx context.Context, sectionID *int, category string) ([]*models.Document, error)
}

func (s *DocumentService) Upload(ctx context.Context, doc *models.Document) (int, error) {
	logger.Log.Info("Сервис: загрузка документа",
		zap.Int("user_id", doc.UserID),
		zap.String("title", doc.Title),
		zap.String("filename", doc.Filename),
		zap.Bool("is_public", doc.IsPublic),
		zap.String("category", doc.Category),
		zap.Any("section_id", doc.SectionID),
		zap.Bool("allow_free_download", doc.AllowFreeDownload),
	)

	id, err := s.repo.SaveDocument(ctx, doc)
	if err != nil {
		logger.Log.Error("Сервис: ошибка сохранения документа",
			zap.Error(err),
			zap.String("filename", doc.Filename),
		)
		return 0, err
	}

	logger.Log.Info("Сервис: документ сохранён", zap.Int("doc_id", id))
	return id, nil
}

func (s *DocumentService) GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error) {
	logger.Log.Info("Сервис: получение публичных документов (пагинация)",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("category", category),
	)

	docs, total, err := s.repo.GetPublicDocumentsPaginated(ctx, limit, offset, category)
	if err != nil {
		logger.Log.Error("Сервис: ошибка получения публичных документов", zap.Error(err))
		return nil, 0, err
	}

	logger.Log.Info("Сервис: публичные документы получены",
		zap.Int("count", len(docs)),
		zap.Int("total", total),
	)
	return docs, total, nil
}

func (s *DocumentService) GetDocumentByID(ctx context.Context, id int) (*models.Document, error) {
	logger.Log.Info("Сервис: получение документа по ID", zap.Int("doc_id", id))

	doc, err := s.repo.GetDocumentByID(ctx, id)
	if err != nil {
		logger.Log.Error("Сервис: ошибка получения документа по ID",
			zap.Int("doc_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Log.Info("Сервис: документ получен", zap.Int("doc_id", id))
	return doc, nil
}

func (s *DocumentService) Delete(ctx context.Context, id int) error {
	logger.Log.Info("Сервис: удаление документа", zap.Int("doc_id", id))

	if err := s.repo.DeleteDocument(ctx, id); err != nil {
		logger.Log.Error("Сервис: ошибка удаления документа",
			zap.Int("doc_id", id),
			zap.Error(err),
		)
		return err
	}

	logger.Log.Info("Сервис: документ удалён", zap.Int("doc_id", id))
	return nil
}

func (s *DocumentService) GetAllDocuments(ctx context.Context, limit int) ([]*models.Document, error) {
	logger.Log.Info("Сервис: получение всех документов", zap.Int("limit", limit))

	docs, err := s.repo.GetAllDocuments(ctx, limit)
	if err != nil {
		logger.Log.Error("Сервис: ошибка получения всех документов", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Сервис: документы получены", zap.Int("count", len(docs)))
	return docs, nil
}

func (s *DocumentService) Search(ctx context.Context, query string) ([]models.Document, error) {
	logger.Log.Info("Сервис: поиск документов", zap.String("query", query))

	res, err := s.repo.Search(ctx, query)
	if err != nil {
		logger.Log.Error("Сервис: ошибка поиска документов", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Сервис: поиск завершён", zap.Int("count", len(res)))
	return res, nil
}

func (s *DocumentService) GetPublicDocumentsByFilterPaginated(
	ctx context.Context,
	limit, offset int,
	sectionID *int,
	category string,
) ([]*models.Document, int, error) {
	logger.Log.Info("Сервис: публичные документы по фильтру (пагинация)",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Any("section_id", sectionID),
		zap.String("category", category),
	)

	docs, total, err := s.repo.GetPublicDocumentsByFilterPaginated(ctx, limit, offset, sectionID, category)
	if err != nil {
		logger.Log.Error("Сервис: ошибка получения документов по фильтру", zap.Error(err))
		return nil, 0, err
	}

	logger.Log.Info("Сервис: документы по фильтру получены",
		zap.Int("count", len(docs)),
		zap.Int("total", total),
	)
	return docs, total, nil
}

func (s *DocumentService) GetPublicDocuments(
	ctx context.Context,
	sectionID *int,
	category string,
) ([]*models.Document, error) {
	logger.Log.Info("Сервис: публичные документы (без пагинации)",
		zap.Any("section_id", sectionID),
		zap.String("category", category),
	)

	docs, err := s.repo.GetPublicDocuments(ctx, sectionID, category)
	if err != nil {
		logger.Log.Error("Сервис: ошибка получения публичных документов", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Сервис: публичные документы получены", zap.Int("count", len(docs)))
	return docs, nil
}
