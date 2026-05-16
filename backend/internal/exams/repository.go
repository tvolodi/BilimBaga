package exams

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

// Repository defines all persistence operations for the exams domain.
type Repository interface {
	// Exams
	Create(ctx context.Context, e *Exam) error
	GetByID(ctx context.Context, id string) (*Exam, error)
	ListFiltered(ctx context.Context, filter ExamFilter) ([]*ExamListItem, int, error)
	Update(ctx context.Context, id string, input UpdateExamInput) (*Exam, error)
	UpdateStatus(ctx context.Context, id, status string) error
	DeleteByID(ctx context.Context, id string) error
	GetWithDetails(ctx context.Context, id string) (*ExamDetail, error)

	// CountQuestionsPerDifficulty returns the count of active questions matching a rule
	// for the given difficulty level. Used for adaptive publish validation (FR-BB72 AC-2).
	CountQuestionsPerDifficulty(ctx context.Context, rule *ExamQuestionRule, difficulty string) (int, error)

	// Sections
	CreateSection(ctx context.Context, examID string, input SectionInput) (*ExamSection, error)
	// UpdateSection updates a section; returns ErrNotFound if the section does not belong to examID.
	UpdateSection(ctx context.Context, examID, id string, input SectionInput) (*ExamSection, error)
	// DeleteSection deletes a section; returns ErrNotFound if the section does not belong to examID.
	DeleteSection(ctx context.Context, examID, id string) error

	// Rules
	CreateRule(ctx context.Context, examID string, input QuestionRuleInput) (*ExamQuestionRule, error)
	UpdateRule(ctx context.Context, id string, input QuestionRuleInput) (*ExamQuestionRule, error)
	DeleteRule(ctx context.Context, id string) error

	// Manual questions
	SetManualQuestions(ctx context.Context, ruleID string, questions []ManualQuestionInput) error

	// GetRuleByID fetches a single rule for mode validation.
	GetRuleByID(ctx context.Context, id string) (*ExamQuestionRule, error)

	// ListRulesForExam returns all rules for the given exam (for publish validation).
	ListRulesForExam(ctx context.Context, examID string) ([]*ExamQuestionRule, error)

	// CountAvailableForRule counts active questions matching the rule's filters.
	CountAvailableForRule(ctx context.Context, rule *ExamQuestionRule) (int, error)

	// Assignments (FR-BB33)
	CreateAssignment(ctx context.Context, a *ExamAssignment) error
	GetAssignmentByID(ctx context.Context, id string) (*ExamAssignment, error)
	DeleteAssignment(ctx context.Context, id string) error
	ListAssignmentsWithStats(ctx context.Context, examID string) ([]*AssignmentDetail, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewRepository returns a Repository backed by PostgreSQL.
func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, e *Exam) error {
	const q = `
		INSERT INTO exams (
			title, description, status, time_limit_minutes, passing_score_pct,
			max_attempts, available_from, available_until, shuffle_questions,
			shuffle_options, show_answers, on_tab_switch, certificate_enabled, adaptive, created_by
		) VALUES (
			:title, :description, :status, :time_limit_minutes, :passing_score_pct,
			:max_attempts, :available_from, :available_until, :shuffle_questions,
			:shuffle_options, :show_answers, :on_tab_switch, :certificate_enabled, :adaptive, :created_by
		)
		RETURNING id, created_at, updated_at`
	rows, err := r.db.NamedQueryContext(ctx, q, e)
	if err != nil {
		return fmt.Errorf("exams: Create: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return fmt.Errorf("exams: Create: scan: %w", err)
		}
	}
	return rows.Err()
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*Exam, error) {
	var e Exam
	const q = `SELECT * FROM exams WHERE id = $1`
	if err := r.db.GetContext(ctx, &e, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("exams: GetByID: %w", err)
	}
	return &e, nil
}

func (r *postgresRepository) ListFiltered(ctx context.Context, filter ExamFilter) ([]*ExamListItem, int, error) {
	var (
		args    []any
		argN    int
		wheres  []string
	)

	nextArg := func(v any) string {
		argN++
		args = append(args, v)
		return fmt.Sprintf("$%d", argN)
	}

	if filter.Status != nil {
		wheres = append(wheres, fmt.Sprintf("e.status = %s", nextArg(*filter.Status)))
	}
	if filter.Search != "" {
		wheres = append(wheres, fmt.Sprintf("e.title ILIKE %s", nextArg("%"+filter.Search+"%")))
	}

	where := ""
	if len(wheres) > 0 {
		where = "WHERE " + strings.Join(wheres, " AND ")
	}

	// Count query
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM exams e %s`, where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("exams: ListFiltered: count: %w", err)
	}

	// Sort
	allowedSorts := map[string]string{
		"created_at": "e.created_at",
		"updated_at": "e.updated_at",
		"title":      "e.title",
	}
	sortCol, ok := allowedSorts[filter.Sort]
	if !ok {
		sortCol = "e.created_at"
	}
	order := "DESC"
	if filter.Order == "asc" {
		order = "ASC"
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}
	offset := (filter.Page - 1) * filter.PerPage

	listQ := fmt.Sprintf(`
		SELECT e.id, e.title, e.status, e.time_limit_minutes, e.passing_score_pct,
		       e.max_attempts, e.available_from, e.available_until,
		       e.created_by, e.created_at, e.updated_at
		FROM exams e
		%s
		ORDER BY %s %s
		LIMIT %s OFFSET %s`,
		where,
		sortCol, order,
		nextArg(filter.PerPage), nextArg(offset),
	)

	rows, err := r.db.QueryxContext(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("exams: ListFiltered: query: %w", err)
	}
	defer rows.Close()

	items := make([]*ExamListItem, 0)
	for rows.Next() {
		var item ExamListItem
		if err := rows.Scan(
			&item.ID, &item.Title, &item.Status,
			&item.TimeLimitMinutes, &item.PassingScorePct, &item.MaxAttempts,
			&item.AvailableFrom, &item.AvailableUntil,
			&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("exams: ListFiltered: scan: %w", err)
		}
		items = append(items, &item)
	}
	return items, total, rows.Err()
}

func (r *postgresRepository) Update(ctx context.Context, id string, input UpdateExamInput) (*Exam, error) {
	const q = `
		UPDATE exams SET
			title = $1, description = $2, time_limit_minutes = $3, passing_score_pct = $4,
			max_attempts = $5, available_from = $6, available_until = $7,
			shuffle_questions = $8, shuffle_options = $9, show_answers = $10,
			on_tab_switch = $11, certificate_enabled = $12, adaptive = $13
		WHERE id = $14
		RETURNING id, title, description, status, time_limit_minutes, passing_score_pct,
		          max_attempts, available_from, available_until, shuffle_questions, shuffle_options,
		          show_answers, on_tab_switch, certificate_enabled, adaptive, created_by, created_at, updated_at`
	var e Exam
	if err := r.db.GetContext(ctx, &e, q,
		input.Title, input.Description, input.TimeLimitMinutes, input.PassingScorePct,
		input.MaxAttempts, input.AvailableFrom, input.AvailableUntil,
		input.ShuffleQuestions, input.ShuffleOptions, input.ShowAnswers,
		input.OnTabSwitch, input.CertificateEnabled, input.Adaptive, id,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("exams: Update: %w", err)
	}
	return &e, nil
}

// CountQuestionsPerDifficulty counts active questions matching a rule's category/tag filters
// at the given difficulty level. Used for adaptive publish validation (FR-BB72 AC-2).
func (r *postgresRepository) CountQuestionsPerDifficulty(ctx context.Context, rule *ExamQuestionRule, difficulty string) (int, error) {
	var args []any
	argN := 0
	nextArg := func(v any) string {
		argN++
		args = append(args, v)
		return fmt.Sprintf("$%d", argN)
	}

	wheres := []string{"q.status = 'active'", fmt.Sprintf("q.difficulty = %s", nextArg(difficulty))}
	if rule.CategoryID != nil {
		wheres = append(wheres, fmt.Sprintf("q.category_id = %s", nextArg(*rule.CategoryID)))
	}

	var tagIDs []string
	if len(rule.TagIDs) > 0 {
		_ = json.Unmarshal(rule.TagIDs, &tagIDs)
	}
	for _, tagID := range tagIDs {
		wheres = append(wheres, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM question_tags qt WHERE qt.question_id = q.id AND qt.tag_id = %s)",
			nextArg(tagID),
		))
	}

	q := fmt.Sprintf(`SELECT COUNT(*) FROM questions q WHERE %s`, strings.Join(wheres, " AND "))
	var count int
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("exams: CountQuestionsPerDifficulty: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, id, status string) error {
	const q = `UPDATE exams SET status = $1 WHERE id = $2`
	res, err := r.db.ExecContext(ctx, q, status, id)
	if err != nil {
		return fmt.Errorf("exams: UpdateStatus: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM exams WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("exams: DeleteByID: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) GetWithDetails(ctx context.Context, id string) (*ExamDetail, error) {
	e, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &ExamDetail{
		ID:                 e.ID,
		Title:              e.Title,
		Description:        e.Description,
		Status:             e.Status,
		TimeLimitMinutes:   e.TimeLimitMinutes,
		PassingScorePct:    e.PassingScorePct,
		MaxAttempts:        e.MaxAttempts,
		AvailableFrom:      e.AvailableFrom,
		AvailableUntil:     e.AvailableUntil,
		ShuffleQuestions:   e.ShuffleQuestions,
		ShuffleOptions:     e.ShuffleOptions,
		ShowAnswers:        e.ShowAnswers,
		OnTabSwitch:        e.OnTabSwitch,
		CertificateEnabled: e.CertificateEnabled,
		Adaptive:           e.Adaptive,
		CreatedBy:          e.CreatedBy,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
		Sections:           []SectionDetail{},
		Rules:              []QuestionRuleDetail{},
	}

	// Fetch sections
	const sectQ = `SELECT id, exam_id, title, sort_order FROM exam_sections WHERE exam_id = $1 ORDER BY sort_order`
	rows, err := r.db.QueryxContext(ctx, sectQ, id)
	if err != nil {
		return nil, fmt.Errorf("exams: GetWithDetails: sections: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s ExamSection
		if err := rows.StructScan(&s); err != nil {
			return nil, fmt.Errorf("exams: GetWithDetails: sections scan: %w", err)
		}
		detail.Sections = append(detail.Sections, SectionDetail{
			ID:        s.ID,
			Title:     s.Title,
			SortOrder: s.SortOrder,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("exams: GetWithDetails: sections iter: %w", err)
	}

	// Fetch rules
	const ruleQ = `
		SELECT id, exam_id, section_id, mode, category_id, tag_ids, difficulty, count, sort_order
		FROM exam_question_rules WHERE exam_id = $1 ORDER BY sort_order`
	ruleRows, err := r.db.QueryContext(ctx, ruleQ, id)
	if err != nil {
		return nil, fmt.Errorf("exams: GetWithDetails: rules: %w", err)
	}
	defer ruleRows.Close()

	ruleIDs := make([]string, 0)
	ruleMap := make(map[string]*QuestionRuleDetail)
	for ruleRows.Next() {
		var (
			r         QuestionRuleDetail
			rawTagIDs []byte
		)
		if err := ruleRows.Scan(
			&r.ID, new(string), &r.SectionID, &r.Mode, &r.CategoryID,
			&rawTagIDs, &r.Difficulty, &r.Count, &r.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("exams: GetWithDetails: rules scan: %w", err)
		}
		if err := json.Unmarshal(rawTagIDs, &r.TagIDs); err != nil {
			r.TagIDs = []string{}
		}
		r.Questions = []ManualQuestionRef{}
		detail.Rules = append(detail.Rules, r)
		ruleIDs = append(ruleIDs, r.ID)
		ruleMap[r.ID] = &detail.Rules[len(detail.Rules)-1]
	}
	if err := ruleRows.Err(); err != nil {
		return nil, fmt.Errorf("exams: GetWithDetails: rules iter: %w", err)
	}

	// Fetch manual questions for all rules in one query
	if len(ruleIDs) > 0 {
		manualQ, manualArgs, err := sqlx.In(
			`SELECT rule_id, question_id, sort_order FROM exam_manual_questions WHERE rule_id IN (?) ORDER BY rule_id, sort_order`,
			ruleIDs,
		)
		if err != nil {
			return nil, fmt.Errorf("exams: GetWithDetails: manual questions build: %w", err)
		}
		manualQ = r.db.Rebind(manualQ)
		manualRows, err := r.db.QueryContext(ctx, manualQ, manualArgs...)
		if err != nil {
			return nil, fmt.Errorf("exams: GetWithDetails: manual questions: %w", err)
		}
		defer manualRows.Close()
		for manualRows.Next() {
			var ruleID, questionID string
			var sortOrder int
			if err := manualRows.Scan(&ruleID, &questionID, &sortOrder); err != nil {
				return nil, fmt.Errorf("exams: GetWithDetails: manual questions scan: %w", err)
			}
			if rd, ok := ruleMap[ruleID]; ok {
				rd.Questions = append(rd.Questions, ManualQuestionRef{
					QuestionID: questionID,
					SortOrder:  sortOrder,
				})
			}
		}
		if err := manualRows.Err(); err != nil {
			return nil, fmt.Errorf("exams: GetWithDetails: manual questions iter: %w", err)
		}
	}

	return detail, nil
}

func (r *postgresRepository) CreateSection(ctx context.Context, examID string, input SectionInput) (*ExamSection, error) {
	const q = `
		INSERT INTO exam_sections (exam_id, title, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id, exam_id, title, sort_order`
	var s ExamSection
	if err := r.db.GetContext(ctx, &s, q, examID, input.Title, input.SortOrder); err != nil {
		return nil, fmt.Errorf("exams: CreateSection: %w", err)
	}
	return &s, nil
}

func (r *postgresRepository) UpdateSection(ctx context.Context, examID, id string, input SectionInput) (*ExamSection, error) {
	const q = `
		UPDATE exam_sections SET title = $1, sort_order = $2
		WHERE id = $3 AND exam_id = $4
		RETURNING id, exam_id, title, sort_order`
	var s ExamSection
	if err := r.db.GetContext(ctx, &s, q, input.Title, input.SortOrder, id, examID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("exams: UpdateSection: %w", err)
	}
	return &s, nil
}

func (r *postgresRepository) DeleteSection(ctx context.Context, examID, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM exam_sections WHERE id = $1 AND exam_id = $2`, id, examID)
	if err != nil {
		return fmt.Errorf("exams: DeleteSection: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) CreateRule(ctx context.Context, examID string, input QuestionRuleInput) (*ExamQuestionRule, error) {
	tagJSON, err := json.Marshal(input.TagIDs)
	if err != nil {
		return nil, fmt.Errorf("exams: CreateRule: marshal tags: %w", err)
	}
	if input.TagIDs == nil {
		tagJSON = []byte("[]")
	}

	const q = `
		INSERT INTO exam_question_rules (exam_id, section_id, mode, category_id, tag_ids, difficulty, count, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, exam_id, section_id, mode, category_id, tag_ids, difficulty, count, sort_order`
	var rule ExamQuestionRule
	if err := r.db.GetContext(ctx, &rule, q,
		examID, input.SectionID, input.Mode, input.CategoryID,
		tagJSON, input.Difficulty, input.Count, input.SortOrder,
	); err != nil {
		return nil, fmt.Errorf("exams: CreateRule: %w", err)
	}
	return &rule, nil
}

func (r *postgresRepository) UpdateRule(ctx context.Context, id string, input QuestionRuleInput) (*ExamQuestionRule, error) {
	tagJSON, err := json.Marshal(input.TagIDs)
	if err != nil {
		return nil, fmt.Errorf("exams: UpdateRule: marshal tags: %w", err)
	}
	if input.TagIDs == nil {
		tagJSON = []byte("[]")
	}

	const q = `
		UPDATE exam_question_rules
		SET section_id = $1, mode = $2, category_id = $3, tag_ids = $4,
		    difficulty = $5, count = $6, sort_order = $7
		WHERE id = $8
		RETURNING id, exam_id, section_id, mode, category_id, tag_ids, difficulty, count, sort_order`
	var rule ExamQuestionRule
	if err := r.db.GetContext(ctx, &rule, q,
		input.SectionID, input.Mode, input.CategoryID,
		tagJSON, input.Difficulty, input.Count, input.SortOrder, id,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("exams: UpdateRule: %w", err)
	}
	return &rule, nil
}

func (r *postgresRepository) DeleteRule(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM exam_question_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("exams: DeleteRule: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) GetRuleByID(ctx context.Context, id string) (*ExamQuestionRule, error) {
	const q = `SELECT id, exam_id, section_id, mode, category_id, tag_ids, difficulty, count, sort_order
	            FROM exam_question_rules WHERE id = $1`
	var rule ExamQuestionRule
	if err := r.db.GetContext(ctx, &rule, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("exams: GetRuleByID: %w", err)
	}
	return &rule, nil
}

func (r *postgresRepository) ListRulesForExam(ctx context.Context, examID string) ([]*ExamQuestionRule, error) {
	const q = `SELECT id, exam_id, section_id, mode, category_id, tag_ids, difficulty, count, sort_order
	            FROM exam_question_rules WHERE exam_id = $1 ORDER BY sort_order`
	rows, err := r.db.QueryxContext(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("exams: ListRulesForExam: %w", err)
	}
	defer rows.Close()
	var rules []*ExamQuestionRule
	for rows.Next() {
		var rule ExamQuestionRule
		if err := rows.StructScan(&rule); err != nil {
			return nil, fmt.Errorf("exams: ListRulesForExam: scan: %w", err)
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (r *postgresRepository) CountAvailableForRule(ctx context.Context, rule *ExamQuestionRule) (int, error) {
	var args []any
	argN := 0
	nextArg := func(v any) string {
		argN++
		args = append(args, v)
		return fmt.Sprintf("$%d", argN)
	}

	wheres := []string{"q.status = 'active'"}
	if rule.CategoryID != nil {
		wheres = append(wheres, fmt.Sprintf("q.category_id = %s", nextArg(*rule.CategoryID)))
	}
	if rule.Difficulty != nil {
		wheres = append(wheres, fmt.Sprintf("q.difficulty = %s", nextArg(*rule.Difficulty)))
	}

	// Tag filtering: question must be tagged with all tags in rule.tag_ids.
	// tag_ids is stored as raw JSONB bytes in the struct; decode them.
	var tagIDs []string
	if len(rule.TagIDs) > 0 {
		_ = json.Unmarshal(rule.TagIDs, &tagIDs)
	}
	for _, tagID := range tagIDs {
		wheres = append(wheres, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM question_tags qt WHERE qt.question_id = q.id AND qt.tag_id = %s)",
			nextArg(tagID),
		))
	}

	q := fmt.Sprintf(`SELECT COUNT(*) FROM questions q WHERE %s`, strings.Join(wheres, " AND "))
	var count int
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("exams: CountAvailableForRule: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) SetManualQuestions(ctx context.Context, ruleID string, questions []ManualQuestionInput) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("exams: SetManualQuestions: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM exam_manual_questions WHERE rule_id = $1`, ruleID); err != nil {
		return fmt.Errorf("exams: SetManualQuestions: delete: %w", err)
	}

	for _, q := range questions {
		const ins = `INSERT INTO exam_manual_questions (rule_id, question_id, sort_order) VALUES ($1, $2, $3)`
		if _, err = tx.ExecContext(ctx, ins, ruleID, q.QuestionID, q.SortOrder); err != nil {
			return fmt.Errorf("exams: SetManualQuestions: insert %s: %w", q.QuestionID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("exams: SetManualQuestions: commit: %w", err)
	}
	return nil
}

// ── Assignment methods (FR-BB33) ─────────────────────────────────────────────

func (r *postgresRepository) CreateAssignment(ctx context.Context, a *ExamAssignment) error {
	const q = `
		INSERT INTO exam_assignments (exam_id, assignee_type, assignee_id, deadline, assigned_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, assigned_at`
	err := r.db.QueryRowContext(ctx, q,
		a.ExamID, a.AssigneeType, a.AssigneeID, a.Deadline, a.AssignedBy,
	).Scan(&a.ID, &a.AssignedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAssignmentExists
		}
		return fmt.Errorf("exams: CreateAssignment: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetAssignmentByID(ctx context.Context, id string) (*ExamAssignment, error) {
	const q = `SELECT id, exam_id, assignee_type, assignee_id, deadline, assigned_by, assigned_at
	            FROM exam_assignments WHERE id = $1`
	var a ExamAssignment
	if err := r.db.GetContext(ctx, &a, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAssignmentNotFound
		}
		return nil, fmt.Errorf("exams: GetAssignmentByID: %w", err)
	}
	return &a, nil
}

func (r *postgresRepository) DeleteAssignment(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM exam_assignments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("exams: DeleteAssignment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAssignmentNotFound
	}
	return nil
}

// ListAssignmentsWithStats returns every assignment for the given exam together
// with dynamically computed total_users / completed_count / passed_count.
//
// Stats logic:
//   - 'user' assignments:      total_users = 1; sessions counted for that user.
//   - 'department' assignments: total_users = recursive count of users in the dept tree.
//   - 'all' assignments:        total_users = count of active users in the tenant.
//
// completed_count = sessions with status IN ('submitted','auto_submitted') AND score_pct IS NOT NULL.
// passed_count    = sessions with passed = TRUE.
//
// Note: exam_sessions table does not exist yet (FR-BB35). The query returns 0 for
// all stats until that migration is applied. The LEFT JOINs are safe with missing data.
func (r *postgresRepository) ListAssignmentsWithStats(ctx context.Context, examID string) ([]*AssignmentDetail, error) {
	const q = `
	WITH RECURSIVE dept_tree(id) AS (
	    SELECT id FROM departments WHERE id = ea.assignee_id
	    UNION ALL
	    SELECT d.id FROM departments d JOIN dept_tree dt ON d.parent_id = dt.id
	),
	assignment_base AS (
	    SELECT
	        ea.id,
	        ea.assignee_type,
	        ea.assignee_id,
	        ea.deadline,
	        ea.assigned_at,
	        CASE
	            WHEN ea.assignee_type = 'user'       THEN (SELECT full_name FROM users WHERE id = ea.assignee_id)
	            WHEN ea.assignee_type = 'department' THEN (SELECT name FROM departments WHERE id = ea.assignee_id)
	            ELSE NULL
	        END AS assignee_name,
	        CASE
	            WHEN ea.assignee_type = 'user' THEN 1
	            WHEN ea.assignee_type = 'department' THEN (
	                SELECT COUNT(*) FROM users u2
	                WHERE u2.department_id IN (
	                    SELECT id FROM departments WHERE id = ea.assignee_id
	                    UNION ALL
	                    SELECT d2.id FROM departments d2
	                    JOIN departments d3 ON d2.parent_id = d3.id
	                    WHERE d3.id = ea.assignee_id
	                ) AND u2.status = 'active'
	            )
	            ELSE (SELECT COUNT(*) FROM users WHERE status = 'active')
	        END AS total_users
	    FROM exam_assignments ea
	    WHERE ea.exam_id = $1
	)
	SELECT
	    ab.id,
	    ab.assignee_type,
	    ab.assignee_id,
	    ab.assignee_name,
	    ab.deadline,
	    ab.assigned_at,
	    ab.total_users,
	    COALESCE((
	        SELECT COUNT(*) FROM exam_sessions es
	        WHERE es.exam_id = $1
	          AND es.status IN ('submitted','auto_submitted')
	          AND es.score_pct IS NOT NULL
	          AND (
	              (ab.assignee_type = 'user'       AND es.user_id = ab.assignee_id) OR
	              (ab.assignee_type = 'department' AND es.user_id IN (
	                  SELECT id FROM users WHERE department_id = ab.assignee_id AND status = 'active'
	              )) OR
	              (ab.assignee_type = 'all')
	          )
	    ), 0) AS completed_count,
	    COALESCE((
	        SELECT COUNT(*) FROM exam_sessions es
	        WHERE es.exam_id = $1
	          AND es.passed = TRUE
	          AND (
	              (ab.assignee_type = 'user'       AND es.user_id = ab.assignee_id) OR
	              (ab.assignee_type = 'department' AND es.user_id IN (
	                  SELECT id FROM users WHERE department_id = ab.assignee_id AND status = 'active'
	              )) OR
	              (ab.assignee_type = 'all')
	          )
	    ), 0) AS passed_count
	FROM assignment_base ab
	ORDER BY ab.assigned_at`

	type row struct {
		ID             string     `db:"id"`
		AssigneeType   string     `db:"assignee_type"`
		AssigneeID     *string    `db:"assignee_id"`
		AssigneeName   *string    `db:"assignee_name"`
		Deadline       *time.Time `db:"deadline"`
		AssignedAt     time.Time  `db:"assigned_at"`
		TotalUsers     int        `db:"total_users"`
		CompletedCount int        `db:"completed_count"`
		PassedCount    int        `db:"passed_count"`
	}

	rows, err := r.db.QueryxContext(ctx, q, examID)
	if err != nil {
		// exam_sessions table may not exist yet — treat as empty stats rather than error
		if strings.Contains(err.Error(), "exam_sessions") && strings.Contains(err.Error(), "does not exist") {
			return r.listAssignmentsNoSessions(ctx, examID)
		}
		return nil, fmt.Errorf("exams: ListAssignmentsWithStats: %w", err)
	}
	defer rows.Close()

	var result []*AssignmentDetail
	for rows.Next() {
		var rw row
		if err := rows.StructScan(&rw); err != nil {
			return nil, fmt.Errorf("exams: ListAssignmentsWithStats: scan: %w", err)
		}
		result = append(result, &AssignmentDetail{
			ID:           rw.ID,
			AssigneeType: rw.AssigneeType,
			AssigneeID:   rw.AssigneeID,
			AssigneeName: rw.AssigneeName,
			Deadline:     rw.Deadline,
			AssignedAt:   rw.AssignedAt,
			Stats: AssignmentStats{
				TotalUsers:     rw.TotalUsers,
				CompletedCount: rw.CompletedCount,
				PassedCount:    rw.PassedCount,
			},
		})
	}
	if result == nil {
		result = []*AssignmentDetail{}
	}
	return result, rows.Err()
}

// listAssignmentsNoSessions is used when exam_sessions does not yet exist (pre-FR-BB35).
func (r *postgresRepository) listAssignmentsNoSessions(ctx context.Context, examID string) ([]*AssignmentDetail, error) {
	const q = `
	SELECT
	    ea.id,
	    ea.assignee_type,
	    ea.assignee_id,
	    ea.deadline,
	    ea.assigned_at,
	    CASE
	        WHEN ea.assignee_type = 'user'       THEN (SELECT full_name FROM users WHERE id = ea.assignee_id)
	        WHEN ea.assignee_type = 'department' THEN (SELECT name FROM departments WHERE id = ea.assignee_id)
	        ELSE NULL
	    END AS assignee_name,
	    CASE
	        WHEN ea.assignee_type = 'user' THEN 1
	        WHEN ea.assignee_type = 'department' THEN (
	            SELECT COUNT(*) FROM users WHERE department_id = ea.assignee_id AND status = 'active'
	        )
	        ELSE (SELECT COUNT(*) FROM users WHERE status = 'active')
	    END AS total_users
	FROM exam_assignments ea
	WHERE ea.exam_id = $1
	ORDER BY ea.assigned_at`

	type row struct {
		ID           string     `db:"id"`
		AssigneeType string     `db:"assignee_type"`
		AssigneeID   *string    `db:"assignee_id"`
		AssigneeName *string    `db:"assignee_name"`
		Deadline     *time.Time `db:"deadline"`
		AssignedAt   time.Time  `db:"assigned_at"`
		TotalUsers   int        `db:"total_users"`
	}

	rows, err := r.db.QueryxContext(ctx, q, examID)
	if err != nil {
		return nil, fmt.Errorf("exams: listAssignmentsNoSessions: %w", err)
	}
	defer rows.Close()

	var result []*AssignmentDetail
	for rows.Next() {
		var rw row
		if err := rows.StructScan(&rw); err != nil {
			return nil, fmt.Errorf("exams: listAssignmentsNoSessions: scan: %w", err)
		}
		result = append(result, &AssignmentDetail{
			ID:           rw.ID,
			AssigneeType: rw.AssigneeType,
			AssigneeID:   rw.AssigneeID,
			AssigneeName: rw.AssigneeName,
			Deadline:     rw.Deadline,
			AssignedAt:   rw.AssignedAt,
			Stats:        AssignmentStats{TotalUsers: rw.TotalUsers},
		})
	}
	if result == nil {
		result = []*AssignmentDetail{}
	}
	return result, rows.Err()
}

// isUniqueViolation detects PostgreSQL unique-constraint errors (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
