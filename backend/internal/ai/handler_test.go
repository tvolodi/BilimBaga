package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
)

// ctxWithUserID injects a user ID into the context the same way auth middleware does.
func ctxWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxkeys.CtxUserID, userID)
}

// ---- Manual mock for Service ------------------------------------------------

type mockService struct {
	questions []DraftQuestion
	err       error
}

func (m *mockService) GenerateQuestions(_ context.Context, _ string, _ GenerateQuestionsRequest) ([]DraftQuestion, error) {
	return m.questions, m.err
}

func (m *mockService) GetInsights(_ context.Context, _, _, _ string, _ bool) (*InsightResult, error) {
	return nil, ErrAIUnavailable
}

func (m *mockService) GetLoyaltyNarrative(_ context.Context, _, _, _ string) (*LoyaltyNarrativeResult, error) {
	return nil, ErrAIUnavailable
}

// ---- Helpers ----------------------------------------------------------------

func makeRequest(t *testing.T, body interface{}, userID string) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/generate-questions", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	if userID != "" {
		r = r.WithContext(ctxWithUserID(r.Context(), userID))
	}
	return r
}

type apiEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeEnvelope(t *testing.T, body []byte) apiEnvelope {
	t.Helper()
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env
}

// ---- Tests ------------------------------------------------------------------

// 200 success path.
func TestHandler_GenerateQuestions_Success(t *testing.T) {
	svc := &mockService{
		questions: []DraftQuestion{
			{Type: "single_choice", Stem: "Test?", Difficulty: "easy"},
		},
	}
	h := NewHandler(svc)

	reqBody := map[string]interface{}{
		"category_id": "cat-1",
		"difficulty":  "easy",
		"count":       1,
	}
	r := makeRequest(t, reqBody, "user-1")
	w := httptest.NewRecorder()

	h.HandleGenerateQuestions(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error != nil {
		t.Errorf("expected no error in envelope, got %+v", env.Error)
	}
}

// 400 validation failure.
func TestHandler_GenerateQuestions_ValidationError(t *testing.T) {
	svc := &mockService{err: &wrappedValidationError{}}
	h := NewHandler(svc)

	r := makeRequest(t, map[string]interface{}{"category_id": "x", "difficulty": "bad", "count": 0}, "user-1")
	w := httptest.NewRecorder()

	h.HandleGenerateQuestions(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %+v", env.Error)
	}
}

// wrappedValidationError wraps ErrValidation so errors.Is works.
type wrappedValidationError struct{}

func (e *wrappedValidationError) Error() string { return ErrValidation.Error() }
func (e *wrappedValidationError) Is(target error) bool {
	return errors.Is(ErrValidation, target)
}
func (e *wrappedValidationError) Unwrap() error { return ErrValidation }

// 401 when no user in context.
func TestHandler_GenerateQuestions_NoAuth(t *testing.T) {
	h := NewHandler(&mockService{})

	r := makeRequest(t, map[string]interface{}{"category_id": "x", "difficulty": "easy", "count": 1}, "")
	w := httptest.NewRecorder()

	h.HandleGenerateQuestions(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// 429 rate limited.
func TestHandler_GenerateQuestions_RateLimited(t *testing.T) {
	svc := &mockService{err: ErrAIRateLimited}
	h := NewHandler(svc)

	r := makeRequest(t, map[string]interface{}{"category_id": "x", "difficulty": "easy", "count": 1}, "user-1")
	w := httptest.NewRecorder()

	h.HandleGenerateQuestions(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != "AI_RATE_LIMITED" {
		t.Errorf("expected AI_RATE_LIMITED, got %+v", env.Error)
	}
}

// 503 AI unavailable.
func TestHandler_GenerateQuestions_AIUnavailable(t *testing.T) {
	svc := &mockService{err: ErrAIUnavailable}
	h := NewHandler(svc)

	r := makeRequest(t, map[string]interface{}{"category_id": "x", "difficulty": "easy", "count": 1}, "user-1")
	w := httptest.NewRecorder()

	h.HandleGenerateQuestions(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != "AI_UNAVAILABLE" {
		t.Errorf("expected AI_UNAVAILABLE, got %+v", env.Error)
	}
}

// 400 for completely invalid JSON body.
func TestHandler_GenerateQuestions_InvalidBody(t *testing.T) {
	h := NewHandler(&mockService{})

	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/generate-questions",
		bytes.NewBufferString("not json"))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(ctxWithUserID(r.Context(), "user-1"))
	w := httptest.NewRecorder()

	h.HandleGenerateQuestions(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", w.Code)
	}
}

// insightsErrService returns a configurable error from GetInsights.
type insightsErrService struct {
	mockService
	err error
}

func (s *insightsErrService) GetInsights(_ context.Context, _, _, _ string, _ bool) (*InsightResult, error) {
	return nil, s.err
}

// ISS-093: unexpected errors yield 500 and are logged, not swallowed.
func TestHandler_GetInsights_UnexpectedErrorLoggedAnd500(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	h := NewHandler(&insightsErrService{err: errors.New("ai: boom: sql: Scan error")})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai/insights/e1", nil)
	r = r.WithContext(ctxWithUserID(r.Context(), "u1"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("examId", "e1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.HandleGetInsights(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if !strings.Contains(buf.String(), "Scan error") {
		t.Fatalf("error not logged, log=%q", buf.String())
	}
}
