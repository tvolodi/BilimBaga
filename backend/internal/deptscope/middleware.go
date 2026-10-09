package deptscope

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type sessionLookup func(ctx context.Context, store Store, s Scope, sessionID string) (inScope, found bool, err error)

// RequireSessionInScope hides exam sessions outside the caller's department
// subtree (every role except super_admin, ISS-218): the session owner must be
// inside the subtree, otherwise the response is the endpoint's own "session not
// found" (404 SESSION_NOT_FOUND, identical body), so an out-of-scope id is
// indistinguishable from an unknown one (no existence leak). super_admin,
// malformed ids and unknown sessions pass through untouched (the handler
// produces its normal 400/404). Fails closed (500) when no store is available.
func RequireSessionInScope(store Store, param string) func(http.Handler) http.Handler {
	return requireSession(store, param, func(ctx context.Context, st Store, s Scope, id string) (bool, bool, error) {
		return st.SessionUserInScope(ctx, s, id)
	})
}

// RequireGradingSessionInScope is RequireSessionInScope for the manual-grading
// endpoints (grading detail, grade answer). It additionally lets an examiner
// reach sessions of exams it created (Scope.ExamOwnerID), and nothing else.
func RequireGradingSessionInScope(store Store, param string) func(http.Handler) http.Handler {
	return requireSession(store, param, func(ctx context.Context, st Store, s Scope, id string) (bool, bool, error) {
		return st.GradingSessionInScope(ctx, s, id)
	})
}

func requireSession(store Store, param string, lookup sessionLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := FromContext(r.Context())
			if !s.Restricted {
				next.ServeHTTP(w, r)
				return
			}
			id := chi.URLParam(r, param)
			if _, err := uuid.Parse(id); err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if store == nil {
				api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify department scope")
				return
			}
			inScope, found, err := lookup(r.Context(), store, s, id)
			if err != nil {
				slog.Error("deptscope: scope check failed", "error", err, "param", param)
				api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify department scope")
				return
			}
			if found && !inScope {
				api.WriteError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
