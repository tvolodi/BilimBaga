package reports

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

	// ── FR-BB53: Per-Employee Record & Progress ───────────────────────────────

	// GetUserInfo fetches basic user details (id, full_name, department name).
	// Returns ErrNotFound if no user with that ID exists.
	GetUserInfo(ctx context.Context, userID string) (*userInfoRow, error)

	// GetUserSessionHistory returns a page of session history for one employee,
	// excluding in_progress sessions, ordered by started_at DESC (AC-3).
	GetUserSessionHistory(ctx context.Context, userID string, limit, offset int) ([]SessionRecord, error)

	// GetUserSessionCount returns the total non-in_progress session count for
	// one employee (used for pagination metadata, AC-5).
	GetUserSessionCount(ctx context.Context, userID string) (int, error)

	// GetUserTrackActivity returns per-track question counts and last activity
	// for an employee.  Only tracks with activity are returned; the service
	// layer fills in zero-value entries for the other tracks (AC-7 / AC-8).
	GetUserTrackActivity(ctx context.Context, userID string) ([]TrackActivity, error)

	// GetUserRequiredExams returns all active exams assigned to the employee
	// (directly, via their department, or to all) with pass status and attempt
	// count.  Exam track is inferred from the lowest-sort_order question rule
	// category (AC-9).
	GetUserRequiredExams(ctx context.Context, userID string) ([]ExamProgress, error)

	// ── FR-BB54: Export API ───────────────────────────────────────────────────

	// GetExamQuestions returns the ordered list of unique questions for sessions
	// of an exam, used to build the dynamic CSV header (AC-3).
	GetExamQuestions(ctx context.Context, examID, tenantID string) ([]ExamQuestion, error)

	// StreamExamResultSessions returns all submitted/grading_pending sessions for
	// one exam scoped to the given tenant (AC-3, AC-8). The caller must close the
	// returned *sqlx.Rows.
	StreamExamResultSessions(ctx context.Context, examID, tenantID string) (*sqlx.Rows, error)

	// GetSessionQuestionScores returns per-question scores for the given session
	// IDs (AC-3, AC-4).
	GetSessionQuestionScores(ctx context.Context, sessionIDs []string) ([]QuestionScore, error)

	// StreamUserRecordSessions returns all non-in_progress sessions for one user
	// scoped to the given tenant (AC-5, AC-8). The caller must close the returned
	// *sqlx.Rows.
	StreamUserRecordSessions(ctx context.Context, userID, tenantID string) (*sqlx.Rows, error)

	// GetDashboardCompletionRatesForRange returns per-exam completion stats within
	// the given date range for the PDF export (AC-7).
	GetDashboardCompletionRatesForRange(ctx context.Context, tenantID string, from, to time.Time) ([]*ExamCompletionRate, error)

	// GetTopBottomQuestions returns top 5 and bottom 5 questions by correct_rate
	// across all exams in the range for the PDF export (AC-7).
	GetTopBottomQuestions(ctx context.Context, tenantID string, from, to time.Time) (top []QuestionStat, bottom []QuestionStat, err error)
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
LEFT JOIN resolved_assignments ra ON ra.exam_id = e.id
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
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
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

// ── FR-BB53: Per-Employee Record & Progress ──────────────────────────────────

// userInfoRow is the internal scan target for basic user lookup.
type userInfoRow struct {
	ID             string  `db:"id"`
	FullName       string  `db:"full_name"`
	DepartmentName *string `db:"department_name"`
}

// GetUserInfo fetches id, full_name and department name for one user.
// Returns ErrNotFound when no user with that ID exists.
func (r *postgresRepository) GetUserInfo(ctx context.Context, userID string) (*userInfoRow, error) {
	const q = `
SELECT u.id, u.full_name, d.name AS department_name
FROM users u
LEFT JOIN departments d ON d.id = u.department_id
WHERE u.id = $1`
	var row userInfoRow
	if err := r.db.GetContext(ctx, &row, q, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reports: GetUserInfo: %w", err)
	}
	return &row, nil
}

// GetUserSessionHistory returns a page of non-in_progress sessions for one
// employee, ordered by started_at DESC (AC-3, AC-5).
func (r *postgresRepository) GetUserSessionHistory(ctx context.Context, userID string, limit, offset int) ([]SessionRecord, error) {
	const q = `
SELECT
  es.id AS session_id,
  es.exam_id,
  e.title AS exam_title,
  es.started_at,
  es.submitted_at,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT AS time_taken_seconds,
  es.status,
  c.id AS certificate_id,
  NULL::text AS exam_category_track
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
LEFT JOIN certificates c ON c.session_id = es.id
WHERE es.user_id = $1
  AND es.status != 'in_progress'
ORDER BY es.started_at DESC
LIMIT $2 OFFSET $3`

	var result []SessionRecord
	if err := r.db.SelectContext(ctx, &result, q, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("reports: GetUserSessionHistory: %w", err)
	}
	if result == nil {
		result = []SessionRecord{}
	}
	return result, nil
}

// GetUserSessionCount returns the total non-in_progress session count for
// one employee (AC-5 pagination).
func (r *postgresRepository) GetUserSessionCount(ctx context.Context, userID string) (int, error) {
	const q = `
SELECT COUNT(*)
FROM exam_sessions
WHERE user_id = $1
  AND status != 'in_progress'`
	var count int
	if err := r.db.GetContext(ctx, &count, q, userID); err != nil {
		return 0, fmt.Errorf("reports: GetUserSessionCount: %w", err)
	}
	return count, nil
}

// GetUserTrackActivity returns per-track distinct question counts and last
// submitted_at for submitted/grading_pending sessions (AC-7 / AC-8).
// Only tracks with activity are returned.
func (r *postgresRepository) GetUserTrackActivity(ctx context.Context, userID string) ([]TrackActivity, error) {
	const q = `
SELECT
  cat.track,
  COUNT(DISTINCT sqs.question_id) AS questions_answered,
  MAX(es.submitted_at) AS last_activity
FROM exam_sessions es
JOIN session_question_scores sqs ON sqs.session_id = es.id
JOIN questions q ON q.id = sqs.question_id
JOIN categories cat ON cat.id = q.category_id
WHERE es.user_id = $1
  AND es.status IN ('submitted', 'grading_pending')
  AND cat.track IN ('security', 'safety', 'loyalty')
GROUP BY cat.track`

	var result []TrackActivity
	if err := r.db.SelectContext(ctx, &result, q, userID); err != nil {
		return nil, fmt.Errorf("reports: GetUserTrackActivity: %w", err)
	}
	return result, nil
}

// GetUserRequiredExams returns active exams assigned to the employee with pass
// status and attempt count.  Exam track is inferred via a LATERAL join to the
// lowest-sort_order exam_question_rule whose category has a valid track (AC-9).
func (r *postgresRepository) GetUserRequiredExams(ctx context.Context, userID string) ([]ExamProgress, error) {
	const q = `
SELECT
  e.id AS exam_id,
  e.title,
  exam_track.track,
  BOOL_OR(es.passed) AS passed,
  COUNT(es.id) AS attempts
FROM exams e
JOIN LATERAL (
  SELECT cat.track
  FROM exam_question_rules eqr
  JOIN categories cat ON cat.id = eqr.category_id
  WHERE eqr.exam_id = e.id
    AND cat.track IN ('security', 'safety', 'loyalty')
  ORDER BY eqr.sort_order
  LIMIT 1
) AS exam_track ON true
JOIN exam_assignments ea ON ea.exam_id = e.id
  AND (
    (ea.assignee_type = 'user'       AND ea.assignee_id = $1)
    OR (ea.assignee_type = 'department'
        AND ea.assignee_id = (SELECT department_id FROM users WHERE id = $1))
    OR (ea.assignee_type = 'all')
  )
LEFT JOIN exam_sessions es ON es.exam_id = e.id
  AND es.user_id = $1
  AND es.status IN ('submitted', 'grading_pending')
WHERE e.status = 'active'
GROUP BY e.id, e.title, exam_track.track`

	var result []ExamProgress
	if err := r.db.SelectContext(ctx, &result, q, userID); err != nil {
		return nil, fmt.Errorf("reports: GetUserRequiredExams: %w", err)
	}
	if result == nil {
		result = []ExamProgress{}
	}
	return result, nil
}

// ── FR-BB54: Export API ──────────────────────────────────────────────────────

// GetExamQuestions returns the ordered distinct questions that appeared in
// completed sessions of an exam, used to build the dynamic CSV header.
func (r *postgresRepository) GetExamQuestions(ctx context.Context, examID, tenantID string) ([]ExamQuestion, error) {
	const q = `
SELECT
  sq.question_id,
  ROW_NUMBER() OVER (ORDER BY MIN(sq.question_id)) AS position
FROM session_questions sq
JOIN exam_sessions es ON es.id = sq.session_id
WHERE es.exam_id = $1
  AND es.tenant_id = $2
  AND es.status IN ('submitted', 'grading_pending')
GROUP BY sq.question_id
ORDER BY position`

	var result []ExamQuestion
	if err := r.db.SelectContext(ctx, &result, q, examID, tenantID); err != nil {
		return nil, fmt.Errorf("reports: GetExamQuestions: %w", err)
	}
	return result, nil
}

// StreamExamResultSessions returns an open *sqlx.Rows cursor over
// submitted/grading_pending sessions for one exam scoped to the tenant.
// The caller is responsible for closing the rows.
func (r *postgresRepository) StreamExamResultSessions(ctx context.Context, examID, tenantID string) (*sqlx.Rows, error) {
	const q = `
SELECT
  u.full_name                                                                AS employee_name,
  COALESCE(d.name, '')                                                       AS department,
  TO_CHAR(es.started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')  AS started_at,
  TO_CHAR(es.submitted_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS submitted_at,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT                AS time_taken_seconds,
  es.id                                                                      AS session_id
FROM exam_sessions es
JOIN users u ON u.id = es.user_id
LEFT JOIN departments d ON d.id = u.department_id
WHERE es.exam_id = $1
  AND es.tenant_id = $2
  AND es.status IN ('submitted', 'grading_pending')
ORDER BY es.started_at`

	rows, err := r.db.QueryxContext(ctx, q, examID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("reports: StreamExamResultSessions: %w", err)
	}
	return rows, nil
}

// GetSessionQuestionScores returns per-question scores for a batch of session IDs.
func (r *postgresRepository) GetSessionQuestionScores(ctx context.Context, sessionIDs []string) ([]QuestionScore, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	// Use lib/pq array to pass session IDs as a PostgreSQL UUID array.
	q := `
SELECT session_id, question_id, score
FROM session_question_scores
WHERE session_id = ANY($1::UUID[])`

	rows, err := r.db.QueryxContext(ctx, q, pq.Array(sessionIDs))
	if err != nil {
		return nil, fmt.Errorf("reports: GetSessionQuestionScores: %w", err)
	}
	defer rows.Close()

	var result []QuestionScore
	for rows.Next() {
		var qs QuestionScore
		if err := rows.StructScan(&qs); err != nil {
			return nil, fmt.Errorf("reports: GetSessionQuestionScores: scan: %w", err)
		}
		result = append(result, qs)
	}
	return result, rows.Err()
}

// StreamUserRecordSessions returns an open *sqlx.Rows cursor over all
// non-in_progress sessions for one user scoped to the tenant.
// The caller is responsible for closing the rows.
func (r *postgresRepository) StreamUserRecordSessions(ctx context.Context, userID, tenantID string) (*sqlx.Rows, error) {
	const q = `
SELECT
  e.title                                                                     AS exam_title,
  TO_CHAR(es.started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')   AS started_at,
  TO_CHAR(es.submitted_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS submitted_at,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT                 AS time_taken_seconds,
  es.status
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
WHERE es.user_id = $1
  AND es.tenant_id = $2
  AND es.status != 'in_progress'
ORDER BY es.started_at DESC`

	rows, err := r.db.QueryxContext(ctx, q, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("reports: StreamUserRecordSessions: %w", err)
	}
	return rows, nil
}

// GetDashboardCompletionRatesForRange returns completion stats per exam within
// the given UTC date range (inclusive), scoped to the tenant.
func (r *postgresRepository) GetDashboardCompletionRatesForRange(ctx context.Context, tenantID string, from, to time.Time) ([]*ExamCompletionRate, error) {
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
    SELECT ea.exam_id, ea.assignee_id AS user_id
    FROM exam_assignments ea WHERE ea.assignee_type = 'user'
    UNION
    SELECT ea.exam_id, au.user_id
    FROM exam_assignments ea
    JOIN dept_tree dt ON dt.root_id = ea.assignee_id
    JOIN all_users au ON au.department_id = dt.id
    WHERE ea.assignee_type = 'department'
    UNION
    SELECT ea.exam_id, au.user_id
    FROM exam_assignments ea CROSS JOIN all_users au
    WHERE ea.assignee_type = 'all'
)
SELECT
  e.id                                                                           AS exam_id,
  e.title,
  COUNT(DISTINCT ra.user_id)                                                     AS assigned_count,
  COUNT(DISTINCT CASE WHEN es.status IN ('submitted','grading_pending') THEN ra.user_id END)
                                                                                 AS completed_count,
  COUNT(DISTINCT CASE WHEN es.passed = TRUE THEN ra.user_id END)                 AS passed_count
FROM exams e
LEFT JOIN resolved_assignments ra ON ra.exam_id = e.id
LEFT JOIN exam_sessions es
  ON es.exam_id = e.id
  AND es.user_id = ra.user_id
  AND es.submitted_at BETWEEN $2 AND $3
WHERE e.tenant_id = $1
  AND e.status = 'active'
GROUP BY e.id, e.title
ORDER BY e.title`

	rows, err := r.db.QueryxContext(ctx, q, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("reports: GetDashboardCompletionRatesForRange: %w", err)
	}
	defer rows.Close()

	type crRow struct {
		ExamID         string `db:"exam_id"`
		Title          string `db:"title"`
		AssignedCount  int    `db:"assigned_count"`
		CompletedCount int    `db:"completed_count"`
		PassedCount    int    `db:"passed_count"`
	}

	var result []*ExamCompletionRate
	for rows.Next() {
		var row crRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("reports: GetDashboardCompletionRatesForRange: scan: %w", err)
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

// GetTopBottomQuestions returns the top 5 and bottom 5 questions by correct_rate
// across all exams in the date range, scoped to the tenant.
func (r *postgresRepository) GetTopBottomQuestions(ctx context.Context, tenantID string, from, to time.Time) ([]QuestionStat, []QuestionStat, error) {
	const q = `
WITH question_rates AS (
  SELECT
    q.id                                                                AS question_id,
    LEFT(qt.stem, 120)                                                  AS stem_preview,
    ROUND(
      COUNT(sqs.question_id) FILTER (WHERE sqs.score = sqs.max_score)::DECIMAL
      / NULLIF(COUNT(sqs.question_id), 0), 4
    )                                                                   AS correct_rate
  FROM exam_sessions es
  JOIN session_question_scores sqs ON sqs.session_id = es.id
  JOIN questions q                  ON q.id = sqs.question_id
  JOIN question_translations qt     ON qt.question_id = q.id AND qt.locale = q.default_locale
  WHERE es.tenant_id = $1
    AND es.status IN ('submitted', 'grading_pending')
    AND es.submitted_at BETWEEN $2 AND $3
  GROUP BY q.id, qt.stem
),
ranked AS (
  SELECT *, ROW_NUMBER() OVER (ORDER BY correct_rate DESC NULLS LAST) AS top_rank,
            ROW_NUMBER() OVER (ORDER BY correct_rate ASC  NULLS LAST) AS bot_rank
  FROM question_rates
)
SELECT question_id, stem_preview, correct_rate,
       CASE WHEN top_rank <= 5 THEN 'top' ELSE 'bottom' END AS bucket
FROM ranked
WHERE top_rank <= 5 OR bot_rank <= 5
ORDER BY bucket DESC, correct_rate DESC`

	type qRateRow struct {
		QuestionID  string   `db:"question_id"`
		StemPreview string   `db:"stem_preview"`
		CorrectRate *float64 `db:"correct_rate"`
		Bucket      string   `db:"bucket"`
	}

	rows, err := r.db.QueryxContext(ctx, q, tenantID, from, to)
	if err != nil {
		return nil, nil, fmt.Errorf("reports: GetTopBottomQuestions: %w", err)
	}
	defer rows.Close()

	var top, bottom []QuestionStat
	for rows.Next() {
		var row qRateRow
		if err := rows.StructScan(&row); err != nil {
			return nil, nil, fmt.Errorf("reports: GetTopBottomQuestions: scan: %w", err)
		}
		qs := QuestionStat{
			QuestionID:         row.QuestionID,
			StemPreview:        row.StemPreview,
			CorrectRate:        row.CorrectRate,
			AnswerDistribution: []AnswerOptionCount{},
		}
		if row.Bucket == "top" {
			top = append(top, qs)
		} else {
			bottom = append(bottom, qs)
		}
	}
	if top == nil {
		top = []QuestionStat{}
	}
	if bottom == nil {
		bottom = []QuestionStat{}
	}
	return top, bottom, rows.Err()
}
