package sessions

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the sessions domain.
type Handler struct {
	svc Service
}

// NewHandler creates a new Handler backed by the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// CreateSession handles POST /api/v1/portal/exams/:id/sessions.
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	userID := auth.UserIDFromCtx(r.Context())
	deptID := auth.DepartmentIDFromCtx(r.Context())

	resp, err := h.svc.CreateSession(r.Context(), examID, userID, deptID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotAssigned):
			api.WriteError(w, http.StatusForbidden, "EXAM_NOT_ASSIGNED",
				"You do not have access to this exam.")
		case errors.Is(err, ErrExamNotActive):
			api.WriteError(w, http.StatusUnprocessableEntity, "EXAM_NOT_ACTIVE",
				"This exam is not currently active.")
		case errors.Is(err, ErrExamOutsideWindow):
			api.WriteError(w, http.StatusUnprocessableEntity, "EXAM_OUTSIDE_WINDOW",
				"This exam is not available at this time.")
		case errors.Is(err, ErrAttemptsExhausted):
			api.WriteError(w, http.StatusUnprocessableEntity, "ATTEMPTS_EXHAUSTED",
				"You have used all allowed attempts for this exam.")
		case errors.Is(err, ErrSessionAlreadyOpen):
			api.WriteError(w, http.StatusConflict, "SESSION_ALREADY_OPEN",
				"You already have an active session for this exam. Resume it instead.")
		case errors.Is(err, ErrInsufficientQuestions):
			api.WriteError(w, http.StatusUnprocessableEntity, "INSUFFICIENT_QUESTIONS",
				"This exam cannot be started because the question pool is too small.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL",
				"failed to create session")
		}
		return
	}

	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": resp, "error": nil})
}

// SaveAnswer handles PUT /api/v1/portal/sessions/:id/answers/:questionId.
func (h *Handler) SaveAnswer(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	questionID := chi.URLParam(r, "questionId")
	userID := auth.UserIDFromCtx(r.Context())

	var input SaveAnswerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "request body is not valid JSON")
		return
	}
	if input.SelectedOptionIDs == nil {
		input.SelectedOptionIDs = []string{}
	}

	resp, err := h.svc.SaveAnswer(r.Context(), sessionID, questionID, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrNegativeTimeSpent):
			api.WriteError(w, http.StatusBadRequest, "INVALID_TIME_SPENT",
				"time_spent_seconds must be non-negative")
		case errors.Is(err, ErrSessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND",
				"Session not found.")
		case errors.Is(err, ErrSessionForbidden):
			api.WriteError(w, http.StatusForbidden, "SESSION_FORBIDDEN",
				"You do not have access to this session.")
		case errors.Is(err, ErrSessionNotActive):
			api.WriteError(w, http.StatusUnprocessableEntity, "SESSION_NOT_ACTIVE",
				"Session is not in progress.")
		case errors.Is(err, ErrSessionExpired):
			api.WriteError(w, http.StatusUnprocessableEntity, "SESSION_EXPIRED",
				"Your exam session has expired.")
		case errors.Is(err, ErrQuestionNotInSession):
			api.WriteError(w, http.StatusNotFound, "QUESTION_NOT_IN_SESSION",
				"Question does not belong to this session.")
		case errors.Is(err, ErrInvalidOption):
			api.WriteError(w, http.StatusBadRequest, "INVALID_OPTION",
				"One or more selected option IDs do not belong to this question.")
		case errors.Is(err, ErrInvalidAnswerFormat):
			api.WriteError(w, http.StatusBadRequest, "INVALID_ANSWER_FORMAT",
				"Invalid answer format for this question type.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to save answer")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// ReportEvent handles POST /api/v1/portal/sessions/:id/events (FR-BB38).
func (h *Handler) ReportEvent(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := auth.UserIDFromCtx(r.Context())

	var input ReportEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "request body is not valid JSON")
		return
	}

	resp, err := h.svc.ReportEvent(r.Context(), sessionID, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEventType):
			api.WriteError(w, http.StatusBadRequest, "INVALID_EVENT_TYPE",
				"Event type must be one of: tab_switch, blur, fullscreen_exit.")
		case errors.Is(err, ErrSessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
		case errors.Is(err, ErrSessionForbidden):
			api.WriteError(w, http.StatusForbidden, "SESSION_FORBIDDEN",
				"You do not have access to this session.")
		case errors.Is(err, ErrSessionNotActive):
			api.WriteError(w, http.StatusUnprocessableEntity, "SESSION_NOT_ACTIVE",
				"Session is not in progress.")
		case errors.Is(err, ErrSessionExpired):
			api.WriteError(w, http.StatusUnprocessableEntity, "SESSION_EXPIRED",
				"Your exam session has expired.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to report event")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// SubmitSession handles POST /api/v1/portal/sessions/:id/submit (FR-BB39).
func (h *Handler) SubmitSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := auth.UserIDFromCtx(r.Context())
	tenantID := ctxkeys.TenantIDFromCtx(r.Context())
	actorIP := r.RemoteAddr

	resp, err := h.svc.SubmitSession(r.Context(), sessionID, userID, tenantID, actorIP)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
		case errors.Is(err, ErrSessionForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN",
				"You do not have access to this session.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to submit session")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// GetSessionState handles GET /api/v1/portal/sessions/:id.
func (h *Handler) GetSessionState(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := auth.UserIDFromCtx(r.Context())

	resp, err := h.svc.GetSessionState(r.Context(), sessionID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
		case errors.Is(err, ErrSessionForbidden):
			api.WriteError(w, http.StatusForbidden, "SESSION_FORBIDDEN",
				"You do not have access to this session.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get session state")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}
