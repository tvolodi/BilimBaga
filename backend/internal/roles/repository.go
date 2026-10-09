package roles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository is the data-access interface for the roles domain.
type Repository interface {
	List(ctx context.Context) ([]Role, error)
	GetByID(ctx context.Context, id string) (*Role, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	PermissionsByIDs(ctx context.Context, ids []string) ([]Permission, error)
	Create(ctx context.Context, name, description string, permissionIDs []string) (string, error)
	Update(ctx context.Context, id, description string, permissionIDs []string) error
	Delete(ctx context.Context, id string) error
	CountUsers(ctx context.Context, id string) (int, error)
}

type pgRepository struct{ db *sqlx.DB }

// NewRepository creates a PostgreSQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository { return &pgRepository{db: db} }

const roleSelect = `
	SELECT r.id, r.name, r.description, r.is_system, r.created_at,
	       (SELECT COUNT(*) FROM users u WHERE u.role_id = r.id) AS user_count
	  FROM roles r`

func (r *pgRepository) attachPermissions(ctx context.Context, list []Role) error {
	var rows []RolePermission
	const q = `
		SELECT rp.role_id, p.resource, p.action
		  FROM role_permissions rp
		  JOIN permissions p ON p.id = rp.permission_id
		 ORDER BY p.resource, p.action`
	if err := r.db.SelectContext(ctx, &rows, q); err != nil {
		return fmt.Errorf("roles.attachPermissions: %w", err)
	}
	byRole := make(map[string][]string)
	for _, rp := range rows {
		byRole[rp.RoleID] = append(byRole[rp.RoleID], rp.Resource+":"+rp.Action)
	}
	for i := range list {
		list[i].Permissions = byRole[list[i].ID]
		if list[i].Permissions == nil {
			list[i].Permissions = []string{}
		}
	}
	return nil
}

func (r *pgRepository) List(ctx context.Context) ([]Role, error) {
	var out []Role
	if err := r.db.SelectContext(ctx, &out, roleSelect+` ORDER BY r.is_system DESC, r.name`); err != nil {
		return nil, fmt.Errorf("roles.List: %w", err)
	}
	if err := r.attachPermissions(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *pgRepository) GetByID(ctx context.Context, id string) (*Role, error) {
	var role Role
	if err := r.db.GetContext(ctx, &role, roleSelect+` WHERE r.id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("roles.GetByID: %w", err)
	}
	list := []Role{role}
	if err := r.attachPermissions(ctx, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (r *pgRepository) ListPermissions(ctx context.Context) ([]Permission, error) {
	var out []Permission
	if err := r.db.SelectContext(ctx, &out, `SELECT id, resource, action FROM permissions ORDER BY resource, action`); err != nil {
		return nil, fmt.Errorf("roles.ListPermissions: %w", err)
	}
	return out, nil
}

func (r *pgRepository) PermissionsByIDs(ctx context.Context, ids []string) ([]Permission, error) {
	var out []Permission
	if len(ids) == 0 {
		return out, nil
	}
	if err := r.db.SelectContext(ctx, &out,
		`SELECT id, resource, action FROM permissions WHERE id = ANY($1::uuid[])`, pq.Array(ids)); err != nil {
		return nil, fmt.Errorf("roles.PermissionsByIDs: %w", err)
	}
	return out, nil
}

func setPermissions(ctx context.Context, tx *sqlx.Tx, roleID string, ids []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return fmt.Errorf("clear permissions: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO role_permissions (role_id, permission_id)
		 SELECT $1, UNNEST($2::uuid[])`, roleID, pq.Array(ids)); err != nil {
		return fmt.Errorf("insert permissions: %w", err)
	}
	return nil
}

func (r *pgRepository) Create(ctx context.Context, name, description string, permissionIDs []string) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("roles.Create: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var id string
	if err := tx.GetContext(ctx, &id,
		`INSERT INTO roles (name, description, is_system) VALUES ($1, $2, false) RETURNING id`,
		name, description); err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", ErrNameTaken
		}
		return "", fmt.Errorf("roles.Create: insert: %w", err)
	}
	if err := setPermissions(ctx, tx, id, permissionIDs); err != nil {
		return "", fmt.Errorf("roles.Create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("roles.Create: commit: %w", err)
	}
	return id, nil
}

func (r *pgRepository) Update(ctx context.Context, id, description string, permissionIDs []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("roles.Update: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Only custom roles are ever modified; the guard is repeated in SQL so a
	// concurrent change can never touch a system role.
	res, err := tx.ExecContext(ctx,
		`UPDATE roles SET description = $2 WHERE id = $1 AND is_system = false`, id, description)
	if err != nil {
		return fmt.Errorf("roles.Update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := setPermissions(ctx, tx, id, permissionIDs); err != nil {
		return fmt.Errorf("roles.Update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("roles.Update: commit: %w", err)
	}
	return nil
}

func (r *pgRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM roles WHERE id = $1 AND is_system = false`, id)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return &InUseError{}
		}
		return fmt.Errorf("roles.Delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) CountUsers(ctx context.Context, id string) (int, error) {
	var n int
	if err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM users WHERE role_id = $1`, id); err != nil {
		return 0, fmt.Errorf("roles.CountUsers: %w", err)
	}
	return n, nil
}
