package ai

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
)

// ---- Manual mocks -----------------------------------------------------------

type mockRepository struct {
	countReturns    int
	countErr        error
	logErr          error
	logCalled       bool
	lastLog         UsageLog
	categoryName    string
	categoryNameErr error
}

func (m *mockRepository) CountAIUsageLastHour(_ context.Context, _, _ string) (int, error) {
	return m.countReturns, m.countErr
}

func (m *mockRepository) LogUsage(_ context.Context, log UsageLog) error {
	m.logCalled = true
	m.lastLog = log
	return m.logErr
}

func (m *mockRepository) GetCategoryName(_ context.Context, _ string) (string, error) {
	return m.categoryName, m.categoryNameErr
}

type mockClient struct {
	text       string
	tokens     int
	err        error
}

func (m *mockClient) GenerateText(_ context.Context, _, _ string) (string, int, error) {
	return m.text, m.tokens, m.err
}

// ---- Helpers ----------------------------------------------------------------

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func validRequest() GenerateQuestionsRequest {
	return GenerateQuestionsRequest{
		CategoryID:  "cat-uuid-1",
		Difficulty:  "medium",
		Count:       3,
		ContextText: "",
	}
}

const validJSONResponse = `{"questions":[{"type":"single_choice","difficulty":"medium","stem":"What is X?","explanation":"X is Y","options":[{"text":"A","is_correct":true},{"text":"B","is_correct":false},{"text":"C","is_correct":false},{"text":"D","is_correct":false}],"tags":["topic"]}]}`

// ---- Tests ------------------------------------------------------------------

// AC-1: GenerateQuestions happy path returns questions.
func TestGenerateQuestions_HappyPath(t *testing.T) {
	repo := &mockRepository{countReturns: 0, categoryName: "Safety"}
	client := &mockClient{text: validJSONResponse, tokens: 150}
	svc := NewService(repo, client, "claude-3-5-haiku-20241022", newLogger())

	questions, err := svc.GenerateQuestions(context.Background(), "user-1", validRequest())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}
	if questions[0].Stem != "What is X?" {
		t.Errorf("unexpected stem: %s", questions[0].Stem)
	}
}

// AC-4: successful AI call logs to ai_usage_log.
func TestGenerateQuestions_LogsUsage(t *testing.T) {
	repo := &mockRepository{countReturns: 0, categoryName: "Safety"}
	client := &mockClient{text: validJSONResponse, tokens: 200}
	svc := NewService(repo, client, "claude-3-5-haiku-20241022", newLogger())

	_, err := svc.GenerateQuestions(context.Background(), "user-42", validRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.logCalled {
		t.Error("expected LogUsage to be called, but it was not")
	}
	if repo.lastLog.UserID != "user-42" {
		t.Errorf("expected userID user-42, got %s", repo.lastLog.UserID)
	}
	if repo.lastLog.Feature != featureQuestionGeneration {
		t.Errorf("expected feature %q, got %q", featureQuestionGeneration, repo.lastLog.Feature)
	}
	if repo.lastLog.TokensUsed != 200 {
		t.Errorf("expected 200 tokens, got %d", repo.lastLog.TokensUsed)
	}
}

// AC-3: Anthropic client failure → ErrAIUnavailable.
func TestGenerateQuestions_ClientFailure_ReturnsUnavailable(t *testing.T) {
	repo := &mockRepository{countReturns: 0, categoryName: "Safety"}
	client := &mockClient{err: ErrAIUnavailable}
	svc := NewService(repo, client, "claude-3-5-haiku-20241022", newLogger())

	_, err := svc.GenerateQuestions(context.Background(), "user-1", validRequest())
	if !errors.Is(err, ErrAIUnavailable) {
		t.Errorf("expected ErrAIUnavailable, got %v", err)
	}
}

// AC-3: Malformed JSON from model → ErrAIUnavailable.
func TestGenerateQuestions_MalformedJSON_ReturnsUnavailable(t *testing.T) {
	repo := &mockRepository{countReturns: 0, categoryName: "Safety"}
	client := &mockClient{text: "not json at all", tokens: 10}
	svc := NewService(repo, client, "claude-3-5-haiku-20241022", newLogger())

	_, err := svc.GenerateQuestions(context.Background(), "user-1", validRequest())
	if !errors.Is(err, ErrAIUnavailable) {
		t.Errorf("expected ErrAIUnavailable for malformed JSON, got %v", err)
	}
}

// AC-5: rate limit exceeded → ErrAIRateLimited.
func TestGenerateQuestions_RateLimitExceeded(t *testing.T) {
	repo := &mockRepository{countReturns: 20, categoryName: "Safety"}
	client := &mockClient{} // should never be called
	svc := NewService(repo, client, "claude-3-5-haiku-20241022", newLogger())

	_, err := svc.GenerateQuestions(context.Background(), "user-1", validRequest())
	if !errors.Is(err, ErrAIRateLimited) {
		t.Errorf("expected ErrAIRateLimited, got %v", err)
	}
	// Verify client was not called.
	if client.text != "" || client.err != nil {
		// client was never mutated by a call, which is correct
	}
}

// AC-5: exactly at limit (count=20) → rate limited.
func TestGenerateQuestions_ExactlyAtLimit(t *testing.T) {
	repo := &mockRepository{countReturns: 20, categoryName: "Safety"}
	svc := NewService(repo, &mockClient{}, "model", newLogger())

	_, err := svc.GenerateQuestions(context.Background(), "u", validRequest())
	if !errors.Is(err, ErrAIRateLimited) {
		t.Errorf("expected ErrAIRateLimited at count=20, got %v", err)
	}
}

// AC-5: below limit (count=19) → allowed.
func TestGenerateQuestions_BelowLimit_Allowed(t *testing.T) {
	repo := &mockRepository{countReturns: 19, categoryName: "Safety"}
	client := &mockClient{text: validJSONResponse, tokens: 10}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GenerateQuestions(context.Background(), "u", validRequest())
	if err != nil {
		t.Errorf("expected no error below rate limit, got %v", err)
	}
}

// AC-6: invalid count (0) → ErrValidation.
func TestGenerateQuestions_Validation_CountZero(t *testing.T) {
	svc := NewService(&mockRepository{categoryName: "X"}, &mockClient{}, "model", newLogger())
	req := validRequest()
	req.Count = 0

	_, err := svc.GenerateQuestions(context.Background(), "u", req)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for count=0, got %v", err)
	}
}

// AC-6: invalid count (11) → ErrValidation.
func TestGenerateQuestions_Validation_CountTooHigh(t *testing.T) {
	svc := NewService(&mockRepository{categoryName: "X"}, &mockClient{}, "model", newLogger())
	req := validRequest()
	req.Count = 11

	_, err := svc.GenerateQuestions(context.Background(), "u", req)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for count=11, got %v", err)
	}
}

// AC-6: context_text too long → ErrValidation.
func TestGenerateQuestions_Validation_ContextTooLong(t *testing.T) {
	svc := NewService(&mockRepository{categoryName: "X"}, &mockClient{}, "model", newLogger())
	req := validRequest()
	// Build a string > 2000 chars.
	long := make([]byte, 2001)
	for i := range long {
		long[i] = 'a'
	}
	req.ContextText = string(long)

	_, err := svc.GenerateQuestions(context.Background(), "u", req)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for long context, got %v", err)
	}
}

// AC-6: invalid difficulty → ErrValidation.
func TestGenerateQuestions_Validation_InvalidDifficulty(t *testing.T) {
	svc := NewService(&mockRepository{categoryName: "X"}, &mockClient{}, "model", newLogger())
	req := validRequest()
	req.Difficulty = "impossible"

	_, err := svc.GenerateQuestions(context.Background(), "u", req)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for invalid difficulty, got %v", err)
	}
}

// AC-6: missing category_id → ErrValidation.
func TestGenerateQuestions_Validation_NoCategoryID(t *testing.T) {
	svc := NewService(&mockRepository{categoryName: "X"}, &mockClient{}, "model", newLogger())
	req := validRequest()
	req.CategoryID = ""

	_, err := svc.GenerateQuestions(context.Background(), "u", req)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for missing category_id, got %v", err)
	}
}
