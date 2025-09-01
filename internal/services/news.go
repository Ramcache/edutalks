package services

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"go.uber.org/zap"
)

type NewsService struct {
	repo         *repository.NewsRepository
	userRepo     *repository.UserRepository
	emailService *EmailService
	siteURL      string
}

func NewNewsService(
	repo *repository.NewsRepository,
	userRepo *repository.UserRepository,
	emailService *EmailService,
	cfg *config.Config,
) *NewsService {
	return &NewsService{
		repo:         repo,
		userRepo:     userRepo,
		emailService: emailService,
		siteURL:      cfg.SiteURL,
	}
}

func (s *NewsService) Create(ctx context.Context, news *models.News) (int, error) {
	logger.Log.Info("Сервис: создание новости", zap.String("title", news.Title))

	id, err := s.repo.Create(ctx, news)
	if err != nil {
		logger.Log.Error("Ошибка создания новости (service)", zap.Error(err))
		return 0, err
	}
	return id, nil
}

func (s *NewsService) ListPaginated(ctx context.Context, limit, offset int) ([]*models.News, int, error) {
	return s.repo.ListPaginated(ctx, limit, offset)
}

func (s *NewsService) GetByID(ctx context.Context, id int) (*models.News, error) {
	logger.Log.Info("Сервис: получение новости по ID", zap.Int("news_id", id))
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Log.Error("Ошибка получения новости по ID (service)", zap.Error(err), zap.Int("news_id", id))
	}
	return n, err
}

func (s *NewsService) Update(ctx context.Context, id int, title, content, imageURL, color, sticker string) error {
	logger.Log.Info("Сервис: обновление новости", zap.Int("news_id", id))
	err := s.repo.Update(ctx, id, title, content, imageURL, color, sticker)
	if err != nil {
		logger.Log.Error("Ошибка обновления новости (service)", zap.Error(err), zap.Int("news_id", id))
	}
	return err
}

func (s *NewsService) Delete(ctx context.Context, id int) error {
	logger.Log.Info("Сервис: удаление новости", zap.Int("news_id", id))
	err := s.repo.Delete(ctx, id)
	if err != nil {
		logger.Log.Error("Ошибка удаления новости (service)", zap.Error(err), zap.Int("news_id", id))
	}
	return err
}

func (s *NewsService) Search(ctx context.Context, query string) ([]models.News, error) {
	return s.repo.Search(ctx, query)
}
