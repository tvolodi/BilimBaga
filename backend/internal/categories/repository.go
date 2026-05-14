package categories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository interface {
	GetAll(ctx context.Context) ([]Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	Create(ctx context.Context, name string, parentID, track *string, sortOrder int) (*Category, error)
	Update(ctx context.Context, c Category) (*Category, error)
	Delete(ctx context.Context, id string) error
	HasChildren(ctx context.Context, id string) (bool, error)
	HasQuestions(ctx context.Context, id string) (bool, error)
}

type pgRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) GetAll(ctx context.Context) ([]Category, error) {
	const q = `SELECT id, name, parent_id, track, sort_order, created_at, updated_at
	           FROM categories ORDER BY sort_order, name`
	var rows []Category
	if err := r.db.SelectContext(ctx, &rows, q); err != nil {
		return nil, fmt.Errorf("categories.GetAll: %w", err)
	}
	return rows, nil
}

func (r *pgRepository) GetByID(ctx context.Context, id string) (*Category, error) {
	const q = `SELECT id, name, parent_id, track, sort_order, created_at, updated_at
	           FROM categories WHERE id = $1`
	var c Category
	if err := r.db.GetContext(ctx, &c, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("categories.GetByID: %w", err)
	}
	return &c, nil
}

func (r *pgRepository) Create(ctx context.Context, name string, parentID, track *string, sortOrder int) (*Category, error) {
	const q = `
		INSERT INTO categories (name, parent_id, track, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, parent_id, track, sort_order, created_at, updated_at`
	var c Category
	if err := r.db.GetContext(ctx, &c, q, name, parentID, track, sortOrder); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return nil, ErrParentNotFound
		}
		return nil, fmt.Errorf("categories.Create: %w", err)
	}
	return &c, nil
}

func (r *pgRepository) Update(ctx context.Context, c Category) (*Category, error) {
	const q = `
		UPDATE categories
		SET name = $1, parent_id = $2, track = $3, sort_order = $4, updated_at = now()
		WHERE id = $5
		RETURNING id, name, parent_id, track, sort_order, created_at, updated_at`
	var out Category
	if err := r.db.GetContext(ctx, &out, q, c.Name, c.ParentID, c.Track, c.SortOrder, c.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return nil, ErrParentNotFound
		}
		return nil, fmt.Errorf("categories.Update: %w", err)
	}
	return &out, nil
}

func (r *pgRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("categories.Delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) HasChildren(ctx context.Context, id string) (bool, error) {
	var n int
	if err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM categories WHERE parent_id = $1`, id); err != nil {
		return false, fmt.Errorf("categories.HasChildren: %w", err)
	}
	return n > 0, nil
}

// HasQuestions returns true when any question references the category.
// The `questions` table is owned by FR-BB22; if it does not yet exist, this
// returns false so deletes are not falsely blocked.
func (r *pgRepository) HasQuestions(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists,
		`SELECT to_regclass('public.questions') IS NOT NULL`); err != nil {
		return false, fmt.Errorf("categories.HasQuestions: probe: %w", err)
	}
	if !exists {
		return false, nil
	}
	var n int
	if err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM questions WHERE category_id = $1`, id); err != nil {
		return false, fmt.Errorf("categories.HasQuestions: %w", err)
	}
	return n > 0, nil
}
