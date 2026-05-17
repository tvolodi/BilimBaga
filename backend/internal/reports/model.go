// Package reports implements the dashboard metrics API (FR-BB51) and
// per-exam analytics API (FR-BB52).
package reports

import (
	"errors"
	"time"
)

// ErrNotFound is returned by repository methods when no matching row exists.
var ErrNotFound = errors.New("not found")

// DashboardMetrics is the top-level response returned by GET /api/v1/admin/dashboard.
type DashboardMetrics struct {
	CompletionRateByExam []*ExamCompletionRate `json:"completion_rate_by_exam"`
	OverdueEmployees     []*OverdueEmployee    `json:"overdue_employees"`
	RecentActivity       []*RecentActivity     `json:"recent_activity"`
	AvgScoreByTrack      TrackScores           `json:"avg_score_by_track"`
}

// ExamCompletionRate holds completion and pass-rate counts for one active exam.
type ExamCompletionRate struct {
	ExamID         string `json:"exam_id"`
	Title          string `json:"title"`
	AssignedCount  int    `json:"assigned_count"`
	CompletedCount int    `json:"completed_count"`
	PassedCount    int    `json:"passed_count"`
}

// OverdueEmployee represents an employee who has passed the assignment deadline
// without a passing session for the relevant exam.
type OverdueEmployee struct {
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	ExamTitle string    `json:"exam_title"`
	Deadline  time.Time `json:"deadline"`
}

// RecentActivity is one recently submitted or grading-pending session.
type RecentActivity struct {
	SessionID    string    `json:"session_id"`
	EmployeeName string    `json:"employee_name"`
	ExamTitle    string    `json:"exam_title"`
	ScorePct     *float64  `json:"score_pct"`
	Passed       bool      `json:"passed"`
	SubmittedAt  time.Time `json:"submitted_at"`
}

// TrackScores maps the three compliance tracks to their average score (or null).
// A nil pointer serialises as JSON null (AC-5 / note in spec).
type TrackScores struct {
	Security *float64 `json:"security"`
	Safety   *float64 `json:"safety"`
	Loyalty  *float64 `json:"loyalty"`
}

// ── Per-Exam Analytics (FR-BB52) ─────────────────────────────────────────────

// ExamAnalyticsResponse is the top-level response for GET /api/v1/admin/exams/{id}/analytics.
type ExamAnalyticsResponse struct {
	ExamID             string         `json:"exam_id"`
	ExamTitle          string         `json:"exam_title"`
	ScoreDistribution  []BucketCount  `json:"score_distribution"`
	PassRate           float64        `json:"pass_rate"`
	AvgScore           *float64       `json:"avg_score"`
	MedianScore        *float64       `json:"median_score"`
	TotalAttempts      int            `json:"total_attempts"`
	UniqueParticipants int            `json:"unique_participants"`
	PerQuestionStats   []QuestionStat `json:"per_question_stats"`
}

// BucketCount holds the session count for one score-distribution bucket.
type BucketCount struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// QuestionStat holds per-question analytics data.
type QuestionStat struct {
	QuestionID         string              `json:"question_id"`
	StemPreview        string              `json:"stem_preview"`
	CorrectRate        *float64            `json:"correct_rate"`
	AvgTimeSeconds     *float64            `json:"avg_time_seconds"`
	AnswerDistribution []AnswerOptionCount `json:"answer_distribution"`
}

// AnswerOptionCount holds the selection count for one answer option.
type AnswerOptionCount struct {
	OptionID    string `json:"option_id"`
	OptionText  string `json:"option_text"`
	SelectCount int    `json:"select_count"`
}

// ── FR-BB54: Export API ──────────────────────────────────────────────────────

// ExamResultRow is one data row in the exam results CSV export (AC-3).
// QuestionScores maps question position (1-indexed) to the score value (nil = empty cell, AC-4).
type ExamResultRow struct {
	EmployeeName     string
	Department       string
	StartedAt        string
	SubmittedAt      string
	ScorePct         string
	Passed           string
	TimeTakenSeconds string
	QuestionScores   []*float64 // nil = grading_pending, empty cell in CSV (AC-4)
}

// ExamQuestion is minimal question metadata for building CSV column headers.
type ExamQuestion struct {
	QuestionID string `db:"question_id"`
	Position   int    `db:"position"`
}

// ExamResultSessionRow is an intermediate scan result from the exam results streaming query.
type ExamResultSessionRow struct {
	SessionID        string   `db:"session_id"`
	EmployeeName     string   `db:"employee_name"`
	Department       string   `db:"department"`
	StartedAt        string   `db:"started_at"`
	SubmittedAt      string   `db:"submitted_at"`
	ScorePct         *float64 `db:"score_pct"`
	Passed           *bool    `db:"passed"`
	TimeTakenSeconds *int     `db:"time_taken_seconds"`
}

// UserRecordCSVRow is one data row in the user record CSV export (AC-5).
type UserRecordCSVRow struct {
	ExamTitle        string   `db:"exam_title"`
	StartedAt        string   `db:"started_at"`
	SubmittedAt      string   `db:"submitted_at"`
	ScorePct         *float64 `db:"score_pct"`
	Passed           *bool    `db:"passed"`
	TimeTakenSeconds *int     `db:"time_taken_seconds"`
	Status           string   `db:"status"`
}

// QuestionScore is a per-question score record fetched for CSV columns.
type QuestionScore struct {
	SessionID  string   `db:"session_id"`
	QuestionID string   `db:"question_id"`
	Score      *float64 `db:"score"`
}

// DashboardReportData is the structured input to GenerateDashboardPDF (AC-7).
type DashboardReportData struct {
	From            string
	To              string
	CompanyName     string
	LogoBase64      string
	CompletionRates []*ExamCompletionRate
	TopQuestions    []QuestionStat
	BottomQuestions []QuestionStat
}

// ── FR-BB53: Per-Employee Record & Progress ──────────────────────────────────

// SessionRecord is one row in an employee's exam session history.
type SessionRecord struct {
	SessionID         string     `json:"session_id" db:"session_id"`
	ExamID            string     `json:"exam_id" db:"exam_id"`
	ExamTitle         string     `json:"exam_title" db:"exam_title"`
	StartedAt         time.Time  `json:"started_at" db:"started_at"`
	SubmittedAt       *time.Time `json:"submitted_at" db:"submitted_at"`
	ScorePct          *float64   `json:"score_pct" db:"score_pct"`
	Passed            *bool      `json:"passed" db:"passed"`
	TimeTakenSeconds  *int       `json:"time_taken_seconds" db:"time_taken_seconds"`
	Status            string     `json:"status" db:"status"`
	CertificateID     *string    `json:"certificate_id" db:"certificate_id"`
	ExamCategoryTrack *string    `json:"exam_category_track" db:"exam_category_track"`
}

// TrackActivity holds the question count and last activity timestamp for one
// compliance track.
type TrackActivity struct {
	Track             string     `json:"track" db:"track"`
	QuestionsAnswered int        `json:"questions_answered" db:"questions_answered"`
	LastActivity      *time.Time `json:"last_activity" db:"last_activity"`
}

// ExamProgress holds an exam's pass status and attempt count for one employee.
// Track is used internally to group exams into TrackSummary; it is also
// serialised so the frontend can use a single flat list if needed.
type ExamProgress struct {
	ExamID   string `json:"exam_id" db:"exam_id"`
	Title    string `json:"title" db:"title"`
	Track    string `json:"track" db:"track"`
	Passed   *bool  `json:"passed" db:"passed"`
	Attempts int    `json:"attempts" db:"attempts"`
}

// TrackSummary aggregates activity and required exams for one compliance track.
type TrackSummary struct {
	Track             string         `json:"track"`
	QuestionsAnswered int            `json:"questions_answered"`
	LastActivity      *time.Time     `json:"last_activity"`
	RequiredExams     []ExamProgress `json:"required_exams"`
}

// UserRecordResponse is the top-level data object for
// GET /api/v1/admin/users/{id}/record.
type UserRecordResponse struct {
	UserID     string          `json:"user_id"`
	FullName   string          `json:"full_name"`
	Department string          `json:"department"`
	Sessions   []SessionRecord `json:"sessions"`
}

// UserProgressResponse is the top-level data object for
// GET /api/v1/admin/users/{id}/progress.
type UserProgressResponse struct {
	UserID   string         `json:"user_id"`
	FullName string         `json:"full_name"`
	Tracks   []TrackSummary `json:"tracks"`
}
