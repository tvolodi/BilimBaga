package ai

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the AI domain.
type Handler struct {
	svc Service
}

// NewHandler returns a Handler backed by the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// HandleGenerateQuestions handles POST /api/v1/admin/ai/generate-questions.
//
// AC-1: Returns up to 10 draft questions.
// AC-2: JWT auth + examiner role enforced at router level via rbac.RequirePermission.
// AC-3: Anthropic errors → 503 AI_UNAVAILABLE.
// AC-5: Rate limit exceeded → 429 AI_RATE_LIMITED.
// AC-6: Validation errors → 400.
// AC-8: Drafts never inserted — returned in response only.
func (h *Handler) HandleGenerateQuestions(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	if userID == "" {
		api.WriteError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authentication required")
		return
	}

	var req GenerateQuestionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	questions, err := h.svc.GenerateQuestions(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		case errors.Is(err, ErrAIRateLimited):
			api.WriteError(w, http.StatusTooManyRequests, "AI_RATE_LIMITED",
				"You have reached the AI generation limit. Please try again in an hour.")
		case errors.Is(err, ErrAIUnavailable):
			api.WriteError(w, http.StatusServiceUnavailable, "AI_UNAVAILABLE",
				"AI service is temporarily unavailable. Please try again later.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"an unexpected error occurred")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  map[string]interface{}{"questions": questions},
		"error": nil,
	})
}

// HandleGetInsights handles GET /api/v1/admin/ai/insights/{examId}.
//
// AC-2: JWT auth + examiner role enforced at router level via rbac.RequirePermission.
// AC-3: Cache hit within 24 h returns cached result with "cached":true.
// AC-4: ?refresh=true bypasses cache and calls Anthropic.
// AC-6: Anthropic errors → 503 AI_UNAVAILABLE.
func (h *Handler) HandleGetInsights(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	if userID == "" {
		api.WriteError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authentication required")
		return
	}

	examID := chi.URLParam(r, "examId")
	if examID == "" {
		api.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "examId is required")
		return
	}

	tenantID := auth.TenantIDFromCtx(r.Context())
	forceRefresh := r.URL.Query().Get("refresh") == "true"

	result, err := h.svc.GetInsights(r.Context(), examID, tenantID, userID, forceRefresh)
	if err != nil {
		switch {
		case errors.Is(err, ErrExamNotFound):
			api.WriteError(w, http.StatusNotFound, "EXAM_NOT_FOUND", "exam not found or access denied")
		case errors.Is(err, ErrAIUnavailable):
			api.WriteError(w, http.StatusServiceUnavailable, "AI_UNAVAILABLE", "AI service unavailable")
		default:
			slog.Error("ai: get insights failed", "error", err, "examId", examID)
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  result,
		"error": nil,
	})
}

// GetLoyaltyNarrative handles GET /api/v1/admin/ai/loyalty-summary/{sessionId}.
//
// AC-1: Returns 400 NOT_A_LOYALTY_SESSION when session is not loyalty-track.
// AC-2: JWT auth + role ≥ department_admin enforced at router level.
//
//	Department access check enforced in service layer.
//
// AC-6: Anthropic errors → 503 AI_UNAVAILABLE.
func (h *Handler) GetLoyaltyNarrative(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	if userID == "" {
		api.WriteError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authentication required")
		return
	}

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		api.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "sessionId is required")
		return
	}

	role := auth.RoleFromCtx(r.Context())

	result, err := h.svc.GetLoyaltyNarrative(r.Context(), sessionID, userID, role)
	if err != nil {
		switch {
		case errors.Is(err, ErrLoyaltySessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
		case errors.Is(err, ErrNotLoyaltySession):
			api.WriteError(w, http.StatusBadRequest, "NOT_A_LOYALTY_SESSION", "this session does not belong to a loyalty-track exam")
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		case errors.Is(err, ErrAIUnavailable):
			api.WriteError(w, http.StatusServiceUnavailable, "AI_UNAVAILABLE", "AI service is temporarily unavailable")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  result,
		"error": nil,
	})
}
