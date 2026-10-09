package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// ISS-240 / FR-BB117 D-4: the per-request account lookup also checks status, role and
// department against the database, so a demotion, move or deactivation revokes the token
// within the cache TTL instead of the JWT lifetime.

func claimsToken(t *testing.T, role, dept string) string {
	t.Helper()
	c := jwt.MapClaims{"sub": "u1", "role": role, "iat": time.Now().Unix(), "exp": time.Now().Add(15 * time.Minute).Unix()}
	if dept != "" {
		c["department_id"] = dept
	}
	return makeToken(t, testSecret, c)
}

func stateOf(status, role, dept string) AccountStateLookup {
	return func(context.Context, string) (AccountState, error) {
		return AccountState{Status: status, RoleName: role, DepartmentID: dept}, nil
	}
}

func TestAuthenticate_StaleClaims(t *testing.T) {
	cases := []struct {
		name  string
		state AccountStateLookup
		tok   string
		want  int
	}{
		{"matching claims accepted", stateOf("active", "department_admin", "d1"), claimsToken(t, "department_admin", "d1"), http.StatusOK},
		{"no department on both sides accepted", stateOf("active", "super_admin", ""), claimsToken(t, "super_admin", ""), http.StatusOK},
		{"demoted user revoked", stateOf("active", "employee", "d1"), claimsToken(t, "department_admin", "d1"), http.StatusUnauthorized},
		{"promoted user revoked (token role lower)", stateOf("active", "department_admin", "d1"), claimsToken(t, "employee", "d1"), http.StatusUnauthorized},
		{"moved department revoked", stateOf("active", "department_admin", "d2"), claimsToken(t, "department_admin", "d1"), http.StatusUnauthorized},
		{"department removed in DB revoked", stateOf("active", "department_admin", ""), claimsToken(t, "department_admin", "d1"), http.StatusUnauthorized},
		{"department added in DB revoked", stateOf("active", "department_admin", "d1"), claimsToken(t, "department_admin", ""), http.StatusUnauthorized},
		{"deactivated user revoked", stateOf("inactive", "department_admin", "d1"), claimsToken(t, "department_admin", "d1"), http.StatusUnauthorized},
		{"unknown status revoked", stateOf("suspended", "department_admin", "d1"), claimsToken(t, "department_admin", "d1"), http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := runEpoch(t, c.state, c.tok)
			assert.Equal(t, c.want, rec.Code)
			if c.want == http.StatusUnauthorized {
				assert.Equal(t, "TOKEN_REVOKED", decodeErrorEnvelope(t, rec))
			}
		})
	}
}

func TestClaimsStale_StateWithoutIdentityIsNotChecked(t *testing.T) {
	assert.False(t, claimsStale(AccountState{}, "anything", "d9"))
}

// The cache must serve the identity fields too, and refresh them after the TTL.
func TestAccountStateCache_IdentityRefreshedAfterTTL(t *testing.T) {
	clock := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	role := "department_admin"
	c := NewAccountStateCache(func(context.Context, string) (AccountState, error) {
		return AccountState{Status: "active", RoleName: role}, nil
	}, func() time.Time { return clock })

	st, _ := c.Lookup(context.Background(), "u1")
	assert.Equal(t, "department_admin", st.RoleName)
	role = "employee"
	st, _ = c.Lookup(context.Background(), "u1")
	assert.Equal(t, "department_admin", st.RoleName, "served from cache within the TTL")
	clock = clock.Add(accountStateTTL + time.Second)
	st, _ = c.Lookup(context.Background(), "u1")
	assert.Equal(t, "employee", st.RoleName, "demotion visible after the TTL")
}
