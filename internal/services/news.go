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
		logger.Log.Error("Сервис: ошибка создания новости", zap.Error(err))
		return 0, err
	}

	logger.Log.Info("Сервис: новость создана", zap.Int("news_id", id))
	return id, nil
}

func (s *NewsService) ListPaginated(ctx context.Context, limit, offset int) ([]*models.News, int, error) {
	logger.Log.Debug("Сервис: список новостей (пагинация)",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	items, total, err := s.repo.ListPaginated(ctx, limit, offset)
	if err != nil {
		logger.Log.Error("Сервис: ошибка получения списка новостей", zap.Error(err))
		return nil, 0, err
	}

	logger.Log.Debug("Сервис: список новостей получен",
		zap.Int("count", len(items)),
		zap.Int("total", total),
	)
	return items, total, nil
}

func (s *NewsService) GetByID(ctx context.Context, id int) (*models.News, error) {
	logger.Log.Info("Сервис: получение новости по ID", zap.Int("news_id", id))

	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Log.Warn("Сервис: новость не найдена или ошибка выборки",
			zap.Int("news_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Log.Info("Сервис: новость получена", zap.Int("news_id", id))
	return n, nil
}

func (s *NewsService) Update(ctx context.Context, id int, title, content, imageURL, color, sticker string) error {
	logger.Log.Info("Сервис: обновление новости", zap.Int("news_id", id))

	if err := s.repo.Update(ctx, id, title, content, imageURL, color, sticker); err != nil {
		logger.Log.Error("Сервис: ошибка обновления новости",
			zap.Int("news_id", id),
			zap.Error(err),
		)
		return err
	}

	logger.Log.Info("Сервис: новость обновлена", zap.Int("news_id", id))
	return nil
}

func (s *NewsService) Delete(ctx context.Context, id int) error {
	logger.Log.Info("Сервис: удаление новости", zap.Int("news_id", id))

	if err := s.repo.Delete(ctx, id); err != nil {
		logger.Log.Error("Сервис: ошибка удаления новости",
			zap.Int("news_id", id),
			zap.Error(err),
		)
		return err
	}

	logger.Log.Info("Сервис: новость удалена", zap.Int("news_id", id))
	return nil
}

func (s *NewsService) Search(ctx context.Context, query string) ([]models.News, error) {
	logger.Log.Debug("Сервис: поиск новостей", zap.Int("query_len", len(query)))

	items, err := s.repo.Search(ctx, query)
	if err != nil {
		logger.Log.Error("Сервис: ошибка поиска новостей", zap.Error(err))
		return nil, err
	}

	logger.Log.Debug("Сервис: поиск новостей завершён", zap.Int("count", len(items)))
	return items, nil
}
