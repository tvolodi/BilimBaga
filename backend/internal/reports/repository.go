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

	// ── FR-BB52: Per-Exam Analytics ──────────────────────────────────────────

	// GetExamTitle returns the title for the given exam ID, or ErrNotFound if
	// no exam with that ID exists.
	GetExamTitle(ctx context.Context, examID string) (string, error)

	// GetExamScoreDistribution returns raw bucket counts for completed sessions
	// of an exam.  Buckets with zero sessions are absent; the service layer fills
	// them in.
	GetExamScoreDistribution(ctx context.Context, examID string) ([]BucketCount, error)

	// GetExamSummaryStats returns aggregate statistics for completed sessions of
	// an exam.  All numeric fields may be nil when no sessions exist.
	GetExamSummaryStats(ctx context.Context, examID string) (*examSummaryRow, error)

	// GetPerQuestionStats returns per-question analytics rows for all distinct
	// questions that appeared in completed sessions of an exam.
	GetPerQuestionStats(ctx context.Context, examID string) ([]questionStatRow, error)

	// GetAnswerDistribution returns option selection counts for every answer
	// option across all completed sessions for an exam.
	GetAnswerDistribution(ctx context.Context, examID string) ([]answerDistRow, error)
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

// ── FR-BB52: Per-Exam Analytics ──────────────────────────────────────────────

// examSummaryRow is the internal scan target for summary stats.
type examSummaryRow struct {
	AvgScore           *float64 `db:"avg_score"`
	MedianScore        *float64 `db:"median_score"`
	TotalAttempts      int      `db:"total_attempts"`
	UniqueParticipants int      `db:"unique_participants"`
	PassRate           *float64 `db:"pass_rate"`
}

// questionStatRow is the internal scan target for per-question analytics.
type questionStatRow struct {
	QuestionID     string   `db:"question_id"`
	StemPreview    string   `db:"stem_preview"`
	CorrectRate    *float64 `db:"correct_rate"`
	AvgTimeSeconds *float64 `db:"avg_time_seconds"`
}

// answerDistRow is the internal scan target for answer distribution.
type answerDistRow struct {
	QuestionID  string `db:"question_id"`
	OptionID    string `db:"option_id"`
	OptionText  string `db:"option_text"`
	SelectCount int    `db:"select_count"`
}

// GetExamTitle returns the exam title or ErrNotFound.
func (r *postgresRepository) GetExamTitle(ctx context.Context, examID string) (string, error) {
	var title string
	err := r.db.QueryRowContext(ctx, `SELECT title FROM exams WHERE id = $1`, examID).Scan(&title)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", ErrNotFound
		}
		// Check for standard database/sql package sentinel using string match since
		// we cannot import database/sql here without a cycle concern.
		return "", fmt.Errorf("reports: GetExamTitle: %w", err)
	}
	return title, nil
}

// GetExamScoreDistribution returns raw (non-zero) bucket counts.
func (r *postgresRepository) GetExamScoreDistribution(ctx context.Context, examID string) ([]BucketCount, error) {
	const q = `
SELECT
  CASE
    WHEN score_pct >= 90 THEN '90-100'
    WHEN score_pct >= 80 THEN '80-90'
    WHEN score_pct >= 70 THEN '70-80'
    WHEN score_pct >= 60 THEN '60-70'
    WHEN score_pct >= 50 THEN '50-60'
    WHEN score_pct >= 40 THEN '40-50'
    WHEN score_pct >= 30 THEN '30-40'
    WHEN score_pct >= 20 THEN '20-30'
    WHEN score_pct >= 10 THEN '10-20'
    ELSE '0-10'
  END AS bucket,
  COUNT(*) AS count
FROM exam_sessions
WHERE exam_id = $1
  AND status IN ('submitted','auto_submitted','grading_pending')
GROUP BY bucket`

	type bucketRow struct {
		Bucket string `db:"bucket"`
		Count  int    `db:"count"`
	}

	rows, err := r.db.QueryxContext(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetExamScoreDistribution: %w", err)
	}
	defer rows.Close()

	var result []BucketCount
	for rows.Next() {
		var row bucketRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("reports: GetExamScoreDistribution: scan: %w", err)
		}
		result = append(result, BucketCount{Bucket: row.Bucket, Count: row.Count})
	}
	return result, rows.Err()
}

// GetExamSummaryStats returns aggregate stats for completed sessions.
func (r *postgresRepository) GetExamSummaryStats(ctx context.Context, examID string) (*examSummaryRow, error) {
	const q = `
SELECT
  ROUND(AVG(score_pct)::numeric, 1)                                           AS avg_score,
  PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY score_pct)                      AS median_score,
  COUNT(*)                                                                     AS total_attempts,
  COUNT(DISTINCT user_id)                                                      AS unique_participants,
  ROUND(
    COUNT(*) FILTER (WHERE passed = TRUE)::DECIMAL / NULLIF(COUNT(*), 0), 4
  )                                                                            AS pass_rate
FROM exam_sessions
WHERE exam_id = $1
  AND status IN ('submitted','auto_submitted','grading_pending')`

	var row examSummaryRow
	if err := r.db.QueryRowxContext(ctx, q, examID).StructScan(&row); err != nil {
		return nil, fmt.Errorf("reports: GetExamSummaryStats: %w", err)
	}
	return &row, nil
}

// GetPerQuestionStats returns per-question analytics for all questions that
// appeared in at least one completed session.
func (r *postgresRepository) GetPerQuestionStats(ctx context.Context, examID string) ([]questionStatRow, error) {
	const q = `
SELECT
  q.id                                                                AS question_id,
  LEFT(qt.stem, 100)                                                  AS stem_preview,
  ROUND(
    COUNT(sqs.question_id) FILTER (WHERE sqs.score = sqs.max_score)::DECIMAL
    / NULLIF(COUNT(sqs.question_id), 0), 4
  )                                                                   AS correct_rate,
  ROUND(AVG(sqs.time_taken_seconds)::numeric, 1)                     AS avg_time_seconds
FROM questions q
JOIN question_translations qt ON qt.question_id = q.id AND qt.locale = q.default_locale
LEFT JOIN session_question_scores sqs ON sqs.question_id = q.id
  AND sqs.session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND status IN ('submitted','auto_submitted','grading_pending')
  )
WHERE q.id IN (
  SELECT DISTINCT question_id FROM session_questions
  WHERE session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND status IN ('submitted','auto_submitted','grading_pending')
  )
)
GROUP BY q.id, qt.stem
ORDER BY q.id`

	rows, err := r.db.QueryxContext(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetPerQuestionStats: %w", err)
	}
	defer rows.Close()

	var result []questionStatRow
	for rows.Next() {
		var row questionStatRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("reports: GetPerQuestionStats: scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetAnswerDistribution returns option-level selection counts across all completed
// sessions for every question that appeared in those sessions.
func (r *postgresRepository) GetAnswerDistribution(ctx context.Context, examID string) ([]answerDistRow, error) {
	const q = `
SELECT
  q.id                                                               AS question_id,
  ao.id                                                              AS option_id,
  COALESCE(at_t.text, '')                                            AS option_text,
  COUNT(sa.id)                                                       AS select_count
FROM questions q
JOIN answer_options ao ON ao.question_id = q.id
LEFT JOIN answer_translations at_t ON at_t.option_id = ao.id AND at_t.locale = q.default_locale
LEFT JOIN session_answers sa
  ON sa.selected_option_ids @> jsonb_build_array(ao.id::text)
  AND sa.session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND status IN ('submitted','auto_submitted','grading_pending')
  )
WHERE q.id IN (
  SELECT DISTINCT question_id FROM session_questions
  WHERE session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND status IN ('submitted','auto_submitted','grading_pending')
  )
)
GROUP BY q.id, ao.id, at_t.text, ao.sort_order
ORDER BY q.id, ao.sort_order`

	rows, err := r.db.QueryxContext(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("reports: GetAnswerDistribution: %w", err)
	}
	defer rows.Close()

	var result []answerDistRow
	for rows.Next() {
		var row answerDistRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("reports: GetAnswerDistribution: scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
