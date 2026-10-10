package router

import (
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/ai"
	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/categories"
	"github.com/bilimbaga/bilimbaga/internal/certificates"
	"github.com/bilimbaga/bilimbaga/internal/departments"
	"github.com/bilimbaga/bilimbaga/internal/deptscope"
	"github.com/bilimbaga/bilimbaga/internal/email"
	"github.com/bilimbaga/bilimbaga/internal/exams"
	"github.com/bilimbaga/bilimbaga/internal/health"
	appmw "github.com/bilimbaga/bilimbaga/internal/middleware"
	"github.com/bilimbaga/bilimbaga/internal/portal"
	"github.com/bilimbaga/bilimbaga/internal/questions"
	"github.com/bilimbaga/bilimbaga/internal/ratelimit"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/bilimbaga/bilimbaga/internal/reports"
	"github.com/bilimbaga/bilimbaga/internal/roles"
	"github.com/bilimbaga/bilimbaga/internal/sessions"
	"github.com/bilimbaga/bilimbaga/internal/tags"
	"github.com/bilimbaga/bilimbaga/internal/tenant"
	"github.com/bilimbaga/bilimbaga/internal/users"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// New creates and returns a configured Chi router with all registered routes.
// jwtSecret is passed to auth.Authenticate() so the router package never calls os.Getenv().
// rbacCache is the in-memory permission cache loaded at startup.
// db is used by the health endpoint to verify database connectivity.
// version is the build-time Git SHA injected via -ldflags.
// log is the zerolog logger used by the structured middleware chain.
func New(tenantHandler *tenant.Handler, authHandler *auth.Handler, deptHandler *departments.Handler, usersHandler *users.Handler, auditHandler *audit.Handler, categoriesHandler *categories.Handler, tagsHandler *tags.Handler, questionsHandler *questions.Handler, translationsHandler *questions.TranslationHandler, examsHandler *exams.Handler, portalHandler *portal.Handler, sessionsHandler *sessions.Handler, certHandler *certificates.Handler, reportsHandler *reports.Handler, emailHandler *email.Handler, aiHandler *ai.Handler, rolesHandler *roles.Handler, jwtSecret string, rbacCache *rbac.Cache, db *sqlx.DB, version string, log zerolog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Per-user account state (password epoch + force_password_change), shared by the
	// authenticate middleware and the password-change handlers so a successful change
	// takes effect immediately instead of after the cache TTL (ISS-160).
	var accountState *auth.AccountStateCache
	if db != nil {
		accountState = auth.NewDBAccountStateCache(db)
		if authHandler != nil {
			authHandler.SetPasswordChangedHook(accountState.Invalidate)
		}
		// Role/department/status/lock changes by an admin must not wait out the cache TTL (ISS-248).
		if usersHandler != nil {
			usersHandler.SetUserChangedHook(accountState.Invalidate)
		}
	}

	// Structured middleware chain (FR-BB66):
	//   1. RequestID  — assign UUID correlation ID
	//   2. Recovery   — recover panics and return 500 (wraps everything below)
	//   3. clientFromXRealIP — the client address from X-Real-IP (#475)
	//   4. RequestLogger — structured JSON log after response written; runs after the client is resolved, so
	//      its ip field is the real client (#478)
	//   5. Heartbeat — chi built-in
	r.Use(appmw.RequestID)
	r.Use(appmw.Recovery(log))
	r.Use(clientFromXRealIP)
	r.Use(appmw.RequestLogger(log))
	r.Use(chimw.Heartbeat("/ping"))

	// Global middleware — injects tenant_id for every request (public and protected alike).
	r.Use(auth.TenantContext())

	r.Route("/api/v1", func(r chi.Router) {
		// Auth endpoints — tight rate limit: 10 req/min per IP (AC-1).
		r.Group(func(r chi.Router) {
			r.Use(ratelimit.AuthLimiter())
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
			r.Post("/auth/logout", authHandler.Logout)
			// Account recovery (FR-BB115) — public, same tight limiter.
			r.Post("/auth/forgot-password", authHandler.ForgotPassword)
			r.Post("/auth/reset-password", authHandler.ResetPassword)
		})

		// Public non-auth routes — general rate limit (AC-1).
		r.Group(func(r chi.Router) {
			r.Use(ratelimit.GlobalLimiter())
			// Health (#475): not an auth endpoint, so it sits here and not in the auth bucket. Deploy scripts poll it.
			r.Get("/health", health.Handler(db, version))
			r.Get("/tenant/config", tenantHandler.GetConfig)
			r.Get("/tenant/logo", tenantHandler.GetLogo)

			// Certificate verification — public, no auth required (AC-7).
			r.Get("/verify/{code}", certHandler.HandleVerifyCertificate)
		})

		// Protected routes — Bearer JWT required; general rate limit (AC-1).
		r.Group(func(r chi.Router) {
			r.Use(ratelimit.GlobalLimiter())
			r.Use(authenticate(jwtSecret, accountState))
			// Department scoping for session-keyed admin endpoints (ISS-165).
			var scopeStore deptscope.Store
			if db != nil {
				scopeStore = deptscope.NewStore(db)
			}
			sessionScope := func(param string) func(http.Handler) http.Handler {
				return deptscope.RequireSessionInScope(scopeStore, param)
			}
			// Manual grading (ISS-218): also admits sessions of exams the examiner created.
			gradingScope := func(param string) func(http.Handler) http.Handler {
				return deptscope.RequireGradingSessionInScope(scopeStore, param)
			}
			// Malformed UUID path params 404 here instead of reaching Postgres (500).
			// Runs after routing, so chi URL params are resolved (ISS-141).
			r.Use(api.RequireUUIDPathParams(api.UUIDPathParamNames...))
			r.Post("/auth/change-password", authHandler.ChangePassword)

			// Tenant configuration — requires super_admin.
			r.With(rbac.RequirePermission(rbacCache, "tenant", "manage")).
				Put("/tenant/config", tenantHandler.UpdateConfig)

			// Department management.
			r.With(rbac.RequirePermission(rbacCache, "departments", "read")).
				Get("/departments", deptHandler.ListTree)
			r.With(rbac.RequirePermission(rbacCache, "departments", "manage")).
				Post("/departments", deptHandler.Create)
			r.With(rbac.RequirePermission(rbacCache, "departments", "manage")).
				Put("/departments/{id}", deptHandler.Update)
			r.With(rbac.RequirePermission(rbacCache, "departments", "manage")).
				Delete("/departments/{id}", deptHandler.Delete)

			// User management.
			// GetMe is available to every authenticated user — no extra permission required.
			r.Get("/users/me", usersHandler.GetMe)
			// Self-service preferred locale (FR-BB116): own record only, no extra permission.
			r.Patch("/users/me", usersHandler.UpdateMe)
			// Roles lookup — static sub-path, must come before /{id} routes.
			r.With(rbac.RequirePermission(rbacCache, "users", "read")).
				Get("/users/roles", usersHandler.ListRoles)
			// Import is a static sub-path; must be registered before /{id} routes.
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users/import", usersHandler.ImportUsers)
			r.With(rbac.RequirePermission(rbacCache, "users", "read")).
				Get("/users", usersHandler.ListUsers)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users", usersHandler.CreateUser)
			// Per-user routes — GET self-access is enforced inside the handler.
			r.Get("/users/{id}", usersHandler.GetUser)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Put("/users/{id}", usersHandler.UpdateUser)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users/{id}/deactivate", usersHandler.DeactivateUser)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users/{id}/reactivate", usersHandler.ReactivateUser)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users/{id}/reset-password", usersHandler.ResetPassword)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users/{id}/unlock", usersHandler.UnlockUser)

			// Role management (FR-BB117). /roles/permissions is registered before /roles/{id}.
			r.With(rbac.RequirePermission(rbacCache, "roles", "read")).
				Get("/roles", rolesHandler.List)
			r.With(rbac.RequirePermission(rbacCache, "roles", "read")).
				Get("/roles/permissions", rolesHandler.ListPermissions)
			r.With(rbac.RequirePermission(rbacCache, "roles", "manage")).
				Post("/roles", rolesHandler.Create)
			r.With(rbac.RequirePermission(rbacCache, "roles", "read")).
				Get("/roles/{id}", rolesHandler.Get)
			r.With(rbac.RequirePermission(rbacCache, "roles", "manage")).
				Put("/roles/{id}", rolesHandler.Update)
			r.With(rbac.RequirePermission(rbacCache, "roles", "manage")).
				Delete("/roles/{id}", rolesHandler.Delete)

			// Audit log.
			r.With(rbac.RequirePermission(rbacCache, "audit", "read")).
				Get("/audit", auditHandler.List)
			r.With(rbac.RequirePermission(rbacCache, "audit", "read")).
				Get("/audit/export", auditHandler.Export)

			// Categories — tree is readable by any authenticated user;
			// mutations require categories:manage (examiner+ and super_admin).
			r.Get("/categories", categoriesHandler.ListTree)
			r.With(rbac.RequirePermission(rbacCache, "categories", "manage")).
				Post("/categories", categoriesHandler.Create)
			r.With(rbac.RequirePermission(rbacCache, "categories", "manage")).
				Put("/categories/{id}", categoriesHandler.Update)
			r.With(rbac.RequirePermission(rbacCache, "categories", "manage")).
				Delete("/categories/{id}", categoriesHandler.Delete)

			// Tags — read and write require explicit permissions.
			r.With(rbac.RequirePermission(rbacCache, "tags", "read")).
				Get("/tags", tagsHandler.List)
			r.With(rbac.RequirePermission(rbacCache, "tags", "manage")).
				Post("/tags", tagsHandler.Create)
			r.With(rbac.RequirePermission(rbacCache, "tags", "manage")).
				Put("/tags/{id}", tagsHandler.Update)
			r.With(rbac.RequirePermission(rbacCache, "tags", "manage")).
				Delete("/tags/{id}", tagsHandler.Delete)

			// Questions — all endpoints require questions:read or questions:write.
			// Import and export are static sub-paths; registered before /{id} routes.
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Post("/questions/import", questionsHandler.Import)
			r.With(rbac.RequirePermission(rbacCache, "questions", "read")).
				Get("/questions/export", questionsHandler.Export)
			r.With(rbac.RequirePermission(rbacCache, "questions", "read")).
				Get("/questions", questionsHandler.List)
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Post("/questions", questionsHandler.Create)
			r.With(rbac.RequirePermission(rbacCache, "questions", "read")).
				Get("/questions/{id}", questionsHandler.Get)
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Put("/questions/{id}", questionsHandler.Update)
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Post("/questions/{id}/status", questionsHandler.TransitionStatus)
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Delete("/questions/{id}", questionsHandler.Delete)
			r.With(rbac.RequirePermission(rbacCache, "questions", "read")).
				Get("/questions/{id}/versions", questionsHandler.ListVersions)
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Post("/questions/{id}/tags", questionsHandler.AddTag)
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Delete("/questions/{id}/tags/{tagId}", questionsHandler.RemoveTag)

			// Question translations (FR-BB24).
			r.With(rbac.RequirePermission(rbacCache, "questions", "read")).
				Get("/questions/{id}/translations", translationsHandler.List)
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Put("/questions/{id}/translations/{locale}", translationsHandler.Upsert)
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Delete("/questions/{id}/translations/{locale}", translationsHandler.Delete)

			// Exams (FR-BB31).
			r.With(rbac.RequirePermission(rbacCache, "exams", "read")).
				Get("/exams", examsHandler.List)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Post("/exams", examsHandler.Create)
			r.With(rbac.RequirePermission(rbacCache, "exams", "read")).
				Get("/exams/{id}", examsHandler.Get)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Put("/exams/{id}", examsHandler.Update)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Post("/exams/{id}/publish", examsHandler.Publish)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Post("/exams/{id}/unpublish", examsHandler.Unpublish)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Post("/exams/{id}/archive", examsHandler.Archive)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Post("/exams/{id}/status", examsHandler.TransitionStatus)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Delete("/exams/{id}", examsHandler.Delete)

			// Exam sections.
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Post("/exams/{id}/sections", examsHandler.CreateSection)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Put("/exams/{id}/sections/{sectionId}", examsHandler.UpdateSection)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Delete("/exams/{id}/sections/{sectionId}", examsHandler.DeleteSection)

			// Exam question rules.
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Post("/exams/{id}/rules", examsHandler.CreateRule)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Put("/exams/{id}/rules/{ruleId}", examsHandler.UpdateRule)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Delete("/exams/{id}/rules/{ruleId}", examsHandler.DeleteRule)
			r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
				Put("/exams/{id}/rules/{ruleId}/questions", examsHandler.SetManualQuestions)

			// Exam assignments (FR-BB33).
			r.With(rbac.RequirePermission(rbacCache, "exams", "assign")).
				Post("/exams/{id}/assign", examsHandler.Assign)
			r.With(rbac.RequirePermission(rbacCache, "exams", "assign")).
				Delete("/exams/{id}/assign/{assignmentId}", examsHandler.Unassign)
			r.With(rbac.RequirePermission(rbacCache, "exams", "read")).
				Get("/exams/{id}/assignments", examsHandler.ListAssignments)
			r.With(rbac.RequirePermission(rbacCache, "exams", "read")).
				Get("/exams/{id}/rules/eligible-counts", examsHandler.GetEligibleCounts)

			// Employee exam portal (FR-BB34) — any authenticated user.
			r.Get("/portal/exams", portalHandler.ListMyExams)
			r.Get("/portal/exams/{id}", portalHandler.GetMyExam)

			// Session creation (FR-BB35) — any authenticated user.
			r.Post("/portal/exams/{id}/sessions", sessionsHandler.CreateSession)

			// Answer saving and session resume (FR-BB37) — any authenticated user.
			r.Get("/portal/sessions/{id}", sessionsHandler.GetSessionState)
			// Answer-save has a tighter per-session rate limit: 60 req/min (AC-1).
			r.With(ratelimit.AnswerSaveLimiter()).
				Put("/portal/sessions/{id}/answers/{questionId}", sessionsHandler.SaveAnswer)

			// Adaptive next question (FR-BB72 AC-3) — any authenticated user.
			r.Get("/portal/sessions/{id}/next-question", sessionsHandler.GetNextQuestion)

			// Tab-switch event reporting (FR-BB38) — any authenticated user.
			r.Post("/portal/sessions/{id}/events", sessionsHandler.ReportEvent)

			// Session submission (FR-BB39) — any authenticated user.
			r.Post("/portal/sessions/{id}/submit", sessionsHandler.SubmitSession)

			// Result retrieval (FR-BB41) — any authenticated user.
			r.Get("/portal/sessions/{id}/result", sessionsHandler.GetSessionResult)
			r.With(rbac.RequirePermission(rbacCache, "exams", "read"), sessionScope("id")).Get("/admin/sessions/{id}/result", sessionsHandler.GetAdminSessionResult)
			r.Get("/portal/exams/{id}/history", sessionsHandler.GetExamHistory)

			// My Results (FR-BB46) — any authenticated user.
			r.Get("/portal/results", sessionsHandler.HandleGetMyResults)

			// Send overdue reminder (FR-BB510) — examiner+ only.
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Post("/admin/users/{userId}/remind", usersHandler.RemindEmployee)

			// Dashboard metrics (FR-BB51) — examiner+ only.
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/dashboard", reportsHandler.GetDashboard)

			// Per-exam analytics (FR-BB52) — examiner+ only.
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/exams/{id}/analytics", reportsHandler.GetExamAnalytics)

			// Per-employee record and progress (FR-BB53) — examiner+ only.
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/users/{id}/record", reportsHandler.GetUserRecord)
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/users/{id}/progress", reportsHandler.GetUserProgress)

			// Export API (FR-BB54) — examiner+ only.
			// Static sub-paths registered before /{id} routes.
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/dashboard/export", reportsHandler.DashboardExportPDF)
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/exams/{id}/results/export", reportsHandler.ExamResultsCSV)
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/users/{id}/record/export", reportsHandler.UserRecordCSV)

			// Manual grading queue (FR-BB42).
			r.With(rbac.RequirePermission(rbacCache, "grading", "read")).Get("/admin/grading", sessionsHandler.HandleListGradingQueue)
			r.With(rbac.RequirePermission(rbacCache, "grading", "read"), gradingScope("sessionId")).Get("/admin/grading/{sessionId}", sessionsHandler.HandleGetGradingDetail)
			r.With(rbac.RequirePermission(rbacCache, "grading", "write"), gradingScope("sessionId")).Post("/admin/grading/{sessionId}/answers/{questionId}", sessionsHandler.HandleGradeAnswer)

			// Certificate generation (FR-BB43).
			r.Get("/portal/sessions/{id}/certificate", certHandler.HandleGetPortalCertificate)
			r.With(rbac.RequirePermission(rbacCache, "exams", "read"), sessionScope("id")).Get("/admin/sessions/{id}/certificate", certHandler.HandleGetAdminCertificate)

			// Email notification test (FR-BB61) — super_admin only.
			r.With(rbac.RequirePermission(rbacCache, "tenant", "manage")).Post("/admin/notifications/test", emailHandler.HandleTestNotification)

			// AI question generation (FR-BB71) — examiner+ only.
			r.With(rbac.RequirePermission(rbacCache, "questions", "write")).
				Post("/admin/ai/generate-questions", aiHandler.HandleGenerateQuestions)

			// AI performance insight summaries (FR-BB74) — examiner+ only.
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/ai/insights/{examId}", aiHandler.HandleGetInsights)

			// AI loyalty profile narrative (FR-BB75) — department_admin+ only.
			r.With(rbac.RequirePermission(rbacCache, "reports", "read")).
				Get("/admin/ai/loyalty-summary/{sessionId}", aiHandler.GetLoyaltyNarrative)
		})
	})

	return r
}

// authenticate builds the JWT middleware. With an account-state cache it also rejects
// access tokens issued before the user's last password reset (ISS-105) and blocks every
// route except the force-password-change allowlist while users.force_password_change is
// set (ISS-160); with a nil cache (route tests that never reach a handler) only the
// signature/expiry checks apply.
func authenticate(jwtSecret string, state *auth.AccountStateCache) func(http.Handler) http.Handler {
	if state == nil {
		return auth.Authenticate(jwtSecret)
	}
	return auth.Authenticate(jwtSecret, auth.WithAccountState(state.Lookup))
}
