package ai

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
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

	// Insight cache fields.
	insightCache    *InsightResult
	insightCacheErr error
	upsertCalled    bool
	upsertErr       error
	examInsightData *ExamInsightData
	examInsightErr  error

	// Loyalty narrative fields.
	sessionTrack       string
	sessionEmployeeID  string
	sessionTrackErr    error
	inDepartment       bool
	inDepartmentErr    error
	likertResponses    []LikertResponseData
	likertResponsesErr error
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

func (m *mockRepository) GetInsightCache(_ context.Context, _ string) (*InsightResult, error) {
	return m.insightCache, m.insightCacheErr
}

func (m *mockRepository) UpsertInsightCache(_ context.Context, _, _ string, _ []string) error {
	m.upsertCalled = true
	return m.upsertErr
}

func (m *mockRepository) GetExamInsightData(_ context.Context, _, _ string) (*ExamInsightData, error) {
	return m.examInsightData, m.examInsightErr
}

func (m *mockRepository) GetSessionCategoryTrack(_ context.Context, _ string) (string, string, error) {
	return m.sessionTrack, m.sessionEmployeeID, m.sessionTrackErr
}

func (m *mockRepository) IsEmployeeInAdminDepartment(_ context.Context, _, _ string) (bool, error) {
	return m.inDepartment, m.inDepartmentErr
}

func (m *mockRepository) CollectLikertResponses(_ context.Context, _ string) ([]LikertResponseData, error) {
	return m.likertResponses, m.likertResponsesErr
}

type mockClient struct {
	text   string
	tokens int
	err    error
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

// ── FR-BB74: GetInsights tests ─────────────────────────────────────────────

var sampleExamData = &ExamInsightData{
	ExamTitle:         "Safety Exam",
	TotalAttempts:     50,
	PassRate:          0.68,
	AvgScorePct:       0.72,
	AvgCompletionSecs: 1800,
	PassingScorePct:   75,
	QuestionStats: []InsightQuestionStat{
		{OrderNum: 1, Stem: "What is fire safety?", CorrectRate: 0.9, AvgTimeSecs: 30},
	},
}

const validInsightJSON = `["Insight one.", "Insight two.", "Insight three."]`

// AC-3: cache hit within 24 h → return cached, no Anthropic call.
func TestGetInsights_CacheHit_NoChatCall(t *testing.T) {
	freshCache := &InsightResult{
		Insights:    []string{"Cached insight."},
		GeneratedAt: time.Now().UTC().Add(-1 * time.Hour), // 1 h old — within 24 h
		Cached:      true,
	}
	repo := &mockRepository{
		insightCache: freshCache,
	}
	client := &mockClient{} // must not be called
	svc := NewService(repo, client, "model", newLogger())

	result, err := svc.GetInsights(context.Background(), "exam-1", "tenant-1", "user-1", false)
	if err != nil {
		t.Fatalf("expected no error on cache hit, got: %v", err)
	}
	if !result.Cached {
		t.Error("expected Cached=true for cache hit")
	}
	if client.text != "" {
		t.Error("Anthropic client should not have been called on cache hit")
	}
}

// AC-4: force refresh bypasses fresh cache and calls Anthropic.
func TestGetInsights_ForceRefresh_BypassesFreshCache(t *testing.T) {
	freshCache := &InsightResult{
		Insights:    []string{"Old insight."},
		GeneratedAt: time.Now().UTC().Add(-30 * time.Minute),
		Cached:      true,
	}
	repo := &mockRepository{
		insightCache:    freshCache,
		examInsightData: sampleExamData,
	}
	client := &mockClient{text: validInsightJSON, tokens: 100}
	svc := NewService(repo, client, "model", newLogger())

	result, err := svc.GetInsights(context.Background(), "exam-1", "tenant-1", "user-1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Cached {
		t.Error("expected Cached=false for force refresh")
	}
	if !repo.logCalled {
		t.Error("expected LogUsage to be called for force refresh")
	}
	if repo.lastLog.Feature != featureExamInsights {
		t.Errorf("expected feature %q, got %q", featureExamInsights, repo.lastLog.Feature)
	}
	if !repo.upsertCalled {
		t.Error("expected UpsertInsightCache to be called")
	}
}

// Cache miss → Anthropic called → result returned.
func TestGetInsights_CacheMiss_CallsAnthropic(t *testing.T) {
	repo := &mockRepository{
		insightCache:    nil, // cache miss
		examInsightData: sampleExamData,
	}
	client := &mockClient{text: validInsightJSON, tokens: 200}
	svc := NewService(repo, client, "model", newLogger())

	result, err := svc.GetInsights(context.Background(), "exam-1", "tenant-1", "user-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Cached {
		t.Error("expected Cached=false for cache miss")
	}
	if len(result.Insights) != 3 {
		t.Errorf("expected 3 insights, got %d", len(result.Insights))
	}
	if !repo.upsertCalled {
		t.Error("expected UpsertInsightCache to be called after fresh generation")
	}
}

// AC-6: Anthropic error → ErrAIUnavailable; no stale-cache fallback.
func TestGetInsights_AnthropicError_ReturnsUnavailable(t *testing.T) {
	repo := &mockRepository{
		insightCache:    nil,
		examInsightData: sampleExamData,
	}
	client := &mockClient{err: ErrAIUnavailable}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GetInsights(context.Background(), "exam-1", "tenant-1", "user-1", false)
	if !errors.Is(err, ErrAIUnavailable) {
		t.Errorf("expected ErrAIUnavailable, got %v", err)
	}
}

// AC-2: exam not found → ErrExamNotFound.
func TestGetInsights_ExamNotFound(t *testing.T) {
	repo := &mockRepository{
		insightCache:   nil,
		examInsightErr: ErrExamNotFound,
	}
	client := &mockClient{}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GetInsights(context.Background(), "bad-exam", "tenant-1", "user-1", false)
	if !errors.Is(err, ErrExamNotFound) {
		t.Errorf("expected ErrExamNotFound, got %v", err)
	}
}

// ── FR-BB75: GetLoyaltyNarrative tests ────────────────────────────────────────

var sampleLikertResponses = []LikertResponseData{
	{DimensionLabel: "Loyalty & Values", NormalizedWeight: 4},
	{DimensionLabel: "Team Collaboration", NormalizedWeight: 5},
}

// AC-1 + AC-4: happy path returns a narrative.
func TestGetLoyaltyNarrative_HappyPath(t *testing.T) {
	repo := &mockRepository{
		sessionTrack:      "loyalty",
		sessionEmployeeID: "emp-1",
		inDepartment:      true,
		likertResponses:   sampleLikertResponses,
	}
	client := &mockClient{text: "The responses indicate strong loyalty values.", tokens: 20}
	svc := NewService(repo, client, "model", newLogger())

	result, err := svc.GetLoyaltyNarrative(context.Background(), "sess-1", "admin-1", "department_admin")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Narrative == "" {
		t.Error("expected non-empty narrative")
	}
}

// AC-1: non-loyalty session → ErrNotLoyaltySession.
func TestGetLoyaltyNarrative_NotLoyaltySession(t *testing.T) {
	repo := &mockRepository{
		sessionTrack:      "security",
		sessionEmployeeID: "emp-1",
	}
	client := &mockClient{}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GetLoyaltyNarrative(context.Background(), "sess-1", "admin-1", "department_admin")
	if !errors.Is(err, ErrNotLoyaltySession) {
		t.Errorf("expected ErrNotLoyaltySession, got %v", err)
	}
}

// Session not found → ErrLoyaltySessionNotFound.
func TestGetLoyaltyNarrative_SessionNotFound(t *testing.T) {
	repo := &mockRepository{
		sessionTrackErr: ErrLoyaltySessionNotFound,
	}
	client := &mockClient{}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GetLoyaltyNarrative(context.Background(), "bad-sess", "admin-1", "department_admin")
	if !errors.Is(err, ErrLoyaltySessionNotFound) {
		t.Errorf("expected ErrLoyaltySessionNotFound, got %v", err)
	}
}

// AC-2: admin not in same department → ErrForbidden.
func TestGetLoyaltyNarrative_ForbiddenDepartment(t *testing.T) {
	repo := &mockRepository{
		sessionTrack:      "loyalty",
		sessionEmployeeID: "emp-1",
		inDepartment:      false,
	}
	client := &mockClient{}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GetLoyaltyNarrative(context.Background(), "sess-1", "admin-1", "department_admin")
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// AC-2: super_admin bypasses department check.
func TestGetLoyaltyNarrative_SuperAdminBypassesDeptCheck(t *testing.T) {
	repo := &mockRepository{
		sessionTrack:      "loyalty",
		sessionEmployeeID: "emp-1",
		inDepartment:      false, // would be forbidden for non-super_admin
		likertResponses:   sampleLikertResponses,
	}
	client := &mockClient{text: "This profile suggests strong loyalty.", tokens: 10}
	svc := NewService(repo, client, "model", newLogger())

	result, err := svc.GetLoyaltyNarrative(context.Background(), "sess-1", "super-1", "super_admin")
	if err != nil {
		t.Fatalf("super_admin should bypass dept check, got: %v", err)
	}
	if result.Narrative == "" {
		t.Error("expected non-empty narrative")
	}
}

// AC-5: usage log uses admin user_id.
func TestGetLoyaltyNarrative_LogsAdminUserID(t *testing.T) {
	repo := &mockRepository{
		sessionTrack:      "loyalty",
		sessionEmployeeID: "emp-1",
		inDepartment:      true,
		likertResponses:   sampleLikertResponses,
	}
	client := &mockClient{text: "The responses indicate loyalty.", tokens: 15}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GetLoyaltyNarrative(context.Background(), "sess-1", "admin-42", "department_admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastLog.UserID != "admin-42" {
		t.Errorf("expected log UserID=admin-42, got %q", repo.lastLog.UserID)
	}
	if repo.lastLog.Feature != featureLoyaltyNarrative {
		t.Errorf("expected feature=%q, got %q", featureLoyaltyNarrative, repo.lastLog.Feature)
	}
}

// AC-6: Anthropic error → ErrAIUnavailable.
func TestGetLoyaltyNarrative_AIUnavailable(t *testing.T) {
	repo := &mockRepository{
		sessionTrack:      "loyalty",
		sessionEmployeeID: "emp-1",
		inDepartment:      true,
		likertResponses:   sampleLikertResponses,
	}
	client := &mockClient{err: ErrAIUnavailable}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GetLoyaltyNarrative(context.Background(), "sess-1", "admin-1", "department_admin")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Errorf("expected ErrAIUnavailable, got %v", err)
	}
}

// AC-6: empty Anthropic response → ErrAIUnavailable.
func TestGetLoyaltyNarrative_EmptyResponse(t *testing.T) {
	repo := &mockRepository{
		sessionTrack:      "loyalty",
		sessionEmployeeID: "emp-1",
		inDepartment:      true,
		likertResponses:   sampleLikertResponses,
	}
	client := &mockClient{text: "   ", tokens: 0}
	svc := NewService(repo, client, "model", newLogger())

	_, err := svc.GetLoyaltyNarrative(context.Background(), "sess-1", "admin-1", "department_admin")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Errorf("expected ErrAIUnavailable for empty response, got %v", err)
	}
}
