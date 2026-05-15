package certificates

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the certificates domain.
type Handler struct {
	svc     Service
	baseURL string // e.g. "https://app.bilimbaga.kz" — used for verify URL in PDF
}

// NewHandler creates a new Handler.
func NewHandler(svc Service, baseURL string) *Handler {
	return &Handler{svc: svc, baseURL: baseURL}
}

// HandleGetPortalCertificate handles GET /api/v1/portal/sessions/:id/certificate.
// Requires authentication; enforces session ownership (AC-1).
func (h *Handler) HandleGetPortalCertificate(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	callerID := auth.UserIDFromCtx(r.Context())

	cert, snap, err := h.svc.GetOrCreate(r.Context(), sessionID, callerID, false)
	if err != nil {
		h.writeCertError(w, err, true)
		return
	}

	h.streamPDF(w, cert, snap)
}

// HandleGetAdminCertificate handles GET /api/v1/admin/sessions/:id/certificate.
// Requires exams:read permission (enforced at router level); no ownership restriction (AC-8).
func (h *Handler) HandleGetAdminCertificate(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	cert, snap, err := h.svc.GetOrCreate(r.Context(), sessionID, "", true)
	if err != nil {
		h.writeCertError(w, err, false)
		return
	}

	h.streamPDF(w, cert, snap)
}

// HandleVerifyCertificate handles GET /api/v1/verify/:code.
// Public — no authentication required. Always returns HTTP 200 (AC-7).
func (h *Handler) HandleVerifyCertificate(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	result, err := h.svc.GetByVerificationCode(r.Context(), code)
	if err != nil {
		// On unexpected internal error, still return valid=false — never expose error details.
		api.WriteJSON(w, http.StatusOK, map[string]any{"data": &VerifyResponse{Valid: false}, "error": nil})
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": result, "error": nil})
}

// writeCertError maps domain errors to HTTP responses.
// checkOwner must be true for portal endpoints (where ErrNotOwner → 403).
func (h *Handler) writeCertError(w http.ResponseWriter, err error, checkOwner bool) {
	switch {
	case checkOwner && errors.Is(err, ErrNotOwner):
		api.WriteError(w, http.StatusForbidden, "SESSION_FORBIDDEN",
			"You do not have access to this session.")
	case errors.Is(err, ErrNotSubmitted):
		api.WriteError(w, http.StatusUnprocessableEntity, "SESSION_NOT_SUBMITTED",
			"Certificate can only be issued for submitted sessions.")
	case errors.Is(err, ErrNotCertifiable):
		api.WriteError(w, http.StatusUnprocessableEntity, "EXAM_NOT_CERTIFIABLE",
			"This exam does not issue certificates.")
	case errors.Is(err, ErrNotPassed):
		api.WriteError(w, http.StatusUnprocessableEntity, "SESSION_NOT_PASSED",
			"Certificate is only issued for passed sessions.")
	case errors.Is(err, ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND",
			"Session not found.")
	default:
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL",
			"failed to generate certificate")
	}
}

// streamPDF generates and writes the PDF bytes with correct headers (AC-6).
func (h *Handler) streamPDF(w http.ResponseWriter, cert *Certificate, snap TemplateSnapshot) {
	pdfBytes, err := GeneratePDF(cert, snap, h.baseURL)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL",
			"failed to generate PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="certificate-%s.pdf"`, cert.VerificationCode))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes) //nolint:errcheck
}
