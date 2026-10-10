package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/jmoiron/sqlx"
)

// Writer records audit events to the audit_log table.
// It is safe to call Write from any handler; failures are logged, never returned.
type Writer struct {
	db     *sqlx.DB
	logger *slog.Logger
}

// NewWriter creates a new Writer backed by the given database connection.
// If logger is nil, slog.Default() is used.
func NewWriter(db *sqlx.DB, logger *slog.Logger) *Writer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Writer{db: db, logger: logger}
}

// Write records an audit event. It is safe to call from any handler.
// It never returns an error; failures are logged via the injected logger.
// entityType may be empty (stored as NULL); entityID may be nil.
// If w is nil, Write is a no-op (useful in tests).
func (w *Writer) Write(ctx context.Context, r *http.Request, action, entityType string, entityID *string, metadata any) {
	if w == nil {
		return
	}

	tenantID := ctxkeys.TenantIDFromCtx(ctx)
	if tenantID == "" {
		w.logger.Warn("audit: missing tenant_id, dropping event", "action", action)
		return
	}

	actorIDStr := ctxkeys.UserIDFromCtx(ctx)
	var actorNull *string
	if actorIDStr != "" {
		actorNull = &actorIDStr
	}

	ip := api.ClientIP(r) // the resolved client (#475), never a forged X-Forwarded-For (#478)

	var meta []byte
	if metadata != nil {
		var err error
		meta, err = json.Marshal(metadata)
		if err != nil {
			w.logger.Warn("audit: marshal metadata failed", "err", err)
			meta = []byte("{}")
		}
	} else {
		meta = []byte("{}")
	}

	var entityTypeNull *string
	if entityType != "" {
		entityTypeNull = &entityType
	}

	_, err := w.db.ExecContext(ctx,
		`INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, entity_id, ip, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tenantID, actorNull, action, entityTypeNull, entityID, ip, meta,
	)
	if err != nil {
		w.logger.Error("audit: write failed", "action", action, "err", err)
	}
}
