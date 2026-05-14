package rbac

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-rbac-jwt-secret"

// makeTestToken creates a signed JWT with the given role for testing.
func makeTestToken(t *testing.T, role string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  "user-id",
		"role": role,
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeTestToken: %v", err)
	}
	return signed
}

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func decodeErrorCode(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&env); err != nil {
		t.Fatalf("decodeErrorCode: %v", err)
	}
	if env.Error == nil {
		return ""
	}
	return env.Error.Code
}

func TestRequirePermission(t *testing.T) {
	cache := NewCache()
	cache.data = map[string]PermissionSet{
		"examiner": {"questions:write": true, "questions:read": true},
		"employee": {"portal:read": true},
	}

	tests := []struct {
		name       string
		role       string
		skipAuth   bool // if true, RequirePermission is called without auth.Authenticate
		resource   string
		action     string
		wantStatus int
		wantCode   string
	}{
		{
			// No auth.Authenticate in the chain → role "" in context → RequirePermission returns 401
			name:       "no role in context returns 401",
			skipAuth:   true,
			resource:   "questions",
			action:     "write",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "MISSING_TOKEN",
		},
		{
			name:       "role without permission returns 403",
			role:       "employee",
			resource:   "questions",
			action:     "write",
			wantStatus: http.StatusForbidden,
			wantCode:   "FORBIDDEN",
		},
		{
			name:       "role with permission calls next handler",
			role:       "examiner",
			resource:   "questions",
			action:     "write",
			wantStatus: http.StatusOK,
			wantCode:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rbacMW := RequirePermission(cache, tt.resource, tt.action)

			var handler http.Handler
			if tt.skipAuth {
				// Call RequirePermission directly; auth.Authenticate hasn't run so role is "".
				handler = rbacMW(okHandler)
			} else {
				// Wrap with auth.Authenticate so the role is stored in context.
				handler = auth.Authenticate(testSecret)(rbacMW(okHandler))
			}

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if !tt.skipAuth {
				req.Header.Set("Authorization", "Bearer "+makeTestToken(t, tt.role))
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantCode != "" {
				if code := decodeErrorCode(t, rr); code != tt.wantCode {
					t.Errorf("error code = %q, want %q", code, tt.wantCode)
				}
			}
		})
	}
}
