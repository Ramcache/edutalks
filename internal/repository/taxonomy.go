package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"edutalks/internal/logger"
	"edutalks/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type TaxonomyRepo struct {
	db *pgxpool.Pool
}

func NewTaxonomyRepo(db *pgxpool.Pool) *TaxonomyRepo { return &TaxonomyRepo{db: db} }

// ----- Tabs -----

func (r *TaxonomyRepo) CreateTab(ctx context.Context, t *models.Tab) (int, error) {
	log := logger.WithCtx(ctx)

	var id int
	if err := r.db.QueryRow(ctx,
		`INSERT INTO tabs (slug, title, position, is_active) VALUES ($1,$2,$3,$4) RETURNING id`,
		t.Slug, t.Title, t.Position, t.IsActive,
	).Scan(&id); err != nil {
		log.Error("taxonomy repo: create tab failed", zap.Error(err), zap.String("slug", t.Slug))
		return 0, err
	}

	log.Info("taxonomy repo: tab created", zap.Int("id", id), zap.String("slug", t.Slug))
	return id, nil
}

func (r *TaxonomyRepo) UpdateTab(ctx context.Context, t *models.Tab) error {
	log := logger.WithCtx(ctx)

	_, err := r.db.Exec(ctx,
		`UPDATE tabs SET slug=$1, title=$2, position=$3, is_active=$4, updated_at=now() WHERE id=$5`,
		t.Slug, t.Title, t.Position, t.IsActive, t.ID,
	)
	if err != nil {
		log.Error("taxonomy repo: update tab failed", zap.Error(err), zap.Int("id", t.ID))
		return err
	}

	log.Info("taxonomy repo: tab updated", zap.Int("id", t.ID))
	return nil
}

func (r *TaxonomyRepo) DeleteTab(ctx context.Context, id int) error {
	log := logger.WithCtx(ctx)

	_, err := r.db.Exec(ctx, `DELETE FROM tabs WHERE id=$1`, id)
	if err != nil {
		log.Error("taxonomy repo: delete tab failed", zap.Error(err), zap.Int("id", id))
		return err
	}

	log.Info("taxonomy repo: tab deleted", zap.Int("id", id))
	return nil
}

// ----- Sections -----

func (r *TaxonomyRepo) CreateSection(ctx context.Context, s *models.Section) (int, error) {
	log := logger.WithCtx(ctx)

	var id int
	if err := r.db.QueryRow(ctx,
		`INSERT INTO sections (tab_id, slug, title, description, position, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		s.TabID, s.Slug, s.Title, s.Description, s.Position, s.IsActive,
	).Scan(&id); err != nil {
		log.Error("taxonomy repo: create section failed", zap.Error(err), zap.String("slug", s.Slug), zap.Int("tab_id", s.TabID))
		return 0, err
	}

	log.Info("taxonomy repo: section created", zap.Int("id", id), zap.String("slug", s.Slug), zap.Int("tab_id", s.TabID))
	return id, nil
}

func (r *TaxonomyRepo) UpdateSection(ctx context.Context, s *models.Section) error {
	log := logger.WithCtx(ctx)

	_, err := r.db.Exec(ctx,
		`UPDATE sections SET slug=$1, title=$2, description=$3, position=$4, is_active=$5, updated_at=now() WHERE id=$6`,
		s.Slug, s.Title, s.Description, s.Position, s.IsActive, s.ID,
	)
	if err != nil {
		log.Error("taxonomy repo: update section failed", zap.Error(err), zap.Int("id", s.ID))
		return err
	}

	log.Info("taxonomy repo: section updated", zap.Int("id", s.ID))
	return nil
}

func (r *TaxonomyRepo) DeleteSection(ctx context.Context, id int) error {
	log := logger.WithCtx(ctx)

	_, err := r.db.Exec(ctx, `DELETE FROM sections WHERE id=$1`, id)
	if err != nil {
		log.Error("taxonomy repo: delete section failed", zap.Error(err), zap.Int("id", id))
		return err
	}

	log.Info("taxonomy repo: section deleted", zap.Int("id", id))
	return nil
}

// ----- Public tree -----

func (r *TaxonomyRepo) ListTabTree(ctx context.Context) ([]models.TabTree, error) {
	log := logger.WithCtx(ctx)

	const q = `
WITH s AS (
  SELECT s.*, COALESCE(d.cnt,0) AS docs_count
  FROM sections s
  LEFT JOIN (
    SELECT section_id, COUNT(*) cnt FROM documents GROUP BY section_id
  ) d ON d.section_id = s.id
  WHERE s.is_active = true
)
SELECT
  t.id, t.slug, t.title, t.position, t.is_active, t.created_at, t.updated_at,
  s.id, s.tab_id, s.slug, s.title, s.description, s.position, s.is_active, s.created_at, s.updated_at, s.docs_count
FROM tabs t
LEFT JOIN s ON s.tab_id = t.id
WHERE t.is_active = true
ORDER BY t.position, t.id, s.position, s.id;
`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		log.Error("taxonomy repo: list tree query failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var out []models.TabTree
	var cur *models.TabTree

	for rows.Next() {
		var t models.Tab

		var (
			secID        sql.NullInt32
			secTabID     sql.NullInt32
			secSlug      sql.NullString
			secTitle     sql.NullString
			secDesc      sql.NullString
			secPos       sql.NullInt32
			secActive    sql.NullBool
			secCreatedAt sql.NullTime
			secUpdatedAt sql.NullTime
			docsCount    sql.NullInt64
		)

		if err := rows.Scan(
			&t.ID, &t.Slug, &t.Title, &t.Position, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
			&secID, &secTabID, &secSlug, &secTitle, &secDesc, &secPos, &secActive, &secCreatedAt, &secUpdatedAt, &docsCount,
		); err != nil {
			log.Error("taxonomy repo: scan tree row failed", zap.Error(err))
			return nil, err
		}

		if cur == nil || cur.Tab.ID != t.ID {
			out = append(out, models.TabTree{Tab: t})
			cur = &out[len(out)-1]
		}

		if secID.Valid {
			s := models.Section{
				ID:          int(secID.Int32),
				TabID:       int(secTabID.Int32),
				Slug:        secSlug.String,
				Title:       secTitle.String,
				Description: secDesc.String,
				Position:    int(secPos.Int32),
				IsActive:    secActive.Bool,
				CreatedAt:   secCreatedAt.Time,
				UpdatedAt:   secUpdatedAt.Time,
			}
			cnt := 0
			if docsCount.Valid {
				cnt = int(docsCount.Int64)
			}
			cur.Sections = append(cur.Sections, models.SectionWithCount{
				Section:   s,
				DocsCount: cnt,
			})
		}
	}

	if err := rows.Err(); err != nil {
		log.Error("taxonomy repo: rows error list tree", zap.Error(err))
		return nil, err
	}

	log.Debug("taxonomy repo: list tree done", zap.Int("tabs", len(out)))
	return out, nil
}

// ListTabTreeFilter — дерево по ID/slug вкладки (любой из них, опционально).
func (r *TaxonomyRepo) ListTabTreeFilter(ctx context.Context, tabID *int, tabSlug *string) ([]models.TabTree, error) {
	log := logger.WithCtx(ctx)

	q := `
WITH s AS (
  SELECT s.*, COALESCE(d.cnt,0) AS docs_count
  FROM sections s
  LEFT JOIN (SELECT section_id, COUNT(*) cnt FROM documents GROUP BY section_id) d
    ON d.section_id = s.id
  WHERE s.is_active = true
)
SELECT
  t.id, t.slug, t.title, t.position, t.is_active, t.created_at, t.updated_at,
  s.id, s.tab_id, s.slug, s.title, s.description, s.position, s.is_active, s.created_at, s.updated_at, s.docs_count
FROM tabs t
LEFT JOIN s ON s.tab_id = t.id
WHERE t.is_active = true
`
	args := []any{}
	conds := []string{}

	if tabID != nil {
		conds = append(conds, "t.id = $"+itoa(len(args)+1))
		args = append(args, *tabID)
	}
	if tabSlug != nil && *tabSlug != "" {
		conds = append(conds, "t.slug = $"+itoa(len(args)+1))
		args = append(args, *tabSlug)
	}
	if len(conds) > 0 {
		q += " AND (" + strings.Join(conds, " OR ") + ")"
	}

	q += " ORDER BY t.position, t.id, s.position, s.id;"

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		log.Error("taxonomy repo: list tree filter query failed", zap.Error(err),
			zap.Any("tab_id", tabID), zap.Any("tab_slug", tabSlug))
		return nil, err
	}
	defer rows.Close()

	var out []models.TabTree
	var cur *models.TabTree

	for rows.Next() {
		var t models.Tab

		var (
			secID        sql.NullInt32
			secTabID     sql.NullInt32
			secSlug      sql.NullString
			secTitle     sql.NullString
			secDesc      sql.NullString
			secPos       sql.NullInt32
			secActive    sql.NullBool
			secCreatedAt sql.NullTime
			secUpdatedAt sql.NullTime
			docsCount    sql.NullInt64
		)

		if err := rows.Scan(
			&t.ID, &t.Slug, &t.Title, &t.Position, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
			&secID, &secTabID, &secSlug, &secTitle, &secDesc, &secPos, &secActive, &secCreatedAt, &secUpdatedAt, &docsCount,
		); err != nil {
			log.Error("taxonomy repo: scan tree filter row failed", zap.Error(err))
			return nil, err
		}

		if cur == nil || cur.Tab.ID != t.ID {
			out = append(out, models.TabTree{Tab: t})
			cur = &out[len(out)-1]
		}
		if secID.Valid {
			s := models.Section{
				ID:          int(secID.Int32),
				TabID:       int(secTabID.Int32),
				Slug:        secSlug.String,
				Title:       secTitle.String,
				Description: secDesc.String,
				Position:    int(secPos.Int32),
				IsActive:    secActive.Bool,
				CreatedAt:   secCreatedAt.Time,
				UpdatedAt:   secUpdatedAt.Time,
			}
			cnt := 0
			if docsCount.Valid {
				cnt = int(docsCount.Int64)
			}
			cur.Sections = append(cur.Sections, models.SectionWithCount{
				Section:   s,
				DocsCount: cnt,
			})
		}
	}
	if err := rows.Err(); err != nil {
		log.Error("taxonomy repo: rows error list tree filter", zap.Error(err))
		return nil, err
	}

	log.Debug("taxonomy repo: list tree filter done",
		zap.Any("tab_id", tabID), zap.Any("tab_slug", tabSlug), zap.Int("tabs", len(out)))
	return out, nil
}

// ----- Utils -----

func itoa(i int) string { return fmt.Sprintf("%d", i) }

func (r *TaxonomyRepo) TabSlugExists(ctx context.Context, slug string) (bool, error) {
	log := logger.WithCtx(ctx)

	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tabs WHERE slug=$1)`, slug).Scan(&exists); err != nil {
		log.Error("taxonomy repo: tab slug exists check failed", zap.Error(err), zap.String("slug", slug))
		return false, err
	}
	log.Debug("taxonomy repo: tab slug exists", zap.String("slug", slug), zap.Bool("exists", exists))
	return exists, nil
}

// SectionSlugExists — проверка уникальности slug в рамках вкладки.
func (r *TaxonomyRepo) SectionSlugExists(ctx context.Context, tabID int, slug string) (bool, error) {
	log := logger.WithCtx(ctx)

	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sections WHERE tab_id=$1 AND slug=$2)`,
		tabID, slug,
	).Scan(&exists); err != nil {
		log.Error("taxonomy repo: section slug exists check failed", zap.Error(err), zap.Int("tab_id", tabID), zap.String("slug", slug))
		return false, err
	}
	log.Debug("taxonomy repo: section slug exists", zap.Int("tab_id", tabID), zap.String("slug", slug), zap.Bool("exists", exists))
	return exists, nil
}

func (r *TaxonomyRepo) GetSectionSlugByID(ctx context.Context, id int) (string, error) {
	log := logger.WithCtx(ctx)

	var slug string
	if err := r.db.QueryRow(ctx, `SELECT slug FROM sections WHERE id=$1`, id).Scan(&slug); err != nil {
		if err == pgx.ErrNoRows {
			log.Warn("taxonomy repo: section slug not found", zap.Int("id", id))
		} else {
			log.Error("taxonomy repo: get section slug failed", zap.Error(err), zap.Int("id", id))
		}
		return "", err
	}
	log.Debug("taxonomy repo: got section slug", zap.Int("id", id), zap.String("slug", slug))
	return slug, nil
}

func (r *TaxonomyRepo) GetTabSlugByID(ctx context.Context, id int) (string, error) {
	log := logger.WithCtx(ctx)

	var slug string
	if err := r.db.QueryRow(ctx, `SELECT slug FROM tabs WHERE id = $1`, id).Scan(&slug); err != nil {
		if err == pgx.ErrNoRows {
			log.Warn("taxonomy repo: tab slug not found", zap.Int("id", id))
		} else {
			log.Error("taxonomy repo: get tab slug failed", zap.Error(err), zap.Int("id", id))
		}
		return "", err
	}
	log.Debug("taxonomy repo: got tab slug", zap.Int("id", id), zap.String("slug", slug))
	return slug, nil
}

func (r *TaxonomyRepo) GetTabIDBySectionID(ctx context.Context, sectionID int) (int, error) {
	log := logger.WithCtx(ctx)

	var id int
	if err := r.db.QueryRow(ctx, `SELECT tab_id FROM sections WHERE id = $1`, sectionID).Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			log.Warn("taxonomy repo: tab id by section not found", zap.Int("section_id", sectionID))
		} else {
			log.Error("taxonomy repo: get tab id by section failed", zap.Error(err), zap.Int("section_id", sectionID))
		}
		return 0, err
	}
	log.Debug("taxonomy repo: got tab id by section", zap.Int("section_id", sectionID), zap.Int("tab_id", id))
	return id, nil
}
