package questions

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/go-chi/chi/v5"
)

// TranslationHandler exposes the FR-BB24 endpoints.
type TranslationHandler struct {
	svc    TranslationService
	writer *audit.Writer
}

// NewTranslationHandler builds a TranslationHandler backed by the given service
// and audit Writer.
func NewTranslationHandler(svc TranslationService, writer *audit.Writer) *TranslationHandler {
	return &TranslationHandler{svc: svc, writer: writer}
}

type upsertTranslationReq struct {
	Stem        string                       `json:"stem"`
	Explanation *string                      `json:"explanation"`
	Options     []upsertOptionTranslationReq `json:"options"`
}

type upsertOptionTranslationReq struct {
	OptionID string `json:"option_id"`
	Text     string `json:"text"`
}

// List handles GET /api/v1/questions/:id/translations.
func (h *TranslationHandler) List(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	out, err := h.svc.GetAll(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			api.WriteError(w, http.StatusNotFound, "ERR_QUESTION_NOT_FOUND", "question not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to load translations")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": out, "error": nil})
}

// Upsert handles PUT /api/v1/questions/:id/translations/:locale.
func (h *TranslationHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	locale := chi.URLParam(r, "locale")

	var req upsertTranslationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}
	if req.Stem == "" {
		api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"data": nil,
			"error": map[string]any{
				"code":    "ERR_VALIDATION",
				"message": "stem is required",
				"fields":  []fieldError{{Field: "stem", Message: "stem is required"}},
			},
		})
		return
	}

	options := make([]AnswerTextTranslation, 0, len(req.Options))
	for _, o := range req.Options {
		options = append(options, AnswerTextTranslation(o))
	}

	before, after, err := h.svc.Upsert(r.Context(), id, locale, UpsertTranslationInput{
		Stem:        req.Stem,
		Explanation: req.Explanation,
		Options:     options,
	})
	if err != nil {
		var missing *MissingOptionsError
		switch {
		case errors.Is(err, ErrQuestionNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_QUESTION_NOT_FOUND", "question not found")
		case errors.Is(err, ErrUnsupportedLocale):
			api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "ERR_UNSUPPORTED_LOCALE",
					"message": err.Error(),
				},
			})
		case errors.As(err, &missing):
			api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"data": nil,
				"error": map[string]any{
					"code":              "ERR_MISSING_OPTION_TRANSLATIONS",
					"message":           "translation missing for one or more answer options",
					"missing_option_ids": missing.MissingIDs,
				},
			})
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to upsert translation")
		}
		return
	}

	h.writer.Write(r.Context(), r, "question_translation.upsert", "question_translation", composeEntityID(id, locale), map[string]any{
		"question_id": id,
		"locale":      locale,
		"before":      translationAuditPayload(before),
		"after":       translationAuditPayload(after),
	})

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": after, "error": nil})
}

// Delete handles DELETE /api/v1/questions/:id/translations/:locale.
func (h *TranslationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	locale := chi.URLParam(r, "locale")

	before, err := h.svc.Delete(r.Context(), id, locale)
	if err != nil {
		switch {
		case errors.Is(err, ErrQuestionNotFound):
			api.WriteError(w, http.StatusNotFound, "ERR_QUESTION_NOT_FOUND", "question not found")
		case errors.Is(err, ErrCannotDeleteDefaultLocale):
			api.WriteJSON(w, http.StatusConflict, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "ERR_CANNOT_DELETE_DEFAULT_LOCALE",
					"message": err.Error(),
				},
			})
		default:
			api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to delete translation")
		}
		return
	}

	h.writer.Write(r.Context(), r, "question_translation.delete", "question_translation", composeEntityID(id, locale), map[string]any{
		"question_id": id,
		"locale":      locale,
		"before":      translationAuditPayload(before),
		"after":       nil,
	})

	w.WriteHeader(http.StatusNoContent)
}

// composeEntityID returns a pointer to "<questionID>:<locale>" for the audit log.
func composeEntityID(questionID, locale string) *string {
	s := questionID + ":" + locale
	return &s
}

// translationAuditPayload reduces a *LocaleTranslation to a JSON-serialisable
// before/after diff target. Returns nil if the input is nil.
func translationAuditPayload(t *LocaleTranslation) any {
	if t == nil {
		return nil
	}
	return map[string]any{
		"stem":        t.Stem,
		"explanation": t.Explanation,
		"options":     t.Options,
	}
}
