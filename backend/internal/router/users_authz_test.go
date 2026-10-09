package router_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/deptscope"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/bilimbaga/bilimbaga/internal/router"
	"github.com/bilimbaga/bilimbaga/internal/users"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-250 (follow-up of ISS-240 / PR #246): FR-BB117 D-4 users authz exercised through the
// real route tree -- real auth.Authenticate (with the DB-backed account-state cache), real
// rbac.RequirePermission, real users.Handler and users.Service -- over an in-memory
// users.Repository and a recording database/sql driver. No live database.

const (
	authzDeptA = "aaaaaaaa-0000-4000-8000-00000000000a"
	authzDeptB = "bbbbbbbb-0000-4000-8000-00000000000b"

	idSuper   = "10000000-0000-4000-8000-000000000001"
	idDeptAdm = "10000000-0000-4000-8000-000000000002"
	idExam    = "10000000-0000-4000-8000-000000000003"
	idEmp     = "10000000-0000-4000-8000-000000000004"
	idFar     = "10000000-0000-4000-8000-000000000005" // employee in the other department
	idUnknown = "10000000-0000-4000-8000-0000000000ff"

	roleSuperID  = "20000000-0000-4000-8000-000000000001"
	roleDAID     = "20000000-0000-4000-8000-000000000002"
	roleExamID   = "20000000-0000-4000-8000-000000000003"
	roleEmpID    = "20000000-0000-4000-8000-000000000004"
	roleCloneID  = "20000000-0000-4000-8000-000000000005" // exactly department_admin's permissions
	roleNarrowID = "20000000-0000-4000-8000-000000000006" // strict subset of department_admin
)

var authzPerms = map[string]rbac.PermissionSet{
	"super_admin":      {"users:read": true, "users:manage": true, "roles:read": true},
	"department_admin": {"users:read": true, "users:manage": true, "exams:read": true},
	"da_clone":         {"users:read": true, "users:manage": true, "exams:read": true},
	"narrow_custom":    {"users:read": true},
	"examiner":         {"exams:read": true, "exams:write": true},
	"employee":         {"portal:read": true},
}

// ---- in-memory users.Repository ---------------------------------------------

type authzRepo struct {
	mu       sync.Mutex
	users    map[string]*users.User
	roleName map[string]string // role id -> name
	roleID   map[string]string // role name -> id
	writes   int
}

func newAuthzRepo() *authzRepo {
	r := &authzRepo{users: map[string]*users.User{}, roleName: map[string]string{}, roleID: map[string]string{}}
	for id, n := range map[string]string{roleSuperID: "super_admin", roleDAID: "department_admin", roleExamID: "examiner",
		roleEmpID: "employee", roleCloneID: "da_clone", roleNarrowID: "narrow_custom"} {
		r.roleName[id], r.roleID[n] = n, id
	}
	add := func(id, dept, roleID string) {
		d := dept
		r.users[id] = &users.User{ID: id, Email: id + "@example.com", FullName: "U " + id, DepartmentID: &d,
			RoleID: roleID, RoleName: r.roleName[roleID], Status: "active", CreatedAt: time.Now()}
	}
	add(idSuper, authzDeptA, roleSuperID)
	add(idDeptAdm, authzDeptA, roleDAID)
	add(idExam, authzDeptA, roleExamID)
	add(idEmp, authzDeptA, roleEmpID)
	add(idFar, authzDeptB, roleEmpID)
	return r
}

func (r *authzRepo) snapshot(id string) users.User {
	r.mu.Lock()
	defer r.mu.Unlock()
	return *r.users[id]
}

func (r *authzRepo) List(context.Context, users.ListFilters, *string) ([]users.User, int, error) {
	return nil, 0, nil
}
func (r *authzRepo) GetByID(_ context.Context, id string) (*users.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, users.ErrNotFound
	}
	c := *u
	return &c, nil
}
func (r *authzRepo) Create(context.Context, string, string, string, *string, string) (*users.User, error) {
	r.mu.Lock()
	r.writes++
	r.mu.Unlock()
	return &users.User{ID: "new"}, nil
}
func (r *authzRepo) Update(_ context.Context, id, fullName string, dept *string, roleID string) (*users.User, error) {
	// The real pgRepository validates uuid columns before writing (users.validateIDs).
	if dept != nil && !api.IsUUID(*dept) {
		return nil, fmt.Errorf("%w: department_id must be a valid UUID", users.ErrValidation)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, users.ErrNotFound
	}
	r.writes++
	u.FullName, u.DepartmentID, u.RoleID, u.RoleName = fullName, dept, roleID, r.roleName[roleID]
	c := *u
	return &c, nil
}
func (r *authzRepo) Deactivate(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes++
	r.users[id].Status = "inactive"
	return nil
}
func (r *authzRepo) RevokeAllTokens(context.Context, string) error { return nil }
func (r *authzRepo) UpdatePassword(context.Context, string, string) error {
	r.mu.Lock()
	r.writes++
	r.mu.Unlock()
	return nil
}
func (r *authzRepo) Unlock(context.Context, string) error {
	r.mu.Lock()
	r.writes++
	r.mu.Unlock()
	return nil
}
func (r *authzRepo) GetDepartmentIDByName(context.Context, string) (string, error) {
	return "", users.ErrNotFound
}
func (r *authzRepo) GetRoleIDByName(_ context.Context, n string) (string, error) {
	if id, ok := r.roleID[n]; ok {
		return id, nil
	}
	return "", users.ErrNotFound
}
func (r *authzRepo) GetRoleNameByID(_ context.Context, id string) (string, error) {
	if n, ok := r.roleName[id]; ok {
		return n, nil
	}
	return "", users.ErrNotFound
}
func (r *authzRepo) ListRoles(context.Context) ([]users.RoleRow, error) { return nil, nil }

// SetPreferredLocale mirrors the pgRepository contract: nil clears, unknown id is ErrNotFound (FR-BB116).
func (r *authzRepo) SetPreferredLocale(_ context.Context, id string, locale *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return users.ErrNotFound
	}
	r.writes++
	if locale == nil {
		u.PreferredLocale = nil
	} else {
		v := *locale
		u.PreferredLocale = &v
	}
	return nil
}

// ---- recording driver answering the auth account-state query -----------------

type stateConnector struct{ repo *authzRepo }
type stateConn struct{ repo *authzRepo }
type stateStmt struct{ repo *authzRepo }
type stateRows struct {
	data [][]driver.Value
	i    int
}

func (c stateConnector) Connect(context.Context) (driver.Conn, error) { return stateConn(c), nil }
func (c stateConnector) Driver() driver.Driver                        { return nil }
func (c stateConn) Prepare(string) (driver.Stmt, error)               { return stateStmt(c), nil }
func (c stateConn) Close() error                                      { return nil }
func (c stateConn) Begin() (driver.Tx, error)                         { return nil, io.ErrUnexpectedEOF }
func (s stateStmt) Close() error                                      { return nil }
func (s stateStmt) NumInput() int                                     { return -1 }
func (s stateStmt) Exec([]driver.Value) (driver.Result, error)        { return driver.RowsAffected(0), nil }

// Query serves accountStateSQL: columns password_changed_at, force_password_change,
// status, role_name, department_id; the single arg is the user id.
func (s stateStmt) Query(args []driver.Value) (driver.Rows, error) {
	rows := &stateRows{}
	id, _ := args[0].(string)
	s.repo.mu.Lock()
	defer s.repo.mu.Unlock()
	if u, ok := s.repo.users[id]; ok {
		var dept driver.Value
		if u.DepartmentID != nil {
			dept = *u.DepartmentID
		}
		rows.data = [][]driver.Value{{nil, false, u.Status, u.RoleName, dept}}
	}
	return rows, nil
}
func (r *stateRows) Columns() []string {
	return []string{"password_changed_at", "force_password_change", "status", "role_name", "department_id"}
}
func (r *stateRows) Close() error { return nil }
func (r *stateRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.i])
	r.i++
	return nil
}

// ---- harness -----------------------------------------------------------------

func newUsersRouter(t *testing.T) (http.Handler, *authzRepo) {
	t.Helper()
	repo := newAuthzRepo()
	db := sqlx.NewDb(sql.OpenDB(stateConnector{repo}), "postgres")
	t.Cleanup(func() { _ = db.Close() })

	cache := rbac.NewCache()
	cache.LoadFromMap(authzPerms)
	svc := users.WithPermissionsLookup(users.WithPermissionChecker(users.NewService(repo), cache.Has), cache.PermissionsFor)
	svc = users.WithLocaleSource(svc, localeList{"kk", "ru", "en"})
	uh := users.NewHandler(svc, nil)

	h := router.New(nil, nil, nil, uh, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"test-secret", cache, db, "test", zerolog.Nop())
	return h, repo
}

// localeList is the tenant available_locales stand-in for the router harness.
type localeList []string

func (l localeList) GetAvailableLocales() []string { return l }

func tokenFor(t *testing.T, sub, role, dept string) string {
	t.Helper()
	c := jwt.MapClaims{"sub": sub, "role": role, "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix()}
	if dept != "" {
		c["department_id"] = dept
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte("test-secret"))
	require.NoError(t, err)
	return s
}

func call(h http.Handler, method, path, tok string, body any) *httptest.ResponseRecorder {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), rec.Body.String())
	if env.Error == nil {
		return ""
	}
	return env.Error.Code
}

func updateBody(roleID string) map[string]any {
	return map[string]any{"full_name": "Renamed", "department_id": authzDeptA, "role_id": roleID}
}

func daToken(t *testing.T) string  { return tokenFor(t, idDeptAdm, "department_admin", authzDeptA) }
func supToken(t *testing.T) string { return tokenFor(t, idSuper, "super_admin", authzDeptA) }

// ---- strict subset -------------------------------------------------------------

func TestRouterUsersAuthz_StrictSubset(t *testing.T) {
	h, repo := newUsersRouter(t)
	tok := daToken(t)

	// A custom role with exactly the caller's permissions is a peer: refused on update and create.
	rec := call(h, http.MethodPut, "/api/v1/users/"+idEmp, tok, updateBody(roleCloneID))
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	assert.Equal(t, "FORBIDDEN", errCode(t, rec))
	assert.Equal(t, "employee", repo.snapshot(idEmp).RoleName, "no write on refusal")

	rec = call(h, http.MethodPost, "/api/v1/users", tok, map[string]any{
		"email": "x@example.com", "full_name": "X", "department_id": authzDeptA, "role_id": roleCloneID})
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	assert.Zero(t, repo.writes)

	// The peer's own record is unreachable too (managing an equal-power role).
	repo.users[idEmp].RoleID, repo.users[idEmp].RoleName = roleCloneID, "da_clone"
	rec = call(h, http.MethodPost, "/api/v1/users/"+idEmp+"/deactivate", tok, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	assert.Equal(t, "active", repo.snapshot(idEmp).Status)
	repo.users[idEmp].RoleID, repo.users[idEmp].RoleName = roleEmpID, "employee"

	// A proper subset is still assignable.
	rec = call(h, http.MethodPut, "/api/v1/users/"+idEmp, tok, updateBody(roleNarrowID))
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "narrow_custom", repo.snapshot(idEmp).RoleName)
}

func TestRouterUsersAuthz_ExaminerGetsRBAC403OnManageRoutes(t *testing.T) {
	h, repo := newUsersRouter(t)
	tok := tokenFor(t, idExam, "examiner", authzDeptA)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/users"},
		{http.MethodPut, "/api/v1/users/" + idEmp},
		{http.MethodPost, "/api/v1/users/" + idEmp + "/deactivate"},
		{http.MethodPost, "/api/v1/users/" + idEmp + "/reset-password"},
		{http.MethodPost, "/api/v1/users/" + idEmp + "/unlock"},
		{http.MethodPost, "/api/v1/users/import"},
	} {
		rec := call(h, tc.method, tc.path, tok, updateBody(roleEmpID))
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s", tc.method, tc.path)
		assert.Equal(t, "FORBIDDEN", errCode(t, rec), "%s %s", tc.method, tc.path)
	}
	assert.Zero(t, repo.writes)
}

// ---- no self role change / self deactivate -------------------------------------

func TestRouterUsersAuthz_NoSelfRoleChangeOrDeactivate(t *testing.T) {
	h, repo := newUsersRouter(t)
	for _, c := range []struct {
		name, id, role, newRole string
		tok                     string
	}{
		{"super_admin", idSuper, "super_admin", roleDAID, supToken(t)},
		{"department_admin", idDeptAdm, "department_admin", roleEmpID, daToken(t)},
	} {
		t.Run(c.name, func(t *testing.T) {
			rec := call(h, http.MethodPut, "/api/v1/users/"+c.id, c.tok, updateBody(c.newRole))
			assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
			assert.Equal(t, c.role, repo.snapshot(c.id).RoleName)

			rec = call(h, http.MethodPost, "/api/v1/users/"+c.id+"/deactivate", c.tok, nil)
			assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
			assert.Equal(t, "active", repo.snapshot(c.id).Status)
		})
	}
	assert.Zero(t, repo.writes)

	// Self-service that does not change the role still works.
	rec := call(h, http.MethodPut, "/api/v1/users/"+idSuper, supToken(t), updateBody(roleSuperID))
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// ---- 404 parity ------------------------------------------------------------------

func TestRouterUsersAuthz_UnknownAndOutOfScopeAreIdentical404(t *testing.T) {
	h, repo := newUsersRouter(t)
	tok := daToken(t)
	ops := []struct{ method, suffix string }{
		{http.MethodGet, ""}, {http.MethodPut, ""}, {http.MethodPost, "/deactivate"},
		{http.MethodPost, "/reset-password"}, {http.MethodPost, "/unlock"},
	}
	for _, op := range ops {
		unknown := call(h, op.method, "/api/v1/users/"+idUnknown+op.suffix, tok, updateBody(roleEmpID))
		far := call(h, op.method, "/api/v1/users/"+idFar+op.suffix, tok, updateBody(roleEmpID))
		assert.Equal(t, http.StatusNotFound, unknown.Code, "%s %s unknown", op.method, op.suffix)
		assert.Equal(t, unknown.Code, far.Code, "%s %s status parity", op.method, op.suffix)
		assert.Equal(t, unknown.Body.String(), far.Body.String(), "%s %s body parity", op.method, op.suffix)
	}
	assert.Zero(t, repo.writes)
	assert.Equal(t, "active", repo.snapshot(idFar).Status)
}

// ---- malformed ids -----------------------------------------------------------------

func TestRouterUsersAuthz_MalformedUUIDs(t *testing.T) {
	h, repo := newUsersRouter(t)
	tok := daToken(t)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users/not-a-uuid"},
		{http.MethodPut, "/api/v1/users/not-a-uuid"},
		{http.MethodPost, "/api/v1/users/not-a-uuid/deactivate"},
		{http.MethodPost, "/api/v1/users/not-a-uuid/reset-password"},
		{http.MethodPost, "/api/v1/users/not-a-uuid/unlock"},
	} {
		rec := call(h, tc.method, tc.path, tok, updateBody(roleEmpID))
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s", tc.method, tc.path)
		assert.Equal(t, "NOT_FOUND", errCode(t, rec))
	}

	// Malformed ids in the body / query are 422, never a 500 from Postgres.
	rec := call(h, http.MethodPut, "/api/v1/users/"+idEmp, tok, updateBody("not-a-uuid"))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	// (a scoped caller is refused first for a department outside its scope, so use super_admin.)
	rec = call(h, http.MethodPut, "/api/v1/users/"+idEmp, supToken(t), map[string]any{
		"full_name": "X", "department_id": "not-a-uuid", "role_id": roleEmpID})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	for _, q := range []string{"department_id=nope", "role_id=nope"} {
		rec = call(h, http.MethodGet, "/api/v1/users?"+q, tok, nil)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, q)
	}
	assert.Zero(t, repo.writes)
}

// ---- stale claims ---------------------------------------------------------------------

func TestRouterUsersAuthz_StaleClaimsAreTokenRevoked(t *testing.T) {
	path := "/api/v1/users/" + idEmp

	cases := []struct {
		name string
		prep func(repo *authzRepo)
	}{
		{"demoted since token issued", func(repo *authzRepo) { // DB says employee, token says department_admin
			repo.users[idDeptAdm].RoleID, repo.users[idDeptAdm].RoleName = roleEmpID, "employee"
		}},
		{"deactivated since token issued", func(repo *authzRepo) { repo.users[idDeptAdm].Status = "inactive" }},
		{"moved department since token issued", func(repo *authzRepo) {
			d := authzDeptB
			repo.users[idDeptAdm].DepartmentID = &d
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, repo := newUsersRouter(t)
			c.prep(repo)
			rec := call(h, http.MethodGet, path, daToken(t), nil)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
			assert.Equal(t, "TOKEN_REVOKED", errCode(t, rec))
			rec = call(h, http.MethodPut, path, daToken(t), updateBody(roleEmpID))
			assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
			assert.Equal(t, "TOKEN_REVOKED", errCode(t, rec))
			assert.Zero(t, repo.writes)
		})
	}
}

// ISS-248: an admin changing a user's role drops that user's cached account state, so the
// user's old token is revoked on the very next request (not after the cache TTL).
func TestRouterUsersAuthz_RoleChangeRevokesOldTokenImmediately(t *testing.T) {
	h, _ := newUsersRouter(t)
	empTok := tokenFor(t, idEmp, "employee", authzDeptA)

	rec := call(h, http.MethodGet, "/api/v1/users/"+idEmp, empTok, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String()) // populates the account-state cache

	rec = call(h, http.MethodPut, "/api/v1/users/"+idEmp, daToken(t), updateBody(roleNarrowID))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = call(h, http.MethodGet, "/api/v1/users/"+idEmp, empTok, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	assert.Equal(t, "TOKEN_REVOKED", errCode(t, rec))
}

// FR-BB510 overdue reminders (unused in the authz tests).
func (r *authzRepo) ExamExists(context.Context, string) (bool, error) { return false, nil }
func (r *authzRepo) IsOverdueTarget(context.Context, string, string) (bool, error) {
	return false, nil
}
func (r *authzRepo) LastReminderAt(context.Context, string, string) (*time.Time, error) {
	return nil, nil
}
func (r *authzRepo) InsertReminder(context.Context, string, string, string) error { return nil }

// ---- FR-BB116: PATCH /users/me through the real route tree ---------------------

// AC-2/AC-3 end to end: an employee (no users permissions) may set its own locale; the
// stored value changes and only the caller's record is touched.
func TestRouterUsersAuthz_PatchMeSetsOwnLocaleForAnyRole(t *testing.T) {
	h, repo := newUsersRouter(t)
	before := repo.snapshot(idFar)

	rec := call(h, http.MethodPatch, "/api/v1/users/me", tokenFor(t, idEmp, "employee", authzDeptA), map[string]any{"preferred_locale": "ru"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var env struct {
		Data struct {
			ID              string  `json:"id"`
			PreferredLocale *string `json:"preferred_locale"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, idEmp, env.Data.ID)
	require.NotNil(t, env.Data.PreferredLocale)
	assert.Equal(t, "ru", *env.Data.PreferredLocale)
	require.NotNil(t, repo.snapshot(idEmp).PreferredLocale)
	assert.Equal(t, "ru", *repo.snapshot(idEmp).PreferredLocale)

	// A department admin can use the same endpoint on its own record.
	rec = call(h, http.MethodPatch, "/api/v1/users/me", daToken(t), map[string]any{"preferred_locale": nil})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Nil(t, repo.snapshot(idDeptAdm).PreferredLocale)

	// The other user is untouched.
	assert.Equal(t, before.PreferredLocale, repo.snapshot(idFar).PreferredLocale)
}

// AC-2: a code outside the tenant available_locales is refused and nothing is written.
func TestRouterUsersAuthz_PatchMeRejectsUnavailableLocale(t *testing.T) {
	h, repo := newUsersRouter(t)

	rec := call(h, http.MethodPatch, "/api/v1/users/me", tokenFor(t, idEmp, "employee", authzDeptA), map[string]any{"preferred_locale": "de"})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec))
	assert.Nil(t, repo.snapshot(idEmp).PreferredLocale)
	assert.Zero(t, repo.writes)
}

// AC-3: role_id, department_id, email and status cannot ride along on the self-service endpoint.
// Each attempt is 400 VALIDATION_ERROR and persists nothing, including the locale in the same body.
func TestRouterUsersAuthz_PatchMeRejectsPrivilegeFields(t *testing.T) {
	cases := map[string]map[string]any{
		"role_id escalation":     {"preferred_locale": "ru", "role_id": roleSuperID},
		"department move":        {"preferred_locale": "ru", "department_id": authzDeptB},
		"email change":           {"preferred_locale": "ru", "email": "attacker@example.com"},
		"status change":          {"preferred_locale": "ru", "status": "inactive"},
		"role_id without locale": {"role_id": roleSuperID},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			h, repo := newUsersRouter(t)
			rec := call(h, http.MethodPatch, "/api/v1/users/me", tokenFor(t, idEmp, "employee", authzDeptA), body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec))

			after := repo.snapshot(idEmp)
			assert.Equal(t, "employee", after.RoleName)
			assert.Equal(t, authzDeptA, *after.DepartmentID)
			assert.Equal(t, idEmp+"@example.com", after.Email)
			assert.Equal(t, "active", after.Status)
			assert.Nil(t, after.PreferredLocale)
			assert.Zero(t, repo.writes)
		})
	}
}

// UserInDeptScope: this file tests role gates, not department scope, so every target is in scope.
func (r *authzRepo) UserInDeptScope(context.Context, deptscope.Scope, string) (bool, error) {
	return true, nil
}
