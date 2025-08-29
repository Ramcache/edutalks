package services

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"

	"go.uber.org/zap"
)

type DocumentService struct {
	repo *repository.DocumentRepository
}

func NewDocumentService(repo *repository.DocumentRepository) *DocumentService {
	return &DocumentService{repo: repo}
}

type DocumentServiceInterface interface {
	Upload(ctx context.Context, doc *models.Document) (int, error) // было error
	GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error)
	GetDocumentByID(ctx context.Context, id int) (*models.Document, error)
	Delete(ctx context.Context, id int) error
	GetAllDocuments(ctx context.Context) ([]*models.Document, error)
	Search(ctx context.Context, query string) ([]models.Document, error) // было []models.News
	GetPublicDocumentsByFilterPaginated(ctx context.Context, limit, offset int, sectionID *int, category string) ([]*models.Document, int, error)
}

func (s *DocumentService) Upload(ctx context.Context, doc *models.Document) (int, error) {
	return s.repo.SaveDocument(ctx, doc)
}

func (s *DocumentService) GetPublicDocumentsPaginated(ctx context.Context, limit, offset int, category string) ([]*models.Document, int, error) {
	return s.repo.GetPublicDocumentsPaginated(ctx, limit, offset, category)
}

func (s *DocumentService) GetDocumentByID(ctx context.Context, id int) (*models.Document, error) {
	logger.Log.Info("Сервис: получение документа по ID", zap.Int("doc_id", id))
	doc, err := s.repo.GetDocumentByID(ctx, id)
	if err != nil {
		logger.Log.Error("Ошибка получения документа по ID (service)", zap.Int("doc_id", id), zap.Error(err))
	}
	return doc, err
}

func (s *DocumentService) Delete(ctx context.Context, id int) error {
	logger.Log.Info("Сервис: удаление документа", zap.Int("doc_id", id))
	err := s.repo.DeleteDocument(ctx, id)
	if err != nil {
		logger.Log.Error("Ошибка удаления документа (service)", zap.Int("doc_id", id), zap.Error(err))
	}
	return err
}

func (s *DocumentService) GetAllDocuments(ctx context.Context) ([]*models.Document, error) {
	logger.Log.Info("Сервис: получение всех документов")
	return s.repo.GetAllDocuments(ctx)
}

func (s *DocumentService) Search(ctx context.Context, query string) ([]models.Document, error) {
	return s.repo.Search(ctx, query)
}

func (s *DocumentService) GetPublicDocumentsByFilterPaginated(
	ctx context.Context, limit, offset int, sectionID *int, category string,
) ([]*models.Document, int, error) {
	return s.repo.GetPublicDocumentsByFilterPaginated(ctx, limit, offset, sectionID, category)
}
