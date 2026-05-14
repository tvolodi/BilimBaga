package tenant

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository defines the data access layer for tenant configuration.
type Repository interface {
	GetAll(ctx context.Context) (map[string]json.RawMessage, error)
	Upsert(ctx context.Context, key string, value json.RawMessage) error
}

type pgRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository {
	return &pgRepository{db: db}
}

type tenantConfigRow struct {
	Key   string          `db:"key"`
	Value json.RawMessage `db:"value"`
}

// GetAll reads all rows from tenant_config and returns them as a keyed map.
func (r *pgRepository) GetAll(ctx context.Context) (map[string]json.RawMessage, error) {
	var rows []tenantConfigRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT key, value FROM tenant_config`); err != nil {
		return nil, fmt.Errorf("tenant.GetAll: %w", err)
	}
	result := make(map[string]json.RawMessage, len(rows))
	for _, row := range rows {
		result[row.Key] = row.Value
	}
	return result, nil
}

// Upsert inserts or updates a single key in tenant_config.
func (r *pgRepository) Upsert(ctx context.Context, key string, value json.RawMessage) error {
	const q = `
		INSERT INTO tenant_config (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE
			SET value      = EXCLUDED.value,
			    updated_at = now()`
	if _, err := r.db.ExecContext(ctx, q, key, value); err != nil {
		return fmt.Errorf("tenant.Upsert: %w", err)
	}
	return nil
}
