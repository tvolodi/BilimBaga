package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// IsUUID reports whether s is a canonical 36-character hyphenated UUID.
// uuid.Parse alone also accepts "urn:uuid:" / braced / unhyphenated forms,
// which Postgres does not all accept, so the length is pinned.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

// UUIDQuery reads the query parameter name and validates it as a UUID.
// An absent/empty parameter returns ("", true). A malformed value writes a
// 422 VALIDATION_ERROR response and returns ("", false); callers must return.
func UUIDQuery(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return "", true
	}
	if !IsUUID(v) {
		WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", name+" must be a valid UUID")
		return "", false
	}
	return v, true
}

// ValidateUUIDList validates every element of ids (e.g. a comma-separated
// query value already split). On the first malformed element it writes a 422
// VALIDATION_ERROR naming param and returns false.
func ValidateUUIDList(w http.ResponseWriter, param string, ids []string) bool {
	for _, id := range ids {
		if !IsUUID(id) {
			WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", param+" must contain only valid UUIDs")
			return false
		}
	}
	return true
}

// RequireUUIDPathParams returns chi middleware that responds 404 NOT_FOUND when
// any of the named URL path parameters present on the matched route is not a
// valid UUID (such a resource cannot exist). It must be mounted where chi has
// already resolved URL params (inline r.Use inside r.Group / r.With), not on
// the root mux.
func RequireUUIDPathParams(names ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, n := range names {
				v := chi.URLParam(r, n)
				if v != "" && !IsUUID(v) {
					WriteError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
