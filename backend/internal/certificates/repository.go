package certificates

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository defines the persistence operations for the certificates domain.
type Repository interface {
	// GetSessionForCertificate fetches session+exam+user data needed for certificate validation.
	// Returns ErrNotFound if the session does not exist.
	GetSessionForCertificate(ctx context.Context, sessionID string) (*certSessionRow, error)

	// GetBySessionID returns an existing certificate for the given session.
	// Returns ErrNotFound if no certificate exists yet.
	GetBySessionID(ctx context.Context, sessionID string) (*Certificate, error)

	// Create inserts a new certificate using ON CONFLICT (session_id) DO NOTHING for idempotency,
	// then re-fetches the row (handles concurrent first-request races).
	Create(ctx context.Context, input certInsert) (*Certificate, error)

	// IssueLocked issues the certificate of a session in one transaction (FR-BB73 AC-20): FOR SHARE on the session
	// first, then the session, the existing certificate and the decision are read after the lock, then the insert.
	IssueLocked(ctx context.Context, sessionID string, decide IssueDecision) (*Certificate, error)

	// GetByVerificationCode returns the certificate with the given verification code.
	// Returns ErrNotFound if the code is unknown.
	GetByVerificationCode(ctx context.Context, code string) (*Certificate, error)
}

type pgRepository struct {
	db *sqlx.DB
}

// NewRepository creates a PostgreSQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository {
	return &pgRepository{db: db}
}

// certSessionRow holds the joined data needed to validate and issue a certificate.
type certSessionRow struct {
	SessionID          string   `db:"session_id"`
	UserID             string   `db:"user_id"`
	Status             string   `db:"status"`
	Passed             bool     `db:"passed"`
	ScorePct           *float64 `db:"score_pct"`
	CertificateEnabled bool     `db:"certificate_enabled"`
	ExamTitle          string   `db:"exam_title"`
	EmployeeName       string   `db:"employee_name"`
}

// IssueDecision is the service's rule for one issuance. It sees the session and the existing certificate, both read
// under the session lock. It returns the insert, or nil when the certificate already exists (then existing is returned).
type IssueDecision func(row certSessionRow, existing *Certificate) (*certInsert, error)

// certInsert carries the data for a new certificate row.
type certInsert struct {
	SessionID        string
	EmployeeName     string
	ExamTitle        string
	ScorePct         float64
	TemplateSnapshot TemplateSnapshot
}

func (r *pgRepository) GetSessionForCertificate(ctx context.Context, sessionID string) (*certSessionRow, error) {
	const q = `
		SELECT
			s.id            AS session_id,
			s.user_id       AS user_id,
			s.status::text  AS status,
			s.passed        AS passed,
			s.score_pct     AS score_pct,
			e.certificate_enabled AS certificate_enabled,
			e.title         AS exam_title,
			u.full_name     AS employee_name
		FROM exam_sessions s
		JOIN exams e ON e.id = s.exam_id
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1`

	var row certSessionRow
	if err := r.db.GetContext(ctx, &row, q, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("certificates: GetSessionForCertificate: %w", err)
	}
	return &row, nil
}

func (r *pgRepository) GetBySessionID(ctx context.Context, sessionID string) (*Certificate, error) {
	const q = `
		SELECT id, session_id, verification_code, issued_at,
		       employee_name, exam_title, score_pct, template_snapshot
		FROM certificates
		WHERE session_id = $1`

	var cert Certificate
	if err := r.db.GetContext(ctx, &cert, q, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("certificates: GetBySessionID: %w", err)
	}
	return &cert, nil
}

func (r *pgRepository) Create(ctx context.Context, input certInsert) (*Certificate, error) {
	snapshotJSON, err := json.Marshal(input.TemplateSnapshot)
	if err != nil {
		return nil, fmt.Errorf("certificates: Create: marshal snapshot: %w", err)
	}

	const q = `
		INSERT INTO certificates (session_id, employee_name, exam_title, score_pct, template_snapshot)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id) DO NOTHING`

	if _, err := r.db.ExecContext(ctx, q,
		input.SessionID,
		input.EmployeeName,
		input.ExamTitle,
		input.ScorePct,
		snapshotJSON,
	); err != nil {
		return nil, fmt.Errorf("certificates: Create: insert: %w", err)
	}

	// Re-fetch after insert (handles both first-insert and concurrent conflict).
	cert, err := r.GetBySessionID(ctx, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("certificates: Create: re-fetch: %w", err)
	}
	return cert, nil
}

// IssueLocked issues the certificate of a session in one transaction (FR-BB73 AC-20). The first statement takes
// FOR SHARE on the session, so the status and passed are read after the session lock. An override of the same session
// (GradeAnswer, FOR UPDATE on the session) therefore sees either the certificate, or the certificate's decision
// happens after the override has set passed. The insert is ON CONFLICT (session_id) DO NOTHING.
func (r *pgRepository) IssueLocked(ctx context.Context, sessionID string, decide IssueDecision) (*Certificate, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("certificates: IssueLocked: begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const lockQ = `
		SELECT
			s.id            AS session_id,
			s.user_id       AS user_id,
			s.status::text  AS status,
			s.passed        AS passed,
			s.score_pct     AS score_pct,
			e.certificate_enabled AS certificate_enabled,
			e.title         AS exam_title,
			u.full_name     AS employee_name
		FROM exam_sessions s
		JOIN exams e ON e.id = s.exam_id
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1
		FOR SHARE OF s`
	var row certSessionRow
	if err := tx.GetContext(ctx, &row, lockQ, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("certificates: IssueLocked: lock session: %w", err)
	}

	var existing *Certificate
	const existingQ = `
		SELECT id, session_id, verification_code, issued_at,
		       employee_name, exam_title, score_pct, template_snapshot
		FROM certificates
		WHERE session_id = $1`
	var found Certificate
	switch err := tx.GetContext(ctx, &found, existingQ, sessionID); {
	case err == nil:
		existing = &found
	case errors.Is(err, sql.ErrNoRows):
	default:
		return nil, fmt.Errorf("certificates: IssueLocked: existing: %w", err)
	}

	ins, err := decide(row, existing)
	if err != nil {
		return nil, err
	}
	if ins == nil {
		if existing == nil {
			return nil, fmt.Errorf("certificates: IssueLocked: no decision for session %s", sessionID)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("certificates: IssueLocked: commit: %w", err)
		}
		return existing, nil
	}

	snapshotJSON, err := json.Marshal(ins.TemplateSnapshot)
	if err != nil {
		return nil, fmt.Errorf("certificates: IssueLocked: marshal snapshot: %w", err)
	}
	const insertQ = `
		INSERT INTO certificates (session_id, employee_name, exam_title, score_pct, template_snapshot)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id) DO NOTHING`
	if _, err := tx.ExecContext(ctx, insertQ, ins.SessionID, ins.EmployeeName, ins.ExamTitle, ins.ScorePct, snapshotJSON); err != nil {
		return nil, fmt.Errorf("certificates: IssueLocked: insert: %w", err)
	}
	var cert Certificate
	if err := tx.GetContext(ctx, &cert, existingQ, sessionID); err != nil {
		return nil, fmt.Errorf("certificates: IssueLocked: re-fetch: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("certificates: IssueLocked: commit: %w", err)
	}
	return &cert, nil
}

func (r *pgRepository) GetByVerificationCode(ctx context.Context, code string) (*Certificate, error) {
	const q = `
		SELECT id, session_id, verification_code, issued_at,
		       employee_name, exam_title, score_pct, template_snapshot
		FROM certificates
		WHERE verification_code = $1`

	var cert Certificate
	if err := r.db.GetContext(ctx, &cert, q, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("certificates: GetByVerificationCode: %w", err)
	}
	return &cert, nil
}
