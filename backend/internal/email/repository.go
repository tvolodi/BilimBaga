package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// UserEmailData is the minimal user data needed to send an email.
type UserEmailData struct {
	Email  string
	Locale string
}

// ReminderRow represents one assignment that needs a deadline reminder.
type ReminderRow struct {
	UserID   string
	Deadline time.Time
	Email    string
	Locale   string
	Title    string
}

// SessionEmailData contains all the information needed to send a session result email.
type SessionEmailData struct {
	UserEmail       string
	UserLocale      string
	ExamTitle       string
	ScorePct        float64
	PassingScorePct float64
	MaxAttempts     int
	AttemptNumber   int
	Passed          bool
	SessionID       string
}

// Repository defines the data access contract for the email package.
type Repository interface {
	LogAttempt(ctx context.Context, to, tmpl string, sendErr error) error
	GetTenantDefaultLocale(ctx context.Context) (string, error)
	FetchDeadlineReminderTargets(ctx context.Context, from, to time.Time) ([]ReminderRow, error)
	GetUserForEmail(ctx context.Context, userID string) (UserEmailData, error)
	GetDepartmentUsersForEmail(ctx context.Context, deptID string) ([]UserEmailData, error)
	GetAllActiveUsersForEmail(ctx context.Context) ([]UserEmailData, error)
	GetSessionEmailData(ctx context.Context, sessionID string) (*SessionEmailData, error)
}

// ── Postgres implementation ───────────────────────────────────────────────────

type postgresRepository struct {
	db *sqlx.DB
}

func newPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

// internal scan struct for user rows
type userEmailRow struct {
	Email  string `db:"email"`
	Locale string `db:"preferred_locale"`
}

// internal scan struct for session result rows
type sessionEmailRow struct {
	UserEmail       string  `db:"user_email"`
	UserLocale      string  `db:"user_locale"`
	ExamTitle       string  `db:"exam_title"`
	ScorePct        float64 `db:"score_pct"`
	PassingScorePct float64 `db:"passing_score_pct"`
	MaxAttempts     int     `db:"max_attempts"`
	AttemptNumber   int     `db:"attempt_number"`
	Passed          bool    `db:"passed"`
	SessionID       string  `db:"session_id"`
}

// internal scan struct for reminder rows
type reminderScanRow struct {
	UserID   string    `db:"user_id"`
	Deadline time.Time `db:"deadline"`
	Email    string    `db:"email"`
	Locale   string    `db:"preferred_locale"`
	Title    string    `db:"title"`
}

func (r *postgresRepository) LogAttempt(ctx context.Context, to, tmpl string, sendErr error) error {
	var errStr *string
	if sendErr != nil {
		s := sendErr.Error()
		errStr = &s
	}
	const q = `INSERT INTO email_log (recipient_email, template, error) VALUES ($1, $2, $3)`
	if _, err := r.db.ExecContext(ctx, q, to, tmpl, errStr); err != nil {
		return fmt.Errorf("email: log attempt: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetTenantDefaultLocale(ctx context.Context) (string, error) {
	const q = `SELECT value#>>'{}' FROM tenant_config WHERE key = 'default_locale' LIMIT 1`
	var locale string
	if err := r.db.GetContext(ctx, &locale, q); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "en", nil
		}
		return "", fmt.Errorf("email: get tenant locale: %w", err)
	}
	return locale, nil
}

func (r *postgresRepository) FetchDeadlineReminderTargets(ctx context.Context, from, to time.Time) ([]ReminderRow, error) {
	const q = `
		WITH assigned_users AS (
			SELECT ea.exam_id, ea.assignee_id AS user_id, ea.deadline
			FROM exam_assignments ea
			WHERE ea.assignee_type = 'user' AND ea.deadline BETWEEN $1 AND $2
			UNION
			SELECT ea.exam_id, u.id AS user_id, ea.deadline
			FROM exam_assignments ea
			JOIN users u ON u.department_id = ea.assignee_id
			WHERE ea.assignee_type = 'department' AND ea.deadline BETWEEN $1 AND $2
			UNION
			SELECT ea.exam_id, u.id AS user_id, ea.deadline
			FROM exam_assignments ea
			CROSS JOIN users u
			WHERE ea.assignee_type = 'all' AND ea.deadline BETWEEN $1 AND $2
		)
		SELECT
			au.user_id,
			au.deadline,
			u.email,
			COALESCE(u.preferred_locale, '') AS preferred_locale,
			e.title
		FROM assigned_users au
		JOIN users u ON u.id = au.user_id
		JOIN exams e ON e.id = au.exam_id
		WHERE NOT EXISTS (
			SELECT 1 FROM exam_sessions es
			WHERE es.exam_id = au.exam_id
			  AND es.user_id = au.user_id
			  AND es.passed = TRUE
		)
	`
	var rows []reminderScanRow
	if err := r.db.SelectContext(ctx, &rows, q, from, to); err != nil {
		return nil, fmt.Errorf("email: fetch deadline reminder targets: %w", err)
	}
	result := make([]ReminderRow, len(rows))
	for i, rr := range rows {
		result[i] = ReminderRow{
			UserID:   rr.UserID,
			Deadline: rr.Deadline,
			Email:    rr.Email,
			Locale:   rr.Locale,
			Title:    rr.Title,
		}
	}
	return result, nil
}

func (r *postgresRepository) GetUserForEmail(ctx context.Context, userID string) (UserEmailData, error) {
	const q = `SELECT email, COALESCE(preferred_locale, '') AS preferred_locale FROM users WHERE id = $1`
	var row userEmailRow
	if err := r.db.GetContext(ctx, &row, q, userID); err != nil {
		return UserEmailData{}, fmt.Errorf("email: get user %s: %w", userID, err)
	}
	return UserEmailData{Email: row.Email, Locale: row.Locale}, nil
}

func (r *postgresRepository) GetDepartmentUsersForEmail(ctx context.Context, deptID string) ([]UserEmailData, error) {
	const q = `
		SELECT email, COALESCE(preferred_locale, '') AS preferred_locale
		FROM users
		WHERE department_id = $1 AND status = 'active'
	`
	var rows []userEmailRow
	if err := r.db.SelectContext(ctx, &rows, q, deptID); err != nil {
		return nil, fmt.Errorf("email: get department users %s: %w", deptID, err)
	}
	result := make([]UserEmailData, len(rows))
	for i, rr := range rows {
		result[i] = UserEmailData{Email: rr.Email, Locale: rr.Locale}
	}
	return result, nil
}

func (r *postgresRepository) GetAllActiveUsersForEmail(ctx context.Context) ([]UserEmailData, error) {
	const q = `SELECT email, COALESCE(preferred_locale, '') AS preferred_locale FROM users WHERE status = 'active'`
	var rows []userEmailRow
	if err := r.db.SelectContext(ctx, &rows, q); err != nil {
		return nil, fmt.Errorf("email: get all active users: %w", err)
	}
	result := make([]UserEmailData, len(rows))
	for i, rr := range rows {
		result[i] = UserEmailData{Email: rr.Email, Locale: rr.Locale}
	}
	return result, nil
}

func (r *postgresRepository) GetSessionEmailData(ctx context.Context, sessionID string) (*SessionEmailData, error) {
	const q = `
		SELECT
			u.email                                                         AS user_email,
			COALESCE(u.preferred_locale, '')                                AS user_locale,
			e.title                                                         AS exam_title,
			COALESCE(es.score_pct, 0)                                       AS score_pct,
			e.passing_score_pct,
			COALESCE(e.max_attempts, 0)                                     AS max_attempts,
			es.passed,
			es.id                                                           AS session_id,
			(SELECT COUNT(*) FROM exam_sessions es2
			 WHERE es2.exam_id = es.exam_id
			   AND es2.user_id = es.user_id
			   AND es2.started_at <= es.started_at)::int                    AS attempt_number
		FROM exam_sessions es
		JOIN users u ON u.id = es.user_id
		JOIN exams e ON e.id = es.exam_id
		WHERE es.id = $1
	`
	var row sessionEmailRow
	if err := r.db.GetContext(ctx, &row, q, sessionID); err != nil {
		return nil, fmt.Errorf("email: get session data %s: %w", sessionID, err)
	}
	return &SessionEmailData{
		UserEmail:       row.UserEmail,
		UserLocale:      row.UserLocale,
		ExamTitle:       row.ExamTitle,
		ScorePct:        row.ScorePct,
		PassingScorePct: row.PassingScorePct,
		MaxAttempts:     row.MaxAttempts,
		AttemptNumber:   row.AttemptNumber,
		Passed:          row.Passed,
		SessionID:       row.SessionID,
	}, nil
}
