package deptscope

import (
	"context"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	deptA   = "11111111-1111-1111-1111-111111111111"
	callerU = "22222222-2222-2222-2222-222222222222"
	target  = "33333333-3333-3333-3333-333333333333"
)

func principal(role, dept string) context.Context {
	ctx := context.WithValue(context.Background(), ctxkeys.CtxRole, role)
	ctx = context.WithValue(ctx, ctxkeys.CtxUserID, callerU)
	return context.WithValue(ctx, ctxkeys.CtxDepartmentID, dept)
}

func TestFromContext(t *testing.T) {
	s := FromContext(principal("department_admin", deptA))
	assert.True(t, s.Restricted)
	assert.Equal(t, deptA, s.DepartmentID)
	assert.Equal(t, callerU, s.UserID)
	assert.Equal(t, deptA, s.Arg())

	assert.Equal(t, "00000000-0000-0000-0000-000000000000", FromContext(principal("department_admin", "")).Arg(),
		"a department_admin without a department matches nothing")

	for _, role := range []string{"super_admin", "examiner", "employee", ""} {
		s := FromContext(principal(role, deptA))
		assert.False(t, s.Restricted, role)
		assert.Nil(t, s.Arg(), role)
	}
	assert.False(t, FromContext(context.Background()).Restricted)
}

func TestPredicate_IncludesDescendantsAndNullBypass(t *testing.T) {
	p := Predicate("es.user_id", "$2")
	assert.Contains(t, p, "$2::uuid IS NULL")
	assert.Contains(t, p, "es.user_id IN (")
	assert.Contains(t, p, "WITH RECURSIVE sc_t")
	assert.Contains(t, p, "sc_c.parent_id = sc_t.id", "descendant departments must be included")
	assert.Contains(t, p, "sc_u.department_id IN")
}

func TestStore_UserInScope(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"x"}, [][]driver.Value{{int64(1)}})
	in, found, err := NewStore(db).UserInScope(context.Background(), FromContext(principal("department_admin", deptA)), target)
	require.NoError(t, err)
	assert.True(t, in)
	assert.True(t, found)
	assert.Contains(t, f.queries[0], "FROM users u WHERE u.id = $1")
	assert.Contains(t, f.queries[0], "WITH RECURSIVE sc_t")
	assert.Equal(t, []driver.Value{target, deptA}, f.args[0])

	db, _ = newFakeDB(t) // no row: unknown user
	in, found, err = NewStore(db).UserInScope(context.Background(), FromContext(principal("department_admin", deptA)), target)
	require.NoError(t, err)
	assert.False(t, in)
	assert.False(t, found)

	db, f = newFakeDB(t)
	f.qErr = errors.New("boom")
	_, _, err = NewStore(db).UserInScope(context.Background(), Scope{}, target)
	require.Error(t, err)
}

func TestStore_SessionUserInScope(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"x"}, [][]driver.Value{{int64(0)}})
	in, found, err := NewStore(db).SessionUserInScope(context.Background(), FromContext(principal("department_admin", deptA)), target)
	require.NoError(t, err)
	assert.False(t, in)
	assert.True(t, found)
	assert.Contains(t, f.queries[0], "FROM exam_sessions es WHERE es.id = $1")
	assert.Contains(t, f.queries[0], "es.user_id IN (")
	assert.Equal(t, []driver.Value{target, deptA}, f.args[0])
}

// ── Middleware ────────────────────────────────────────────────────────────────

type fakeStore struct {
	inScope, found bool
	err            error
	calls          int
}

func (s *fakeStore) UserInScope(context.Context, Scope, string) (bool, bool, error) {
	s.calls++
	return s.inScope, s.found, s.err
}
func (s *fakeStore) SessionUserInScope(context.Context, Scope, string) (bool, bool, error) {
	s.calls++
	return s.inScope, s.found, s.err
}

func serve(mw func(http.Handler) http.Handler, role, dept, id string) (*httptest.ResponseRecorder, bool) {
	reached := false
	r := chi.NewRouter()
	r.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ctx := context.WithValue(r.Context(), ctxkeys.CtxRole, role)
			ctx = context.WithValue(ctx, ctxkeys.CtxUserID, callerU)
			ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, dept)
			next.ServeHTTP(w, r.WithContext(ctx)) })
	}).With(mw).Get("/x/{id}", func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x/"+id, nil))
	return w, reached
}

func TestMiddleware_Session(t *testing.T) {
	cases := []struct {
		name     string
		role     string
		dept     string
		store    *fakeStore
		id       string
		wantCode int
		reached  bool
		lookup   bool
	}{
		{"own department passes", "department_admin", deptA, &fakeStore{inScope: true, found: true}, target, 200, true, true},
		{"other department 404", "department_admin", deptA, &fakeStore{inScope: false, found: true}, target, 404, false, true},
		{"unknown session passes to handler 404", "department_admin", deptA, &fakeStore{}, target, 200, true, true},
		{"malformed id passes to handler", "department_admin", deptA, &fakeStore{}, "not-a-uuid", 200, true, false},
		{"lookup error 500", "department_admin", deptA, &fakeStore{err: errors.New("db")}, target, 500, false, true},
		{"super_admin never checked", "super_admin", "", &fakeStore{inScope: false, found: true}, target, 200, true, false},
		{"examiner never checked", "examiner", deptA, &fakeStore{inScope: false, found: true}, target, 200, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, reached := serve(RequireSessionInScope(tc.store, "id"), tc.role, tc.dept, tc.id)
			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, tc.reached, reached)
			assert.Equal(t, tc.lookup, tc.store.calls == 1)
			if tc.wantCode == 404 {
				assert.Contains(t, w.Body.String(), `"code":"SESSION_NOT_FOUND"`)
			}
		})
	}
}

// An out-of-scope session must be indistinguishable from an unknown one: the
// guard's 404 is byte-identical to the sessions/certificates handlers' own
// "session not found" response.
func TestMiddleware_OutOfScopeIdenticalToUnknown(t *testing.T) {
	w, _ := serve(RequireSessionInScope(&fakeStore{inScope: false, found: true}, "id"), "department_admin", deptA, target)
	want := httptest.NewRecorder()
	api.WriteError(want, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found.")
	assert.Equal(t, want.Code, w.Code)
	assert.Equal(t, want.Body.String(), w.Body.String())
	assert.Equal(t, want.Header().Get("Content-Type"), w.Header().Get("Content-Type"))
}

func TestMiddleware_NilStoreFailsClosedForDepartmentAdmin(t *testing.T) {
	w, reached := serve(RequireSessionInScope(nil, "id"), "department_admin", deptA, target)
	assert.Equal(t, 500, w.Code)
	assert.False(t, reached)
	assert.True(t, strings.Contains(w.Body.String(), "INTERNAL_ERROR"))

	w, reached = serve(RequireSessionInScope(nil, "id"), "super_admin", "", target)
	assert.Equal(t, 200, w.Code)
	assert.True(t, reached)
}
