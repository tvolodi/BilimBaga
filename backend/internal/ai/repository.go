package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/deptscope"
	"github.com/jmoiron/sqlx"
)

// Repository defines the data-access operations required by the AI service.
type Repository interface {
	// CountAIUsageLastHour returns the number of AI calls made by userID for feature
	// in the last 60 minutes.
	CountAIUsageLastHour(ctx context.Context, userID, feature string) (int, error)

	// LogUsage inserts a row into ai_usage_log.
	LogUsage(ctx context.Context, log UsageLog) error

	// GetCategoryName returns the name of a category by its UUID.
	GetCategoryName(ctx context.Context, categoryID string) (string, error)

	// ── FR-BB74: Insight cache ────────────────────────────────────────────────

	// GetInsightCache returns the cached InsightResult for the given exam and scope
	// key (deptscope.ScopeKey; "all" for super_admin), or nil when no entry exists.
	GetInsightCache(ctx context.Context, examID, scopeKey string) (*InsightResult, error)

	// UpsertInsightCache writes (or overwrites) the cache entry for the exam and scope key.
	UpsertInsightCache(ctx context.Context, examID, scopeKey, userID string, insights []string) error

	// CountAIUsageLastDay returns the number of AI calls made by userID for
	// feature in the last 24 hours (ISS-232 daily cap).
	CountAIUsageLastDay(ctx context.Context, userID, feature string) (int, error)

	// GetScopeDepartmentIDs returns the sorted department ids of the caller's
	// scope (its subtree; nil for an unrestricted scope). The AI service hashes
	// them into the scoped insight-cache key (ISS-218).
	GetScopeDepartmentIDs(ctx context.Context, scope deptscope.Scope) ([]string, error)

	// GetExamInsightData gathers all anonymised aggregate statistics needed to
	// build the AI prompt.  Returns ErrExamNotFound when the exam does not exist
	// (tenantID is unused: exams has no tenant_id column; single-tenant, ISS-82).
	GetExamInsightData(ctx context.Context, examID, tenantID string) (*ExamInsightData, error)

	// ── FR-BB75: Loyalty Profile Narrative ───────────────────────────────────

	// GetSessionCategoryTrack loads the exam's category track and the session
	// employee's user ID for the given session. Returns ErrLoyaltySessionNotFound
	// when no session row exists.
	GetSessionCategoryTrack(ctx context.Context, sessionID string) (track string, employeeUserID string, err error)

	// IsEmployeeInAdminDepartment returns true if employeeUserID belongs to the
	// same department as adminUserID.
	IsEmployeeInAdminDepartment(ctx context.Context, adminUserID, employeeUserID string) (bool, error)

	// CollectLikertResponses returns anonymised, polarity-inverted Likert
	// response data for the given session.
	CollectLikertResponses(ctx context.Context, sessionID string) ([]LikertResponseData, error)
}

// withScope substitutes @SCOPE@ with the department-subtree predicate on
// exam_sessions.user_id bound to $2 (NULL = unrestricted; ISS-165).
func withScope(q string) string {
	return strings.ReplaceAll(q, "@SCOPE@", deptscope.Predicate("user_id", "$2"))
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewRepository returns a Repository backed by PostgreSQL.
func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CountAIUsageLastHour(ctx context.Context, userID, feature string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ai_usage_log
		 WHERE user_id = $1 AND feature = $2 AND created_at >= $3`,
		userID, feature, time.Now().UTC().Add(-time.Hour),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ai: CountAIUsageLastHour: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) CountAIUsageLastDay(ctx context.Context, userID, feature string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ai_usage_log
		 WHERE user_id = $1 AND feature = $2 AND created_at >= $3`,
		userID, feature, time.Now().UTC().Add(-24*time.Hour),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ai: CountAIUsageLastDay: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) LogUsage(ctx context.Context, log UsageLog) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ai_usage_log (user_id, feature, tokens_used, model, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		log.UserID, log.Feature, log.TokensUsed, log.Model, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("ai: LogUsage: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	var name string
	err := r.db.QueryRowContext(ctx,
		`SELECT name FROM categories WHERE id = $1`, categoryID,
	).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("ai: GetCategoryName: %w", err)
	}
	return name, nil
}

// ── FR-BB74: Insight cache ────────────────────────────────────────────────────

func (r *postgresRepository) GetInsightCache(ctx context.Context, examID, scopeKey string) (*InsightResult, error) {
	var raw struct {
		Insights    []byte    `db:"insights"`
		GeneratedAt time.Time `db:"generated_at"`
	}
	err := r.db.QueryRowxContext(ctx,
		`SELECT insights, generated_at FROM ai_insight_cache WHERE exam_id = $1 AND scope_key = $2`,
		examID, scopeKey,
	).StructScan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // nil means cache miss
		}
		return nil, fmt.Errorf("ai: GetInsightCache: %w", err)
	}

	var insights []string
	if err := json.Unmarshal(raw.Insights, &insights); err != nil {
		return nil, fmt.Errorf("ai: GetInsightCache: unmarshal insights: %w", err)
	}

	return &InsightResult{
		Insights:    insights,
		GeneratedAt: raw.GeneratedAt,
		Cached:      true,
	}, nil
}

func (r *postgresRepository) UpsertInsightCache(ctx context.Context, examID, scopeKey, userID string, insights []string) error {
	insightsJSON, err := json.Marshal(insights)
	if err != nil {
		return fmt.Errorf("ai: UpsertInsightCache: marshal insights: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO ai_insight_cache (exam_id, scope_key, insights, generated_at, generated_by)
VALUES ($1, $2, $3, NOW(), $4)
ON CONFLICT (exam_id, scope_key)
DO UPDATE SET insights      = EXCLUDED.insights,
              generated_at  = EXCLUDED.generated_at,
              generated_by  = EXCLUDED.generated_by`,
		examID, scopeKey, insightsJSON, userID,
	)
	if err != nil {
		return fmt.Errorf("ai: UpsertInsightCache: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetScopeDepartmentIDs(ctx context.Context, scope deptscope.Scope) ([]string, error) {
	ids, err := deptscope.SubtreeIDs(ctx, r.db, scope)
	if err != nil {
		return nil, fmt.Errorf("ai: GetScopeDepartmentIDs: %w", err)
	}
	return ids, nil
}

// examInsightRow is the scan target for the exam header query.
type examInsightRow struct {
	Title string `db:"title"`
	// exams.passing_score_pct is numeric(5,2); pgx returns it as "70.00", which
	// cannot scan into an int (ISS-093), so scan as float64 and round.
	PassingScorePct float64 `db:"passing_score_pct"`
}

// insightSummaryRow is the scan target for the session aggregate query.
type insightSummaryRow struct {
	TotalAttempts     int      `db:"total_attempts"`
	PassRate          *float64 `db:"pass_rate"`
	AvgScorePct       *float64 `db:"avg_score_pct"`
	AvgCompletionSecs *float64 `db:"avg_completion_secs"`
}

// insightQuestionRow is the scan target for per-question stats.
type insightQuestionRow struct {
	OrderNum    int      `db:"order_num"`
	Stem        string   `db:"stem"`
	CorrectRate *float64 `db:"correct_rate"`
	AvgTimeSecs *float64 `db:"avg_time_secs"`
}

// GetExamInsightData gathers the aggregate statistics for an exam.
//
// tenantID is accepted to keep the Repository interface stable but is unused:
// the exams table has no tenant_id column (the deployment is single-tenant and
// the tenant middleware injects the constant "public"), so there is nothing to
// compare it against (ISS-82; same root cause as ISS-75).
func (r *postgresRepository) GetExamInsightData(ctx context.Context, examID, _ string) (*ExamInsightData, error) {
	// Verify the exam exists.
	var header examInsightRow
	err := r.db.QueryRowxContext(ctx,
		`SELECT title, passing_score_pct FROM exams WHERE id = $1`, examID,
	).StructScan(&header)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExamNotFound
		}
		return nil, fmt.Errorf("ai: GetExamInsightData: fetch exam: %w", err)
	}

	// Aggregate session statistics.
	var summary insightSummaryRow
	scopeArg := deptscope.FromContext(ctx).Arg()
	err = r.db.QueryRowxContext(ctx, withScope(`
SELECT
  COUNT(*)                                                                            AS total_attempts,
  ROUND(
    COUNT(*) FILTER (WHERE passed = TRUE)::DECIMAL / NULLIF(COUNT(*), 0), 4
  )                                                                                   AS pass_rate,
  ROUND(AVG(score_pct)::numeric / 100, 4)                                            AS avg_score_pct,
  ROUND(AVG(
    EXTRACT(EPOCH FROM (submitted_at - started_at))
  )::numeric, 0)                                                                      AS avg_completion_secs
FROM exam_sessions
WHERE exam_id = $1
  AND status IN ('submitted', 'auto_submitted', 'grading_pending') AND @SCOPE@`), examID, scopeArg,
	).StructScan(&summary)
	if err != nil {
		return nil, fmt.Errorf("ai: GetExamInsightData: aggregate sessions: %w", err)
	}

	// Per-question stats — use sort_order from session_questions to preserve display order.
	rows, err := r.db.QueryxContext(ctx, withScope(`
SELECT
  MIN(sq.sort_order)                                                              AS order_num,
  LEFT(qt.stem, 100)                                                              AS stem,
  ROUND(
    COUNT(sqs.question_id) FILTER (WHERE sqs.score = sqs.max_score)::DECIMAL
    / NULLIF(COUNT(sqs.question_id), 0), 4
  )                                                                               AS correct_rate,
  ROUND(AVG(sqs.time_taken_seconds)::numeric, 0)                                 AS avg_time_secs
FROM questions q
JOIN question_translations qt ON qt.question_id = q.id AND qt.locale = q.default_locale
LEFT JOIN session_question_scores sqs ON sqs.question_id = q.id
  AND sqs.session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND status IN ('submitted', 'auto_submitted', 'grading_pending') AND @SCOPE@
  )
LEFT JOIN session_questions sq ON sq.question_id = q.id
  AND sq.session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND status IN ('submitted', 'auto_submitted', 'grading_pending') AND @SCOPE@
  )
WHERE q.id IN (
  SELECT DISTINCT question_id FROM session_questions
  WHERE session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND status IN ('submitted', 'auto_submitted', 'grading_pending') AND @SCOPE@
  )
)
GROUP BY q.id, qt.stem
ORDER BY order_num`), examID, scopeArg,
	)
	if err != nil {
		return nil, fmt.Errorf("ai: GetExamInsightData: per-question stats: %w", err)
	}
	defer rows.Close()

	var qStats []InsightQuestionStat
	for rows.Next() {
		var row insightQuestionRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("ai: GetExamInsightData: scan question row: %w", err)
		}
		stat := InsightQuestionStat{
			OrderNum: row.OrderNum,
			Stem:     row.Stem,
		}
		if row.CorrectRate != nil {
			stat.CorrectRate = *row.CorrectRate
		}
		if row.AvgTimeSecs != nil {
			stat.AvgTimeSecs = int(*row.AvgTimeSecs)
		}
		qStats = append(qStats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ai: GetExamInsightData: iterate question rows: %w", err)
	}

	data := &ExamInsightData{
		ExamTitle:       header.Title,
		PassingScorePct: int(math.Round(header.PassingScorePct)),
		TotalAttempts:   summary.TotalAttempts,
		QuestionStats:   qStats,
	}
	if summary.PassRate != nil {
		data.PassRate = *summary.PassRate
	}
	if summary.AvgScorePct != nil {
		data.AvgScorePct = *summary.AvgScorePct
	}
	if summary.AvgCompletionSecs != nil {
		data.AvgCompletionSecs = int(*summary.AvgCompletionSecs)
	}

	return data, nil
}

// ── FR-BB75: Loyalty Profile Narrative ───────────────────────────────────────

func (r *postgresRepository) GetSessionCategoryTrack(ctx context.Context, sessionID string) (track string, employeeUserID string, err error) {
	var row struct {
		Track          sql.NullString `db:"track"`
		EmployeeUserID string         `db:"user_id"`
	}
	scanErr := r.db.QueryRowxContext(ctx, `
SELECT t.track, es.user_id
FROM exam_sessions es
LEFT JOIN LATERAL (
  SELECT c.track
  FROM exam_question_rules r
  JOIN categories c ON c.id = r.category_id
  WHERE r.exam_id = es.exam_id AND c.track IS NOT NULL
  ORDER BY r.sort_order
  LIMIT 1
) t ON TRUE
WHERE es.id = $1`, sessionID).StructScan(&row)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return "", "", ErrLoyaltySessionNotFound
		}
		return "", "", fmt.Errorf("ai: GetSessionCategoryTrack: %w", scanErr)
	}
	if row.Track.Valid {
		track = row.Track.String
	}
	return track, row.EmployeeUserID, nil
}

func (r *postgresRepository) IsEmployeeInAdminDepartment(ctx context.Context, adminUserID, employeeUserID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM users u_emp
  JOIN users u_adm ON u_adm.department_id = u_emp.department_id
  WHERE u_emp.id = $1 AND u_adm.id = $2
)`, employeeUserID, adminUserID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ai: IsEmployeeInAdminDepartment: %w", err)
	}
	return exists, nil
}

type likertRow struct {
	DimensionLabel string `db:"dimension_label"`
	RawWeight      int    `db:"raw_weight"`
	LikertPolarity string `db:"likert_polarity"`
}

func (r *postgresRepository) CollectLikertResponses(ctx context.Context, sessionID string) ([]LikertResponseData, error) {
	rows, err := r.db.QueryxContext(ctx, `
SELECT
  COALESCE(c.name, '') AS dimension_label,
  ROUND(ao.likert_weight)::int AS raw_weight,
  COALESCE(ao.likert_polarity, 'positive') AS likert_polarity
FROM session_answers sa
JOIN questions q ON q.id = sa.question_id
JOIN session_questions sq ON sq.question_id = q.id AND sq.session_id = $1
JOIN answer_options ao ON ao.id = (sa.selected_option_ids->>0)::uuid
LEFT JOIN categories c ON c.id = q.category_id
WHERE sa.session_id = $1
  AND q.type = 'likert'
ORDER BY sq.sort_order`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ai: CollectLikertResponses: %w", err)
	}
	defer rows.Close()

	var result []LikertResponseData
	for rows.Next() {
		var row likertRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("ai: CollectLikertResponses: scan: %w", err)
		}
		normalized := row.RawWeight
		if row.LikertPolarity == "negative" {
			normalized = 6 - row.RawWeight
		}
		result = append(result, LikertResponseData{
			DimensionLabel:   row.DimensionLabel,
			NormalizedWeight: normalized,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ai: CollectLikertResponses: iterate: %w", err)
	}
	return result, nil
}
