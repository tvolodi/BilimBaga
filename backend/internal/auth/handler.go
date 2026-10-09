package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/bilimbaga/bilimbaga/internal/audit"
)

// Handler handles HTTP requests for the auth domain.
type Handler struct {
	svc    Service
	writer auditWriter
}

// auditWriter is the subset of *audit.Writer the handler uses (allows a fake in tests).
type auditWriter interface {
	Write(ctx context.Context, r *http.Request, action, entityType string, entityID *string, metadata any)
}

// NewHandler creates a new Handler backed by the given Service and audit Writer.
func NewHandler(svc Service, writer *audit.Writer) *Handler {
	return &Handler{svc: svc, writer: writer}
}

// apiResponse is the standard JSON envelope for all API responses.
type apiResponse struct {
	Data  any       `json:"data"`
	Error *apiError `json:"error"`
}

// apiError is the structured error body in the response envelope.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, data any, apiErr *apiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{Data: data, Error: apiErr})
}

func handleServiceError(w http.ResponseWriter, err error) {
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		writeJSON(w, svcErr.HTTPStatus, nil, &apiError{Code: svcErr.Code, Message: svcErr.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, nil, &apiError{
		Code:    "INTERNAL_ERROR",
		Message: "an unexpected error occurred",
	})
}

// clientIP extracts the caller's IP address, preferring X-Forwarded-For.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may be a comma-separated list; take the first entry.
		if ip := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, nil, &apiError{Code: "INVALID_BODY", Message: "invalid JSON body"})
		return
	}

	resp, cookie, err := h.svc.Login(r.Context(), &req, clientIP(r))
	if err != nil {
		h.writer.Write(r.Context(), r, "auth.login.failure", "user", nil, map[string]any{"email": req.Email})
		handleServiceError(w, err)
		return
	}

	h.writer.Write(r.Context(), r, "auth.login.success", "user", &resp.User.ID, map[string]any{"email": req.Email})
	http.SetCookie(w, cookie)
	writeJSON(w, http.StatusOK, resp, nil)
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, nil, &apiError{
			Code:    "INVALID_REFRESH_TOKEN",
			Message: "refresh token is invalid or expired",
		})
		return
	}

	resp, newCookie, err := h.svc.Refresh(r.Context(), cookie.Value, clientIP(r))
	if err != nil {
		handleServiceError(w, err)
		return
	}

	http.SetCookie(w, newCookie)
	writeJSON(w, http.StatusOK, resp, nil)
}

// Logout handles POST /api/v1/auth/logout.
// Returns 200 even if no cookie is present (AC-7).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var rawToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		rawToken = cookie.Value
	}

	clearCookie, err := h.svc.Logout(r.Context(), rawToken, clientIP(r))
	if err != nil {
		// Even on unexpected errors, we still clear the cookie and return success per AC-7.
		http.SetCookie(w, clearCookie)
		writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"}, nil)
		return
	}

	h.writer.Write(r.Context(), r, "auth.logout", "", nil, nil)
	http.SetCookie(w, clearCookie)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"}, nil)
}

// ChangePassword handles POST /api/v1/auth/change-password.
// Requires a valid Bearer JWT in the Authorization header.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, nil, &apiError{
			Code:    "UNAUTHORIZED",
			Message: "missing or invalid authorization header",
		})
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := h.svc.ParseAccessToken(tokenStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, nil, &apiError{
			Code:    "UNAUTHORIZED",
			Message: "invalid or expired access token",
		})
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, nil, &apiError{Code: "INVALID_BODY", Message: "invalid JSON body"})
		return
	}

	userID, err := claims.GetSubject()
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, nil, &apiError{
			Code:    "UNAUTHORIZED",
			Message: "invalid or expired access token",
		})
		return
	}

	if err := h.svc.ChangePassword(r.Context(), userID, &req, clientIP(r)); err != nil {
		handleServiceError(w, err)
		return
	}

	h.writer.Write(r.Context(), r, "auth.password_change", "user", &userID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"message": "password changed"}, nil)
}
