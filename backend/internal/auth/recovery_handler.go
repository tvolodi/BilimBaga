package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// maxRecoveryBody bounds the JSON bodies of the public recovery endpoints.
const maxRecoveryBody = 8 << 10

// forgotPasswordMessage is the one response body returned for every valid request.
const forgotPasswordMessage = "If an account exists for that email, a password reset link has been sent."

// ForgotPassword handles POST /api/v1/auth/forgot-password (public).
// The response is identical whether or not the email belongs to an active user.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRecoveryBody)).Decode(&req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, nil, &apiError{Code: "VALIDATION_ERROR", Message: "a valid email is required"})
		return
	}

	started := time.Now()
	userID, err := h.svc.ForgotPassword(r.Context(), &req, clientIP(r))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	// Pad to a constant minimum so residual DB-work differences between known and unknown
	// emails are not observable (ISS-105), then answer before the audit write below.
	h.padForgot(started)
	writeJSON(w, http.StatusOK, map[string]string{"message": forgotPasswordMessage}, nil)
	if userID != "" {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Actor is the user the link was requested for; the token is never audited.
		ctx := context.WithValue(r.Context(), ctxkeys.CtxUserID, userID)
		h.writer.Write(ctx, r.WithContext(ctx), "auth.password_reset_requested", "user", &userID, nil)
	}
}

// forgotMinDuration is the minimum time a forgot-password request takes to answer.
const forgotMinDuration = 400 * time.Millisecond

// padForgot sleeps until forgotMinDuration has elapsed since started. A zero Handler
// (tests) pads nothing; NewHandler enables it.
func (h *Handler) padForgot(started time.Time) {
	if h.forgotMin <= 0 {
		return
	}
	sleep := h.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	if rest := h.forgotMin - time.Since(started); rest > 0 {
		sleep(rest)
	}
}

// ResetPassword handles POST /api/v1/auth/reset-password (public).
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRecoveryBody)).Decode(&req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, nil, &apiError{Code: "VALIDATION_ERROR", Message: "invalid request body"})
		return
	}

	userID, err := h.svc.ResetPassword(r.Context(), &req, clientIP(r))
	if err != nil {
		var svcErr *ServiceError
		if errors.As(err, &svcErr) && svcErr.Code == "INVALID_TOKEN" {
			h.writer.Write(r.Context(), r, "auth.password_reset_failed", "user", nil, nil)
		}
		handleServiceError(w, err)
		return
	}

	h.passwordChanged(userID)
	ctx := context.WithValue(r.Context(), ctxkeys.CtxUserID, userID)
	h.writer.Write(ctx, r.WithContext(ctx), "auth.password_reset_completed", "user", &userID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"message": "password has been reset"}, nil)
}
