package audit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// filterOptionsTTL bounds how stale a tenant's filter options can be (#563). A new action code or entity type is offered
// by the filters at most this long after its first audit row, and the tenant's log is scanned once per TTL, not per request.
const filterOptionsTTL = 5 * time.Minute

// Service defines the business-logic interface for the audit domain.
type Service interface {
	List(ctx context.Context, tenantID string, filters AuditFilters, page, perPage int) ([]AuditEntry, int, error)
	Export(ctx context.Context, tenantID string, filters AuditFilters) ([]AuditEntry, error)
	// FilterOptions returns the action codes and entity types the tenant's audit log holds (#557).
	FilterOptions(ctx context.Context, tenantID string) (AuditFilterOptions, error)
}

type service struct {
	db *sqlx.DB
	// loadFilterOptions reads a tenant's options from the log. Production uses ListFilterOptions; tests replace it.
	loadFilterOptions func(ctx context.Context, tenantID string) (AuditFilterOptions, error)
	now               func() time.Time

	mu            sync.Mutex
	filterOptions map[string]cachedFilterOptions
}

// cachedFilterOptions is one tenant's options and the time until which they are served from the cache.
type cachedFilterOptions struct {
	opts    AuditFilterOptions
	expires time.Time
}

// NewService creates a new Service backed by the given database connection.
func NewService(db *sqlx.DB) Service {
	return &service{
		db: db,
		loadFilterOptions: func(ctx context.Context, tenantID string) (AuditFilterOptions, error) {
			return ListFilterOptions(ctx, db, tenantID)
		},
		now:           time.Now,
		filterOptions: map[string]cachedFilterOptions{},
	}
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

// FilterOptions returns the distinct action codes and entity types of the tenant's audit log (#557), cached per tenant for
// filterOptionsTTL (#563). Callers must not modify the returned slices: they are shared with later callers.
func (s *service) FilterOptions(ctx context.Context, tenantID string) (AuditFilterOptions, error) {
	now := s.now()
	s.mu.Lock()
	cached, ok := s.filterOptions[tenantID]
	s.mu.Unlock()
	if ok && now.Before(cached.expires) {
		return cached.opts, nil
	}

	opts, err := s.loadFilterOptions(ctx, tenantID)
	if err != nil {
		// Errors are not cached: the next call reads the log again.
		return AuditFilterOptions{}, fmt.Errorf("audit.Service.FilterOptions: %w", err)
	}
	s.mu.Lock()
	s.filterOptions[tenantID] = cachedFilterOptions{opts: opts, expires: now.Add(filterOptionsTTL)}
	s.mu.Unlock()
	return opts, nil
}
