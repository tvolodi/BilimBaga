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
