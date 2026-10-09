package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

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
		writeJSON(w, http.StatusBadRequest, nil, &apiError{Code: "VALIDATION_ERROR", Message: "a valid email is required"})
		return
	}

	userID, err := h.svc.ForgotPassword(r.Context(), &req, clientIP(r))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if userID != "" {
		// Actor is the user the link was requested for; the token is never audited.
		ctx := context.WithValue(r.Context(), ctxkeys.CtxUserID, userID)
		h.writer.Write(ctx, r.WithContext(ctx), "auth.password_reset_requested", "user", &userID, nil)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": forgotPasswordMessage}, nil)
}

// ResetPassword handles POST /api/v1/auth/reset-password (public).
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRecoveryBody)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, nil, &apiError{Code: "VALIDATION_ERROR", Message: "invalid request body"})
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

	ctx := context.WithValue(r.Context(), ctxkeys.CtxUserID, userID)
	h.writer.Write(ctx, r.WithContext(ctx), "auth.password_reset_completed", "user", &userID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"message": "password has been reset"}, nil)
}
