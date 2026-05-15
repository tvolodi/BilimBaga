package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines all read-only persistence operations for the reports domain.
type Repository interface {
	// GetCompletionRateByExam returns per-exam assignment/completion/pass counts
	// for every active exam (AC-2).
	GetCompletionRateByExam(ctx context.Context) ([]*ExamCompletionRate, error)

	// GetOverdueEmployees returns at most 20 employees with overdue, unpassed
	// assignments ordered by deadline ASC (AC-3).
	GetOverdueEmployees(ctx context.Context) ([]*OverdueEmployee, error)

	// GetRecentActivity returns the last 20 submitted/grading_pending sessions
	// ordered by submitted_at DESC (AC-4).
	GetRecentActivity(ctx context.Context) ([]*RecentActivity, error)

	// GetAvgScoreByTrack returns average session scores per track over the last
	// 90 days. Tracks with no data are absent from the returned map; the service
	// layer converts the map to the TrackScores struct with explicit nulls (AC-5/AC-6).
	GetAvgScoreByTrack(ctx context.Context) (map[string]*float64, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewRepository returns a Repository backed by PostgreSQL.
func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

// ── Completion rate ───────────────────────────────────────────────────────────

// completionRateRow maps directly to the SQL result row.
type completionRateRow struct {
	ExamID         string `db:"exam_id"`
	Title          string `db:"title"`
	AssignedCount  int    `db:"assigned_count"`
	CompletedCount int    `db:"completed_count"`
	PassedCount    int    `db:"passed_count"`
}

// GetCompletionRateByExam resolves per-exam counts by expanding every
// exam_assignments row into individual user rows before counting.  Direct-user
// assignments contribute the assigned user; department assignments contribute
// every user whose department is in the department subtree; 'all' assignments
// contribute every active user in the system.
//
// Using a CTE that resolves assignments to individual users lets the outer
// query run a single GROUP BY without correlated sub-queries.
func (r *postgresRepository) GetCompletionRateByExam(ctx context.Context) ([]*ExamCompletionRate, error) {
	const q = `
WITH RECURSIVE dept_tree(id, root_id) AS (
    SELECT d.id, d.id AS root_id FROM departments d
    UNION ALL
    SELECT d.id, dt.root_id FROM departments d JOIN dept_tree dt ON d.parent_id = dt.id
),
all_users AS (
    SELECT u.id AS user_id, u.department_id FROM users u WHERE u.status = 'active'
),
-- Expand each assignment row into (exam_id, user_id) pairs.
resolved_assignments AS (
    -- Direct user assignments.
    SELECT ea.exam_id, ea.assignee_id AS user_id
    FROM exam_assignments ea
    WHERE ea.assignee_type = 'user'

    UNION

    -- Department assignments: every user whose dept is in the subtree rooted at assignee_id.
    SELECT ea.exam_id, au.user_id
    FROM exam_assignments ea
    JOIN dept_tree dt ON dt.root_id = ea.assignee_id
    JOIN all_users au ON au.department_id = dt.id
    WHERE ea.assignee_type = 'department'

    UNION

    -- 'all' assignments: every active user.
    SELECT ea.exam_id, au.user_id
    FROM exam_assignments ea
    CROSS JOIN all_users au
    WHERE ea.assignee_type = 'all'
)
SELECT
    e.id                                                                         AS exam_id,
    e.title,
    COUNT(DISTINCT ra.user_id)                                                   AS assigned_count,
    COUNT(DISTINCT CASE WHEN es.status IN ('submitted','grading_pending') THEN ra.user_id END)
                                                                                 AS completed_count,
    COUNT(DISTINCT CASE WHEN es.passed = TRUE THEN ra.user_id END)               AS passed_count
FROM exams e
JOIN resolved_assignments ra ON ra.exam_id = e.id
LEFT JOIN exam_sessions es ON es.exam_id = e.id AND es.user_id = ra.user_id
WHERE e.status = 'active'
GROUP BY e.id, e.title
ORDER BY e.title`

	rows, err := r.db.QueryxContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("reports: GetCompletionRateByExam: %w", err)
	}
	defer rows.Close()

	var result []*ExamCompletionRate
	for rows.Next() {
		var row completionRateRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("reports: GetCompletionRateByExam: scan: %w", err)
		}
		result = append(result, &ExamCompletionRate{
			ExamID:         row.ExamID,
			Title:          row.Title,
			AssignedCount:  row.AssignedCount,
			CompletedCount: row.CompletedCount,
			PassedCount:    row.PassedCount,
		})
	}
	if result == nil {
		result = []*ExamCompletionRate{}
	}
	return result, rows.Err()
}

// ── Overdue employees ─────────────────────────────────────────────────────────

// overdueRow maps directly to the SQL result row.
type overdueRow struct {
	UserID    string    `db:"user_id"`
	Name      string    `db:"name"`
	ExamTitle string    `db:"exam_title"`
	Deadline  time.Time `db:"deadline"`
}

// GetOverdueEmployees returns users who have a user-direct assignment with a past
// deadline and no passing session.  Department and 'all' assignments may also be
// overdue, but resolving them requires expanding to individual users (done in a CTE
// identical to GetCompletionRateByExam).  We limit to 20 rows ordered by deadline ASC
// so the most-overdue appear first (AC-3).
func (r *postgresRepository) GetOverdueEmployees(ctx context.Context) ([]*OverdueEmployee, error) {
	const q = `
WITH RECURSIVE dept_tree(id, root_id) AS (
    SELECT d.id, d.id AS root_id FROM departments d
    UNION ALL
    SELECT d.id, dt.root_id FROM departments d JOIN dept_tree dt ON d.parent_id = dt.id
),
all_users AS (
    SELECT u.id AS user_id, u.department_id FROM users u WHERE u.status = 'active'
),
resolved_assignments AS (
    SELECT ea.exam_id, ea.assignee_id AS user_id, ea.deadline
    FROM exam_assignments ea
    WHERE ea.assignee_type = 'user'

    UNION

    SELECT ea.exam_id, au.user_id, ea.deadline
    FROM exam_assignments ea
    JOIN dept_tree dt ON dt.root_id = ea.assignee_id
    JOIN all_users au ON au.department_id = dt.id
    WHERE ea.assignee_type = 'department'

    UNION

    SELECT ea.exam_id, au.user_id, ea.deadline
    FROM exam_assignments ea
    CROSS JOIN all_users au
    WHERE ea.assignee_type = 'all'
)
SELECT DISTINCT ON (ra.user_id, ra.exam_id)
    u.id                 AS user_id,
    u.full_name          AS name,
    e.title              AS exam_title,
    ra.deadline
FROM resolved_assignments ra
JOIN users u ON u.id = ra.user_id
JOIN exams e ON e.id = ra.exam_id
WHERE ra.deadline IS NOT NULL
  AND ra.deadline < NOW()
  AND NOT EXISTS (
      SELECT 1 FROM exam_sessions es
      WHERE es.exam_id = ra.exam_id
        AND es.user_id = ra.user_id
        AND es.passed = TRUE
  )
ORDER BY ra.user_id, ra.exam_id, ra.deadline ASC
LIMIT 20`

	rows, err := r.db.QueryxContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("reports: GetOverdueEmployees: %w", err)
	}
	defer rows.Close()

	var result []*OverdueEmployee
	for rows.Next() {
		var row overdueRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("reports: GetOverdueEmployees: scan: %w", err)
		}
		result = append(result, &OverdueEmployee{
			UserID:    row.UserID,
			Name:      row.Name,
			ExamTitle: row.ExamTitle,
			Deadline:  row.Deadline,
		})
	}
	if result == nil {
		result = []*OverdueEmployee{}
	}
	return result, rows.Err()
}

// ── Recent activity ───────────────────────────────────────────────────────────

// recentActivityRow maps directly to the SQL result row.
type recentActivityRow struct {
	SessionID    string    `db:"session_id"`
	EmployeeName string    `db:"employee_name"`
	ExamTitle    string    `db:"exam_title"`
	ScorePct     *float64  `db:"score_pct"`
	Passed       bool      `db:"passed"`
	SubmittedAt  time.Time `db:"submitted_at"`
}

// GetRecentActivity returns the 20 most recently completed sessions (AC-4).
func (r *postgresRepository) GetRecentActivity(ctx context.Context) ([]*RecentActivity, error) {
	const q = `
SELECT
    es.id           AS session_id,
    u.full_name     AS employee_name,
    e.title         AS exam_title,
    es.score_pct,
    es.passed,
    es.submitted_at
FROM exam_sessions es
JOIN users u ON u.id = es.user_id
JOIN exams e ON e.id = es.exam_id
WHERE es.status IN ('submitted', 'grading_pending')
ORDER BY es.submitted_at DESC
LIMIT 20`

	rows, err := r.db.QueryxContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("reports: GetRecentActivity: %w", err)
	}
	defer rows.Close()

	var result []*RecentActivity
	for rows.Next() {
		var row recentActivityRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("reports: GetRecentActivity: scan: %w", err)
		}
		result = append(result, &RecentActivity{
			SessionID:    row.SessionID,
			EmployeeName: row.EmployeeName,
			ExamTitle:    row.ExamTitle,
			ScorePct:     row.ScorePct,
			Passed:       row.Passed,
			SubmittedAt:  row.SubmittedAt,
		})
	}
	if result == nil {
		result = []*RecentActivity{}
	}
	return result, rows.Err()
}

// ── Average score by track ────────────────────────────────────────────────────

// trackScoreRow maps one row from the avg-by-track query.
type trackScoreRow struct {
	Track    string  `db:"track"`
	AvgScore float64 `db:"avg_score"`
}

// GetAvgScoreByTrack returns a map of track → average score for sessions submitted
// within the last 90 days (AC-5/AC-6).  Tracks absent from the result had no data.
func (r *postgresRepository) GetAvgScoreByTrack(ctx context.Context) (map[string]*float64, error) {
	const q = `
SELECT
    c.track,
    ROUND(AVG(es.score_pct)::numeric, 1) AS avg_score
FROM exam_sessions es
JOIN session_question_scores sqs ON sqs.session_id = es.id
JOIN questions q                  ON q.id = sqs.question_id
JOIN categories c                 ON c.id = q.category_id
WHERE es.status = 'submitted'
  AND es.submitted_at >= NOW() - INTERVAL '90 days'
  AND c.track IN ('security', 'safety', 'loyalty')
GROUP BY c.track`

	rows, err := r.db.QueryxContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("reports: GetAvgScoreByTrack: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*float64)
	for rows.Next() {
		var row trackScoreRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("reports: GetAvgScoreByTrack: scan: %w", err)
		}
		v := row.AvgScore
		result[row.Track] = &v
	}
	return result, rows.Err()
}
