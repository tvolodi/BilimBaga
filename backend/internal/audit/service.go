package audit

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Service defines the business-logic interface for the audit domain.
type Service interface {
	List(ctx context.Context, tenantID string, filters AuditFilters, page, perPage int) ([]AuditEntry, int, error)
	Export(ctx context.Context, tenantID string, filters AuditFilters) ([]AuditEntry, error)
}

type service struct {
	db *sqlx.DB
}

// NewService creates a new Service backed by the given database connection.
func NewService(db *sqlx.DB) Service {
	return &service{db: db}
}

// List returns a paginated, tenant-scoped slice of audit entries.
func (s *service) List(ctx context.Context, tenantID string, filters AuditFilters, page, perPage int) ([]AuditEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}
	entries, total, err := ListAudit(ctx, s.db, tenantID, filters, page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("audit.Service.List: %w", err)
	}
	return entries, total, nil
}

// Export returns up to maxExportRows audit entries for CSV export, tenant-scoped.
func (s *service) Export(ctx context.Context, tenantID string, filters AuditFilters) ([]AuditEntry, error) {
	entries, _, err := ListAudit(ctx, s.db, tenantID, filters, 1, maxExportRows)
	if err != nil {
		return nil, fmt.Errorf("audit.Service.Export: %w", err)
	}
	return entries, nil
}
