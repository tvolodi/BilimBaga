package ai

import (
	"errors"
	"time"
)

// Sentinel errors returned by the service layer.
var (
	// ErrAIRateLimited is returned when the user exceeds the per-hour AI generation limit.
	ErrAIRateLimited = errors.New("ai: rate limit exceeded")

	// ErrAIUnavailable is returned when the Anthropic API is unreachable or returns an error.
	ErrAIUnavailable = errors.New("ai: upstream service unavailable")

	// ErrValidation is returned for invalid request inputs.
	ErrValidation = errors.New("ai: invalid input")
)

// GenerateQuestionsRequest is the body received from the HTTP client.
type GenerateQuestionsRequest struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"-"` // resolved from DB after validation
	Difficulty   string `json:"difficulty"`
	Count        int    `json:"count"`
	ContextText  string `json:"context_text"`
}

// DraftOption represents a single answer option in a draft question.
type DraftOption struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

// DraftQuestion is a single AI-generated draft question. It is never persisted automatically.
type DraftQuestion struct {
	Type        string        `json:"type"`
	Difficulty  string        `json:"difficulty"`
	Stem        string        `json:"stem"`
	Explanation string        `json:"explanation"`
	Options     []DraftOption `json:"options"`
	Tags        []string      `json:"tags"`
}

// GeneratedQuestionsResponse is what the Anthropic model returns (parsed from JSON text).
type GeneratedQuestionsResponse struct {
	Questions []DraftQuestion `json:"questions"`
}

// UsageLog is a record inserted into ai_usage_log for each successful AI call.
type UsageLog struct {
	UserID     string
	Feature    string
	TokensUsed int
	Model      string
	CreatedAt  time.Time
}

// ── FR-BB74: Performance Insight Summaries ───────────────────────────────────

// ExamInsightData holds anonymised aggregate statistics used to build the AI prompt.
// No employee names or user IDs are included — only aggregate values.
type ExamInsightData struct {
	ExamTitle         string
	TotalAttempts     int
	PassRate          float64 // 0.0–1.0
	AvgScorePct       float64 // 0.0–1.0
	AvgCompletionSecs int
	PassingScorePct   int
	QuestionStats     []InsightQuestionStat
}

// InsightQuestionStat holds per-question analytics used in the AI prompt.
type InsightQuestionStat struct {
	OrderNum    int
	Stem        string  // truncated to 100 chars
	CorrectRate float64 // 0.0–1.0
	AvgTimeSecs int
}

// InsightResult is the response returned by the GetInsights service method.
type InsightResult struct {
	Insights    []string  `json:"insights"`
	GeneratedAt time.Time `json:"generated_at"`
	Cached      bool      `json:"cached"`
}

// ErrExamNotFound is returned when the exam does not exist or belongs to a different tenant.
var ErrExamNotFound = errors.New("ai: exam not found")

// ── FR-BB75: Loyalty Profile Narrative ───────────────────────────────────────

// ErrLoyaltySessionNotFound is returned by GetSessionCategoryTrack when no session row exists.
var ErrLoyaltySessionNotFound = errors.New("ai: loyalty session not found")

// ErrNotLoyaltySession is returned when the session belongs to a non-loyalty-track exam.
var ErrNotLoyaltySession = errors.New("ai: not a loyalty session")

// ErrForbidden is returned when the admin does not have department access to the employee.
var ErrForbidden = errors.New("ai: forbidden")

const featureLoyaltyNarrative = "loyalty_narrative"

// LikertResponseData holds one polarity-inverted Likert answer for prompt construction.
// DimensionLabel comes from categories.name; no question text or PII is included.
type LikertResponseData struct {
	DimensionLabel   string // e.g. "Loyalty & Values" — from categories.name
	NormalizedWeight int    // 1–5 after polarity inversion
}

// LoyaltyPromptData is passed to BuildLoyaltyPrompt in internal/ai/prompt.go.
type LoyaltyPromptData struct {
	Responses []LikertResponseData
}

// LoyaltyNarrativeResult is returned by GetLoyaltyNarrative.
type LoyaltyNarrativeResult struct {
	Narrative   string    `json:"narrative"`
	GeneratedAt time.Time `json:"generated_at"`
}
