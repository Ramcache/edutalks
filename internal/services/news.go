package services

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	helpers "edutalks/internal/utils/helpres"

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

func (s *NewsService) Create(ctx context.Context, news *models.News) error {
	logger.Log.Info("Сервис: создание новости", zap.String("title", news.Title))

	err := s.repo.Create(ctx, news)
	if err != nil {
		logger.Log.Error("Ошибка создания новости (service)", zap.Error(err))
		return err
	}

	// Получаем подписчиков для рассылки
	subscribers, err := s.userRepo.GetSubscribedEmails(ctx)
	if err != nil {
		logger.Log.Warn("Ошибка получения подписчиков", zap.Error(err))
		return nil
	}

	subject := "Новая новость: " + news.Title
	url := s.siteURL + "/news" // Или "/news/" + strconv.Itoa(news.ID)
	htmlBody := helpers.BuildNewsHTML(news.Title, news.Content, url)

	for _, email := range subscribers {
		EmailQueue <- EmailJob{
			To:      []string{email},
			Subject: subject,
			Body:    htmlBody,
			IsHTML:  true,
		}
	}

	return nil
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
