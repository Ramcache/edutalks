// internal/services/taxonomy_service.go

package services

import (
	"context"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"fmt"
	"regexp"
	"strings"
)

type TaxonomyService struct{ repo *repository.TaxonomyRepo }

func NewTaxonomyService(r *repository.TaxonomyRepo) *TaxonomyService {
	return &TaxonomyService{repo: r}
}

func (s *TaxonomyService) CreateTab(ctx context.Context, t *models.Tab) (int, error) {
	// если slug пуст — генерируем из title
	if strings.TrimSpace(t.Slug) == "" {
		base := slugify(t.Title)
		unique, err := s.ensureUniqueTabSlug(ctx, base)
		if err != nil {
			return 0, err
		}
		t.Slug = unique
	}
	return s.repo.CreateTab(ctx, t)
}

func (s *TaxonomyService) UpdateTab(ctx context.Context, t *models.Tab) error {
	// тут намеренно НЕ трогаем slug — можно менять вручную с фронта/админки
	return s.repo.UpdateTab(ctx, t)
}

func (s *TaxonomyService) DeleteTab(ctx context.Context, id int) error {
	return s.repo.DeleteTab(ctx, id)
}

func (s *TaxonomyService) CreateSection(ctx context.Context, sec *models.Section) (int, error) {
	// автогенерация, если пуст
	if strings.TrimSpace(sec.Slug) == "" {
		base := slugify(sec.Title)
		unique, err := s.ensureUniqueSectionSlug(ctx, sec.TabID, base)
		if err != nil {
			return 0, err
		}
		sec.Slug = unique
	}
	return s.repo.CreateSection(ctx, sec)
}

func (s *TaxonomyService) UpdateSection(ctx context.Context, sec *models.Section) error {
	// slug тут тоже не трогаем — можно менять руками
	return s.repo.UpdateSection(ctx, sec)
}

func (s *TaxonomyService) DeleteSection(ctx context.Context, id int) error {
	return s.repo.DeleteSection(ctx, id)
}

func (s *TaxonomyService) PublicTree(ctx context.Context) ([]models.TabTree, error) {
	return s.repo.ListTabTree(ctx)
}

func (s *TaxonomyService) PublicTreeFiltered(ctx context.Context, tabID *int, tabSlug *string) ([]models.TabTree, error) {
	return s.repo.ListTabTreeFilter(ctx, tabID, tabSlug)
}

// ----------------- helpers -----------------

var nonWord = regexp.MustCompile(`[^\p{L}\p{N}]+`) // всё, что не буквы/цифры, в дефисы

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonWord.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	// защитимся от пустых после очистки
	if s == "" {
		s = "item"
	}
	return s
}

func (s *TaxonomyService) ensureUniqueTabSlug(ctx context.Context, base string) (string, error) {
	slug := base
	i := 1
	for {
		exists, err := s.repo.TabSlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		i++
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func (s *TaxonomyService) ensureUniqueSectionSlug(ctx context.Context, tabID int, base string) (string, error) {
	slug := base
	i := 1
	for {
		exists, err := s.repo.SectionSlugExists(ctx, tabID, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		i++
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}
