package auth

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// #455: a no-Postgres simulation of one account. Sign-ins, admin-style resets and self password
// changes run at random application times a few hundred milliseconds apart, so many of them share a
// second. After every step, a token must be rejected by the real Authenticate middleware exactly when
// it was issued before the latest stamping write (reset or change), and accepted otherwise.

const (
	simStartPassword  = "PassAlpha1"
	simOtherPassword  = "PassBeta2"
	simTempPassword   = "TempPass9"
	simSequences      = 300
	simStepsPerSeq    = 12
	simAccountEmail   = "a@example.com"
	simAccountUserID  = "u1"
	simRandomSeed     = 455
	simBcryptCostTest = bcrypt.MinCost
)

// simStepGapsMs are the gaps between steps. Most are short, so steps land in the same second often.
var simStepGapsMs = []int{0, 100, 300, 500, 700, 950}

type simToken struct {
	tok      string
	issuedBy int // index of the step that issued it
}

// simAccount drives one account through the real service and the real middleware. The fake repository
// holds the user's password hash and password_changed_at, the state a request reads.
type simAccount struct {
	t            *testing.T
	svc          *service
	clock        time.Time
	passwordHash string
	password     string
	stamp        *time.Time
}

func newSimAccount(t *testing.T, start time.Time) *simAccount {
	t.Helper()
	a := &simAccount{t: t, clock: start, password: simStartPassword, passwordHash: simHash(t, simStartPassword)}
	repo := &mockRepository{
		getUserByEmailFn: func(context.Context, string) (*User, error) { return a.user(), nil },
		getUserByIDFn:    func(context.Context, string) (*User, error) { return a.user(), nil },
		passwordStamp: func() *time.Time { return a.stamp },
		updatePasswordFn: func(_ context.Context, _ string, hash string, at time.Time) error {
			a.passwordHash = hash
			a.stamp = &at
			return nil
		},
	}
	a.svc = NewService(ServiceConfig{JWTSecret: selfChangeSecret, JWTAccessTTLMin: 15, JWTRefreshTTLDays: 7, BcryptCost: simBcryptCostTest}, repo).(*service)
	a.svc.now = func() time.Time { return a.clock }
	return a
}

func simHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), simBcryptCostTest)
	require.NoError(t, err)
	return string(h)
}

func (a *simAccount) user() *User {
	return &User{
		ID: simAccountUserID, Email: simAccountEmail, RoleName: "employee", Status: "active",
		PasswordHash: a.passwordHash, PasswordChangedAt: a.stamp,
	}
}

// modelResetStamp is the stamp an admin or recovery reset writes: NextPasswordStamp over the
// application clock and the account's previous stamp, as the repositories compute it under their lock.
func modelResetStamp(now time.Time, previous *time.Time) time.Time { return NextPasswordStamp(now, previous) }

func (a *simAccount) signIn() string {
	resp, _, err := a.svc.Login(context.Background(), &LoginRequest{Email: simAccountEmail, Password: a.password}, "127.0.0.1")
	require.NoError(a.t, err)
	return resp.AccessToken
}

func (a *simAccount) change() string {
	next := simOtherPassword
	if a.password == simOtherPassword {
		next = simStartPassword
	}
	resp, _, err := a.svc.ChangePassword(context.Background(), simAccountUserID,
		&ChangePasswordRequest{CurrentPassword: a.password, NewPassword: next}, "127.0.0.1")
	require.NoError(a.t, err)
	a.password = next
	return resp.AccessToken
}

func (a *simAccount) reset() {
	a.password = simTempPassword
	a.passwordHash = simHash(a.t, simTempPassword)
	stamp := modelResetStamp(a.clock, a.stamp)
	a.stamp = &stamp
}

// accepts reports whether the real Authenticate middleware, over this account's state, lets tok through.
func (a *simAccount) accepts(tok string) bool {
	lookup := func(context.Context, string) (AccountState, error) {
		st := AccountState{}
		if a.stamp != nil {
			st.PasswordChangedAt = *a.stamp
		}
		return st, nil
	}
	mw := Authenticate(selfChangeSecret, WithAccountState(lookup))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	return rec.Code == http.StatusOK
}

func TestStampSimulation_RandomSameSecondSequences(t *testing.T) {
	rng := rand.New(rand.NewSource(simRandomSeed))
	base := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	for seq := 0; seq < simSequences; seq++ {
		a := newSimAccount(t, base.Add(time.Duration(rng.Intn(1000))*time.Millisecond))
		var tokens []simToken
		var log []string
		writer := -1 // index of the latest step that wrote password_changed_at
		for i := 0; i < simStepsPerSeq; i++ {
			a.clock = a.clock.Add(time.Duration(simStepGapsMs[rng.Intn(len(simStepGapsMs))]) * time.Millisecond)
			op := []string{"sign-in", "reset", "change"}[rng.Intn(3)]
			switch op {
			case "sign-in":
				tokens = append(tokens, simToken{tok: a.signIn(), issuedBy: i})
			case "reset":
				a.reset()
				writer = i
			case "change":
				tokens = append(tokens, simToken{tok: a.change(), issuedBy: i})
				writer = i
			}
			log = append(log, fmt.Sprintf("step %d %s at %s", i, op, a.clock.Format("05.000")))
			for _, tk := range tokens {
				want := tk.issuedBy >= writer
				if got := a.accepts(tk.tok); got != want {
					t.Fatalf("sequence %d: token from step %d accepted=%v, want %v (latest write: step %d)\n  %s",
						seq, tk.issuedBy, got, want, writer, strings.Join(log, "\n  "))
				}
			}
		}
	}
}
