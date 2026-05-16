package audit

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// ListAudit returns paginated audit entries scoped to tenantID.
// tenantID is ALWAYS the first WHERE clause — cross-tenant reads are impossible.
// The users table is LEFT JOINed to populate actor_name for display.
func ListAudit(
	ctx context.Context,
	db *sqlx.DB,
	tenantID string,
	filters AuditFilters,
	page, perPage int,
) ([]AuditEntry, int, error) {
	args := []any{tenantID}
	where := []string{"al.tenant_id = $1"}
	idx := 2

	if filters.ActorID != nil {
		where = append(where, fmt.Sprintf("al.actor_id = $%d", idx))
		args = append(args, *filters.ActorID)
		idx++
	}
	if filters.Actor != nil {
		where = append(where, fmt.Sprintf("u.full_name ILIKE $%d", idx))
		args = append(args, "%"+*filters.Actor+"%")
		idx++
	}
	if filters.Action != nil {
		where = append(where, fmt.Sprintf("al.action = $%d", idx))
		args = append(args, *filters.Action)
		idx++
	}
	if filters.EntityType != nil {
		where = append(where, fmt.Sprintf("al.entity_type = $%d", idx))
		args = append(args, *filters.EntityType)
		idx++
	}
	if filters.From != nil {
		where = append(where, fmt.Sprintf("al.created_at >= $%d", idx))
		args = append(args, *filters.From)
		idx++
	}
	if filters.To != nil {
		where = append(where, fmt.Sprintf("al.created_at <= $%d", idx))
		args = append(args, *filters.To)
		idx++
	}

	join := "FROM audit_log al LEFT JOIN users u ON u.id = al.actor_id WHERE " + strings.Join(where, " AND ")

	var total int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) "+join, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("audit list count: %w", err)
	}

	offset := (page - 1) * perPage
	query := fmt.Sprintf(
		"SELECT al.id, al.tenant_id, al.actor_id, u.full_name AS actor_name, al.action, "+
			"al.entity_type, al.entity_id, al.ip, al.metadata, al.created_at "+
			join+" ORDER BY al.created_at DESC LIMIT $%d OFFSET $%d",
		idx, idx+1,
	)
	args = append(args, perPage, offset)

	rows, err := db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("audit list query: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.StructScan(&e); err != nil {
			return nil, 0, fmt.Errorf("audit list scan: %w", err)
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	return entries, total, nil
}
