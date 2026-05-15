package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the persistence operations for session creation.
type Repository interface {
	// GetExamConfig fetches the fields needed for pre-flight checks.
	GetExamConfig(ctx context.Context, examID string) (*examConfig, error)

	// IsAssigned returns true if the exam is assigned to the user (directly,
	// via department, or via 'all').
	IsAssigned(ctx context.Context, examID, userID, deptID string) (bool, error)

	// CountFinishedSessions returns the number of non-in_progress sessions for user+exam.
	CountFinishedSessions(ctx context.Context, examID, userID string) (int, error)

	// HasOpenSession returns true if there is an in_progress session with expires_at > NOW().
	HasOpenSession(ctx context.Context, examID, userID string) (bool, error)

	// GetRules returns the question rules for an exam ordered by sort_order.
	GetRules(ctx context.Context, examID string) ([]questionRule, error)

	// GetManualQuestions returns ordered question IDs for a manual rule.
	GetManualQuestions(ctx context.Context, ruleID string) ([]poolQuestion, error)

	// GetEligibleQuestions returns all active questions matching a random rule's filters.
	GetEligibleQuestions(ctx context.Context, rule questionRule) ([]poolQuestion, error)

	// GetOptionIDs returns the option IDs for a question in sort_order.
	GetOptionIDs(ctx context.Context, questionID string) ([]poolOption, error)

	// GetQuestionDetails fetches stem and type for a set of question IDs.
	// Returns a map[questionID]questionDetail.
	GetQuestionDetails(ctx context.Context, ids []string) (map[string]questionDetail, error)

	// GetOptionTexts fetches option text for a set of option IDs using the default locale.
	// Returns a map[optionID]text.
	GetOptionTexts(ctx context.Context, optionIDs []string) (map[string]string, error)

	// CreateSession inserts exam_sessions and session_questions rows inside a
	// single transaction and returns the new session ID and timestamps.
	CreateSession(ctx context.Context, input createSessionInput) (string, time.Time, time.Time, error)

	// GetSessionForUser fetches a session row by ID, scoped to userID (FR-BB37 AC-7/AC-10).
	// Returns ErrSessionNotFound if the session does not exist,
	// ErrSessionForbidden if it belongs to a different user.
	GetSessionForUser(ctx context.Context, sessionID, userID string) (*sessionStateRow, error)

	// GetSessionQuestions returns the session_questions rows (with type) in sort_order for a session.
	GetSessionQuestions(ctx context.Context, sessionID string) ([]sessionQuestionRow, error)

	// GetSessionAnswers returns all saved answers for a session keyed by question_id.
	GetSessionAnswers(ctx context.Context, sessionID string) (map[string]savedAnswerRow, error)

	// GetValidOptionIDs returns the set of valid option IDs for a question.
	GetValidOptionIDs(ctx context.Context, questionID string) (map[string]struct{}, error)

	// UpsertAnswer performs an INSERT … ON CONFLICT DO UPDATE for session_answers.
	// Returns the saved_at timestamp after upsert.
	UpsertAnswer(ctx context.Context, input upsertAnswerInput) (time.Time, error)

	// InsertTabSwitchEvent inserts one row into tab_switch_events (FR-BB38 AC-1).
	InsertTabSwitchEvent(ctx context.Context, sessionID, eventType, actionTaken string) error

	// CountTabSwitchEvents returns the total number of tab-switch events for a session (FR-BB38 AC-3).
	CountTabSwitchEvents(ctx context.Context, sessionID string) (int, error)

	// AutoSubmitSession sets session status to 'auto_submitted' and submitted_at = NOW(),
	// grades it, and returns the updated session row (FR-BB38 AC-2).
	AutoSubmitSession(ctx context.Context, sessionID string) (*autoSubmitResult, error)

	// SubmitSession explicitly submits an in_progress session (FR-BB39).
	// It runs the full status-transition + grading inside one transaction.
	// Returns ErrSessionForbidden if the session belongs to a different user.
	// Returns the idempotent result (HTTP 200) if the session is already submitted/grading_pending.
	SubmitSession(ctx context.Context, sessionID, userID, tenantID, actorIP string) (*SubmitSessionResponse, error)

	// GetSessionResult fetches session result data scoped to the given user (FR-BB41 AC-1, AC-2, AC-8).
	// Returns ErrSessionNotFound if the session does not exist globally.
	// Returns ErrSessionForbidden if the session belongs to a different user.
	GetSessionResult(ctx context.Context, sessionID, userID string) (*sessionResultRow, error)

	// GetAdminSessionResult fetches session result data for any session (FR-BB41 AC-5, AC-9).
	// Returns ErrSessionNotFound if the session does not exist.
	GetAdminSessionResult(ctx context.Context, sessionID string) (*sessionResultRow, error)

	// GetSectionScores returns per-section scores for a session (FR-BB41 AC-7).
	// Returns an empty slice when no section data exists.
	GetSectionScores(ctx context.Context, sessionID string) ([]SectionScore, error)

	// GetQuestionBreakdown returns per-question score breakdown rows ordered by sort_order (FR-BB41 AC-4).
	GetQuestionBreakdown(ctx context.Context, sessionID, locale string) ([]questionBreakdownRow, error)

	// GetCorrectAnswerTexts returns correct option texts per question keyed by question ID (FR-BB41 AC-4).
	// For short_text questions there are no correct options; those question IDs are absent from the map.
	GetCorrectAnswerTexts(ctx context.Context, questionIDs []string, locale string) (map[string][]string, error)

	// GetExamHistory returns paginated completed sessions for a user+exam (FR-BB41 AC-6, AC-10).
	// Returns the rows, total count (for pagination), and any error.
	GetExamHistory(ctx context.Context, examID, userID string, page, perPage int) ([]historyRow, int, error)

	// GetExamTitleByID returns the title for the given exam ID (FR-BB41).
	// Returns ErrExamNotFound if the exam does not exist.
	GetExamTitleByID(ctx context.Context, examID string) (string, error)
}

// questionDetail is the stem+type data fetched for question display.
type questionDetail struct {
	Stem string
	Type string
}

// sessionStateRow holds the raw session fields needed for FR-BB37.
type sessionStateRow struct {
	ID                 string    `db:"id"`
	ExamID             string    `db:"exam_id"`
	UserID             string    `db:"user_id"`
	Status             string    `db:"status"`
	StartedAt          time.Time `db:"started_at"`
	ExpiresAt          time.Time `db:"expires_at"`
	ExamTitle          string    `db:"exam_title"`
	CertificateEnabled bool      `db:"certificate_enabled"`
}

// sessionQuestionRow holds one session_questions row with question type.
type sessionQuestionRow struct {
	QuestionID   string `db:"question_id"`
	SortOrder    int    `db:"sort_order"`
	OptionsOrder []byte `db:"options_order"`
	QuestionType string `db:"question_type"`
}

// savedAnswerRow holds one session_answers row.
type savedAnswerRow struct {
	SelectedOptionIDs []byte    `db:"selected_option_ids"`
	TextAnswer        *string   `db:"text_answer"`
	TimeSpentSeconds  int       `db:"time_spent_seconds"`
	SavedAt           time.Time `db:"saved_at"`
}

// autoSubmitResult holds the session state after auto-submit (FR-BB38 AC-2).
type autoSubmitResult struct {
	SessionID string
	Status    string
	ScorePct  *float64
	Passed    *bool
}

// upsertAnswerInput holds all fields for the INSERT … ON CONFLICT upsert.
type upsertAnswerInput struct {
	SessionID         string
	QuestionID        string
	SelectedOptionIDs []string
	TextAnswer        *string
	TimeSpentSeconds  int
}

// createSessionInput holds everything needed for the transactional insert.
type createSessionInput struct {
	ExamID    string
	UserID    string
	Seed      int64
	ExpiresAt time.Time
	Questions []resolvedQuestion
}

// resolvedQuestion is a fully resolved question with its final sort_order, shuffled option IDs,
// and the rule that produced it (nil for questions outside any rule).
type resolvedQuestion struct {
	QuestionID   string
	SortOrder    int
	OptionsOrder []string // option UUIDs in shuffled display order
	RuleID       *string
}

// sessionResultRow holds the raw result data for a session (FR-BB41).
type sessionResultRow struct {
	SessionID        string     `db:"session_id"`
	ExamID           string     `db:"exam_id"`
	ExamTitle        string     `db:"exam_title"`
	ScorePct         *float64   `db:"score_pct"`
	Passed           bool       `db:"passed"`
	TimeTakenSeconds *int       `db:"time_taken_seconds"`
	AttemptNumber    int        `db:"attempt_number"`
	SubmittedAt      *time.Time `db:"submitted_at"`
	ShowAnswersMode  string     `db:"show_answers_mode"`
	Status           string     `db:"status"`
	UserID           string     `db:"user_id"`
}

// questionBreakdownRow holds one per-question score breakdown row (FR-BB41).
type questionBreakdownRow struct {
	QuestionID   string  `db:"question_id"`
	QuestionType string  `db:"question_type"`
	Stem         string  `db:"stem"`
	PointsEarned float64 `db:"points_earned"`
	MaxPoints    float64 `db:"max_points"`
	Explanation  *string `db:"explanation"`
}

// historyRow holds one session in an exam's history list (FR-BB41).
type historyRow struct {
	SessionID   string     `db:"session_id"`
	StartedAt   time.Time  `db:"started_at"`
	SubmittedAt *time.Time `db:"submitted_at"`
	ScorePct    *float64   `db:"score_pct"`
	Passed      bool       `db:"passed"`
	Status      string     `db:"status"`
}

type postgresRepository struct {
	db     *sqlx.DB
	engine GradingEngine
}

// NewRepository returns a Repository backed by PostgreSQL.
// engine is used by SubmitSession and AutoSubmitSession to grade sessions.
func NewRepository(db *sqlx.DB, engine GradingEngine) Repository {
	return &postgresRepository{db: db, engine: engine}
}

func (r *postgresRepository) GetExamConfig(ctx context.Context, examID string) (*examConfig, error) {
	const q = `
SELECT id, status, time_limit_minutes, max_attempts, available_from, available_until,
       shuffle_questions, shuffle_options, on_tab_switch
FROM exams WHERE id = $1`

	var cfg examConfig
	if err := r.db.GetContext(ctx, &cfg, q, examID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotAssigned
		}
		return nil, fmt.Errorf("sessions: GetExamConfig: %w", err)
	}
	return &cfg, nil
}

func (r *postgresRepository) IsAssigned(ctx context.Context, examID, userID, deptID string) (bool, error) {
	const q = `
WITH RECURSIVE dept_tree(id) AS (
    SELECT $3::uuid AS id
    UNION ALL
    SELECT d.parent_id FROM departments d JOIN dept_tree dt ON d.id = dt.id WHERE d.parent_id IS NOT NULL
)
SELECT EXISTS (
    SELECT 1 FROM exam_assignments ea
    WHERE ea.exam_id = $1
      AND (
           (ea.assignee_type = 'user' AND ea.assignee_id = $2)
        OR (ea.assignee_type = 'department' AND ea.assignee_id IN (SELECT id FROM dept_tree))
        OR (ea.assignee_type = 'all')
      )
)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, q, examID, userID, deptID); err != nil {
		return false, fmt.Errorf("sessions: IsAssigned: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) CountFinishedSessions(ctx context.Context, examID, userID string) (int, error) {
	const q = `
SELECT COUNT(*) FROM exam_sessions
WHERE exam_id = $1 AND user_id = $2 AND status IN ('submitted', 'auto_submitted')`
	var count int
	if err := r.db.GetContext(ctx, &count, q, examID, userID); err != nil {
		return 0, fmt.Errorf("sessions: CountFinishedSessions: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) HasOpenSession(ctx context.Context, examID, userID string) (bool, error) {
	const q = `
SELECT EXISTS (
    SELECT 1 FROM exam_sessions
    WHERE exam_id = $1 AND user_id = $2 AND status = 'in_progress' AND expires_at > NOW()
)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, q, examID, userID); err != nil {
		return false, fmt.Errorf("sessions: HasOpenSession: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) GetRules(ctx context.Context, examID string) ([]questionRule, error) {
	const q = `
SELECT id, mode, category_id, tag_ids, difficulty, count, sort_order
FROM exam_question_rules
WHERE exam_id = $1 ORDER BY sort_order`

	rows, err := r.db.QueryxContext(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetRules: %w", err)
	}
	defer rows.Close()

	var result []questionRule
	for rows.Next() {
		var rule questionRule
		if err := rows.StructScan(&rule); err != nil {
			return nil, fmt.Errorf("sessions: GetRules: scan: %w", err)
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func (r *postgresRepository) GetManualQuestions(ctx context.Context, ruleID string) ([]poolQuestion, error) {
	const q = `
SELECT q.id, q.type
FROM exam_manual_questions emq
JOIN questions q ON q.id = emq.question_id
WHERE emq.rule_id = $1 AND q.status = 'active'
ORDER BY emq.sort_order`

	rows, err := r.db.QueryxContext(ctx, q, ruleID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetManualQuestions: %w", err)
	}
	defer rows.Close()

	var result []poolQuestion
	for rows.Next() {
		var pq poolQuestion
		if err := rows.StructScan(&pq); err != nil {
			return nil, fmt.Errorf("sessions: GetManualQuestions: scan: %w", err)
		}
		result = append(result, pq)
	}
	return result, rows.Err()
}

func (r *postgresRepository) GetEligibleQuestions(ctx context.Context, rule questionRule) ([]poolQuestion, error) {
	var (
		args  []interface{}
		parts []string
		idx   = 1
	)

	parts = append(parts, "q.status = 'active'")

	if rule.CategoryID != nil {
		args = append(args, *rule.CategoryID)
		parts = append(parts, fmt.Sprintf("q.category_id = $%d", idx))
		idx++
	}
	if rule.Difficulty != nil {
		args = append(args, *rule.Difficulty)
		parts = append(parts, fmt.Sprintf("q.difficulty = $%d", idx))
		idx++
	}

	// Tag containment: question must have ALL tags in the rule's tag_ids array.
	var tagIDs []string
	if len(rule.TagIDs) > 0 {
		if err := json.Unmarshal(rule.TagIDs, &tagIDs); err == nil && len(tagIDs) > 0 {
			args = append(args, rule.TagIDs)
			parts = append(parts, fmt.Sprintf(`(
    SELECT array_agg(qt.tag_id::text) FROM question_tags qt WHERE qt.question_id = q.id
) @> (SELECT array_agg(t) FROM jsonb_array_elements_text($%d::jsonb) t)`, idx))
			idx++
		}
	}

	where := strings.Join(parts, " AND ")
	q := fmt.Sprintf(`SELECT q.id, q.type FROM questions q WHERE %s ORDER BY q.id`, where)

	rows, err := r.db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetEligibleQuestions: %w", err)
	}
	defer rows.Close()

	var result []poolQuestion
	for rows.Next() {
		var pq poolQuestion
		if err := rows.StructScan(&pq); err != nil {
			return nil, fmt.Errorf("sessions: GetEligibleQuestions: scan: %w", err)
		}
		result = append(result, pq)
	}
	return result, rows.Err()
}

func (r *postgresRepository) GetOptionIDs(ctx context.Context, questionID string) ([]poolOption, error) {
	const q = `
SELECT id AS option_id, sort_order
FROM answer_options
WHERE question_id = $1
ORDER BY sort_order`

	rows, err := r.db.QueryxContext(ctx, q, questionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetOptionIDs: %w", err)
	}
	defer rows.Close()

	var result []poolOption
	for rows.Next() {
		var po poolOption
		if err := rows.StructScan(&po); err != nil {
			return nil, fmt.Errorf("sessions: GetOptionIDs: scan: %w", err)
		}
		result = append(result, po)
	}
	return result, rows.Err()
}

func (r *postgresRepository) GetQuestionDetails(ctx context.Context, ids []string) (map[string]questionDetail, error) {
	if len(ids) == 0 {
		return map[string]questionDetail{}, nil
	}

	// Build $1,$2,... placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	q := fmt.Sprintf(`
SELECT q.id, COALESCE(qt.stem, '') AS stem, q.type
FROM questions q
LEFT JOIN question_translations qt
    ON qt.question_id = q.id AND qt.locale = q.default_locale
WHERE q.id IN (%s)`, strings.Join(placeholders, ","))

	type row struct {
		ID   string `db:"id"`
		Stem string `db:"stem"`
		Type string `db:"type"`
	}

	rows, err := r.db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetQuestionDetails: %w", err)
	}
	defer rows.Close()

	result := make(map[string]questionDetail, len(ids))
	for rows.Next() {
		var r row
		if err := rows.StructScan(&r); err != nil {
			return nil, fmt.Errorf("sessions: GetQuestionDetails: scan: %w", err)
		}
		result[r.ID] = questionDetail{Stem: r.Stem, Type: r.Type}
	}
	return result, rows.Err()
}

func (r *postgresRepository) GetOptionTexts(ctx context.Context, optionIDs []string) (map[string]string, error) {
	if len(optionIDs) == 0 {
		return map[string]string{}, nil
	}

	placeholders := make([]string, len(optionIDs))
	args := make([]interface{}, len(optionIDs))
	for i, id := range optionIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Join through questions to reach its default_locale for the translation lookup.
	q := fmt.Sprintf(`
SELECT ao.id AS option_id, COALESCE(at.text, '') AS text
FROM answer_options ao
LEFT JOIN answer_translations at
    ON at.option_id = ao.id
    AND at.locale = (SELECT default_locale FROM questions WHERE id = ao.question_id)
WHERE ao.id IN (%s)`, strings.Join(placeholders, ","))

	type row struct {
		OptionID string `db:"option_id"`
		Text     string `db:"text"`
	}

	rows, err := r.db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetOptionTexts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string, len(optionIDs))
	for rows.Next() {
		var r row
		if err := rows.StructScan(&r); err != nil {
			return nil, fmt.Errorf("sessions: GetOptionTexts: scan: %w", err)
		}
		result[r.OptionID] = r.Text
	}
	return result, rows.Err()
}

func (r *postgresRepository) CreateSession(ctx context.Context, input createSessionInput) (string, time.Time, time.Time, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("sessions: CreateSession: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const insertSession = `
INSERT INTO exam_sessions (exam_id, user_id, seed, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, started_at, expires_at`

	type sessionRow struct {
		ID        string    `db:"id"`
		StartedAt time.Time `db:"started_at"`
		ExpiresAt time.Time `db:"expires_at"`
	}
	var sr sessionRow
	if err := tx.QueryRowxContext(ctx, insertSession,
		input.ExamID, input.UserID, input.Seed, input.ExpiresAt,
	).StructScan(&sr); err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("sessions: CreateSession: insert session: %w", err)
	}

	for _, q := range input.Questions {
		optJSON, err := json.Marshal(q.OptionsOrder)
		if err != nil {
			return "", time.Time{}, time.Time{}, fmt.Errorf("sessions: CreateSession: marshal options: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO session_questions (session_id, question_id, sort_order, options_order, rule_id) VALUES ($1, $2, $3, $4, $5)`,
			sr.ID, q.QuestionID, q.SortOrder, optJSON, q.RuleID,
		); err != nil {
			return "", time.Time{}, time.Time{}, fmt.Errorf("sessions: CreateSession: insert question: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("sessions: CreateSession: commit: %w", err)
	}
	return sr.ID, sr.StartedAt, sr.ExpiresAt, nil
}

func (r *postgresRepository) GetSessionForUser(ctx context.Context, sessionID, userID string) (*sessionStateRow, error) {
	const q = `
SELECT es.id, es.exam_id, es.user_id, es.status, es.started_at, es.expires_at,
       e.title AS exam_title, e.certificate_enabled
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
WHERE es.id = $1`

	var row sessionStateRow
	if err := r.db.GetContext(ctx, &row, q, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("sessions: GetSessionForUser: %w", err)
	}
	if row.UserID != userID {
		return nil, ErrSessionForbidden
	}
	return &row, nil
}

func (r *postgresRepository) GetSessionQuestions(ctx context.Context, sessionID string) ([]sessionQuestionRow, error) {
	const q = `
SELECT sq.question_id, sq.sort_order, sq.options_order, q.type AS question_type
FROM session_questions sq
JOIN questions q ON q.id = sq.question_id
WHERE sq.session_id = $1
ORDER BY sq.sort_order`

	rows, err := r.db.QueryxContext(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSessionQuestions: %w", err)
	}
	defer rows.Close()

	var result []sessionQuestionRow
	for rows.Next() {
		var row sessionQuestionRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("sessions: GetSessionQuestions: scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *postgresRepository) GetSessionAnswers(ctx context.Context, sessionID string) (map[string]savedAnswerRow, error) {
	const q = `
SELECT question_id, selected_option_ids, text_answer, time_spent_seconds, saved_at
FROM session_answers
WHERE session_id = $1`

	type dbRow struct {
		QuestionID        string    `db:"question_id"`
		SelectedOptionIDs []byte    `db:"selected_option_ids"`
		TextAnswer        *string   `db:"text_answer"`
		TimeSpentSeconds  int       `db:"time_spent_seconds"`
		SavedAt           time.Time `db:"saved_at"`
	}

	rows, err := r.db.QueryxContext(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSessionAnswers: %w", err)
	}
	defer rows.Close()

	result := make(map[string]savedAnswerRow)
	for rows.Next() {
		var dbr dbRow
		if err := rows.StructScan(&dbr); err != nil {
			return nil, fmt.Errorf("sessions: GetSessionAnswers: scan: %w", err)
		}
		result[dbr.QuestionID] = savedAnswerRow{
			SelectedOptionIDs: dbr.SelectedOptionIDs,
			TextAnswer:        dbr.TextAnswer,
			TimeSpentSeconds:  dbr.TimeSpentSeconds,
			SavedAt:           dbr.SavedAt,
		}
	}
	return result, rows.Err()
}

func (r *postgresRepository) GetValidOptionIDs(ctx context.Context, questionID string) (map[string]struct{}, error) {
	const q = `SELECT id FROM answer_options WHERE question_id = $1`

	rows, err := r.db.QueryxContext(ctx, q, questionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetValidOptionIDs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("sessions: GetValidOptionIDs: scan: %w", err)
		}
		result[id] = struct{}{}
	}
	return result, rows.Err()
}

func (r *postgresRepository) InsertTabSwitchEvent(ctx context.Context, sessionID, eventType, actionTaken string) error {
	const q = `
INSERT INTO tab_switch_events (session_id, occurred_at, action_taken)
VALUES ($1, NOW(), $2)`
	if _, err := r.db.ExecContext(ctx, q, sessionID, actionTaken); err != nil {
		return fmt.Errorf("sessions: InsertTabSwitchEvent: %w", err)
	}
	return nil
}

func (r *postgresRepository) CountTabSwitchEvents(ctx context.Context, sessionID string) (int, error) {
	const q = `SELECT COUNT(*) FROM tab_switch_events WHERE session_id = $1`
	var count int
	if err := r.db.GetContext(ctx, &count, q, sessionID); err != nil {
		return 0, fmt.Errorf("sessions: CountTabSwitchEvents: %w", err)
	}
	return count, nil
}

// AutoSubmitSession transitions the session to 'auto_submitted' (or 'grading_pending'
// if short-text questions exist), runs the GradingEngine, and returns the resulting state.
func (r *postgresRepository) AutoSubmitSession(ctx context.Context, sessionID string) (*autoSubmitResult, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("sessions: AutoSubmitSession: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Detect whether short-text questions exist in this session.
	const hasShortTextQ = `
SELECT EXISTS (
    SELECT 1 FROM session_questions sq
    JOIN questions q ON q.id = sq.question_id
    WHERE sq.session_id = $1 AND q.type = 'shorttext'
)`
	var needsManualGrade bool
	if err := tx.QueryRowContext(ctx, hasShortTextQ, sessionID).Scan(&needsManualGrade); err != nil {
		return nil, fmt.Errorf("sessions: AutoSubmitSession: check shorttext: %w", err)
	}

	newStatus := "auto_submitted"
	if needsManualGrade {
		newStatus = "grading_pending"
	}

	const updateQ = `
UPDATE exam_sessions SET status = $2, submitted_at = NOW()
WHERE id = $1
RETURNING id, status, score_pct, passed`
	var res autoSubmitResult
	if err := tx.QueryRowxContext(ctx, updateQ, sessionID, newStatus).Scan(
		&res.SessionID, &res.Status, &res.ScorePct, &res.Passed,
	); err != nil {
		return nil, fmt.Errorf("sessions: AutoSubmitSession: update status: %w", err)
	}

	// Run the grading engine (handles all question types including shorttext → pending_manual).
	scorePct, passed, err := r.engine.Grade(tx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: AutoSubmitSession: grade: %w", err)
	}
	res.ScorePct = &scorePct
	res.Passed = &passed

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("sessions: AutoSubmitSession: commit: %w", err)
	}
	return &res, nil
}

func (r *postgresRepository) UpsertAnswer(ctx context.Context, input upsertAnswerInput) (time.Time, error) {
	optJSON, err := json.Marshal(input.SelectedOptionIDs)
	if err != nil {
		return time.Time{}, fmt.Errorf("sessions: UpsertAnswer: marshal options: %w", err)
	}

	const q = `
INSERT INTO session_answers
    (id, session_id, question_id, selected_option_ids, text_answer, saved_at, time_spent_seconds)
VALUES
    (gen_random_uuid(), $1, $2, $3::jsonb, $4, NOW(), $5)
ON CONFLICT (session_id, question_id)
DO UPDATE SET
    selected_option_ids = EXCLUDED.selected_option_ids,
    text_answer         = EXCLUDED.text_answer,
    saved_at            = NOW(),
    time_spent_seconds  = EXCLUDED.time_spent_seconds
RETURNING saved_at`

	var savedAt time.Time
	if err := r.db.QueryRowContext(ctx, q,
		input.SessionID, input.QuestionID, optJSON, input.TextAnswer, input.TimeSpentSeconds,
	).Scan(&savedAt); err != nil {
		return time.Time{}, fmt.Errorf("sessions: UpsertAnswer: %w", err)
	}
	return savedAt, nil
}

// SubmitSession explicitly submits a session (FR-BB39).
// All status transitions, grading, and audit entries run inside a single transaction (AC-10).
func (r *postgresRepository) SubmitSession(ctx context.Context, sessionID, userID, tenantID, actorIP string) (*SubmitSessionResponse, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("sessions: SubmitSession: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Fetch the session, locking it for the duration of the transaction.
	const fetchQ = `
SELECT id, user_id, status, submitted_at, score_pct, passed
FROM exam_sessions
WHERE id = $1
FOR UPDATE`

	type fetchRow struct {
		ID          string     `db:"id"`
		UserID      string     `db:"user_id"`
		Status      string     `db:"status"`
		SubmittedAt *time.Time `db:"submitted_at"`
		ScorePct    *float64   `db:"score_pct"`
		Passed      *bool      `db:"passed"`
	}

	var row fetchRow
	if err := tx.QueryRowxContext(ctx, fetchQ, sessionID).StructScan(&row); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("sessions: SubmitSession: fetch: %w", err)
	}

	// AC-1: ownership check.
	if row.UserID != userID {
		return nil, ErrSessionForbidden
	}

	// AC-8: idempotency — already submitted or pending.
	if row.Status != "in_progress" {
		submittedAt := time.Now().UTC()
		if row.SubmittedAt != nil {
			submittedAt = *row.SubmittedAt
		}
		return &SubmitSessionResponse{
			SessionID:   row.ID,
			Status:      row.Status,
			SubmittedAt: submittedAt,
			ScorePct:    row.ScorePct,
			Passed:      row.Passed,
		}, nil
	}

	// AC-4/AC-5: check whether any short_text question is in the session.
	const hasShortTextQ = `
SELECT EXISTS (
    SELECT 1 FROM session_questions sq
    JOIN questions q ON q.id = sq.question_id
    WHERE sq.session_id = $1 AND q.type = 'shorttext'
)`
	var hasShortText bool
	if err := tx.QueryRowContext(ctx, hasShortTextQ, sessionID).Scan(&hasShortText); err != nil {
		return nil, fmt.Errorf("sessions: SubmitSession: check shorttext: %w", err)
	}

	submittedAt := time.Now().UTC()

	// AC-9: audit entry for every explicit submission.
	const auditQ = `
INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, entity_id, ip, metadata)
VALUES ($1, $2, $3, $4, $5, $6, '{}'::jsonb)`
	sid := sessionID
	if _, err := tx.ExecContext(ctx, auditQ, tenantID, userID, "session.submit", "exam_session", sid, actorIP); err != nil {
		return nil, fmt.Errorf("sessions: SubmitSession: audit session.submit: %w", err)
	}

	// Determine target status.
	targetStatus := "submitted"
	if hasShortText {
		targetStatus = "grading_pending"
		if _, err := tx.ExecContext(ctx, auditQ, tenantID, userID, "session.pending_manual_grade", "exam_session", sid, actorIP); err != nil {
			return nil, fmt.Errorf("sessions: SubmitSession: audit pending_manual_grade: %w", err)
		}
	}

	const updateQ = `
UPDATE exam_sessions SET status = $2, submitted_at = $3
WHERE id = $1
RETURNING id, status, submitted_at, score_pct, passed`
	var updated fetchRow
	if err := tx.QueryRowxContext(ctx, updateQ, sessionID, targetStatus, submittedAt).StructScan(&updated); err != nil {
		return nil, fmt.Errorf("sessions: SubmitSession: update status: %w", err)
	}

	// Run the grading engine (handles all question types; AC-7: atomic with status update).
	scorePct, passed, err := r.engine.Grade(tx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: SubmitSession: grade: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("sessions: SubmitSession: commit: %w", err)
	}

	return &SubmitSessionResponse{
		SessionID:   updated.ID,
		Status:      updated.Status,
		SubmittedAt: submittedAt,
		ScorePct:    &scorePct,
		Passed:      &passed,
	}, nil
}

// sessionResultCTE is the common CTE fragment used by GetSessionResult and
// GetAdminSessionResult to compute attempt_number correctly.
// Window functions run AFTER WHERE, so we compute attempt_number across all
// sessions for the user+exam pair inside a CTE, then filter outside.
const sessionResultCTE = `
WITH ranked AS (
    SELECT
        es.id                     AS session_id,
        es.exam_id,
        e.title                   AS exam_title,
        e.show_answers_mode,
        es.user_id,
        es.status,
        es.score_pct,
        COALESCE(es.passed, false) AS passed,
        EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::int AS time_taken_seconds,
        es.submitted_at,
        ROW_NUMBER() OVER (
            PARTITION BY es.user_id, es.exam_id
            ORDER BY es.started_at
        )                         AS attempt_number
    FROM exam_sessions es
    JOIN exams e ON e.id = es.exam_id
)
`

// GetSessionResult fetches the result for a session, scoped to the owning user.
func (r *postgresRepository) GetSessionResult(ctx context.Context, sessionID, userID string) (*sessionResultRow, error) {
	q := sessionResultCTE + `
SELECT session_id, exam_id, exam_title, show_answers_mode, user_id, status,
       score_pct, passed, time_taken_seconds, submitted_at, attempt_number::int
FROM ranked WHERE session_id = $1`

	var row sessionResultRow
	if err := r.db.GetContext(ctx, &row, q, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("sessions: GetSessionResult: %w", err)
	}
	// Ownership check done in Go to avoid leaking existence via error type.
	if row.UserID != userID {
		return nil, ErrSessionForbidden
	}
	return &row, nil
}

// GetAdminSessionResult fetches the result for any session (admin use).
func (r *postgresRepository) GetAdminSessionResult(ctx context.Context, sessionID string) (*sessionResultRow, error) {
	q := sessionResultCTE + `
SELECT session_id, exam_id, exam_title, show_answers_mode, user_id, status,
       score_pct, passed, time_taken_seconds, submitted_at, attempt_number::int
FROM ranked WHERE session_id = $1`

	var row sessionResultRow
	if err := r.db.GetContext(ctx, &row, q, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("sessions: GetAdminSessionResult: %w", err)
	}
	return &row, nil
}

// GetSectionScores returns the per-section score aggregations for a session.
// When no section scores exist it returns an empty (non-nil) slice.
func (r *postgresRepository) GetSectionScores(ctx context.Context, sessionID string) ([]SectionScore, error) {
	const q = `
SELECT
    sqs.rule_id::text                          AS section_id,
    COALESCE(eqr.label, '')                    AS title,
    ROUND(
        100.0 * SUM(sqs.points_earned) / NULLIF(SUM(sqs.max_points), 0),
        2
    )                                          AS score_pct
FROM session_question_scores sqs
JOIN session_questions sq ON sq.session_id = sqs.session_id AND sq.question_id = sqs.question_id
LEFT JOIN exam_question_rules eqr ON eqr.id = sq.rule_id
WHERE sqs.session_id = $1
  AND sq.rule_id IS NOT NULL
GROUP BY sqs.rule_id, eqr.label
ORDER BY MIN(sq.sort_order)`

	type row struct {
		SectionID string  `db:"section_id"`
		Title     string  `db:"title"`
		ScorePct  float64 `db:"score_pct"`
	}

	rows, err := r.db.QueryxContext(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSectionScores: %w", err)
	}
	defer rows.Close()

	result := []SectionScore{} // non-nil so JSON serialises as [] not null
	for rows.Next() {
		var r row
		if err := rows.StructScan(&r); err != nil {
			return nil, fmt.Errorf("sessions: GetSectionScores: scan: %w", err)
		}
		result = append(result, SectionScore{
			SectionID: r.SectionID,
			Title:     r.Title,
			ScorePct:  r.ScorePct,
		})
	}
	return result, rows.Err()
}

// GetQuestionBreakdown returns per-question score rows ordered by sort_order.
func (r *postgresRepository) GetQuestionBreakdown(ctx context.Context, sessionID, locale string) ([]questionBreakdownRow, error) {
	const q = `
SELECT
    sqs.question_id,
    q.type                              AS question_type,
    COALESCE(qt.stem, '')               AS stem,
    sqs.points_earned,
    sqs.max_points,
    qt.explanation
FROM session_question_scores sqs
JOIN session_questions sq   ON sq.session_id = sqs.session_id AND sq.question_id = sqs.question_id
JOIN questions q            ON q.id = sqs.question_id
LEFT JOIN question_translations qt
    ON qt.question_id = sqs.question_id
   AND qt.locale = CASE WHEN $2 = '' THEN q.default_locale ELSE $2 END
WHERE sqs.session_id = $1
ORDER BY sq.sort_order`

	rows, err := r.db.QueryxContext(ctx, q, sessionID, locale)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetQuestionBreakdown: %w", err)
	}
	defer rows.Close()

	var result []questionBreakdownRow
	for rows.Next() {
		var r questionBreakdownRow
		if err := rows.StructScan(&r); err != nil {
			return nil, fmt.Errorf("sessions: GetQuestionBreakdown: scan: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// GetCorrectAnswerTexts returns correct option texts for a set of question IDs.
// The locale falls back to the question's default_locale when empty or not found.
func (r *postgresRepository) GetCorrectAnswerTexts(ctx context.Context, questionIDs []string, locale string) (map[string][]string, error) {
	if len(questionIDs) == 0 {
		return map[string][]string{}, nil
	}

	placeholders := make([]string, len(questionIDs))
	args := make([]interface{}, len(questionIDs)+1)
	args[0] = locale
	for i, id := range questionIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	q := fmt.Sprintf(`
SELECT ao.question_id::text, COALESCE(at.text, '') AS text
FROM answer_options ao
JOIN questions q ON q.id = ao.question_id
LEFT JOIN answer_translations at
    ON at.option_id = ao.id
   AND at.locale = CASE WHEN $1 = '' THEN q.default_locale ELSE $1 END
WHERE ao.is_correct = true
  AND ao.question_id IN (%s)
ORDER BY ao.sort_order`, strings.Join(placeholders, ","))

	type row struct {
		QuestionID string `db:"question_id"`
		Text       string `db:"text"`
	}

	rows, err := r.db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetCorrectAnswerTexts: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var r row
		if err := rows.StructScan(&r); err != nil {
			return nil, fmt.Errorf("sessions: GetCorrectAnswerTexts: scan: %w", err)
		}
		result[r.QuestionID] = append(result[r.QuestionID], r.Text)
	}
	return result, rows.Err()
}

// GetExamHistory returns paginated sessions for a user+exam pair.
// Only submitted/auto_submitted sessions are included (AC-6: excludes in_progress).
func (r *postgresRepository) GetExamHistory(ctx context.Context, examID, userID string, page, perPage int) ([]historyRow, int, error) {
	const countQ = `
SELECT COUNT(*) FROM exam_sessions
WHERE exam_id = $1 AND user_id = $2 AND status IN ('submitted', 'auto_submitted')`
	var total int
	if err := r.db.GetContext(ctx, &total, countQ, examID, userID); err != nil {
		return nil, 0, fmt.Errorf("sessions: GetExamHistory: count: %w", err)
	}

	const rowsQ = `
SELECT id AS session_id, started_at, submitted_at, score_pct,
       COALESCE(passed, false) AS passed, status
FROM exam_sessions
WHERE exam_id = $1 AND user_id = $2 AND status IN ('submitted', 'auto_submitted')
ORDER BY started_at DESC
LIMIT $3 OFFSET $4`

	offset := (page - 1) * perPage
	rows, err := r.db.QueryxContext(ctx, rowsQ, examID, userID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("sessions: GetExamHistory: query: %w", err)
	}
	defer rows.Close()

	var result []historyRow
	for rows.Next() {
		var row historyRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, fmt.Errorf("sessions: GetExamHistory: scan: %w", err)
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

// GetExamTitleByID returns the title of an exam or ErrExamNotFound.
func (r *postgresRepository) GetExamTitleByID(ctx context.Context, examID string) (string, error) {
	const q = `SELECT title FROM exams WHERE id = $1`
	var title string
	if err := r.db.GetContext(ctx, &title, q, examID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrExamNotFound
		}
		return "", fmt.Errorf("sessions: GetExamTitleByID: %w", err)
	}
	return title, nil
}
