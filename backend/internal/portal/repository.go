package portal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines all read-only persistence operations for the portal domain.
type Repository interface {
	// ListAssignedExams returns all active exams assigned to the given user (directly,
	// via department membership, or via 'all'), together with session data needed for
	// status computation.
	ListAssignedExams(ctx context.Context, userID, deptID string) ([]*portalExamRow, error)

	// GetAssignedExam fetches one active exam assigned to the given user.
	// Returns ErrNotAssigned if the exam exists but is not assigned.
	GetAssignedExam(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error)

	// ListUserSessions returns all sessions the given user has for the given exam.
	ListUserSessions(ctx context.Context, examID, userID string) ([]sessionRow, error)
}

// portalExamRow is the raw DB row for one assigned exam (exam data + earliest deadline).
type portalExamRow struct {
	ID                 string     `db:"id"`
	Title              string     `db:"title"`
	Description        *string    `db:"description"`
	TimeLimitMinutes   int        `db:"time_limit_minutes"`
	PassingScorePct    float64    `db:"passing_score_pct"`
	MaxAttempts        int        `db:"max_attempts"`
	ShuffleQuestions   bool       `db:"shuffle_questions"`
	ShuffleOptions     bool       `db:"shuffle_options"`
	ShowAnswers        string     `db:"show_answers"`
	CertificateEnabled bool       `db:"certificate_enabled"`
	AvailableFrom      *time.Time `db:"available_from"`
	AvailableUntil     *time.Time `db:"available_until"`
	Deadline           *time.Time `db:"deadline"`
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewRepository returns a Repository backed by PostgreSQL.
func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

// assignmentUnionCTE builds the CTE text that resolves all assignments for a user.
// It returns the CTE body (without the WITH keyword) together with args [userID, deptID, userID].
//
// The CTE emits (exam_id, deadline) for each matching assignment.
// When the same exam has multiple assignments the outer query picks the earliest deadline.
const assignmentCTE = `
WITH RECURSIVE dept_tree(id) AS (
    SELECT NULLIF($2, '')::uuid AS id
    UNION ALL
    SELECT d.parent_id FROM departments d JOIN dept_tree dt ON d.id = dt.id WHERE d.parent_id IS NOT NULL
),
user_assignments AS (
    SELECT ea.exam_id, ea.deadline FROM exam_assignments ea
    WHERE ea.assignee_type = 'user' AND ea.assignee_id = $1
    UNION ALL
    SELECT ea.exam_id, ea.deadline FROM exam_assignments ea
    WHERE ea.assignee_type = 'department' AND ea.assignee_id IN (SELECT id FROM dept_tree)
    UNION ALL
    SELECT ea.exam_id, ea.deadline FROM exam_assignments ea
    WHERE ea.assignee_type = 'all'
),
resolved AS (
    SELECT ua.exam_id, MIN(ua.deadline) AS deadline
    FROM user_assignments ua
    GROUP BY ua.exam_id
)`

func (r *postgresRepository) ListAssignedExams(ctx context.Context, userID, deptID string) ([]*portalExamRow, error) {
	q := assignmentCTE + `
SELECT
    e.id, e.title, e.description, e.time_limit_minutes, e.passing_score_pct,
    e.max_attempts, e.shuffle_questions, e.shuffle_options, e.show_answers,
    e.certificate_enabled, e.available_from, e.available_until, res.deadline
FROM resolved res
JOIN exams e ON e.id = res.exam_id
WHERE e.status = 'active'
ORDER BY e.title`

	rows, err := r.db.QueryxContext(ctx, q, userID, deptID)
	if err != nil {
		return nil, fmt.Errorf("portal: ListAssignedExams: %w", err)
	}
	defer rows.Close()

	var result []*portalExamRow
	for rows.Next() {
		var row portalExamRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("portal: ListAssignedExams: scan: %w", err)
		}
		result = append(result, &row)
	}
	if result == nil {
		result = []*portalExamRow{}
	}
	return result, rows.Err()
}

func (r *postgresRepository) GetAssignedExam(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error) {
	q := assignmentCTE + `
SELECT
    e.id, e.title, e.description, e.time_limit_minutes, e.passing_score_pct,
    e.max_attempts, e.shuffle_questions, e.shuffle_options, e.show_answers,
    e.certificate_enabled, e.available_from, e.available_until, res.deadline
FROM resolved res
JOIN exams e ON e.id = res.exam_id
WHERE e.status = 'active' AND e.id = $3`

	var row portalExamRow
	if err := r.db.GetContext(ctx, &row, q, userID, deptID, examID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotAssigned
		}
		return nil, fmt.Errorf("portal: GetAssignedExam: %w", err)
	}
	return &row, nil
}

// ListUserSessions returns all sessions for a user+exam. If exam_sessions does not
// exist yet (pre-FR-BB35) the function returns an empty slice rather than an error.
func (r *postgresRepository) ListUserSessions(ctx context.Context, examID, userID string) ([]sessionRow, error) {
	const q = `
SELECT id AS session_id, status, expires_at, passed, submitted_at, score_pct, started_at
FROM exam_sessions
WHERE exam_id = $1 AND user_id = $2
ORDER BY started_at`

	rows, err := r.db.QueryxContext(ctx, q, examID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "exam_sessions") && strings.Contains(err.Error(), "does not exist") {
			return []sessionRow{}, nil
		}
		return nil, fmt.Errorf("portal: ListUserSessions: %w", err)
	}
	defer rows.Close()

	type dbRow struct {
		SessionID   string     `db:"session_id"`
		Status      string     `db:"status"`
		ExpiresAt   *time.Time `db:"expires_at"`
		Passed      bool       `db:"passed"`
		SubmittedAt *time.Time `db:"submitted_at"`
		ScorePct    *float64   `db:"score_pct"`
		StartedAt   time.Time  `db:"started_at"`
	}

	var result []sessionRow
	for rows.Next() {
		var dbr dbRow
		if err := rows.StructScan(&dbr); err != nil {
			return nil, fmt.Errorf("portal: ListUserSessions: scan: %w", err)
		}
		result = append(result, sessionRow(dbr))
	}
	return result, rows.Err()
}
