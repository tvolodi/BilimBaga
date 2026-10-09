package certificates

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock Service ──────────────────────────────────────────────────────────────

type mockSvc struct {
	getOrCreateFn           func(ctx context.Context, sessionID, callerID string, isAdmin bool) (*Certificate, TemplateSnapshot, error)
	getByVerificationCodeFn func(ctx context.Context, code string) (*VerifyResponse, error)
}

func (m *mockSvc) GetOrCreate(ctx context.Context, sessionID, callerID string, isAdmin bool) (*Certificate, TemplateSnapshot, error) {
	if m.getOrCreateFn != nil {
		return m.getOrCreateFn(ctx, sessionID, callerID, isAdmin)
	}
	return nil, TemplateSnapshot{}, errors.New("not configured")
}

func (m *mockSvc) GetByVerificationCode(ctx context.Context, code string) (*VerifyResponse, error) {
	if m.getByVerificationCodeFn != nil {
		return m.getByVerificationCodeFn(ctx, code)
	}
	return &VerifyResponse{Valid: false}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// newRouter wraps the handler in a Chi router that provides URL parameters.
func newRouter(handler *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/portal/sessions/{id}/certificate", handler.HandleGetPortalCertificate)
	r.Get("/admin/sessions/{id}/certificate", handler.HandleGetAdminCertificate)
	r.Get("/verify/{code}", handler.HandleVerifyCertificate)
	return r
}

// withUser injects a user ID into the request context (simulates auth middleware).
func withUser(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), ctxkeys.CtxUserID, userID)
	return r.WithContext(ctx)
}

func successCert() *Certificate {
	snap, _ := json.Marshal(TemplateSnapshot{CompanyName: "TestCorp"})
	return &Certificate{
		ID:               "cert-1",
		SessionID:        "sess-1",
		VerificationCode: "code-abc",
		IssuedAt:         time.Now().UTC(),
		EmployeeName:     "Alice",
		ExamTitle:        "Safety",
		ScorePct:         90.0,
		TemplateSnapshot: snap,
	}
}

// ── Portal certificate tests ──────────────────────────────────────────────────

// AC-6: successful portal request returns PDF content-type and disposition.
func TestHandleGetPortalCertificate_200_PDFHeaders(t *testing.T) {
	cert := successCert()
	h := NewHandler(&mockSvc{
		getOrCreateFn: func(_ context.Context, _, _ string, _ bool) (*Certificate, TemplateSnapshot, error) {
			return cert, TemplateSnapshot{CompanyName: "TestCorp"}, nil
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1/certificate", nil)
	req = withUser(req, testUserID)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "certificate-code-abc.pdf")
	assert.True(t, w.Body.Len() > 0)
}

// AC-1: portal endpoint returns 403 when session belongs to different user.
func TestHandleGetPortalCertificate_403_OnOwnershipMismatch(t *testing.T) {
	h := NewHandler(&mockSvc{
		getOrCreateFn: func(_ context.Context, _, _ string, _ bool) (*Certificate, TemplateSnapshot, error) {
			return nil, TemplateSnapshot{}, ErrNotOwner
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1/certificate", nil)
	req = withUser(req, "some-user")
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assertErrorCode(t, w.Body.String(), "SESSION_FORBIDDEN")
}

// AC-2: portal endpoint returns 422 EXAM_NOT_CERTIFIABLE.
func TestHandleGetPortalCertificate_422_ExamNotCertifiable(t *testing.T) {
	h := NewHandler(&mockSvc{
		getOrCreateFn: func(_ context.Context, _, _ string, _ bool) (*Certificate, TemplateSnapshot, error) {
			return nil, TemplateSnapshot{}, ErrNotCertifiable
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1/certificate", nil)
	req = withUser(req, testUserID)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assertErrorCode(t, w.Body.String(), "EXAM_NOT_CERTIFIABLE")
}

// AC-3: portal endpoint returns 422 SESSION_NOT_PASSED.
func TestHandleGetPortalCertificate_422_SessionNotPassed(t *testing.T) {
	h := NewHandler(&mockSvc{
		getOrCreateFn: func(_ context.Context, _, _ string, _ bool) (*Certificate, TemplateSnapshot, error) {
			return nil, TemplateSnapshot{}, ErrNotPassed
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1/certificate", nil)
	req = withUser(req, testUserID)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assertErrorCode(t, w.Body.String(), "SESSION_NOT_PASSED")
}

// AC-4: portal endpoint returns 422 SESSION_NOT_SUBMITTED.
func TestHandleGetPortalCertificate_422_SessionNotSubmitted(t *testing.T) {
	h := NewHandler(&mockSvc{
		getOrCreateFn: func(_ context.Context, _, _ string, _ bool) (*Certificate, TemplateSnapshot, error) {
			return nil, TemplateSnapshot{}, ErrNotSubmitted
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/portal/sessions/sess-1/certificate", nil)
	req = withUser(req, testUserID)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assertErrorCode(t, w.Body.String(), "SESSION_NOT_SUBMITTED")
}

// ── Admin certificate tests ───────────────────────────────────────────────────

// AC-8: admin endpoint returns PDF with no ownership restriction.
func TestHandleGetAdminCertificate_200_PDFHeaders(t *testing.T) {
	cert := successCert()
	h := NewHandler(&mockSvc{
		getOrCreateFn: func(_ context.Context, _, callerID string, isAdmin bool) (*Certificate, TemplateSnapshot, error) {
			assert.True(t, isAdmin, "admin handler must pass isAdmin=true")
			assert.Empty(t, callerID, "admin handler must pass empty callerID")
			return cert, TemplateSnapshot{CompanyName: "TestCorp"}, nil
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/admin/sessions/sess-1/certificate", nil)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
}

// ── Verify endpoint tests ─────────────────────────────────────────────────────

// AC-7: verify endpoint returns 200 with valid=true for known code.
func TestHandleVerifyCertificate_200_ValidTrue(t *testing.T) {
	issuedAt := time.Now().UTC()
	score := 90.0
	h := NewHandler(&mockSvc{
		getByVerificationCodeFn: func(_ context.Context, code string) (*VerifyResponse, error) {
			return &VerifyResponse{
				Valid:        true,
				EmployeeName: "Alice",
				ExamTitle:    "Safety",
				ScorePct:     &score,
				IssuedAt:     &issuedAt,
			}, nil
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/verify/some-code", nil)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var env struct {
		Data  *VerifyResponse `json:"data"`
		Error interface{}     `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	assert.Nil(t, env.Error)
	require.NotNil(t, env.Data)
	assert.True(t, env.Data.Valid)
	assert.Equal(t, "Alice", env.Data.EmployeeName)
}

// AC-7: verify endpoint returns 200 with valid=false for unknown code — never 404.
func TestHandleVerifyCertificate_200_ValidFalse_ForUnknownCode(t *testing.T) {
	h := NewHandler(&mockSvc{
		getByVerificationCodeFn: func(_ context.Context, _ string) (*VerifyResponse, error) {
			return &VerifyResponse{Valid: false}, nil
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/verify/unknown-code", nil)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var env struct {
		Data  *VerifyResponse `json:"data"`
		Error interface{}     `json:"error"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	assert.Nil(t, env.Error)
	require.NotNil(t, env.Data)
	assert.False(t, env.Data.Valid)
	assert.Empty(t, env.Data.EmployeeName)
}

// Internal/repository error → 500 standard envelope, error logged, no internals leaked.
func TestHandleVerifyCertificate_500_OnInternalError_LoggedNoLeak(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	h := NewHandler(&mockSvc{
		getByVerificationCodeFn: func(_ context.Context, _ string) (*VerifyResponse, error) {
			return nil, fmt.Errorf("certificates: GetByVerificationCode: %w", errors.New("pq: connection refused secret-host"))
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/verify/some-code", nil)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := w.Body.String()
	assertErrorCode(t, body, "INTERNAL_ERROR")
	assert.NotContains(t, body, "secret-host")
	assert.NotContains(t, body, "pq:")
	assert.Contains(t, body, `"data":null`)
	assert.Contains(t, buf.String(), "connection refused secret-host")
}

// ErrNotFound surfaced directly by a service is still a 200 valid=false.
func TestHandleVerifyCertificate_200_ValidFalse_OnErrNotFound(t *testing.T) {
	h := NewHandler(&mockSvc{
		getByVerificationCodeFn: func(_ context.Context, _ string) (*VerifyResponse, error) {
			return nil, fmt.Errorf("wrapped: %w", ErrNotFound)
		},
	}, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/verify/nope", nil)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"valid":false`)
}

// ── Assertion helper ──────────────────────────────────────────────────────────

func assertErrorCode(t *testing.T, body, expectedCode string) {
	t.Helper()
	var env struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(strings.NewReader(body)).Decode(&env))
	require.NotNil(t, env.Error)
	assert.Equal(t, expectedCode, env.Error.Code)
}
