package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-jwt-secret-key"

// makeToken creates a signed JWT for testing.
func makeToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// okHandler is a trivial next handler that returns 200.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func decodeErrorEnvelope(t *testing.T, body *httptest.ResponseRecorder) (code string) {
	t.Helper()
	var env struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(body.Body).Decode(&env))
	require.NotNil(t, env.Error)
	return env.Error.Code
}

func TestAuthenticate(t *testing.T) {
	validClaims := jwt.MapClaims{
		"sub":           "user-uuid-1",
		"role":          "employee",
		"department_id": "dept-uuid-1",
		"exp":           time.Now().Add(15 * time.Minute).Unix(),
	}

	tests := []struct {
		name           string
		authHeader     string
		wantStatus     int
		wantCode       string // empty means no error envelope check
		checkCtx       func(t *testing.T, r *http.Request)
	}{
		{
			name:       "AC-1: missing Authorization header → 401 MISSING_TOKEN",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "MISSING_TOKEN",
		},
		{
			name:       "AC-8: non-Bearer prefix → 401 INVALID_TOKEN",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_TOKEN",
		},
		{
			name: "AC-2: expired token → 401 TOKEN_EXPIRED",
			authHeader: "Bearer " + makeToken(t, testSecret, jwt.MapClaims{
				"sub":  "user-uuid-1",
				"role": "employee",
				"exp":  time.Now().Add(-10 * time.Minute).Unix(),
			}),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "TOKEN_EXPIRED",
		},
		{
			name:       "AC-3: wrong signature → 401 INVALID_TOKEN",
			authHeader: "Bearer " + makeToken(t, "wrong-secret", validClaims),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_TOKEN",
		},
		{
			name: "AC-9: missing role claim → 401 INVALID_TOKEN",
			authHeader: "Bearer " + makeToken(t, testSecret, jwt.MapClaims{
				"sub": "user-uuid-1",
				"exp": time.Now().Add(15 * time.Minute).Unix(),
			}),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_TOKEN",
		},
		{
			name: "AC-9: missing sub claim → 401 INVALID_TOKEN",
			authHeader: "Bearer " + makeToken(t, testSecret, jwt.MapClaims{
				"role": "employee",
				"exp":  time.Now().Add(15 * time.Minute).Unix(),
			}),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_TOKEN",
		},
		{
			name:       "AC-4: valid token → 200, claims injected into context",
			authHeader: "Bearer " + makeToken(t, testSecret, validClaims),
			wantStatus: http.StatusOK,
			checkCtx: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "user-uuid-1", UserIDFromCtx(r.Context()))
				assert.Equal(t, "employee", RoleFromCtx(r.Context()))
				assert.Equal(t, "dept-uuid-1", DepartmentIDFromCtx(r.Context()))
			},
		},
		{
			name: "AC-4+AC-9: valid token without department_id → 200, empty department",
			authHeader: "Bearer " + makeToken(t, testSecret, jwt.MapClaims{
				"sub":  "user-uuid-2",
				"role": "admin",
				"exp":  time.Now().Add(15 * time.Minute).Unix(),
			}),
			wantStatus: http.StatusOK,
			checkCtx: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "user-uuid-2", UserIDFromCtx(r.Context()))
				assert.Equal(t, "admin", RoleFromCtx(r.Context()))
				assert.Equal(t, "", DepartmentIDFromCtx(r.Context()))
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// capture request inside handler to inspect context
			var captured *http.Request
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = r
				w.WriteHeader(http.StatusOK)
			})

			handler := Authenticate(testSecret)(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantCode != "" {
				assert.Equal(t, tc.wantCode, decodeErrorEnvelope(t, w))
			}
			if tc.checkCtx != nil {
				require.NotNil(t, captured)
				tc.checkCtx(t, captured)
			}
		})
	}
}
