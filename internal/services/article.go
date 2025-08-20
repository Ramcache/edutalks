package services

import (
	"context"
	"edutalks/internal/logger"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"

	"edutalks/internal/models"
	"edutalks/internal/repository"
)

type ArticleService interface {
	Create(ctx context.Context, authorID *int64, req models.CreateArticleRequest) (*models.Article, error)
	PreviewHTML(rawHTML string) string
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
	return s.policy.Sanitize(rawHTML)
}

func (s *articleService) Create(ctx context.Context, authorID *int64, req models.CreateArticleRequest) (*models.Article, error) {
	if l := utf8.RuneCountInString(strings.TrimSpace(req.Title)); l < 3 || l > 255 {
		return nil, errors.New("title length must be 3..255")
	}
	if strings.TrimSpace(req.BodyHTML) == "" || utf8.RuneCountInString(req.BodyHTML) < 30 {
		return nil, errors.New("content is too short")
	}
	if len(req.Tags) > 5 {
		return nil, errors.New("max 5 tags")
	}
	safe := s.policy.Sanitize(req.BodyHTML)

	a := &models.Article{
		AuthorID:    authorID,
		Title:       req.Title,
		Summary:     strPtr(req.Summary),
		BodyHTML:    safe,
		Tags:        normalizeTags(req.Tags),
		IsPublished: req.Publish,
	}
	created, err := s.repo.Create(ctx, a)
	if err != nil {
		logger.Log.Error("create article failed", zap.Error(err))
		return nil, err
	}
	return created, nil
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
