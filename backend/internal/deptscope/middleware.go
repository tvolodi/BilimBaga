package deptscope

import (
	"log/slog"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RequireUserInScope rejects (403 FORBIDDEN, same shape as GET /users/{id})
// requests whose {param} path user lies outside a department_admin's subtree.
// Other roles, the caller's own id, malformed ids and unknown users pass
// through untouched (the handler produces its normal 400/404).
func RequireUserInScope(store Store, param string) func(http.Handler) http.Handler {
	return guard(store, param, func(st Store, r *http.Request, s Scope, id string) (bool, bool, error) {
		return st.UserInScope(r.Context(), s, id)
	}, true)
}

// RequireSessionInScope is the exam-session counterpart: the session's owner
// must be inside the department_admin's subtree.
func RequireSessionInScope(store Store, param string) func(http.Handler) http.Handler {
	return guard(store, param, func(st Store, r *http.Request, s Scope, id string) (bool, bool, error) {
		return st.SessionUserInScope(r.Context(), s, id)
	}, false)
}

type lookup func(st Store, r *http.Request, s Scope, id string) (inScope, found bool, err error)

func guard(store Store, param string, look lookup, selfAllowed bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := FromContext(r.Context())
			if !s.Restricted {
				next.ServeHTTP(w, r)
				return
			}
			id := chi.URLParam(r, param)
			if _, err := uuid.Parse(id); err != nil || (selfAllowed && id == ctxkeys.UserIDFromCtx(r.Context())) {
				next.ServeHTTP(w, r)
				return
			}
			if store == nil {
				api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify department scope")
				return
			}
			inScope, found, err := look(store, r, s, id)
			if err != nil {
				slog.Error("deptscope: scope check failed", "error", err, "param", param)
				api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify department scope")
				return
			}
			if found && !inScope {
				api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
