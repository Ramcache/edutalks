package repository

import (
	"context"
	"database/sql"
	"edutalks/internal/models"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TaxonomyRepo struct {
	db *pgxpool.Pool
}

func NewTaxonomyRepo(db *pgxpool.Pool) *TaxonomyRepo { return &TaxonomyRepo{db: db} }

// ----- Tabs -----

func (r *TaxonomyRepo) CreateTab(ctx context.Context, t *models.Tab) (int, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`INSERT INTO tabs (slug, title, position, is_active) VALUES ($1,$2,$3,$4) RETURNING id`,
		t.Slug, t.Title, t.Position, t.IsActive,
	).Scan(&id)
	return id, err
}

func (r *TaxonomyRepo) UpdateTab(ctx context.Context, t *models.Tab) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tabs SET slug=$1, title=$2, position=$3, is_active=$4, updated_at=now() WHERE id=$5`,
		t.Slug, t.Title, t.Position, t.IsActive, t.ID,
	)
	return err
}

func (r *TaxonomyRepo) DeleteTab(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tabs WHERE id=$1`, id)
	return err
}

// ----- Sections -----

func (r *TaxonomyRepo) CreateSection(ctx context.Context, s *models.Section) (int, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`INSERT INTO sections (tab_id, slug, title, description, position, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		s.TabID, s.Slug, s.Title, s.Description, s.Position, s.IsActive,
	).Scan(&id)
	return id, err
}

func (r *TaxonomyRepo) UpdateSection(ctx context.Context, s *models.Section) error {
	_, err := r.db.Exec(ctx,
		`UPDATE sections SET slug=$1, title=$2, description=$3, position=$4, is_active=$5, updated_at=now() WHERE id=$6`,
		s.Slug, s.Title, s.Description, s.Position, s.IsActive, s.ID,
	)
	return err
}

func (r *TaxonomyRepo) DeleteSection(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sections WHERE id=$1`, id)
	return err
}

// ----- Public tree -----

func (r *TaxonomyRepo) ListTabTree(ctx context.Context) ([]models.TabTree, error) {
	q := `
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
  -- nullable поля раздела (из LEFT JOIN)
  s.id, s.tab_id, s.slug, s.title, s.description, s.position, s.is_active, s.created_at, s.updated_at, s.docs_count
FROM tabs t
LEFT JOIN s ON s.tab_id = t.id
WHERE t.is_active = true
ORDER BY t.position, t.id, s.position, s.id;
`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.TabTree
	var cur *models.TabTree

	for rows.Next() {
		// вкладка — без NULL
		var t models.Tab

		// раздел — всё nullable, т.к. LEFT JOIN
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
			// tab
			&t.ID, &t.Slug, &t.Title, &t.Position, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
			// section (nullable)
			&secID, &secTabID, &secSlug, &secTitle, &secDesc, &secPos, &secActive, &secCreatedAt, &secUpdatedAt, &docsCount,
		); err != nil {
			return nil, err
		}

		// новая вкладка?
		if cur == nil || cur.Tab.ID != t.ID {
			out = append(out, models.TabTree{Tab: t})
			cur = &out[len(out)-1]
		}

		// добавляем раздел только если он действительно есть (secID.Valid)
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
		return nil, err
	}
	return out, nil
}

// ListTabTreeFilter возвращает дерево по фильтру: по ID или по slug вкладки (любой из них, опционально).
func (r *TaxonomyRepo) ListTabTreeFilter(ctx context.Context, tabID *int, tabSlug *string) ([]models.TabTree, error) {
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
		return nil, err
	}
	return out, nil
}

// простой хелпер без strconv импортов
func itoa(i int) string { return fmt.Sprintf("%d", i) }

func (r *TaxonomyRepo) TabSlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tabs WHERE slug=$1)`, slug).Scan(&exists)
	return exists, err
}

// Для секций — проверяем уникальность в рамках конкретной вкладки
func (r *TaxonomyRepo) SectionSlugExists(ctx context.Context, tabID int, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sections WHERE tab_id=$1 AND slug=$2)`, tabID, slug).Scan(&exists)
	return exists, err
}

func (r *TaxonomyRepo) GetSectionSlugByID(ctx context.Context, id int) (string, error) {
	var slug string
	err := r.db.QueryRow(ctx, `SELECT slug FROM sections WHERE id=$1`, id).Scan(&slug)
	return slug, err
}

func (r *TaxonomyRepo) GetTabSlugByID(ctx context.Context, id int) (string, error) {
	var slug string
	err := r.db.QueryRow(ctx, `SELECT slug FROM tabs WHERE id = $1`, id).Scan(&slug)
	if err != nil {
		return "", err
	}
	return slug, nil
}

func (r *TaxonomyRepo) GetTabIDBySectionID(ctx context.Context, sectionID int) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, `SELECT tab_id FROM sections WHERE id = $1`, sectionID).Scan(&id)
	return id, err
}
