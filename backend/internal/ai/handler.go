package ai

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
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
