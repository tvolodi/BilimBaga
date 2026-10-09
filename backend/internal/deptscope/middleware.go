package deptscope

import (
	"log/slog"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RequireSessionInScope hides exam sessions outside a department_admin's
// department subtree: the session owner must be inside the subtree, otherwise
// the response is the endpoint's own "session not found" (404 SESSION_NOT_FOUND,
// identical body), so an out-of-scope id is indistinguishable from an unknown
// one (no existence leak). Other roles, malformed ids and unknown sessions pass
// through untouched (the handler produces its normal 400/404). Fails closed
// (500) when no store is available.
func RequireSessionInScope(store Store, param string) func(http.Handler) http.Handler {
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
			inScope, found, err := store.SessionUserInScope(r.Context(), s, id)
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
