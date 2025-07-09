package services

import (
	"context"
	"edutalks/internal/models"
	"edutalks/internal/repository"
)

type DocumentService struct {
	repo *repository.DocumentRepository
}

func NewDocumentService(repo *repository.DocumentRepository) *DocumentService {
	return &DocumentService{repo: repo}
}

func (s *DocumentService) Upload(ctx context.Context, doc *models.Document) error {
	return s.repo.SaveDocument(ctx, doc)
}

func (s *DocumentService) GetPublicDocuments(ctx context.Context) ([]*models.Document, error) {
	return s.repo.GetPublicDocuments(ctx)
}

func (s *DocumentService) GetDocumentByID(ctx context.Context, id int) (*models.Document, error) {
	return s.repo.GetDocumentByID(ctx, id)
}

func (s *DocumentService) Delete(ctx context.Context, id int) error {
	return s.repo.DeleteDocument(ctx, id)
}
