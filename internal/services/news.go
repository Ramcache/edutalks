package services

import (
	"context"
	"edutalks/internal/models"
	"edutalks/internal/repository"
)

type NewsService struct {
	repo *repository.NewsRepository
}

func NewNewsService(repo *repository.NewsRepository) *NewsService {
	return &NewsService{repo: repo}
}

func (s *NewsService) Create(ctx context.Context, news *models.News) error {
	return s.repo.Create(ctx, news)
}

func (s *NewsService) List(ctx context.Context) ([]*models.News, error) {
	return s.repo.List(ctx)
}

func (s *NewsService) GetByID(ctx context.Context, id int) (*models.News, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *NewsService) Update(ctx context.Context, id int, title, content string) error {
	return s.repo.Update(ctx, id, title, content)
}

func (s *NewsService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
