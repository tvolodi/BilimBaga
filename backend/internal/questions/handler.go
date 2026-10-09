package questions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the questions domain.
type Handler struct {
	svc    Service
	writer *audit.Writer
}

// NewHandler creates a new Handler backed by the given Service and audit Writer.
func NewHandler(svc Service, writer *audit.Writer) *Handler {
	return &Handler{svc: svc, writer: writer}
}

// ── Request DTOs ─────────────────────────────────────────────────────────────

type createQuestionReq struct {
	CategoryID    string                    `json:"category_id"`
	Difficulty    string                    `json:"difficulty"`
	Type          string                    `json:"type"`
	DefaultLocale string                    `json:"default_locale"`
	AutoGrade     bool                      `json:"auto_grade"`
	ModelAnswer   *string                   `json:"model_answer"`
	Translations  map[string]translationReq `json:"translations"`
	AnswerOptions []answerOptionReq         `json:"answer_options"`
	TagIDs        []string                  `json:"tag_ids"`
}

type updateQuestionReq struct {
	CategoryID    string                    `json:"category_id"`
	Difficulty    string                    `json:"difficulty"`
	AutoGrade     bool                      `json:"auto_grade"`
	ModelAnswer   *string                   `json:"model_answer"`
	Translations  map[string]translationReq `json:"translations"`
	AnswerOptions []answerOptionReq         `json:"answer_options"`
	TagIDs        []string                  `json:"tag_ids"`
}

type translationReq struct {
	Stem        string  `json:"stem"`
	Explanation *string `json:"explanation"`
}

type answerOptionReq struct {
	SortOrder      int                             `json:"sort_order"`
	IsCorrect      bool                            `json:"is_correct"`
	LikertWeight   *float64                        `json:"likert_weight"`
	LikertPolarity *string                         `json:"likert_polarity"`
	Translations   map[string]answerTranslationReq `json:"translations"`
}

type answerTranslationReq struct {
	Text string `json:"text"`
}

type statusTransitionReq struct {
	Status string `json:"status"`
}

type addTagReq struct {
	TagID string `json:"tag_id"`
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

func validateCreateRequest(req createQuestionReq) []fieldError {
	var errs []fieldError

	if req.CategoryID == "" {
		errs = append(errs, fieldError{"category_id", "category_id is required"})
	}
	validDifficulties := map[string]bool{"easy": true, "medium": true, "hard": true}
	if !validDifficulties[req.Difficulty] {
		errs = append(errs, fieldError{"difficulty", "difficulty must be one of: easy, medium, hard"})
	}
	validTypes := map[string]bool{"single": true, "multiple": true, "truefalse": true, "likert": true, "shorttext": true}
	if !validTypes[req.Type] {
		errs = append(errs, fieldError{"type", "type must be one of: single, multiple, truefalse, likert, shorttext"})
	}
	if req.DefaultLocale == "" {
		errs = append(errs, fieldError{"default_locale", "default_locale is required"})
	}

	// Stem must exist for default_locale.
	if req.DefaultLocale != "" {
		tr, ok := req.Translations[req.DefaultLocale]
		if !ok || tr.Stem == "" {
			errs = append(errs, fieldError{
				fmt.Sprintf("translations.%s.stem", req.DefaultLocale),
				"stem is required for the default locale",
			})
		}
	}

	// Answer option constraints by type.
	switch req.Type {
	case "single", "multiple", "truefalse":
		if len(req.AnswerOptions) < 2 {
			errs = append(errs, fieldError{"answer_options", "at least 2 answer options are required for this question type"})
		}
	case "likert":
		if len(req.AnswerOptions) < 2 {
			errs = append(errs, fieldError{"answer_options", "at least 2 answer options are required for likert questions"})
		}
		for i, opt := range req.AnswerOptions {
			if opt.LikertWeight == nil {
				errs = append(errs, fieldError{
					fmt.Sprintf("answer_options[%d].likert_weight", i),
					"likert_weight is required for each option in a likert question",
				})
			}
		}
	}

	// Every option of a choice question needs non-blank text in the default locale.
	if validTypes[req.Type] {
		errs = append(errs, optionTextFieldErrors(req.Type, req.DefaultLocale, answerOptionInputsFromReq(req.AnswerOptions))...)
	}

	// auto_grade / model_answer are only valid for shorttext questions.
	if req.AutoGrade || (req.ModelAnswer != nil && strings.TrimSpace(*req.ModelAnswer) != "") {
		if req.Type != "shorttext" {
			errs = append(errs, fieldError{"auto_grade", "INVALID_FIELD_FOR_TYPE"})
		}
	}
	if req.AutoGrade && req.Type == "shorttext" {
		if req.ModelAnswer == nil || strings.TrimSpace(*req.ModelAnswer) == "" {
			errs = append(errs, fieldError{"model_answer", "MISSING_MODEL_ANSWER"})
		}
	}

	return errs
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

// List handles GET /api/v1/questions.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := QuestionFilter{
		Page:    parseIntParam(r, "page", 1),
		PerPage: parseIntParam(r, "per_page", 20),
	}
	categoryID, ok := api.UUIDQuery(w, r, "category_id")
	if !ok {
		return
	}
	if categoryID != "" {
		filter.CategoryID = &categoryID
	}
	// tag_ids accepts comma-separated UUIDs; tag_id is a backward-compat alias.
	if v := r.URL.Query().Get("tag_ids"); v != "" {
		filter.TagIDs = splitCommaSeparated(v)
	} else if v := r.URL.Query().Get("tag_id"); v != "" {
		filter.TagIDs = []string{v}
	}
	if !api.ValidateUUIDList(w, "tag_ids", filter.TagIDs) {
		return
	}
	// difficulties accepts comma-separated values; difficulty is a backward-compat alias.
	if v := r.URL.Query().Get("difficulties"); v != "" {
		filter.Difficulties = splitCommaSeparated(v)
	} else if v := r.URL.Query().Get("difficulty"); v != "" {
		filter.Difficulties = []string{v}
	}
	if v := r.URL.Query().Get("type"); v != "" {
		filter.Type = &v
	}
	// statuses accepts comma-separated values; status is a backward-compat alias.
	if v := r.URL.Query().Get("statuses"); v != "" {
		filter.Statuses = splitCommaSeparated(v)
	} else if v := r.URL.Query().Get("status"); v != "" {
		filter.Statuses = []string{v}
	}
	if v := r.URL.Query().Get("locale"); v != "" {
		filter.Locale = &v
	}
	if v := r.URL.Query().Get("locale_missing"); v != "" {
		filter.LocaleMissing = &v
	}
	filter.IncludeSuperseded = r.URL.Query().Get("include_versions") == "true"
	filter.Search = r.URL.Query().Get("search")
	filter.Sort = r.URL.Query().Get("sort")
	filter.Order = r.URL.Query().Get("order")

	items, total, err := h.svc.ListFiltered(r.Context(), filter)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to list questions")
		return
	}
	if items == nil {
		items = []*QuestionListItem{}
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

// Create handles POST /api/v1/questions.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createQuestionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	if errs := validateCreateRequest(req); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	actorID := actorFromCtx(r)

	translations := make(map[string]TranslationInput, len(req.Translations))
	for locale, t := range req.Translations {
		translations[locale] = TranslationInput(t)
	}
	options := answerOptionInputsFromReq(req.AnswerOptions)
	tagIDs := req.TagIDs
	if tagIDs == nil {
		tagIDs = []string{}
	}

	detail, err := h.svc.CreateQuestionFull(r.Context(), CreateQuestionFullInput{
		CategoryID:    req.CategoryID,
		Difficulty:    req.Difficulty,
		Type:          req.Type,
		DefaultLocale: req.DefaultLocale,
		CreatedBy:     actorID,
		AutoGrade:     req.AutoGrade,
		ModelAnswer:   req.ModelAnswer,
		Translations:  translations,
		AnswerOptions: options,
		TagIDs:        tagIDs,
	})
	if err != nil {
		var optErr *OptionValidationError
		if errors.As(err, &optErr) {
			writeValidationErrors(w, optErr.Fields)
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to create question")
		return
	}

	h.writer.Write(r.Context(), r, "question.create", "question", &detail.ID, map[string]any{
		"type":        detail.Type,
		"difficulty":  detail.Difficulty,
		"category_id": detail.CategoryID,
	})
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": detail, "error": nil})
}

// Get handles GET /api/v1/questions/:id.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.svc.GetQuestionWithDetails(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "question not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get question")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": detail, "error": nil})
}

// Update handles PUT /api/v1/questions/:id.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateQuestionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	actorID := actorFromCtx(r)

	translations := make(map[string]TranslationInput, len(req.Translations))
	for locale, t := range req.Translations {
		translations[locale] = TranslationInput(t)
	}
	options := answerOptionInputsFromReq(req.AnswerOptions)
	tagIDs := req.TagIDs
	if tagIDs == nil {
		tagIDs = []string{}
	}

	q, err := h.svc.UpdateQuestion(r.Context(), id, UpdateQuestionInput{
		CategoryID:    req.CategoryID,
		Difficulty:    req.Difficulty,
		UpdatedBy:     actorID,
		AutoGrade:     req.AutoGrade,
		ModelAnswer:   req.ModelAnswer,
		Translations:  translations,
		AnswerOptions: options,
		TagIDs:        tagIDs,
	})
	if err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "question not found")
			return
		}
		if errors.Is(err, ErrMissingModelAnswer) {
			api.WriteError(w, http.StatusBadRequest, "MISSING_MODEL_ANSWER", "model_answer is required when auto_grade is true")
			return
		}
		var optErr *OptionValidationError
		if errors.As(err, &optErr) {
			writeValidationErrors(w, optErr.Fields)
			return
		}
		if errors.Is(err, ErrInvalidFieldForType) {
			api.WriteError(w, http.StatusBadRequest, "INVALID_FIELD_FOR_TYPE", "auto_grade and model_answer are only valid for shorttext questions")
			return
		}
		if errors.Is(err, ErrInvalidInput) {
			api.WriteError(w, http.StatusUnprocessableEntity, "ERR_INVALID_STATUS", "question status does not allow updates")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to update question")
		return
	}

	// Compute locale_coverage from the provided translations.
	lc := make([]string, 0, len(translations))
	for locale := range translations {
		lc = append(lc, locale)
	}

	h.writer.Write(r.Context(), r, "question.update", "question", &q.ID, map[string]any{
		"difficulty":  q.Difficulty,
		"category_id": q.CategoryID,
		"version":     q.Version,
	})
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":              q.ID,
			"type":            q.Type,
			"difficulty":      q.Difficulty,
			"status":          q.Status,
			"category_id":     q.CategoryID,
			"default_locale":  q.DefaultLocale,
			"version":         q.Version,
			"parent_id":       q.ParentID,
			"locale_coverage": lc,
			"updated_at":      q.UpdatedAt,
		},
		"error": nil,
	})
}

// TransitionStatus handles POST /api/v1/questions/:id/status.
func (h *Handler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req statusTransitionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}
	if req.Status == "" {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "status is required")
		return
	}

	q, err := h.svc.TransitionStatus(r.Context(), id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrQuestionNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "question not found")
		case errors.Is(err, ErrInvalidTransition):
			api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "ERR_INVALID_TRANSITION",
					"message": err.Error(),
				},
			})
		case errors.Is(err, ErrStemRequired):
			api.WriteError(w, http.StatusBadRequest, "ERR_STEM_REQUIRED", "default locale stem is required before transitioning to this status")
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to transition status")
		}
		return
	}

	h.writer.Write(r.Context(), r, "question.status_transition", "question", &q.ID, map[string]any{
		"status": map[string]string{"new": q.Status},
	})
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data":  map[string]string{"id": q.ID, "status": q.Status},
		"error": nil,
	})
}

// Delete handles DELETE /api/v1/questions/:id.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteQuestion(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrQuestionNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "question not found")
		case errors.Is(err, ErrNotDraft):
			api.WriteJSON(w, http.StatusConflict, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "ERR_NOT_DRAFT",
					"message": "only draft questions can be deleted",
				},
			})
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to delete question")
		}
		return
	}
	h.writer.Write(r.Context(), r, "question.delete", "question", &id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// ListVersions handles GET /api/v1/questions/:id/versions.
func (h *Handler) ListVersions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	versions, err := h.svc.ListVersions(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "question not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to list versions")
		return
	}
	if versions == nil {
		versions = []*VersionEntry{}
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data":  map[string]any{"versions": versions},
		"error": nil,
	})
}

// AddTag handles POST /api/v1/questions/:id/tags.
func (h *Handler) AddTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify question exists.
	if _, err := h.svc.GetQuestion(r.Context(), id); err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "question not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to find question")
		return
	}

	var req addTagReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}
	if req.TagID == "" {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "tag_id is required")
		return
	}

	if err := h.svc.TagQuestion(r.Context(), id, req.TagID); err != nil {
		if errors.Is(err, ErrTagNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "tag not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to add tag")
		return
	}

	tags, err := h.svc.GetQuestionTags(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "tag added but failed to retrieve updated list")
		return
	}
	if tags == nil {
		tags = []string{}
	}

	h.writer.Write(r.Context(), r, "question.tag_add", "question", &id, map[string]any{"tag_id": req.TagID})
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data":  map[string]any{"question_id": id, "tag_ids": tags},
		"error": nil,
	})
}

// RemoveTag handles DELETE /api/v1/questions/:id/tags/:tagId.
func (h *Handler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tagID := chi.URLParam(r, "tagId")

	// Verify question exists.
	if _, err := h.svc.GetQuestion(r.Context(), id); err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "question not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to find question")
		return
	}

	if err := h.svc.UntagQuestion(r.Context(), id, tagID); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to remove tag")
		return
	}

	h.writer.Write(r.Context(), r, "question.tag_remove", "question", &id, map[string]any{"tag_id": tagID})
	w.WriteHeader(http.StatusNoContent)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func actorFromCtx(r *http.Request) string {
	return auth.UserIDFromCtx(r.Context())
}

// splitCommaSeparated splits a comma-separated string, trimming spaces and dropping empty parts.
func splitCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
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
