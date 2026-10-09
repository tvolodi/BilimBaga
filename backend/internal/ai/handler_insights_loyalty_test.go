package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
)

// fnService is a Service whose insight/loyalty behaviour is injectable.
type fnService struct {
	mockService
	insights func(examID, tenantID, userID string, refresh bool) (*InsightResult, error)
	loyalty  func(sessionID, userID, role string) (*LoyaltyNarrativeResult, error)
}

func (s *fnService) GetInsights(_ context.Context, examID, tenantID, userID string, refresh bool) (*InsightResult, error) {
	return s.insights(examID, tenantID, userID, refresh)
}

func (s *fnService) GetLoyaltyNarrative(_ context.Context, sessionID, userID, role string) (*LoyaltyNarrativeResult, error) {
	return s.loyalty(sessionID, userID, role)
}

// paramRequest builds a GET request with one chi URL param. An empty userID
// means an unauthenticated request; an empty value means the param is absent.
func paramRequest(url, param, value, userID string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, url, nil)
	ctx := r.Context()
	if userID != "" {
		ctx = context.WithValue(ctx, ctxkeys.CtxUserID, userID)
		ctx = context.WithValue(ctx, ctxkeys.CtxRole, "department_admin")
		ctx = context.WithValue(ctx, ctxkeys.CtxTenantID, "public")
	}
	rctx := chi.NewRouteContext()
	if value != "" {
		rctx.URLParams.Add(param, value)
	}
	return r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
}

func TestHandler_GetInsights_Success_PassesArgsAndEnvelope(t *testing.T) {
	var gotExam, gotTenant, gotUser string
	var gotRefresh bool
	svc := &fnService{insights: func(e, tn, u string, rf bool) (*InsightResult, error) {
		gotExam, gotTenant, gotUser, gotRefresh = e, tn, u, rf
		return &InsightResult{Insights: []string{"a", "b", "c"}, GeneratedAt: time.Now().UTC(), Cached: true}, nil
	}}
	w := httptest.NewRecorder()
	NewHandler(svc).HandleGetInsights(w, paramRequest("/x?refresh=true", "examId", "exam-1", "u1"))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if gotExam != "exam-1" || gotTenant != "public" || gotUser != "u1" || !gotRefresh {
		t.Fatalf("args = %q %q %q %v", gotExam, gotTenant, gotUser, gotRefresh)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error != nil || len(env.Data) == 0 {
		t.Fatalf("envelope = %+v", env)
	}
	if body := w.Body.String(); !strings.Contains(body, `"cached":true`) || !strings.Contains(body, `"insights":["a","b","c"]`) {
		t.Fatalf("body = %s", body)
	}
}

func TestHandler_GetInsights_RefreshDefaultsFalse(t *testing.T) {
	gotRefresh := true
	svc := &fnService{insights: func(_, _, _ string, rf bool) (*InsightResult, error) {
		gotRefresh = rf
		return &InsightResult{}, nil
	}}
	w := httptest.NewRecorder()
	NewHandler(svc).HandleGetInsights(w, paramRequest("/x?refresh=1", "examId", "e", "u1"))
	if gotRefresh {
		t.Fatal("only refresh=true may force a refresh")
	}
}

func TestHandler_GetInsights_ErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"exam not found", ErrExamNotFound, http.StatusNotFound, "EXAM_NOT_FOUND"},
		{"wrapped exam not found", fmt.Errorf("wrap: %w", ErrExamNotFound), http.StatusNotFound, "EXAM_NOT_FOUND"},
		{"ai unavailable", ErrAIUnavailable, http.StatusServiceUnavailable, "AI_UNAVAILABLE"},
		{"other", errors.New("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fnService{insights: func(_, _, _ string, _ bool) (*InsightResult, error) { return nil, tc.err }}
			w := httptest.NewRecorder()
			NewHandler(svc).HandleGetInsights(w, paramRequest("/x", "examId", "e", "u1"))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			env := decodeEnvelope(t, w.Body.Bytes())
			if env.Error == nil || env.Error.Code != tc.code || string(env.Data) != "null" {
				t.Fatalf("envelope = %+v data=%s", env.Error, env.Data)
			}
		})
	}
}

func TestHandler_GetInsights_NoAuthAndMissingParam(t *testing.T) {
	called := false
	svc := &fnService{insights: func(_, _, _ string, _ bool) (*InsightResult, error) { called = true; return nil, nil }}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	h.HandleGetInsights(w, paramRequest("/x", "examId", "e", ""))
	if w.Code != http.StatusUnauthorized || decodeEnvelope(t, w.Body.Bytes()).Error.Code != "MISSING_TOKEN" {
		t.Fatalf("no auth: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	h.HandleGetInsights(w, paramRequest("/x", "examId", "", "u1"))
	if w.Code != http.StatusBadRequest || decodeEnvelope(t, w.Body.Bytes()).Error.Code != "INVALID_PARAM" {
		t.Fatalf("no param: %d %s", w.Code, w.Body.String())
	}
	if called {
		t.Fatal("service must not be called on rejected requests")
	}
}

func TestHandler_GetLoyaltyNarrative_Success(t *testing.T) {
	var gotSession, gotUser, gotRole string
	svc := &fnService{loyalty: func(s, u, r string) (*LoyaltyNarrativeResult, error) {
		gotSession, gotUser, gotRole = s, u, r
		return &LoyaltyNarrativeResult{Narrative: "A fine profile.", GeneratedAt: time.Now().UTC()}, nil
	}}
	w := httptest.NewRecorder()
	NewHandler(svc).GetLoyaltyNarrative(w, paramRequest("/x", "sessionId", "s1", "admin1"))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if gotSession != "s1" || gotUser != "admin1" || gotRole != "department_admin" {
		t.Fatalf("args = %q %q %q", gotSession, gotUser, gotRole)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error != nil || !strings.Contains(string(env.Data), "A fine profile.") {
		t.Fatalf("envelope = %+v data=%s", env.Error, env.Data)
	}
}

func TestHandler_GetLoyaltyNarrative_ErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"session not found", ErrLoyaltySessionNotFound, http.StatusNotFound, "SESSION_NOT_FOUND"},
		{"not loyalty", ErrNotLoyaltySession, http.StatusBadRequest, "NOT_A_LOYALTY_SESSION"},
		{"forbidden", ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{"ai unavailable", fmt.Errorf("x: %w", ErrAIUnavailable), http.StatusServiceUnavailable, "AI_UNAVAILABLE"},
		{"other", errors.New("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fnService{loyalty: func(_, _, _ string) (*LoyaltyNarrativeResult, error) { return nil, tc.err }}
			w := httptest.NewRecorder()
			NewHandler(svc).GetLoyaltyNarrative(w, paramRequest("/x", "sessionId", "s1", "a1"))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			env := decodeEnvelope(t, w.Body.Bytes())
			if env.Error == nil || env.Error.Code != tc.code || string(env.Data) != "null" {
				t.Fatalf("envelope = %+v data=%s", env.Error, env.Data)
			}
		})
	}
}

func TestHandler_GetLoyaltyNarrative_NoAuthAndMissingParam(t *testing.T) {
	svc := &fnService{loyalty: func(_, _, _ string) (*LoyaltyNarrativeResult, error) {
		t.Fatal("service must not be called")
		return nil, nil
	}}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	h.GetLoyaltyNarrative(w, paramRequest("/x", "sessionId", "s1", ""))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no auth: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.GetLoyaltyNarrative(w, paramRequest("/x", "sessionId", "", "a1"))
	if w.Code != http.StatusBadRequest || decodeEnvelope(t, w.Body.Bytes()).Error.Code != "INVALID_PARAM" {
		t.Fatalf("no param: %d %s", w.Code, w.Body.String())
	}
}

func TestHandler_GenerateQuestions_InternalError(t *testing.T) {
	h := NewHandler(&mockService{err: errors.New("db down")})
	w := httptest.NewRecorder()
	h.HandleGenerateQuestions(w, makeRequest(t, map[string]any{"count": 1}, "u1"))
	if w.Code != http.StatusInternalServerError || decodeEnvelope(t, w.Body.Bytes()).Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}
