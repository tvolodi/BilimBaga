package exams

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the exams domain.
type Handler struct {
	svc    Service
	writer *audit.Writer
}

// NewHandler creates a new Handler backed by the given Service and audit Writer.
func NewHandler(svc Service, writer *audit.Writer) *Handler {
	return &Handler{svc: svc, writer: writer}
}

// ── Request DTOs ─────────────────────────────────────────────────────────────

type createExamReq struct {
	Title              string     `json:"title"`
	Description        *string    `json:"description"`
	TimeLimitMinutes   int        `json:"time_limit_minutes"`
	PassingScorePct    float64    `json:"passing_score_pct"`
	MaxAttempts        int        `json:"max_attempts"`
	AvailableFrom      *time.Time `json:"available_from"`
	AvailableUntil     *time.Time `json:"available_until"`
	ShuffleQuestions   bool       `json:"shuffle_questions"`
	ShuffleOptions     bool       `json:"shuffle_options"`
	ShowAnswers        string     `json:"show_answers"`
	OnTabSwitch        string     `json:"on_tab_switch"`
	CertificateEnabled bool       `json:"certificate_enabled"`
	Adaptive           bool       `json:"adaptive"`
}

type updateExamReq struct {
	Title              string     `json:"title"`
	Description        *string    `json:"description"`
	TimeLimitMinutes   int        `json:"time_limit_minutes"`
	PassingScorePct    float64    `json:"passing_score_pct"`
	MaxAttempts        int        `json:"max_attempts"`
	AvailableFrom      *time.Time `json:"available_from"`
	AvailableUntil     *time.Time `json:"available_until"`
	ShuffleQuestions   bool       `json:"shuffle_questions"`
	ShuffleOptions     bool       `json:"shuffle_options"`
	ShowAnswers        string     `json:"show_answers"`
	OnTabSwitch        string     `json:"on_tab_switch"`
	CertificateEnabled bool       `json:"certificate_enabled"`
	Adaptive           bool       `json:"adaptive"`
}

type statusTransitionReq struct {
	Status string `json:"status"`
}

type sectionReq struct {
	Title     *string `json:"title"`
	SortOrder int     `json:"sort_order"`
}

type manualQuestionReq struct {
	QuestionID string `json:"question_id"`
	SortOrder  int    `json:"sort_order"`
}

type ruleReq struct {
	SectionID  *string             `json:"section_id"`
	Mode       string              `json:"mode"`
	CategoryID *string             `json:"category_id"`
	TagIDs     []string            `json:"tag_ids"`
	Difficulty *string             `json:"difficulty"`
	Count      int                 `json:"count"`
	SortOrder  int                 `json:"sort_order"`
	Questions  []manualQuestionReq `json:"questions"`
}

type setManualQuestionsReq struct {
	Questions []manualQuestionReq `json:"questions"`
}

// ── Validation helpers ───────────────────────────────────────────────────────

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func writeValidationErrors(w http.ResponseWriter, errs []fieldError) {
	api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"data": nil,
		"error": map[string]any{
			"code":    "ERR_VALIDATION",
			"message": "validation failed",
			"fields":  errs,
		},
	})
}

var (
	validShowAnswers = map[string]bool{"never": true, "after_completion": true, "after_all_attempts": true}
	validTabSwitch   = map[string]bool{"log": true, "warn": true, "submit": true}
)

func validateCreateExam(req createExamReq) []fieldError {
	var errs []fieldError
	if req.Title == "" {
		errs = append(errs, fieldError{"title", "title is required"})
	}
	if req.TimeLimitMinutes <= 0 {
		errs = append(errs, fieldError{"time_limit_minutes", "time_limit_minutes must be greater than 0"})
	}
	if req.PassingScorePct < 0 || req.PassingScorePct > 100 {
		errs = append(errs, fieldError{"passing_score_pct", "passing_score_pct must be between 0 and 100"})
	}
	if req.MaxAttempts <= 0 {
		errs = append(errs, fieldError{"max_attempts", "max_attempts must be greater than 0"})
	}
	if req.ShowAnswers != "" && !validShowAnswers[req.ShowAnswers] {
		errs = append(errs, fieldError{"show_answers", "show_answers must be one of: never, after_completion, after_all_attempts"})
	}
	if req.OnTabSwitch != "" && !validTabSwitch[req.OnTabSwitch] {
		errs = append(errs, fieldError{"on_tab_switch", "on_tab_switch must be one of: log, warn, submit"})
	}
	if req.AvailableFrom != nil && req.AvailableUntil != nil && !req.AvailableFrom.Before(*req.AvailableUntil) {
		errs = append(errs, fieldError{"available_until", "available_until must be after available_from"})
	}
	return errs
}

func validateRule(req ruleReq) []fieldError {
	var errs []fieldError
	if req.Mode != "manual" && req.Mode != "random" {
		errs = append(errs, fieldError{"mode", "mode must be one of: manual, random"})
	}
	if req.Count <= 0 {
		errs = append(errs, fieldError{"count", "count must be greater than 0"})
	}
	return errs
}

func actorFromCtx(r *http.Request) string {
	return auth.UserIDFromCtx(r.Context())
}

func parseIntParam(r *http.Request, name string, defaultVal int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// List returns a paginated list of exams.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ExamFilter{
		Page:    parseIntParam(r, "page", 1),
		PerPage: parseIntParam(r, "per_page", 20),
		Search:  r.URL.Query().Get("search"),
		Sort:    r.URL.Query().Get("sort"),
		Order:   r.URL.Query().Get("order"),
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}

	items, total, err := h.svc.ListExams(r.Context(), filter)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to list exams")
		return
	}
	if items == nil {
		items = []*ExamListItem{}
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"items": items,
			"meta": map[string]int{
				"page":     filter.Page,
				"per_page": filter.PerPage,
				"total":    total,
			},
		},
		"error": nil,
	})
}

// Create creates a new exam in draft status.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createExamReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	if errs := validateCreateExam(req); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	// Default enum values
	showAnswers := req.ShowAnswers
	if showAnswers == "" {
		showAnswers = "never"
	}
	onTabSwitch := req.OnTabSwitch
	if onTabSwitch == "" {
		onTabSwitch = "log"
	}
	maxAttempts := req.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 1
	}

	exam, err := h.svc.CreateExam(r.Context(), CreateExamInput{
		Title:              req.Title,
		Description:        req.Description,
		TimeLimitMinutes:   req.TimeLimitMinutes,
		PassingScorePct:    req.PassingScorePct,
		MaxAttempts:        maxAttempts,
		AvailableFrom:      req.AvailableFrom,
		AvailableUntil:     req.AvailableUntil,
		ShuffleQuestions:   req.ShuffleQuestions,
		ShuffleOptions:     req.ShuffleOptions,
		ShowAnswers:        showAnswers,
		OnTabSwitch:        onTabSwitch,
		CertificateEnabled: req.CertificateEnabled,
		Adaptive:           req.Adaptive,
		CreatedBy:          actorFromCtx(r),
	})
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to create exam")
		return
	}

	h.writer.Write(r.Context(), r, "exam.create", "exam", &exam.ID, map[string]any{
		"title":  exam.Title,
		"status": exam.Status,
	})

	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": exam, "error": nil})
}

// Get returns the full exam detail including sections and rules.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.svc.GetExam(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get exam")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": detail, "error": nil})
}

// Update replaces all updatable fields on a non-archived exam.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateExamReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	createLike := createExamReq{
		Title:            req.Title,
		TimeLimitMinutes: req.TimeLimitMinutes,
		PassingScorePct:  req.PassingScorePct,
		MaxAttempts:      req.MaxAttempts,
		ShowAnswers:      req.ShowAnswers,
		OnTabSwitch:      req.OnTabSwitch,
		AvailableFrom:    req.AvailableFrom,
		AvailableUntil:   req.AvailableUntil,
	}
	if errs := validateCreateExam(createLike); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	exam, err := h.svc.UpdateExam(r.Context(), id, UpdateExamInput{
		Title:              req.Title,
		Description:        req.Description,
		TimeLimitMinutes:   req.TimeLimitMinutes,
		PassingScorePct:    req.PassingScorePct,
		MaxAttempts:        req.MaxAttempts,
		AvailableFrom:      req.AvailableFrom,
		AvailableUntil:     req.AvailableUntil,
		ShuffleQuestions:   req.ShuffleQuestions,
		ShuffleOptions:     req.ShuffleOptions,
		ShowAnswers:        req.ShowAnswers,
		OnTabSwitch:        req.OnTabSwitch,
		CertificateEnabled: req.CertificateEnabled,
		Adaptive:           req.Adaptive,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
		case errors.Is(err, ErrNotDraft):
			api.WriteJSON(w, http.StatusConflict, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "EXAM_NOT_DRAFT",
					"message": "This operation is only allowed on exams in draft status.",
				},
			})
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to update exam")
		}
		return
	}

	h.writer.Write(r.Context(), r, "exam.update", "exam", &exam.ID, map[string]any{
		"title": exam.Title,
	})

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": exam, "error": nil})
}

// Publish validates rules and transitions an exam from draft to active.
func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	exam, err := h.svc.Publish(r.Context(), id)
	if err != nil {
		var pve *PublishValidationError
		var apve *AdaptivePublishValidationError
		switch {
		case errors.As(err, &apve):
			api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"data": nil,
				"error": map[string]any{
					"code":    "INSUFFICIENT_ADAPTIVE_QUESTIONS",
					"message": "Adaptive exam requires at least 5 questions per difficulty level per rule.",
					"details": apve,
				},
			})
		case errors.As(err, &pve):
			api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"data": nil,
				"error": map[string]any{
					"code":    "EXAM_RULES_UNSATISFIED",
					"message": "One or more question rules cannot be satisfied.",
					"details": pve.Details,
				},
			})
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
		case errors.Is(err, ErrNotDraft):
			api.WriteJSON(w, http.StatusConflict, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "EXAM_NOT_DRAFT",
					"message": "This operation is only allowed on exams in draft status.",
				},
			})
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to publish exam")
		}
		return
	}

	h.writer.Write(r.Context(), r, "exam.publish", "exam", &exam.ID, map[string]any{
		"status": exam.Status,
	})
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data":  map[string]string{"id": exam.ID, "status": exam.Status},
		"error": nil,
	})
}

// Archive transitions an exam to archived status from draft or active.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	exam, err := h.svc.Archive(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
		case errors.Is(err, ErrInvalidTransition):
			api.WriteError(w, http.StatusConflict, "ERR_INVALID_TRANSITION", "exam is already archived")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to archive exam")
		}
		return
	}

	h.writer.Write(r.Context(), r, "exam.archive", "exam", &exam.ID, map[string]any{
		"status": exam.Status,
	})
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data":  map[string]string{"id": exam.ID, "status": exam.Status},
		"error": nil,
	})
}

// TransitionStatus advances the exam through draft → active → archived.
func (h *Handler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req statusTransitionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	exam, err := h.svc.TransitionStatus(r.Context(), id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
		case errors.Is(err, ErrInvalidTransition):
			api.WriteError(w, http.StatusConflict, "ERR_INVALID_TRANSITION", err.Error())
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to transition exam status")
		}
		return
	}

	h.writer.Write(r.Context(), r, "exam.status", "exam", &exam.ID, map[string]any{
		"status": exam.Status,
	})

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": exam, "error": nil})
}

// Delete hard-deletes a draft exam.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteExam(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
		case errors.Is(err, ErrNotDraft):
			api.WriteJSON(w, http.StatusConflict, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "ERR_NOT_DRAFT",
					"message": "only draft exams can be deleted",
				},
			})
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to delete exam")
		}
		return
	}
	h.writer.Write(r.Context(), r, "exam.delete", "exam", &id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// CreateSection adds a section to an exam.
func (h *Handler) CreateSection(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	var req sectionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	section, err := h.svc.CreateSection(r.Context(), examID, SectionInput{
		Title:     req.Title,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to create section")
		return
	}

	h.writer.Write(r.Context(), r, "exam.section.create", "exam_section", &section.ID, map[string]any{
		"exam_id": examID,
	})
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": section, "error": nil})
}

// UpdateSection updates a section.
func (h *Handler) UpdateSection(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	sectionID := chi.URLParam(r, "sectionId")
	var req sectionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	section, err := h.svc.UpdateSection(r.Context(), examID, sectionID, SectionInput{
		Title:     req.Title,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam or section not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to update section")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": section, "error": nil})
}

// DeleteSection removes a section from an exam.
func (h *Handler) DeleteSection(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	sectionID := chi.URLParam(r, "sectionId")
	if err := h.svc.DeleteSection(r.Context(), examID, sectionID); err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam or section not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to delete section")
		return
	}
	h.writer.Write(r.Context(), r, "exam.section.delete", "exam_section", &sectionID, map[string]any{
		"exam_id": examID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// CreateRule adds a question selection rule to an exam.
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	var req ruleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	if errs := validateRule(req); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	questions := make([]ManualQuestionInput, len(req.Questions))
	for i, q := range req.Questions {
		questions[i] = ManualQuestionInput{QuestionID: q.QuestionID, SortOrder: q.SortOrder}
	}

	rule, err := h.svc.CreateRule(r.Context(), examID, QuestionRuleInput{
		SectionID:  req.SectionID,
		Mode:       req.Mode,
		CategoryID: req.CategoryID,
		TagIDs:     req.TagIDs,
		Difficulty: req.Difficulty,
		Count:      req.Count,
		SortOrder:  req.SortOrder,
		Questions:  questions,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
		case errors.Is(err, ErrInvalidInput):
			writeValidationErrors(w, []fieldError{{"tag_ids", "tag_ids must contain valid UUID strings"}})
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to create rule")
		}
		return
	}

	h.writer.Write(r.Context(), r, "exam.rule.create", "exam_question_rule", &rule.ID, map[string]any{
		"exam_id": examID,
		"mode":    rule.Mode,
	})
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": rule, "error": nil})
}

// UpdateRule replaces all fields of a question rule.
func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	ruleID := chi.URLParam(r, "ruleId")
	var req ruleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	if errs := validateRule(req); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	questions := make([]ManualQuestionInput, len(req.Questions))
	for i, q := range req.Questions {
		questions[i] = ManualQuestionInput{QuestionID: q.QuestionID, SortOrder: q.SortOrder}
	}

	rule, err := h.svc.UpdateRule(r.Context(), examID, ruleID, QuestionRuleInput{
		SectionID:  req.SectionID,
		Mode:       req.Mode,
		CategoryID: req.CategoryID,
		TagIDs:     req.TagIDs,
		Difficulty: req.Difficulty,
		Count:      req.Count,
		SortOrder:  req.SortOrder,
		Questions:  questions,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam or rule not found")
		case errors.Is(err, ErrInvalidInput):
			writeValidationErrors(w, []fieldError{{"tag_ids", "tag_ids must contain valid UUID strings"}})
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to update rule")
		}
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": rule, "error": nil})
}

// DeleteRule removes a question rule from an exam.
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	ruleID := chi.URLParam(r, "ruleId")
	if err := h.svc.DeleteRule(r.Context(), examID, ruleID); err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam or rule not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to delete rule")
		return
	}
	h.writer.Write(r.Context(), r, "exam.rule.delete", "exam_question_rule", &ruleID, map[string]any{
		"exam_id": examID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// ── Assignment handlers (FR-BB33) ─────────────────────────────────────────────

type assignReq struct {
	AssigneeType string     `json:"assignee_type"`
	AssigneeID   *string    `json:"assignee_id"`
	Deadline     *time.Time `json:"deadline"`
}

// Assign creates an exam assignment.
func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	var req assignReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	validTypes := map[string]bool{"user": true, "department": true, "all": true}
	if !validTypes[req.AssigneeType] {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "assignee_type must be one of: user, department, all")
		return
	}
	if req.AssigneeType != "all" && req.AssigneeID == nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "assignee_id is required for assignee_type user or department")
		return
	}

	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())

	a, err := h.svc.CreateAssignment(r.Context(), CreateAssignmentInput{
		ExamID:       examID,
		AssigneeType: req.AssigneeType,
		AssigneeID:   req.AssigneeID,
		Deadline:     req.Deadline,
		AssignedBy:   actorFromCtx(r),
		CallerRole:   callerRole,
		CallerDeptID: callerDeptID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
		case errors.Is(err, ErrNotActive):
			api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "EXAM_NOT_ACTIVE",
					"message": "Exams must be in active status before they can be assigned.",
				},
			})
		case errors.Is(err, ErrAssignmentExists):
			api.WriteJSON(w, http.StatusConflict, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "ASSIGNMENT_ALREADY_EXISTS",
					"message": "This exam is already assigned to the specified target.",
				},
			})
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "ERR_FORBIDDEN", "you do not have permission to assign to this target")
		case errors.Is(err, ErrDeadlineInPast):
			api.WriteError(w, http.StatusBadRequest, "ERR_DEADLINE_IN_PAST", "deadline must be in the future")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to create assignment")
		}
		return
	}

	h.writer.Write(r.Context(), r, "exam.assign", "exam_assignment", &a.ID, map[string]any{
		"exam_id":       examID,
		"assignee_type": a.AssigneeType,
		"assignee_id":   a.AssigneeID,
	})
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": a, "error": nil})
}

// Unassign removes an exam assignment.
func (h *Handler) Unassign(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	assignmentID := chi.URLParam(r, "assignmentId")

	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())

	if err := h.svc.DeleteAssignment(r.Context(), examID, assignmentID, callerRole, callerDeptID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
		case errors.Is(err, ErrAssignmentNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "assignment not found")
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "ERR_FORBIDDEN", "you do not have permission to remove this assignment")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to delete assignment")
		}
		return
	}

	h.writer.Write(r.Context(), r, "exam.unassign", "exam_assignment", &assignmentID, map[string]any{
		"exam_id": examID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// ListAssignments returns all assignments for an exam with stats.
func (h *Handler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	details, err := h.svc.ListAssignments(r.Context(), examID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to list assignments")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": details, "error": nil})
}

// SetManualQuestions replaces the question list for a manual-mode rule.
func (h *Handler) SetManualQuestions(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleId")
	var req setManualQuestionsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	questions := make([]ManualQuestionInput, len(req.Questions))
	for i, q := range req.Questions {
		questions[i] = ManualQuestionInput{QuestionID: q.QuestionID, SortOrder: q.SortOrder}
	}

	if err := h.svc.SetManualQuestions(r.Context(), ruleID, questions); err != nil {
		switch {
		case errors.Is(err, ErrRulesModeConflict):
			api.WriteError(w, http.StatusBadRequest, "RULE_NOT_MANUAL", "manual questions can only be set on manual-mode rules")
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "rule not found")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to set manual questions")
		}
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]int{"count": len(questions)}, "error": nil})
}

// GetEligibleCounts returns the eligible question count for each rule of an exam (FR-BB315).
func (h *Handler) GetEligibleCounts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	counts, err := h.svc.GetEligibleCounts(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "EXAM_NOT_FOUND", "exam not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get eligible counts")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data":  GetEligibleCountsResponse{Counts: counts},
		"error": nil,
	})
}
