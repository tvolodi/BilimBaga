package departments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository is the data-access interface for the departments domain.
type Repository interface {
	GetAll(ctx context.Context) ([]Department, error)
	GetByID(ctx context.Context, id string) (*Department, error)
	Create(ctx context.Context, name string, parentID *string) (*Department, error)
	Update(ctx context.Context, id, name string) (*Department, error)
	HasUsers(ctx context.Context, id string) (bool, error)
	HasChildren(ctx context.Context, id string) (bool, error)
	Delete(ctx context.Context, id string) error
}

type pgRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository {
	return &pgRepository{db: db}
}

// GetAll fetches all departments as a flat list ordered by name.
func (r *pgRepository) GetAll(ctx context.Context) ([]Department, error) {
	const q = `SELECT id, name, parent_id, created_at, updated_at FROM departments ORDER BY name`
	var depts []Department
	if err := r.db.SelectContext(ctx, &depts, q); err != nil {
		return nil, fmt.Errorf("departments.GetAll: %w", err)
	}
	return depts, nil
}

// GetByID fetches a single department by its UUID.
func (r *pgRepository) GetByID(ctx context.Context, id string) (*Department, error) {
	const q = `SELECT id, name, parent_id, created_at, updated_at FROM departments WHERE id = $1`
	var dept Department
	if err := r.db.GetContext(ctx, &dept, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("departments.GetByID: %w", err)
	}
	return &dept, nil
}

// Create inserts a new department and returns the persisted row.
func (r *pgRepository) Create(ctx context.Context, name string, parentID *string) (*Department, error) {
	const q = `
		INSERT INTO departments (name, parent_id)
		VALUES ($1, $2)
		RETURNING id, name, parent_id, created_at, updated_at`
	var dept Department
	if err := r.db.GetContext(ctx, &dept, q, name, parentID); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("departments.Create: %w", err)
	}
	return &dept, nil
}

// Update renames a department and refreshes updated_at, returning the updated row.
func (r *pgRepository) Update(ctx context.Context, id, name string) (*Department, error) {
	const q = `
		UPDATE departments SET name = $1, updated_at = now()
		WHERE id = $2
		RETURNING id, name, parent_id, created_at, updated_at`
	var dept Department
	if err := r.db.GetContext(ctx, &dept, q, name, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("departments.Update: %w", err)
	}
	return &dept, nil
}

// HasUsers returns true if any user record references this department.
func (r *pgRepository) HasUsers(ctx context.Context, id string) (bool, error) {
	var count int
	const q = `SELECT COUNT(*) FROM users WHERE department_id = $1`
	if err := r.db.GetContext(ctx, &count, q, id); err != nil {
		return false, fmt.Errorf("departments.HasUsers: %w", err)
	}
	return count > 0, nil
}

// HasChildren returns true if any department has this department as its parent.
func (r *pgRepository) HasChildren(ctx context.Context, id string) (bool, error) {
	var count int
	const q = `SELECT COUNT(*) FROM departments WHERE parent_id = $1`
	if err := r.db.GetContext(ctx, &count, q, id); err != nil {
		return false, fmt.Errorf("departments.HasChildren: %w", err)
	}
	return count > 0, nil
}

// Delete removes a department by ID.
func (r *pgRepository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM departments WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return fmt.Errorf("departments.Delete: %w", err)
	}
	return nil
}

// isUniqueViolation returns true for PostgreSQL unique constraint violation (code 23505).
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
