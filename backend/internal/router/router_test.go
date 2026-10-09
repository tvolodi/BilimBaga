package router_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/bilimbaga/bilimbaga/internal/router"
	"github.com/rs/zerolog"
)

// newRouter builds the full route tree with nil handlers. Handlers are only
// dereferenced when a route executes, so routes that are rejected by middleware
// (auth, heartbeat) can be exercised without a database.
func newRouter() http.Handler {
	return router.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"test-secret", nil, nil, "test", zerolog.Nop())
}

func TestRouter_HeartbeatPing(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/ping status = %d; want 200", rec.Code)
	}
}

func TestRouter_ProtectedRoutesRequireBearerToken(t *testing.T) {
	h := newRouter()
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/auth/change-password"},
		{http.MethodPut, "/api/v1/tenant/config"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d; want 401", tc.method, tc.path, rec.Code)
		}
	}
}

func TestRouter_UnknownRouteIs404(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404", rec.Code)
	}
}

// uuidTestToken returns a valid HS256 JWT accepted by auth.Authenticate("test-secret").
func uuidTestToken(t *testing.T) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "11111111-1111-4111-8111-111111111111",
		"role": "super_admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

// ISS-141: RequireUUIDPathParams must be mounted in the authenticated group.
// Handlers are nil, so a request that reaches one would panic (Recovery -> 500);
// a 404 NOT_FOUND body proves the middleware short-circuited first.
func TestRouter_MalformedUUIDPathParamIs404(t *testing.T) {
	h := newRouter()
	tok := uuidTestToken(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users/not-a-uuid"},
		{http.MethodPut, "/api/v1/users/not-a-uuid"},
		{http.MethodGet, "/api/v1/exams/not-a-uuid"},
		{http.MethodGet, "/api/v1/questions/not-a-uuid"},
		{http.MethodGet, "/api/v1/portal/sessions/not-a-uuid"},
		{http.MethodGet, "/api/v1/portal/sessions/not-a-uuid/result"},
		{http.MethodGet, "/api/v1/admin/grading/not-a-uuid"},
		{http.MethodGet, "/api/v1/admin/exams/not-a-uuid/results/export"}, // ISS-210 G2
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"NOT_FOUND"`) {
			t.Errorf("%s %s = %d %s; want 404 NOT_FOUND from RequireUUIDPathParams", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}

// Routes whose params are not UUIDs must not be rejected by the middleware.
func TestRouter_NonUUIDParamRoutesUnaffected(t *testing.T) {
	h := newRouter()
	tok := uuidTestToken(t)
	const qid = "22222222-2222-4222-8222-222222222222"

	// Authenticated {locale} route with a valid {id}: must pass the middleware
	// (nil handler/cache then panics -> Recovery 500, never the 404 body).
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/"+qid+"/translations/kk", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), "resource not found") {
		t.Errorf("{locale} route rejected by UUID middleware: %d %s", rec.Code, rec.Body.String())
	}

	// Public /verify/{code}: no auth, non-UUID code must reach the handler (nil -> 500), not 401/404.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/verify/ABC-123", nil))
	if rec.Code == http.StatusUnauthorized || strings.Contains(rec.Body.String(), "resource not found") {
		t.Errorf("/verify/{code} affected by UUID middleware or auth: %d %s", rec.Code, rec.Body.String())
	}
}

// nonUUIDPathParamExceptions are URL param names that legitimately hold
// non-UUID values and are therefore NOT validated by RequireUUIDPathParams.
var nonUUIDPathParamExceptions = map[string]string{
	"locale": "language code (kk/ru/en) in /questions/{id}/translations/{locale}",
	"code":   "certificate verification code on the public /verify/{code} route",
}

// ISS-141 drift guard: every path param of every registered route must either
// be covered by api.UUIDPathParamNames or be an explicit documented exception.
func TestRouter_AllPathParamsAreUUIDOrDocumentedExceptions(t *testing.T) {
	uuidNames := map[string]bool{}
	for _, n := range api.UUIDPathParamNames {
		uuidNames[n] = true
	}
	paramRe := regexp.MustCompile(`\{([^}/]+)\}`)
	routes := 0
	err := chi.Walk(newRouter().(*chi.Mux), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes++
		for _, m := range paramRe.FindAllStringSubmatch(route, -1) {
			name := m[1]
			if uuidNames[name] {
				continue
			}
			if _, ok := nonUUIDPathParamExceptions[name]; ok {
				continue
			}
			t.Errorf("%s %s uses path param {%s} which is neither in api.UUIDPathParamNames nor in nonUUIDPathParamExceptions. "+
				"If it always holds a UUID, add it to api.UUIDPathParamNames (backend/internal/api/uuid.go); "+
				"otherwise add it to nonUUIDPathParamExceptions in this test with a justification.", method, route, name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if routes < 50 {
		t.Fatalf("walked only %d routes; router construction looks wrong", routes)
	}
}

// ISS-165: session-keyed admin endpoints are guarded by the department-scope
// middleware. The test router has no database, so a department_admin request
// that reaches the guard fails closed with a 500 before the (nil) handler runs.
func TestRouter_SessionKeyedAdminRoutesAreDepartmentScoped(t *testing.T) {
	cache := rbac.NewCache()
	perms := rbac.PermissionSet{"exams:read": true, "grading:read": true, "grading:write": true}
	cache.LoadFromMap(map[string]rbac.PermissionSet{"department_admin": perms})
	h := router.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"test-secret", cache, nil, "test", zerolog.Nop())

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":           "11111111-1111-4111-8111-111111111111",
		"role":          "department_admin",
		"department_id": "22222222-2222-4222-8222-222222222222",
		"exp":           time.Now().Add(time.Hour).Unix(),
	})
	token, err := tok.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	sid := "33333333-3333-4333-8333-333333333333"
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/sessions/" + sid + "/result"},
		{http.MethodGet, "/api/v1/admin/sessions/" + sid + "/certificate"},
		{http.MethodGet, "/api/v1/admin/grading/" + sid},
		{http.MethodPost, "/api/v1/admin/grading/" + sid + "/answers/" + sid},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "failed to verify department scope") {
			t.Errorf("%s %s: department_admin not guarded (status %d body %s)", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}
