// internal/services/taxonomy_service.go
package services

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

type TaxonomyService struct{ repo *repository.TaxonomyRepo }

func NewTaxonomyService(r *repository.TaxonomyRepo) *TaxonomyService {
	return &TaxonomyService{repo: r}
}

// CreateTab — создаёт вкладку. Если slug пуст — генерируем и гарантируем уникальность.
func (s *TaxonomyService) CreateTab(ctx context.Context, t *models.Tab) (int, error) {
	title := strings.TrimSpace(t.Title)
	if title == "" {
		logger.Log.Warn("Пустой title при создании вкладки")
		return 0, fmt.Errorf("title is required")
	}

	// если slug пуст — генерируем из title
	if strings.TrimSpace(t.Slug) == "" {
		base := slugify(title)
		unique, err := s.ensureUniqueTabSlug(ctx, base)
		if err != nil {
			logger.Log.Error("Не удалось подобрать уникальный slug для вкладки", zap.String("base", base), zap.Error(err))
			return 0, err
		}
		t.Slug = unique
	}

	logger.Log.Info("Создание вкладки", zap.String("title", t.Title), zap.String("slug", t.Slug))
	id, err := s.repo.CreateTab(ctx, t)
	if err != nil {
		logger.Log.Error("Ошибка создания вкладки", zap.String("slug", t.Slug), zap.Error(err))
		return 0, err
	}
	return id, nil
}

// UpdateTab — обновляет вкладку (slug оставляем как прислал фронт).
func (s *TaxonomyService) UpdateTab(ctx context.Context, t *models.Tab) error {
	logger.Log.Info("Обновление вкладки", zap.Int("id", t.ID))
	if err := s.repo.UpdateTab(ctx, t); err != nil {
		logger.Log.Error("Ошибка обновления вкладки", zap.Int("id", t.ID), zap.Error(err))
		return err
	}
	return nil
}

// DeleteTab — удаляет вкладку.
func (s *TaxonomyService) DeleteTab(ctx context.Context, id int) error {
	logger.Log.Info("Удаление вкладки", zap.Int("id", id))
	if err := s.repo.DeleteTab(ctx, id); err != nil {
		logger.Log.Error("Ошибка удаления вкладки", zap.Int("id", id), zap.Error(err))
		return err
	}
	return nil
}

// CreateSection — создаёт раздел. Если slug пуст — генерируем и гарантируем уникальность в пределах вкладки.
func (s *TaxonomyService) CreateSection(ctx context.Context, sec *models.Section) (int, error) {
	title := strings.TrimSpace(sec.Title)
	if title == "" {
		logger.Log.Warn("Пустой title при создании раздела", zap.Int("tab_id", sec.TabID))
		return 0, fmt.Errorf("title is required")
	}

	if strings.TrimSpace(sec.Slug) == "" {
		base := slugify(title)
		unique, err := s.ensureUniqueSectionSlug(ctx, sec.TabID, base)
		if err != nil {
			logger.Log.Error("Не удалось подобрать уникальный slug для раздела", zap.Int("tab_id", sec.TabID), zap.String("base", base), zap.Error(err))
			return 0, err
		}
		sec.Slug = unique
	}

	logger.Log.Info("Создание раздела", zap.String("title", sec.Title), zap.String("slug", sec.Slug), zap.Int("tab_id", sec.TabID))
	id, err := s.repo.CreateSection(ctx, sec)
	if err != nil {
		logger.Log.Error("Ошибка создания раздела", zap.Int("tab_id", sec.TabID), zap.String("slug", sec.Slug), zap.Error(err))
		return 0, err
	}
	return id, nil
}

// UpdateSection — обновляет раздел (slug не трогаем).
func (s *TaxonomyService) UpdateSection(ctx context.Context, sec *models.Section) error {
	logger.Log.Info("Обновление раздела", zap.Int("id", sec.ID), zap.Int("tab_id", sec.TabID))
	if err := s.repo.UpdateSection(ctx, sec); err != nil {
		logger.Log.Error("Ошибка обновления раздела", zap.Int("id", sec.ID), zap.Error(err))
		return err
	}
	return nil
}

// DeleteSection — удаляет раздел.
func (s *TaxonomyService) DeleteSection(ctx context.Context, id int) error {
	logger.Log.Info("Удаление раздела", zap.Int("id", id))
	if err := s.repo.DeleteSection(ctx, id); err != nil {
		logger.Log.Error("Ошибка удаления раздела", zap.Int("id", id), zap.Error(err))
		return err
	}
	return nil
}

// PublicTree — полное дерево вкладок и разделов.
func (s *TaxonomyService) PublicTree(ctx context.Context) ([]models.TabTree, error) {
	items, err := s.repo.ListTabTree(ctx)
	if err != nil {
		logger.Log.Error("Ошибка получения дерева таксономии", zap.Error(err))
		return nil, err
	}
	return items, nil
}

// PublicTreeFiltered — дерево по конкретной вкладке (ID или slug).
func (s *TaxonomyService) PublicTreeFiltered(ctx context.Context, tabID *int, tabSlug *string) ([]models.TabTree, error) {
	var normSlug *string
	if tabSlug != nil {
		slug := normalizeSlug(*tabSlug)
		normSlug = &slug
	}
	items, err := s.repo.ListTabTreeFilter(ctx, tabID, normSlug)
	if err != nil {
		logger.Log.Error("Ошибка выборки дерева по фильтру", zap.Intp("tab_id", tabID), zap.Stringp("tab_slug", normSlug), zap.Error(err))
		return nil, err
	}
	return items, nil
}

// ----------------- helpers -----------------

var nonWord = regexp.MustCompile(`[^\p{L}\p{N}]+`) // всё, что не буквы/цифры, -> дефисы

// некоторые зарезервированные пути сайта — не позволяем чистому совпадению
var reservedSlugs = map[string]struct{}{
	"api": {}, "admin": {}, "auth": {}, "uploads": {}, "static": {},
	"documents": {}, "news": {}, "zavuch": {}, "recomm": {},
}

func slugify(s string) string {
	s = transliterate(s)
	s = strings.TrimSpace(s)
	s = nonWord.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "item"
	}
	// защищаемся от зарезервированных путей
	if _, bad := reservedSlugs[s]; bad {
		s = "tab-" + s
	}
	return s
}

func normalizeSlug(s string) string {
	return slugify(strings.ToLower(strings.TrimSpace(s)))
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

var translitMap = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
	'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i",
	'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
	'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
	'у': "u", 'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch",
	'ш': "sh", 'щ': "sch", 'ъ': "", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "yu", 'я': "ya",
}

func transliterate(input string) string {
	input = strings.ToLower(input)
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		if tr, ok := translitMap[r]; ok {
			b.WriteString(tr)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
