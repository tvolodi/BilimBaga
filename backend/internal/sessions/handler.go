package sessions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

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
		case errors.Is(err, ErrExamArchived):
			api.WriteError(w, http.StatusForbidden, "EXAM_ARCHIVED",
				"This exam has been archived and is no longer available.")
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

// GetSessionResult handles GET /api/v1/portal/sessions/:id/result (FR-BB41).
func (h *Handler) GetSessionResult(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := auth.UserIDFromCtx(r.Context())

	resp, err := h.svc.GetSessionResult(r.Context(), sessionID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
		case errors.Is(err, ErrSessionForbidden):
			api.WriteError(w, http.StatusForbidden, "SESSION_FORBIDDEN",
				"You do not have access to this session.")
		case errors.Is(err, ErrSessionInProgress):
			api.WriteError(w, http.StatusUnprocessableEntity, "SESSION_IN_PROGRESS",
				"Session result is not available while the session is in progress.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get session result")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// GetAdminSessionResult handles GET /api/v1/admin/sessions/:id/result (FR-BB41).
func (h *Handler) GetAdminSessionResult(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	resp, err := h.svc.GetAdminSessionResult(r.Context(), sessionID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
		case errors.Is(err, ErrSessionInProgress):
			api.WriteError(w, http.StatusUnprocessableEntity, "SESSION_IN_PROGRESS",
				"Session result is not available while the session is in progress.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get admin session result")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// GetExamHistory handles GET /api/v1/portal/exams/:id/history (FR-BB41).
func (h *Handler) GetExamHistory(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	userID := auth.UserIDFromCtx(r.Context())
	page, perPage := parsePagination(r)

	resp, err := h.svc.GetExamHistory(r.Context(), examID, userID, page, perPage)
	if err != nil {
		switch {
		case errors.Is(err, ErrExamNotFound):
			api.WriteError(w, http.StatusNotFound, "EXAM_NOT_FOUND", "Exam not found.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get exam history")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// parsePagination parses ?page= and ?per_page= query params with safe defaults.
// page defaults to 1, per_page defaults to 20 and is capped at 100.
func parsePagination(r *http.Request) (page, perPage int) {
	page = 1
	perPage = 20

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

// HandleListGradingQueue handles GET /api/v1/admin/grading (FR-BB42 AC-1/AC-2/AC-3).
func (h *Handler) HandleListGradingQueue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var examID *string
	if v := q.Get("exam_id"); v != "" {
		examID = &v
	}

	var dateFrom, dateTo *time.Time
	if v := q.Get("date_from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			api.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "date_from must be YYYY-MM-DD")
			return
		}
		dateFrom = &t
	}
	if v := q.Get("date_to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			api.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "date_to must be YYYY-MM-DD")
			return
		}
		dateTo = &t
	}

	page, perPage := parsePagination(r)

	resp, err := h.svc.ListGradingQueue(r.Context(), examID, dateFrom, dateTo, page, perPage)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to list grading queue")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// HandleGetGradingDetail handles GET /api/v1/admin/grading/:sessionId (FR-BB42 AC-4).
func (h *Handler) HandleGetGradingDetail(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")

	resp, err := h.svc.GetGradingDetail(r.Context(), sessionID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get grading detail")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// HandleGradeAnswer handles POST /api/v1/admin/grading/:sessionId/answers/:questionId (FR-BB42 AC-5/AC-6/AC-7/AC-8/AC-10).
func (h *Handler) HandleGradeAnswer(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	questionID := chi.URLParam(r, "questionId")
	graderID := auth.UserIDFromCtx(r.Context())
	tenantID := ctxkeys.TenantIDFromCtx(r.Context())
	actorIP := r.RemoteAddr

	var req GradeAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "request body is not valid JSON")
		return
	}

	resp, err := h.svc.GradeAnswer(r.Context(), sessionID, questionID, graderID, tenantID, actorIP, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidScore):
			api.WriteError(w, http.StatusUnprocessableEntity, "INVALID_SCORE", "score_pct must be between 0 and 100")
		case errors.Is(err, ErrSessionNotFound):
			api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to grade answer")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}
