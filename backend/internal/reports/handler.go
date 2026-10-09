package reports

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/go-chi/chi/v5"
)

// TenantConfigProvider provides the tenant configuration cache.
// Satisfied by *tenant.service.
type TenantConfigProvider interface {
	GetAllConfig() map[string]json.RawMessage
}

// Handler handles HTTP requests for the reports domain.
type Handler struct {
	svc       Service
	tenantCfg TenantConfigProvider
}

// NewHandler creates a new Handler backed by the given Service.
// tenantCfg may be nil; when nil the PDF export will use empty company name and no logo.
func NewHandler(svc Service, tenantCfg TenantConfigProvider) *Handler {
	return &Handler{svc: svc, tenantCfg: tenantCfg}
}

// GetDashboard handles GET /api/v1/admin/dashboard (FR-BB51).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.svc.GetDashboardMetrics(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to load dashboard metrics")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": metrics, "error": nil})
}

// GetExamAnalytics handles GET /api/v1/admin/exams/{id}/analytics (FR-BB52).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) GetExamAnalytics(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	if examID == "" {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_PARAM", "exam id is required")
		return
	}

	result, err := h.svc.GetExamAnalytics(r.Context(), examID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "EXAM_NOT_FOUND", "exam not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load exam analytics")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": result, "error": nil})
}

// GetUserRecord handles GET /api/v1/admin/users/{id}/record (FR-BB53).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) GetUserRecord(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_PARAM", "user id is required")
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	perPage := 20
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 {
			if v > 100 {
				v = 100
			}
			perPage = v
		}
	}

	record, total, err := h.svc.GetUserRecord(r.Context(), userID, page, perPage)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user record")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data": record,
		"meta": map[string]any{
			"page":     page,
			"per_page": perPage,
			"total":    total,
		},
		"error": nil,
	})
}

// GetUserProgress handles GET /api/v1/admin/users/{id}/progress (FR-BB53).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) GetUserProgress(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_PARAM", "user id is required")
		return
	}

	result, err := h.svc.GetUserProgress(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user progress")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": result, "error": nil})
}

// ── FR-BB54: Export API ──────────────────────────────────────────────────────

// ExamResultsCSV handles GET /api/v1/admin/exams/:id/results/export (FR-BB54 AC-2,3,4,8,10).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) ExamResultsCSV(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	if examID == "" {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_PARAM", "exam id is required")
		return
	}

	tenantID := auth.TenantIDFromCtx(r.Context())
	filename := fmt.Sprintf("results-%s-%s.csv", examID, time.Now().UTC().Format("20060102"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	if err := h.svc.StreamExamResultsCSV(r.Context(), w, examID, tenantID); err != nil {
		// Headers already sent — we cannot change status code. Log and return.
		slog.Error("reports: exam results CSV export failed", "error", err, "examId", examID)
	}
}

// UserRecordCSV handles GET /api/v1/admin/users/:id/record/export (FR-BB54 AC-2,5,8,10).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) UserRecordCSV(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_PARAM", "user id is required")
		return
	}

	tenantID := auth.TenantIDFromCtx(r.Context())
	filename := fmt.Sprintf("record-%s-%s.csv", userID, time.Now().UTC().Format("20060102"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	if err := h.svc.StreamUserRecordCSV(r.Context(), w, userID, tenantID); err != nil {
		// Headers already sent — we cannot change status code; log it.
		slog.Error("reports: user record CSV export failed", "error", err, "userId", userID)
	}
}

// DashboardExportPDF handles GET /api/v1/admin/dashboard/export (FR-BB54 AC-6,7,9,10).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) DashboardExportPDF(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantIDFromCtx(r.Context())

	// AC-6: parse from/to query params; default to last 30 days.
	now := time.Now().UTC()
	fromTime := now.AddDate(0, 0, -30)
	toTime := now

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			fromTime = t.UTC()
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			// Include the full "to" day.
			toTime = t.UTC().Add(24*time.Hour - time.Second)
		}
	}

	// Extract company name and logo from tenant config.
	companyName := ""
	logoBase64 := ""
	if h.tenantCfg != nil {
		cfg := h.tenantCfg.GetAllConfig()
		if raw, ok := cfg["app_name"]; ok {
			var name string
			if err := json.Unmarshal(raw, &name); err == nil {
				companyName = name
			}
		}
		if raw, ok := cfg["logo"]; ok {
			var logo string
			if err := json.Unmarshal(raw, &logo); err == nil {
				logoBase64 = logo
			}
		}
	}

	data, err := h.svc.BuildDashboardReport(r.Context(), tenantID, fromTime, toTime, companyName, logoBase64)
	if err != nil {
		slog.Error("reports: build dashboard report failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to build dashboard report")
		return
	}

	pdfBytes, err := GenerateDashboardPDF(data)
	if err != nil {
		slog.Error("reports: generate dashboard PDF failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate PDF")
		return
	}

	// AC-10: deterministic filename.
	fromLabel := fromTime.Format("20060102")
	toLabel := toTime.Format("20060102")
	filename := fmt.Sprintf("dashboard-report-%s-%s.pdf", fromLabel, toLabel)

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}
