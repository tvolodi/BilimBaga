package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type auditEntry struct {
	action   string
	entityID *string
	actorID  string
	metadata any
}

type fakeAuditWriter struct{ entries []auditEntry }

func (f *fakeAuditWriter) Write(ctx context.Context, _ *http.Request, action, _ string, entityID *string, metadata any) {
	f.entries = append(f.entries, auditEntry{action: action, entityID: entityID, actorID: ctxkeys.UserIDFromCtx(ctx), metadata: metadata})
}

func newRecoveryHandler(svc Service) (*Handler, *fakeAuditWriter) {
	aw := &fakeAuditWriter{}
	return &Handler{svc: svc, writer: aw}, aw
}

func postJSON(h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, req)
	return w
}

// --- ForgotPassword (AC-1, AC-6) ---

func TestForgotPassword_ExistingAndUnknownEmail_IdenticalResponse(t *testing.T) {
	svc := &mockService{forgotFn: func(_ context.Context, req *ForgotPasswordRequest, _ string) (string, error) {
		if req.Email == "known@example.com" {
			return "user-1", nil
		}
		return "", nil
	}}
	h, aw := newRecoveryHandler(svc)

	known := postJSON(h.ForgotPassword, `{"email":"known@example.com"}`)
	unknown := postJSON(h.ForgotPassword, `{"email":"nobody@example.com"}`)

	assert.Equal(t, http.StatusOK, known.Code)
	assert.Equal(t, http.StatusOK, unknown.Code)
	assert.Equal(t, known.Body.String(), unknown.Body.String(), "responses must be byte-identical")

	data, apiErr := decodeEnvelope(t, known)
	assert.Nil(t, apiErr)
	var msg map[string]string
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.NotEmpty(t, msg["message"])

	// Only the resolvable user produces an audit entry, with the user as actor and no token.
	require.Len(t, aw.entries, 1)
	assert.Equal(t, "auth.password_reset_requested", aw.entries[0].action)
	assert.Equal(t, "user-1", *aw.entries[0].entityID)
	assert.Equal(t, "user-1", aw.entries[0].actorID)
	assert.Nil(t, aw.entries[0].metadata)
}

func TestForgotPassword_ValidationError_Returns422(t *testing.T) {
	svc := &mockService{forgotFn: func(context.Context, *ForgotPasswordRequest, string) (string, error) {
		return "", &ServiceError{Code: "VALIDATION_ERROR", Message: "a valid email is required", HTTPStatus: http.StatusUnprocessableEntity}
	}}
	h, aw := newRecoveryHandler(svc)

	w := postJSON(h.ForgotPassword, `{"email":"not-an-email"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
	assert.Empty(t, aw.entries)
}

func TestForgotPassword_MalformedJSON_Returns422(t *testing.T) {
	h, _ := newRecoveryHandler(&mockService{})
	w := postJSON(h.ForgotPassword, `{not json`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
}

func TestForgotPassword_InternalError_Returns500(t *testing.T) {
	svc := &mockService{forgotFn: func(context.Context, *ForgotPasswordRequest, string) (string, error) {
		return "", errors.New("db down")
	}}
	h, _ := newRecoveryHandler(svc)
	w := postJSON(h.ForgotPassword, `{"email":"a@b.co"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- ResetPassword (AC-4, AC-6) ---

func TestResetPassword_Success_Returns200AndAudits(t *testing.T) {
	svc := &mockService{resetFn: func(_ context.Context, req *ResetPasswordRequest, _ string) (string, error) {
		assert.Equal(t, "tok", req.Token)
		return "user-1", nil
	}}
	h, aw := newRecoveryHandler(svc)

	w := postJSON(h.ResetPassword, `{"token":"tok","new_password":"Str0ngPass"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	assert.Nil(t, apiErr)
	require.Len(t, aw.entries, 1)
	assert.Equal(t, "auth.password_reset_completed", aw.entries[0].action)
	assert.Equal(t, "user-1", *aw.entries[0].entityID)
	assert.Nil(t, aw.entries[0].metadata)
}

func TestResetPassword_InvalidToken_Returns400AndAuditsFailureWithoutToken(t *testing.T) {
	svc := &mockService{resetFn: func(context.Context, *ResetPasswordRequest, string) (string, error) {
		return "", &ServiceError{Code: "INVALID_TOKEN", Message: "reset link is invalid or has expired", HTTPStatus: http.StatusBadRequest}
	}}
	h, aw := newRecoveryHandler(svc)

	w := postJSON(h.ResetPassword, `{"token":"secret-token-value","new_password":"Str0ngPass"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INVALID_TOKEN", apiErr.Code)
	require.Len(t, aw.entries, 1)
	assert.Equal(t, "auth.password_reset_failed", aw.entries[0].action)
	assert.Nil(t, aw.entries[0].entityID)
	assert.Nil(t, aw.entries[0].metadata)
}

func TestResetPassword_WeakPassword_Returns422ValidationNoFailureAudit(t *testing.T) {
	svc := &mockService{resetFn: func(context.Context, *ResetPasswordRequest, string) (string, error) {
		return "", &ServiceError{Code: "VALIDATION_ERROR", Message: "weak", HTTPStatus: http.StatusUnprocessableEntity}
	}}
	h, aw := newRecoveryHandler(svc)

	w := postJSON(h.ResetPassword, `{"token":"tok","new_password":"weak"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
	assert.Empty(t, aw.entries)
}

func TestResetPassword_MalformedJSON_Returns422(t *testing.T) {
	h, _ := newRecoveryHandler(&mockService{})
	w := postJSON(h.ResetPassword, `nope`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResetPassword_InternalError_Returns500(t *testing.T) {
	svc := &mockService{resetFn: func(context.Context, *ResetPasswordRequest, string) (string, error) {
		return "", errors.New("db down")
	}}
	h, aw := newRecoveryHandler(svc)
	w := postJSON(h.ResetPassword, `{"token":"t","new_password":"Str0ngPass"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, aw.entries)
}
