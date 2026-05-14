package tags

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository interface {
	GetAll(ctx context.Context) ([]Tag, error)
	GetByID(ctx context.Context, id string) (*Tag, error)
	Create(ctx context.Context, name string) (*Tag, error)
	Update(ctx context.Context, id, name string) (*Tag, error)
	Delete(ctx context.Context, id string) error
	HasReferences(ctx context.Context, id string) (bool, error)
}

type pgRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) GetAll(ctx context.Context) ([]Tag, error) {
	// question_tags is owned by FR-BB22; if it does not yet exist, fall back to
	// a usage_count = 0 query so this endpoint stays available during early-phase
	// deployments (matches the defensive probe used by HasReferences).
	var hasJoin bool
	if err := r.db.GetContext(ctx, &hasJoin,
		`SELECT to_regclass('public.question_tags') IS NOT NULL`); err != nil {
		return nil, fmt.Errorf("tags.GetAll: probe: %w", err)
	}
	var q string
	if hasJoin {
		q = `
			SELECT t.id, t.name, t.created_at,
			       COALESCE(qt.usage_count, 0) AS usage_count
			FROM tags t
			LEFT JOIN (
				SELECT tag_id, COUNT(*) AS usage_count
				FROM question_tags
				GROUP BY tag_id
			) qt ON qt.tag_id = t.id
			ORDER BY t.name`
	} else {
		q = `SELECT id, name, created_at, 0 AS usage_count FROM tags ORDER BY name`
	}
	var rows []Tag
	if err := r.db.SelectContext(ctx, &rows, q); err != nil {
		return nil, fmt.Errorf("tags.GetAll: %w", err)
	}
	return rows, nil
}

func (r *pgRepository) GetByID(ctx context.Context, id string) (*Tag, error) {
	const q = `SELECT id, name, created_at, 0 AS usage_count FROM tags WHERE id = $1`
	var t Tag
	if err := r.db.GetContext(ctx, &t, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("tags.GetByID: %w", err)
	}
	return &t, nil
}

func (r *pgRepository) Create(ctx context.Context, name string) (*Tag, error) {
	const q = `
		INSERT INTO tags (name) VALUES ($1)
		RETURNING id, name, created_at, 0 AS usage_count`
	var t Tag
	if err := r.db.GetContext(ctx, &t, q, name); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("tags.Create: %w", err)
	}
	return &t, nil
}

func (r *pgRepository) Update(ctx context.Context, id, name string) (*Tag, error) {
	const q = `
		UPDATE tags SET name = $2 WHERE id = $1
		RETURNING id, name, created_at, 0 AS usage_count`
	var t Tag
	if err := r.db.GetContext(ctx, &t, q, id, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("tags.Update: %w", err)
	}
	return &t, nil
}

func (r *pgRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("tags.Delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// HasReferences returns true when any question_tags row references this tag.
// The `question_tags` table is owned by FR-BB22; if it does not yet exist,
// this returns false so deletes are not falsely blocked.
func (r *pgRepository) HasReferences(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists,
		`SELECT to_regclass('public.question_tags') IS NOT NULL`); err != nil {
		return false, fmt.Errorf("tags.HasReferences: probe: %w", err)
	}
	if !exists {
		return false, nil
	}
	var n int
	if err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM question_tags WHERE tag_id = $1`, id); err != nil {
		return false, fmt.Errorf("tags.HasReferences: %w", err)
	}
	return n > 0, nil
}
