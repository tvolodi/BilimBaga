package sessions

import (
	"errors"
	"time"
)

var (
	ErrNotAssigned           = errors.New("exam not assigned to user")
	ErrExamNotActive         = errors.New("exam is not active")
	ErrExamOutsideWindow     = errors.New("exam outside availability window")
	ErrAttemptsExhausted     = errors.New("all allowed attempts exhausted")
	ErrSessionAlreadyOpen    = errors.New("active session already exists for this exam")
	ErrInsufficientQuestions = errors.New("not enough active questions to satisfy a random rule")

	// FR-BB37 errors.
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionForbidden     = errors.New("session does not belong to user")
	ErrSessionExpired       = errors.New("session has expired")
	ErrSessionNotActive     = errors.New("session status is not in_progress")
	ErrQuestionNotInSession = errors.New("question does not belong to session")
	ErrInvalidOption        = errors.New("selected option does not belong to question")
	ErrInvalidAnswerFormat  = errors.New("invalid answer format for question type")
	ErrNegativeTimeSpent    = errors.New("time_spent_seconds must be non-negative")

	// FR-BB38 errors.
	ErrInvalidEventType = errors.New("event type must be one of: tab_switch, blur, fullscreen_exit")

	// FR-BB39 errors.
	ErrSessionAlreadySubmitted = errors.New("session already submitted")

	// FR-BB41 errors.
	ErrSessionInProgress = errors.New("session result not available while in progress")
	ErrExamNotFound      = errors.New("exam not found")
)

// Session is the persisted exam session row.
type Session struct {
	ID          string     `db:"id"`
	ExamID      string     `db:"exam_id"`
	UserID      string     `db:"user_id"`
	Status      string     `db:"status"`
	Seed        int64      `db:"seed"`
	StartedAt   time.Time  `db:"started_at"`
	ExpiresAt   time.Time  `db:"expires_at"`
	SubmittedAt *time.Time `db:"submitted_at"`
	ScorePct    *float64   `db:"score_pct"`
	Passed      bool       `db:"passed"`
}

// SessionQuestion is one row in session_questions.
type SessionQuestion struct {
	SessionID         string  `db:"session_id"`
	QuestionID        string  `db:"question_id"`
	SortOrder         int     `db:"sort_order"`
	OptionsOrder      []byte  `db:"options_order"` // JSONB: []string of option UUIDs in shuffled order
	QuestionVersionID *string `db:"question_version_id"`
}

// SessionAnswer is one row in session_answers (FR-BB36 AC-3/AC-5/AC-10).
type SessionAnswer struct {
	ID                string  `db:"id"`
	SessionID         string  `db:"session_id"`
	QuestionID        string  `db:"question_id"`
	SelectedOptionIDs []byte  `db:"selected_option_ids"` // JSONB: []string of selected option UUIDs
	TextAnswer        *string `db:"text_answer"`
	SavedAt           string  `db:"saved_at"`
	TimeSpentSeconds  int     `db:"time_spent_seconds"`
}

// TabSwitchEvent is one row in tab_switch_events (FR-BB36 AC-4).
type TabSwitchEvent struct {
	ID          string `db:"id"`
	SessionID   string `db:"session_id"`
	OccurredAt  string `db:"occurred_at"`
	ActionTaken string `db:"action_taken"`
}

// SessionQuestionScore is one row in session_question_scores (FR-BB311).
type SessionQuestionScore struct {
	SessionID     string        `db:"session_id"`
	QuestionID    string        `db:"question_id"`
	Score         float64       `db:"score"`
	MaxScore      float64       `db:"max_score"`
	GradingStatus GradingStatus `db:"grading_status"`
}

// examConfig holds the fields we need from the exams table for session creation.
type examConfig struct {
	ID               string     `db:"id"`
	Status           string     `db:"status"`
	TimeLimitMinutes int        `db:"time_limit_minutes"`
	MaxAttempts      int        `db:"max_attempts"`
	AvailableFrom    *time.Time `db:"available_from"`
	AvailableUntil   *time.Time `db:"available_until"`
	ShuffleQuestions bool       `db:"shuffle_questions"`
	ShuffleOptions   bool       `db:"shuffle_options"`
	OnTabSwitch      string     `db:"on_tab_switch"`
}

// questionRule is a resolved row from exam_question_rules.
type questionRule struct {
	ID         string  `db:"id"`
	Mode       string  `db:"mode"`
	CategoryID *string `db:"category_id"`
	TagIDs     []byte  `db:"tag_ids"` // JSONB
	Difficulty *string `db:"difficulty"`
	Count      int     `db:"count"`
	SortOrder  int     `db:"sort_order"`
}

// poolQuestion is one candidate question row from the question pool.
type poolQuestion struct {
	ID   string `db:"id"`
	Type string `db:"type"`
}

// poolOption is one answer option row (ID only — text comes from translations).
type poolOption struct {
	OptionID  string `db:"option_id"`
	SortOrder int    `db:"sort_order"`
}

// SessionQuestionResponse is the question object returned in the session response.
type SessionQuestionResponse struct {
	ID        string                  `json:"id"`
	SortOrder int                     `json:"sort_order"`
	Stem      string                  `json:"stem"`
	Type      string                  `json:"type"`
	Options   []SessionOptionResponse `json:"options"`
}

// SessionOptionResponse is the option object returned in the session response (no is_correct).
type SessionOptionResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// CreateSessionResponse is the 201 response body.
type CreateSessionResponse struct {
	SessionID        string                    `json:"session_id"`
	ExamID           string                    `json:"exam_id"`
	StartedAt        time.Time                 `json:"started_at"`
	ExpiresAt        time.Time                 `json:"expires_at"`
	RemainingSeconds float64                   `json:"remaining_seconds"`
	Questions        []SessionQuestionResponse `json:"questions"`
}

// SaveAnswerInput is the request body for PUT /portal/sessions/:id/answers/:questionId.
type SaveAnswerInput struct {
	SelectedOptionIDs []string `json:"selected_option_ids"`
	TextAnswer        *string  `json:"text_answer"`
	TimeSpentSeconds  int      `json:"time_spent_seconds"`
}

// SaveAnswerResponse is the 200 response body for the upsert endpoint.
type SaveAnswerResponse struct {
	QuestionID       string    `json:"question_id"`
	SavedAt          time.Time `json:"saved_at"`
	RemainingSeconds float64   `json:"remaining_seconds"`
}

// SavedAnswer holds one answer entry in the resume response.
type SavedAnswer struct {
	SelectedOptionIDs []string  `json:"selected_option_ids"`
	TextAnswer        *string   `json:"text_answer"`
	TimeSpentSeconds  int       `json:"time_spent_seconds"`
	SavedAt           time.Time `json:"saved_at"`
}

// ResumeSessionResponse is the GET /portal/sessions/:id response body.
type ResumeSessionResponse struct {
	SessionID          string                    `json:"session_id"`
	ExamID             string                    `json:"exam_id"`
	ExamTitle          string                    `json:"exam_title"`
	CertificateEnabled bool                      `json:"certificate_enabled"`
	Status             string                    `json:"status"`
	StartedAt          time.Time                 `json:"started_at"`
	ExpiresAt          time.Time                 `json:"expires_at"`
	RemainingSeconds   float64                   `json:"remaining_seconds"`
	Questions          []SessionQuestionResponse `json:"questions"`
	Answers            map[string]SavedAnswer    `json:"answers"`
}

// ReportEventInput is the request body for POST /portal/sessions/:id/events (FR-BB38).
type ReportEventInput struct {
	Type string `json:"type"`
}

// ReportEventResponse is the 200 response body for the tab-switch event endpoint.
// For on_tab_switch='submit' the session fields are populated; otherwise only Warn/EventCount.
type ReportEventResponse struct {
	Warn       bool     `json:"warn"`
	EventCount int      `json:"event_count,omitempty"`
	SessionID  *string  `json:"session_id,omitempty"`
	Status     *string  `json:"status,omitempty"`
	ScorePct   *float64 `json:"score_pct,omitempty"`
	Passed     *bool    `json:"passed,omitempty"`
}

// SubmitSessionResponse is the 200 response body for POST /portal/sessions/:id/submit (FR-BB39).
type SubmitSessionResponse struct {
	SessionID   string    `json:"session_id"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
	ScorePct    *float64  `json:"score_pct"`
	Passed      *bool     `json:"passed"`
}

// SectionScore is one entry in the per-section scores list (FR-BB41).
type SectionScore struct {
	SectionID string  `json:"section_id"`
	Title     string  `json:"title"`
	ScorePct  float64 `json:"score_pct"`
}

// QuestionBreakdownItem is one item in the per-question breakdown (FR-BB41).
type QuestionBreakdownItem struct {
	QuestionID     string   `json:"question_id"`
	Stem           string   `json:"stem"`
	EmployeeAnswer []string `json:"employee_answer"`
	CorrectAnswer  []string `json:"correct_answer"`
	PointsEarned   float64  `json:"points_earned"`
	MaxPoints      float64  `json:"max_points"`
	Explanation    *string  `json:"explanation"`
}

// SessionResultResponse is the response for GET /portal/sessions/:id/result
// and GET /admin/sessions/:id/result (FR-BB41).
// PerQuestionBreakdown uses a pointer so omitempty removes the field entirely
// (not null) when show_answers = 'never' (AC-3).
type SessionResultResponse struct {
	SessionID            string                   `json:"session_id"`
	ExamID               string                   `json:"exam_id"`
	ExamTitle            string                   `json:"exam_title"`
	ScorePct             *float64                 `json:"score_pct"`
	Passed               bool                     `json:"passed"`
	TimeTakenSeconds     *int                     `json:"time_taken_seconds"`
	AttemptNumber        int                      `json:"attempt_number"`
	SubmittedAt          *time.Time               `json:"submitted_at"`
	ShowAnswersMode      string                   `json:"show_answers_mode"`
	PerSectionScores     []SectionScore           `json:"per_section_scores"`
	PerQuestionBreakdown *[]QuestionBreakdownItem `json:"per_question_breakdown,omitempty"`
}

// HistorySession is one session in the exam history list (FR-BB41).
type HistorySession struct {
	SessionID   string     `json:"session_id"`
	StartedAt   time.Time  `json:"started_at"`
	SubmittedAt *time.Time `json:"submitted_at"`
	ScorePct    *float64   `json:"score_pct"`
	Passed      bool       `json:"passed"`
	Status      string     `json:"status"`
}

// ExamHistoryMeta holds pagination metadata for the exam history response (FR-BB41).
type ExamHistoryMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// ExamHistoryResponse is the response for GET /portal/exams/:id/history (FR-BB41).
type ExamHistoryResponse struct {
	ExamID    string           `json:"exam_id"`
	ExamTitle string           `json:"exam_title"`
	Sessions  []HistorySession `json:"sessions"`
	Meta      ExamHistoryMeta  `json:"meta"`
}
