package audit

import (
	"encoding/json"
	"time"
)

// AuditEntry represents a single row in the audit_log table.
type AuditEntry struct {
	ID         string          `db:"id"          json:"id"`
	TenantID   string          `db:"tenant_id"   json:"-"`
	ActorID    *string         `db:"actor_id"    json:"actor_id"`
	Action     string          `db:"action"      json:"action"`
	EntityType *string         `db:"entity_type" json:"entity_type"`
	EntityID   *string         `db:"entity_id"   json:"entity_id"`
	IP         string          `db:"ip"          json:"ip"`
	Metadata   json.RawMessage `db:"metadata"    json:"metadata"`
	CreatedAt  time.Time       `db:"created_at"  json:"created_at"`
}

// AuditFilters holds optional query filters for the audit log list endpoint.
type AuditFilters struct {
	ActorID    *string
	Action     *string
	EntityType *string
	From       *time.Time
	To         *time.Time
}

// maxExportRows caps the number of rows returned by the CSV export.
const maxExportRows = 10_000
