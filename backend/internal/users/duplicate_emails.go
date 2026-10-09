package users

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Names tied to migration 035 (ISS-181).
const (
	// EmailUniqueIndexName is the case-insensitive unique index created by migration 035
	// only when the data had no case-insensitive duplicate emails.
	EmailUniqueIndexName = "idx_users_email_lower_unique"
	// DuplicateEmailsAuditAction is the audit_log action written when twins are detected.
	DuplicateEmailsAuditAction = "users.duplicate_emails_detected"

	// maxReportedGroups caps how many duplicate groups are listed in the WARN log.
	maxReportedGroups = 50
	// maxAuditSampleGroups caps how many groups are copied into audit metadata.
	maxAuditSampleGroups = 3
)

// DuplicateEmailStore is the persistence the startup duplicate-email check needs.
type DuplicateEmailStore interface {
	// EmailUniqueIndexExists reports whether the lower(email) unique index is present.
	EmailUniqueIndexExists(ctx context.Context) (bool, error)
	// DuplicateEmailGroups returns the total number of case-insensitive duplicate groups and
	// up to limit of them, each group being the stored emails that differ only by case.
	DuplicateEmailGroups(ctx context.Context, limit int) (total int, groups [][]string, err error)
	// RecordDuplicateEmails writes one system audit_log entry with the given metadata.
	RecordDuplicateEmails(ctx context.Context, metadata map[string]any) error
}

// CheckDuplicateEmails is the ISS-181 admin report. When migration 035 skipped the unique
// index because of legacy case-insensitive duplicate emails, it logs ONE structured WARN
// listing the twins (capped) and writes ONE audit_log entry. It never returns an error and
// never panics: startup must not be aborted by a reporting problem; failures are logged.
func CheckDuplicateEmails(ctx context.Context, store DuplicateEmailStore, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Error("duplicate email check panicked; continuing startup", "panic", fmt.Sprint(r))
		}
	}()

	exists, err := store.EmailUniqueIndexExists(ctx)
	if err != nil {
		logger.Error("duplicate email check: index lookup failed; continuing startup", "err", err)
		return
	}
	if exists {
		return
	}

	total, groups, err := store.DuplicateEmailGroups(ctx, maxReportedGroups)
	if err != nil {
		logger.Error("duplicate email check: query failed; continuing startup", "err", err)
		return
	}
	if total == 0 {
		return
	}

	more := total - len(groups)
	if more < 0 {
		more = 0
	}
	logger.Warn("users with emails differing only by case exist; unique index "+EmailUniqueIndexName+
		" was not created (migration 035). Resolve manually (no automatic merge), then run: "+
		"CREATE UNIQUE INDEX "+EmailUniqueIndexName+" ON users (lower(email))",
		"duplicate_groups", total,
		"listed_groups", len(groups),
		"more_groups_not_listed", more,
		"groups", groups,
	)

	sample := groups
	if len(sample) > maxAuditSampleGroups {
		sample = sample[:maxAuditSampleGroups]
	}
	meta := map[string]any{
		"duplicate_groups": total,
		"index":            EmailUniqueIndexName,
		"sample":           sample,
	}
	if err := store.RecordDuplicateEmails(ctx, meta); err != nil {
		logger.Error("duplicate email check: audit write failed; continuing startup", "err", err)
	}
}

type pgDuplicateEmailStore struct{ db *sqlx.DB }

// NewDuplicateEmailStore returns the Postgres-backed DuplicateEmailStore.
func NewDuplicateEmailStore(db *sqlx.DB) DuplicateEmailStore { return &pgDuplicateEmailStore{db: db} }

func (s *pgDuplicateEmailStore) EmailUniqueIndexExists(ctx context.Context) (bool, error) {
	var ok bool
	err := s.db.GetContext(ctx, &ok,
		`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND tablename = 'users' AND indexname = $1)`, EmailUniqueIndexName)
	if err != nil {
		return false, fmt.Errorf("users.EmailUniqueIndexExists: %w", err)
	}
	return ok, nil
}

func (s *pgDuplicateEmailStore) DuplicateEmailGroups(ctx context.Context, limit int) (int, [][]string, error) {
	var total int
	if err := s.db.GetContext(ctx, &total,
		`SELECT count(*) FROM (SELECT 1 FROM users GROUP BY lower(email) HAVING count(*) > 1) d`); err != nil {
		return 0, nil, fmt.Errorf("users.DuplicateEmailGroups: count: %w", err)
	}
	if total == 0 {
		return 0, nil, nil
	}
	rows, err := s.db.QueryxContext(ctx,
		`SELECT array_agg(email ORDER BY email) FROM users GROUP BY lower(email) HAVING count(*) > 1 ORDER BY lower(email) LIMIT $1`, limit)
	if err != nil {
		return 0, nil, fmt.Errorf("users.DuplicateEmailGroups: list: %w", err)
	}
	defer rows.Close()
	var groups [][]string
	for rows.Next() {
		var g []string
		if err := rows.Scan(pq.Array(&g)); err != nil {
			return 0, nil, fmt.Errorf("users.DuplicateEmailGroups: scan: %w", err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("users.DuplicateEmailGroups: rows: %w", err)
	}
	return total, groups, nil
}

func (s *pgDuplicateEmailStore) RecordDuplicateEmails(ctx context.Context, metadata map[string]any) error {
	meta, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("users.RecordDuplicateEmails: marshal: %w", err)
	}
	// Same system-actor convention as the session auto-submit job (tenant 'public', no actor).
	const q = `
INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, entity_id, ip, metadata)
VALUES ('public', NULL, $1, 'user', NULL, '', $2::jsonb)`
	if _, err := s.db.ExecContext(ctx, q, DuplicateEmailsAuditAction, string(meta)); err != nil {
		return fmt.Errorf("users.RecordDuplicateEmails: %w", err)
	}
	return nil
}

// duplicateCheckTimeout bounds the startup check.
const duplicateCheckTimeout = 15 * time.Second

// RunStartupDuplicateEmailCheck runs CheckDuplicateEmails against the database with a bounded timeout.
func RunStartupDuplicateEmailCheck(ctx context.Context, db *sqlx.DB, logger *slog.Logger) {
	cctx, cancel := context.WithTimeout(ctx, duplicateCheckTimeout)
	defer cancel()
	CheckDuplicateEmails(cctx, NewDuplicateEmailStore(db), logger)
}
