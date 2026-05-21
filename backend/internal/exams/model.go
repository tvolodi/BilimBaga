package exams

import (
	"errors"
	"time"
)

var (
	ErrNotFound                      = errors.New("exam not found")
	ErrInvalidInput                  = errors.New("invalid input")
	ErrInvalidTransition             = errors.New("invalid status transition")
	ErrNotDraft                      = errors.New("exam is not in draft status")
	ErrRulesModeConflict             = errors.New("operation not allowed on random-mode rule")
	ErrRulesUnsatisfied              = errors.New("one or more question rules cannot be satisfied")
	ErrNotActive                     = errors.New("exam is not in active status")
	ErrAssignmentExists              = errors.New("assignment already exists")
	ErrAssignmentNotFound            = errors.New("assignment not found")
	ErrForbidden                     = errors.New("forbidden")
	ErrDeadlineInPast                = errors.New("deadline must be in the future")
	ErrInsufficientAdaptiveQuestions = errors.New("insufficient questions per difficulty for adaptive exam")
	ErrActiveSessionsExist           = errors.New("exam has active in-progress sessions")
)

// Exam is the core configuration row.
type Exam struct {
	ID                 string     `db:"id"                  json:"id"`
	Title              string     `db:"title"               json:"title"`
	Description        *string    `db:"description"         json:"description"`
	Status             string     `db:"status"              json:"status"`
	TimeLimitMinutes   int        `db:"time_limit_minutes"  json:"time_limit_minutes"`
	PassingScorePct    float64    `db:"passing_score_pct"   json:"passing_score_pct"`
	MaxAttempts        int        `db:"max_attempts"        json:"max_attempts"`
	AvailableFrom      *time.Time `db:"available_from"      json:"available_from"`
	AvailableUntil     *time.Time `db:"available_until"     json:"available_until"`
	ShuffleQuestions   bool       `db:"shuffle_questions"   json:"shuffle_questions"`
	ShuffleOptions     bool       `db:"shuffle_options"     json:"shuffle_options"`
	ShowAnswers        string     `db:"show_answers"        json:"show_answers"`
	OnTabSwitch        string     `db:"on_tab_switch"       json:"on_tab_switch"`
	CertificateEnabled bool       `db:"certificate_enabled" json:"certificate_enabled"`
	Adaptive           bool       `db:"adaptive"            json:"adaptive"`
	CreatedBy          string     `db:"created_by"          json:"created_by"`
	CreatedAt          time.Time  `db:"created_at"          json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"          json:"updated_at"`
}

// ExamSection is an optional grouping of question rules within an exam.
type ExamSection struct {
	ID        string  `db:"id"         json:"id"`
	ExamID    string  `db:"exam_id"    json:"exam_id"`
	Title     *string `db:"title"      json:"title"`
	SortOrder int     `db:"sort_order" json:"sort_order"`
}

// ExamQuestionRule describes how questions are selected for a section or the whole exam.
type ExamQuestionRule struct {
	ID         string  `db:"id"          json:"id"`
	ExamID     string  `db:"exam_id"     json:"exam_id"`
	SectionID  *string `db:"section_id"  json:"section_id"`
	Mode       string  `db:"mode"        json:"mode"`
	CategoryID *string `db:"category_id" json:"category_id"`
	TagIDs     []byte  `db:"tag_ids"     json:"tag_ids"` // raw JSONB
	Difficulty *string `db:"difficulty"  json:"difficulty"`
	Count      int     `db:"count"       json:"count"`
	SortOrder  int     `db:"sort_order"  json:"sort_order"`
}

// ExamManualQuestion links a specific question to a manual rule.
type ExamManualQuestion struct {
	RuleID     string `db:"rule_id"     json:"rule_id"`
	QuestionID string `db:"question_id" json:"question_id"`
	SortOrder  int    `db:"sort_order"  json:"sort_order"`
}

// ExamFilter holds parameters for the paginated exam list.
type ExamFilter struct {
	Status  *string
	Search  string
	Sort    string
	Order   string
	Page    int
	PerPage int
}

// ExamListItem is the summary row for the paginated list endpoint.
type ExamListItem struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Status           string     `json:"status"`
	TimeLimitMinutes int        `json:"time_limit_minutes"`
	PassingScorePct  float64    `json:"passing_score_pct"`
	MaxAttempts      int        `json:"max_attempts"`
	AvailableFrom    *time.Time `json:"available_from"`
	AvailableUntil   *time.Time `json:"available_until"`
	CreatedBy        string     `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ManualQuestionRef is a question entry in a manual rule.
type ManualQuestionRef struct {
	QuestionID string `json:"question_id"`
	SortOrder  int    `json:"sort_order"`
}

// QuestionRuleDetail is the full rule representation in the exam detail response.
type QuestionRuleDetail struct {
	ID         string              `json:"id"`
	SectionID  *string             `json:"section_id"`
	Mode       string              `json:"mode"`
	CategoryID *string             `json:"category_id"`
	TagIDs     []string            `json:"tag_ids"`
	Difficulty *string             `json:"difficulty"`
	Count      int                 `json:"count"`
	SortOrder  int                 `json:"sort_order"`
	Questions  []ManualQuestionRef `json:"questions,omitempty"`
}

// SectionDetail is the full section representation in the exam detail response.
type SectionDetail struct {
	ID        string  `json:"id"`
	Title     *string `json:"title"`
	SortOrder int     `json:"sort_order"`
}

// ExamDetail is the full exam including sections and rules.
type ExamDetail struct {
	ID                 string               `json:"id"`
	Title              string               `json:"title"`
	Description        *string              `json:"description"`
	Status             string               `json:"status"`
	TimeLimitMinutes   int                  `json:"time_limit_minutes"`
	PassingScorePct    float64              `json:"passing_score_pct"`
	MaxAttempts        int                  `json:"max_attempts"`
	AvailableFrom      *time.Time           `json:"available_from"`
	AvailableUntil     *time.Time           `json:"available_until"`
	ShuffleQuestions   bool                 `json:"shuffle_questions"`
	ShuffleOptions     bool                 `json:"shuffle_options"`
	ShowAnswers        string               `json:"show_answers"`
	OnTabSwitch        string               `json:"on_tab_switch"`
	CertificateEnabled bool                 `json:"certificate_enabled"`
	Adaptive           bool                 `json:"adaptive"`
	CreatedBy          string               `json:"created_by"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	Sections           []SectionDetail      `json:"sections"`
	Rules              []QuestionRuleDetail `json:"rules"`
}

// SectionInput is used when creating or updating a section.
type SectionInput struct {
	Title     *string `json:"title"`
	SortOrder int     `json:"sort_order"`
}

// ManualQuestionInput is used when adding a question to a manual rule.
type ManualQuestionInput struct {
	QuestionID string `json:"question_id"`
	SortOrder  int    `json:"sort_order"`
}

// QuestionRuleInput is used when creating or updating a question rule.
type QuestionRuleInput struct {
	SectionID  *string               `json:"section_id"`
	Mode       string                `json:"mode"`
	CategoryID *string               `json:"category_id"`
	TagIDs     []string              `json:"tag_ids"`
	Difficulty *string               `json:"difficulty"`
	Count      int                   `json:"count"`
	SortOrder  int                   `json:"sort_order"`
	Questions  []ManualQuestionInput `json:"questions"`
}

// RuleUnsatisfiedDetail describes a single rule that failed publish validation.
type RuleUnsatisfiedDetail struct {
	RuleID    string         `json:"rule_id"`
	Required  int            `json:"required"`
	Available int            `json:"available"`
	Filter    map[string]any `json:"filter"`
}

// PublishValidationError carries the full list of unsatisfied rules.
type PublishValidationError struct {
	Details []RuleUnsatisfiedDetail
}

func (e *PublishValidationError) Error() string { return ErrRulesUnsatisfied.Error() }
func (e *PublishValidationError) Unwrap() error { return ErrRulesUnsatisfied }

// CreateExamInput bundles all fields needed to create a new exam.
type CreateExamInput struct {
	Title              string
	Description        *string
	TimeLimitMinutes   int
	PassingScorePct    float64
	MaxAttempts        int
	AvailableFrom      *time.Time
	AvailableUntil     *time.Time
	ShuffleQuestions   bool
	ShuffleOptions     bool
	ShowAnswers        string
	OnTabSwitch        string
	CertificateEnabled bool
	Adaptive           bool
	CreatedBy          string
}

// UpdateExamInput bundles all updatable fields for an exam.
type UpdateExamInput struct {
	Title              string
	Description        *string
	TimeLimitMinutes   int
	PassingScorePct    float64
	MaxAttempts        int
	AvailableFrom      *time.Time
	AvailableUntil     *time.Time
	ShuffleQuestions   bool
	ShuffleOptions     bool
	ShowAnswers        string
	OnTabSwitch        string
	CertificateEnabled bool
	Adaptive           bool
}

// AdaptivePublishValidationError is returned when an adaptive exam has insufficient
// questions per difficulty level for publish validation (FR-BB72 AC-2).
type AdaptivePublishValidationError struct {
	RuleID     string `json:"rule_id"`
	Difficulty string `json:"difficulty"`
	Required   int    `json:"required"`
	Available  int    `json:"available"`
}

func (e *AdaptivePublishValidationError) Error() string {
	return ErrInsufficientAdaptiveQuestions.Error()
}
func (e *AdaptivePublishValidationError) Unwrap() error {
	return ErrInsufficientAdaptiveQuestions
}

// ── Assignment types ──────────────────────────────────────────────────────────

// ExamAssignment is a persisted assignment row.
type ExamAssignment struct {
	ID           string     `db:"id"            json:"id"`
	ExamID       string     `db:"exam_id"        json:"exam_id"`
	AssigneeType string     `db:"assignee_type"  json:"assignee_type"`
	AssigneeID   *string    `db:"assignee_id"    json:"assignee_id"`
	Deadline     *time.Time `db:"deadline"       json:"deadline"`
	AssignedBy   string     `db:"assigned_by"    json:"assigned_by"`
	AssignedAt   time.Time  `db:"assigned_at"    json:"assigned_at"`
}

// AssignmentStats holds computed completion counters for a single assignment.
type AssignmentStats struct {
	TotalUsers     int `json:"total_users"`
	CompletedCount int `json:"completed_count"`
	PassedCount    int `json:"passed_count"`
}

// AssignmentDetail is the list-endpoint representation of an assignment with stats.
type AssignmentDetail struct {
	ID           string          `json:"id"`
	AssigneeType string          `json:"assignee_type"`
	AssigneeID   *string         `json:"assignee_id"`
	AssigneeName *string         `json:"assignee_name"`
	Deadline     *time.Time      `json:"deadline"`
	AssignedAt   time.Time       `json:"assigned_at"`
	Stats        AssignmentStats `json:"stats"`
}

// CreateAssignmentInput is the service-layer input for creating an assignment.
type CreateAssignmentInput struct {
	ExamID       string
	AssigneeType string
	AssigneeID   *string
	Deadline     *time.Time
	AssignedBy   string
	CallerRole   string
	CallerDeptID string
}

// RuleEligibleCount holds the eligible question count for a single question rule.
type RuleEligibleCount struct {
	RuleID   string `json:"rule_id"`
	Eligible int    `json:"eligible"`
}

// GetEligibleCountsResponse is the data envelope for GET /exams/{id}/rules/eligible-counts.
type GetEligibleCountsResponse struct {
	Counts []RuleEligibleCount `json:"counts"`
}
