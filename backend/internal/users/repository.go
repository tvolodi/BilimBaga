package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository is the data-access interface for the users domain.
type Repository interface {
	List(ctx context.Context, filters ListFilters, deptScope *string) ([]User, int, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, email, fullName, passwordHash string, departmentID *string, roleID string) (*User, error)
	Update(ctx context.Context, id, fullName string, departmentID *string, roleID string) (*User, error)
	Deactivate(ctx context.Context, id string) error
	RevokeAllTokens(ctx context.Context, userID string) error
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	GetDepartmentIDByName(ctx context.Context, name string) (string, error)
	GetRoleIDByName(ctx context.Context, name string) (string, error)
	GetRoleNameByID(ctx context.Context, roleID string) (string, error)
	WriteAuditLog(ctx context.Context, userID *string, action, ipAddress string, metadata map[string]any) error
}

type pgRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository {
	return &pgRepository{db: db}
}

// List returns a paginated slice of users and the total count matching the filters.
// deptScope, when non-nil, restricts results to a specific department.
func (r *pgRepository) List(ctx context.Context, f ListFilters, deptScope *string) ([]User, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	i := 1

	if deptScope != nil {
		where += fmt.Sprintf(" AND u.department_id = $%d", i)
		args = append(args, *deptScope)
		i++
	}
	if f.DepartmentID != nil {
		where += fmt.Sprintf(" AND u.department_id = $%d", i)
		args = append(args, *f.DepartmentID)
		i++
	}
	if f.RoleID != nil {
		where += fmt.Sprintf(" AND u.role_id = $%d", i)
		args = append(args, *f.RoleID)
		i++
	}
	if f.Status != nil {
		where += fmt.Sprintf(" AND u.status = $%d", i)
		args = append(args, *f.Status)
		i++
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s`, where)
	var total int
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, fmt.Errorf("users.List count: %w", err)
	}

	listQ := fmt.Sprintf(`
		SELECT u.id, u.email, u.full_name, u.department_id, d.name AS department_name,
		       u.role_id, ro.name AS role_name, u.status, u.force_password_change, u.created_at
		FROM   users u
		LEFT JOIN departments d ON d.id = u.department_id
		JOIN  roles ro ON ro.id = u.role_id
		%s
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d`, where, i, i+1)
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)

	var users []User
	if err := r.db.SelectContext(ctx, &users, listQ, args...); err != nil {
		return nil, 0, fmt.Errorf("users.List: %w", err)
	}
	if users == nil {
		users = []User{}
	}
	return users, total, nil
}

// GetByID fetches a single user by UUID, joined with department and role names.
func (r *pgRepository) GetByID(ctx context.Context, id string) (*User, error) {
	const q = `
		SELECT u.id, u.email, u.full_name, u.department_id, d.name AS department_name,
		       u.role_id, ro.name AS role_name, u.status, u.force_password_change, u.created_at
		FROM   users u
		LEFT JOIN departments d ON d.id = u.department_id
		JOIN  roles ro ON ro.id = u.role_id
		WHERE  u.id = $1`
	var u User
	if err := r.db.GetContext(ctx, &u, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("users.GetByID: %w", err)
	}
	return &u, nil
}

// Create inserts a new user with a pre-hashed password and returns the persisted record.
func (r *pgRepository) Create(ctx context.Context, email, fullName, passwordHash string, departmentID *string, roleID string) (*User, error) {
	const q = `
		INSERT INTO users (email, password_hash, full_name, department_id, role_id, force_password_change)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id, email, full_name, department_id, role_id, status, force_password_change, created_at`

	type row struct {
		ID                  string  `db:"id"`
		Email               string  `db:"email"`
		FullName            string  `db:"full_name"`
		DepartmentID        *string `db:"department_id"`
		RoleID              string  `db:"role_id"`
		Status              string  `db:"status"`
		ForcePasswordChange bool    `db:"force_password_change"`
	}
	var inserted row
	if err := r.db.GetContext(ctx, &inserted, q, email, passwordHash, fullName, departmentID, roleID); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("users.Create: %w", err)
	}

	// Re-fetch with joins to get department_name and role_name.
	return r.GetByID(ctx, inserted.ID)
}

// Update sets full_name, department_id, and role_id for an existing user.
func (r *pgRepository) Update(ctx context.Context, id, fullName string, departmentID *string, roleID string) (*User, error) {
	const q = `
		UPDATE users
		SET    full_name = $1, department_id = $2, role_id = $3, updated_at = now()
		WHERE  id = $4`
	result, err := r.db.ExecContext(ctx, q, fullName, departmentID, roleID, id)
	if err != nil {
		return nil, fmt.Errorf("users.Update: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

// Deactivate sets a user's status to 'inactive'.
func (r *pgRepository) Deactivate(ctx context.Context, id string) error {
	const q = `UPDATE users SET status = 'inactive', updated_at = now() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("users.Deactivate: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// RevokeAllTokens marks all refresh tokens for a user as revoked.
func (r *pgRepository) RevokeAllTokens(ctx context.Context, userID string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := r.db.ExecContext(ctx, q, userID); err != nil {
		return fmt.Errorf("users.RevokeAllTokens: %w", err)
	}
	return nil
}

// UpdatePassword stores a new bcrypt hash and forces a password change on next login.
func (r *pgRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	const q = `UPDATE users SET password_hash = $1, force_password_change = true, updated_at = now() WHERE id = $2`
	result, err := r.db.ExecContext(ctx, q, passwordHash, id)
	if err != nil {
		return fmt.Errorf("users.UpdatePassword: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetDepartmentIDByName resolves a department name to its UUID.
func (r *pgRepository) GetDepartmentIDByName(ctx context.Context, name string) (string, error) {
	var id string
	const q = `SELECT id FROM departments WHERE name = $1 LIMIT 1`
	if err := r.db.GetContext(ctx, &id, q, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("users.GetDepartmentIDByName: %w", err)
	}
	return id, nil
}

// GetRoleIDByName resolves a role name to its UUID.
func (r *pgRepository) GetRoleIDByName(ctx context.Context, name string) (string, error) {
	var id string
	const q = `SELECT id FROM roles WHERE name = $1 LIMIT 1`
	if err := r.db.GetContext(ctx, &id, q, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("users.GetRoleIDByName: %w", err)
	}
	return id, nil
}

// GetRoleNameByID resolves a role UUID to its name.
func (r *pgRepository) GetRoleNameByID(ctx context.Context, roleID string) (string, error) {
	var name string
	const q = `SELECT name FROM roles WHERE id = $1`
	if err := r.db.GetContext(ctx, &name, q, roleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("users.GetRoleNameByID: %w", err)
	}
	return name, nil
}

// WriteAuditLog appends an entry to the audit_log table.
func (r *pgRepository) WriteAuditLog(ctx context.Context, userID *string, action, ipAddress string, metadata map[string]any) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("users.WriteAuditLog: marshal metadata: %w", err)
	}
	const q = `INSERT INTO audit_log (user_id, action, ip_address, metadata) VALUES ($1, $2, $3, $4)`
	if _, err := r.db.ExecContext(ctx, q, userID, action, ipAddress, raw); err != nil {
		return fmt.Errorf("users.WriteAuditLog: %w", err)
	}
	return nil
}

// isUniqueViolation returns true when err is a PostgreSQL unique-constraint error.
func isUniqueViolation(err error) bool {
	var pgErr *pq.Error
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
