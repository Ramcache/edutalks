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
