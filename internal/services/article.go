package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"

	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"
)

type ArticleService interface {
	Create(ctx context.Context, authorID *int64, req models.CreateArticleRequest) (*models.Article, error)
	PreviewHTML(rawHTML string) string
	GetAll(ctx context.Context, limit, offset int, tag string, onlyPublished bool) ([]*models.Article, error)
	GetByID(ctx context.Context, id int64) (*models.Article, error)
	Update(ctx context.Context, id int64, req models.CreateArticleRequest) (*models.Article, error)
	Delete(ctx context.Context, id int64) error
	SetPublish(ctx context.Context, id int64, publish bool) (*models.Article, error)
}

type articleService struct {
	repo   repository.ArticleRepo
	policy *bluemonday.Policy
}

func NewArticleService(repo repository.ArticleRepo) ArticleService {
	p := bluemonday.UGCPolicy()
	p.AllowElements("img")
	p.AllowAttrs("src", "alt").OnElements("img")
	return &articleService{repo: repo, policy: p}
}

func (s *articleService) PreviewHTML(rawHTML string) string {
	// безопасно логируем только длины
	log := logger.WithCtx(context.Background())
	clean := s.policy.Sanitize(rawHTML)
	log.Debug("Предпросмотр HTML (sanitize)",
		zap.Int("raw_len", len(rawHTML)),
		zap.Int("clean_len", len(clean)),
	)
	return clean
}

func (s *articleService) Create(ctx context.Context, authorID *int64, req models.CreateArticleRequest) (*models.Article, error) {
	log := logger.WithCtx(ctx)
	log.Info("Создание статьи",
		zap.Any("author_id", authorID),
		zap.String("title", strings.TrimSpace(req.Title)),
		zap.Bool("publish", req.Publish),
		zap.Int("tags_count", len(req.Tags)),
	)

	title := strings.TrimSpace(req.Title)
	if l := utf8.RuneCountInString(title); l < 3 || l > 255 {
		err := errors.New("длина заголовка должна быть от 3 до 255 символов")
		log.Warn("Валидация не пройдена: заголовок", zap.Int("runes", l), zap.Error(err))
		return nil, err
	}
	if body := strings.TrimSpace(req.BodyHTML); body == "" || utf8.RuneCountInString(body) < 30 {
		err := errors.New("контент слишком короткий")
		log.Warn("Валидация не пройдена: контент", zap.Int("runes", utf8.RuneCountInString(req.BodyHTML)), zap.Error(err))
		return nil, err
	}
	if len(req.Tags) > 5 {
		err := errors.New("максимум 5 тегов")
		log.Warn("Валидация не пройдена: слишком много тегов", zap.Int("tags_count", len(req.Tags)), zap.Error(err))
		return nil, err
	}

	safe := s.policy.Sanitize(req.BodyHTML)

	a := &models.Article{
		AuthorID:    authorID,
		Title:       title,
		Summary:     strPtr(req.Summary),
		BodyHTML:    safe,
		Tags:        normalizeTags(req.Tags),
		IsPublished: req.Publish,
	}

	created, err := s.repo.Create(ctx, a)
	if err != nil {
		log.Error("Ошибка создания статьи (repo)", zap.Error(err))
		return nil, err
	}

	log.Info("Статья создана",
		zap.Int64("id", created.ID),
		zap.Bool("published", created.IsPublished),
		zap.Int("tags_count", len(created.Tags)),
	)
	return created, nil
}

func (s *articleService) GetAll(ctx context.Context, limit, offset int, tag string, onlyPublished bool) ([]*models.Article, error) {
	log := logger.WithCtx(ctx)
	log.Debug("Получение списка статей",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("tag", tag),
		zap.Bool("only_published", onlyPublished),
	)

	list, err := s.repo.GetAll(ctx, limit, offset, tag, onlyPublished)
	if err != nil {
		log.Error("Ошибка получения списка статей (repo)", zap.Error(err))
		return nil, err
	}

	log.Debug("Список статей получен", zap.Int("count", len(list)))
	return list, nil
}

func (s *articleService) GetByID(ctx context.Context, id int64) (*models.Article, error) {
	log := logger.WithCtx(ctx)
	log.Debug("Получение статьи по ID", zap.Int64("id", id))

	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Warn("Статья не найдена (repo)", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}

	log.Debug("Статья получена", zap.Int64("id", id))
	return a, nil
}

func (s *articleService) Update(ctx context.Context, id int64, req models.CreateArticleRequest) (*models.Article, error) {
	log := logger.WithCtx(ctx)
	log.Info("Обновление статьи", zap.Int64("id", id), zap.String("title", strings.TrimSpace(req.Title)))

	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Warn("Статья для обновления не найдена (repo)", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}

	a.Title = strings.TrimSpace(req.Title)
	a.Summary = strPtr(req.Summary)
	a.BodyHTML = s.policy.Sanitize(req.BodyHTML)
	a.Tags = normalizeTags(req.Tags)
	a.IsPublished = req.Publish

	if err := s.repo.Update(ctx, a); err != nil {
		log.Error("Ошибка обновления статьи (repo)", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}

	log.Info("Статья обновлена", zap.Int64("id", id), zap.Bool("published", a.IsPublished))
	return a, nil
}

func (s *articleService) Delete(ctx context.Context, id int64) error {
	log := logger.WithCtx(ctx)
	log.Info("Удаление статьи", zap.Int64("id", id))

	if err := s.repo.Delete(ctx, id); err != nil {
		log.Error("Ошибка удаления статьи (repo)", zap.Int64("id", id), zap.Error(err))
		return err
	}

	log.Info("Статья удалена", zap.Int64("id", id))
	return nil
}

func (s *articleService) SetPublish(ctx context.Context, id int64, publish bool) (*models.Article, error) {
	log := logger.WithCtx(ctx)
	log.Info("Изменение статуса публикации", zap.Int64("id", id), zap.Bool("publish", publish))

	exists, err := s.repo.Exists(ctx, id)
	if err != nil {
		log.Error("Ошибка проверки существования статьи (repo)", zap.Int64("id", id), zap.Error(err))
		return nil, fmt.Errorf("ошибка проверки существования статьи: %w", err)
	}
	if !exists {
		log.Warn("Статья не найдена при изменении публикации", zap.Int64("id", id))
		return nil, fmt.Errorf("не найдено")
	}

	if err := s.repo.UpdatePublish(ctx, id, publish); err != nil {
		log.Error("Ошибка обновления статуса публикации (repo)", zap.Int64("id", id), zap.Bool("publish", publish), zap.Error(err))
		return nil, fmt.Errorf("ошибка обновления статуса публикации: %w", err)
	}

	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("Ошибка получения статьи после обновления публикации (repo)", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}

	log.Info("Статус публикации изменён", zap.Int64("id", id), zap.Bool("published", a.IsPublished))
	return a, nil
}

func strPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func normalizeTags(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
