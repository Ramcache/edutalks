package services

import (
	"context"
	"edutalks/internal/models"
	"edutalks/internal/repository"
)

type TaxonomyService struct{ repo *repository.TaxonomyRepo }

func NewTaxonomyService(r *repository.TaxonomyRepo) *TaxonomyService {
	return &TaxonomyService{repo: r}
}

func (s *TaxonomyService) CreateTab(ctx context.Context, t *models.Tab) (int, error) {
	return s.repo.CreateTab(ctx, t)
}
func (s *TaxonomyService) UpdateTab(ctx context.Context, t *models.Tab) error {
	return s.repo.UpdateTab(ctx, t)
}
func (s *TaxonomyService) DeleteTab(ctx context.Context, id int) error {
	return s.repo.DeleteTab(ctx, id)
}

func (s *TaxonomyService) CreateSection(ctx context.Context, sec *models.Section) (int, error) {
	return s.repo.CreateSection(ctx, sec)
}
func (s *TaxonomyService) UpdateSection(ctx context.Context, sec *models.Section) error {
	return s.repo.UpdateSection(ctx, sec)
}
func (s *TaxonomyService) DeleteSection(ctx context.Context, id int) error {
	return s.repo.DeleteSection(ctx, id)
}

func (s *TaxonomyService) PublicTree(ctx context.Context) ([]models.TabTree, error) {
	return s.repo.ListTabTree(ctx)
}
