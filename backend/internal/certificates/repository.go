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
