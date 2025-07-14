package services

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"fmt"

	"go.uber.org/zap"
)

type NewsService struct {
	repo         *repository.NewsRepository
	userRepo     *repository.UserRepository
	emailService *EmailService
}

func NewNewsService(
	repo *repository.NewsRepository,
	userRepo *repository.UserRepository,
	emailService *EmailService,
) *NewsService {
	return &NewsService{
		repo:         repo,
		userRepo:     userRepo,
		emailService: emailService,
	}
}

func (s *NewsService) Create(ctx context.Context, news *models.News) error {
	logger.Log.Info("Сервис: создание новости", zap.String("title", news.Title))
	err := s.repo.Create(ctx, news)
	if err != nil {
		logger.Log.Error("Ошибка создания новости (service)", zap.Error(err))
	}
	subscribers, _ := s.userRepo.GetSubscribedEmails(ctx)
	subject := "Новая новость: " + news.Title
	body := fmt.Sprintf("Прочитайте новость: %s\n\n%s", news.Title, news.Content)

	if err := s.emailService.Send(subscribers, subject, body); err != nil {
		logger.Log.Warn("Ошибка при отправке писем", zap.Error(err))
	}

	return err
}

func (s *NewsService) List(ctx context.Context) ([]*models.News, error) {
	logger.Log.Info("Сервис: получение списка новостей")
	news, err := s.repo.List(ctx)
	if err != nil {
		logger.Log.Error("Ошибка получения списка новостей (service)", zap.Error(err))
	}
	return news, err
}

func (s *NewsService) GetByID(ctx context.Context, id int) (*models.News, error) {
	logger.Log.Info("Сервис: получение новости по ID", zap.Int("news_id", id))
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Log.Error("Ошибка получения новости по ID (service)", zap.Error(err), zap.Int("news_id", id))
	}
	return n, err
}

func (s *NewsService) Update(ctx context.Context, id int, title, content, imageURL string) error {
	logger.Log.Info("Сервис: обновление новости", zap.Int("news_id", id))
	err := s.repo.Update(ctx, id, title, content, imageURL)
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
