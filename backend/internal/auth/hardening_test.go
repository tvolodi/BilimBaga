package auth

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ISS-105 (1): known and unknown emails must do structurally equal work.
// ---------------------------------------------------------------------------

func TestForgotPassword_AllPathsRunTheSameRepositorySequence(t *testing.T) {
	emails := map[string]string{
		"known":     "alice@example.com",
		"unknown":   "nobody@example.com",
		"inactive":  "gone@example.com",
		"throttled": "alice@example.com",
	}
	want := []string{"GetUserByEmail", "PurgeExpiredResetTokens", "IssueResetToken"}
	for name, email := range emails {
		f := newRecoveryFixture(t)
		if name == "throttled" {
			for i := 0; i < 3; i++ {
				_, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: email}, "")
				require.NoError(t, err)
			}
		}
		f.repo.calls = nil
		_, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: email}, "")
		require.NoError(t, err)
		assert.Equal(t, want, f.repo.calls, "path %q must run the identical repository sequence", name)
	}
}

func TestForgotPassword_Handler_PadsToMinimumDuration(t *testing.T) {
	h, _ := newRecoveryHandler(&mockService{forgotFn: func(context.Context, *ForgotPasswordRequest, string) (string, error) { return "", nil }})
	h.forgotMin = 400 * time.Millisecond
	var slept []time.Duration
	h.sleep = func(d time.Duration) { slept = append(slept, d) }

	w := postJSON(h.ForgotPassword, `{"email":"nobody@example.com"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, slept, 1, "fast (unknown email) path is padded")
	assert.Greater(t, slept[0], 300*time.Millisecond)
	assert.LessOrEqual(t, slept[0], 400*time.Millisecond)
}

func TestForgotPassword_Handler_NoPaddingBeyondBudget(t *testing.T) {
	h, _ := newRecoveryHandler(&mockService{forgotFn: func(context.Context, *ForgotPasswordRequest, string) (string, error) {
		time.Sleep(20 * time.Millisecond)
		return "", nil
	}})
	h.forgotMin = 10 * time.Millisecond
	called := false
	h.sleep = func(time.Duration) { called = true }
	postJSON(h.ForgotPassword, `{"email":"a@b.co"}`)
	assert.False(t, called, "no sleep when the request already exceeded the minimum")
}

func TestNewHandler_EnablesForgotPadding(t *testing.T) {
	assert.Equal(t, forgotMinDuration, NewHandler(nil, nil).forgotMin)
}

// ---------------------------------------------------------------------------
// ISS-105 (2): the 3/hour throttle is atomic.
// ---------------------------------------------------------------------------

func TestForgotPassword_ConcurrentRequests_CannotExceedLimit(t *testing.T) {
	f := newRecoveryFixture(t)
	var wg sync.WaitGroup
	var issued int32
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")
			if err == nil && id != "" {
				atomic.AddInt32(&issued, 1)
			}
		}()
	}
	wg.Wait()
	assert.EqualValues(t, resetMaxPerHour, issued, "only 3 of 40 concurrent requests may issue a token")
}

// txFakeDB is a transaction-aware database/sql driver that records statements in order
// and serves canned results for the row-returning ones.
type txFakeDB struct {
	mu        sync.Mutex
	stmts     []string
	lockRows  int // rows returned by the FOR UPDATE select (0 = no such active user)
	count     int64
	begun     int
	commits   int
	isolation sql.IsolationLevel
}

type txConn struct{ f *txFakeDB }
type txStmt struct {
	f *txFakeDB
	q string
}
type txTx struct{ f *txFakeDB }
type txRows struct {
	cols []string
	data [][]driver.Value
	i    int
}

func (c txConn) Prepare(q string) (driver.Stmt, error) { return txStmt{c.f, q}, nil }
func (c txConn) Close() error                          { return nil }
func (c txConn) Begin() (driver.Tx, error) {
	c.f.mu.Lock()
	c.f.begun++
	c.f.mu.Unlock()
	return txTx(c), nil
}
func (t txTx) Commit() error {
	t.f.mu.Lock()
	t.f.commits++
	t.f.mu.Unlock()
	return nil
}
func (t txTx) Rollback() error { return nil }
func (s txStmt) Close() error  { return nil }
func (s txStmt) NumInput() int { return -1 }
func (s txStmt) Exec([]driver.Value) (driver.Result, error) {
	s.f.mu.Lock()
	s.f.stmts = append(s.f.stmts, s.q)
	s.f.mu.Unlock()
	return driver.RowsAffected(1), nil
}
func (s txStmt) Query([]driver.Value) (driver.Rows, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.stmts = append(s.f.stmts, s.q)
	switch {
	case strings.Contains(s.q, "FOR UPDATE"):
		r := &txRows{cols: []string{"id"}}
		for i := 0; i < s.f.lockRows; i++ {
			r.data = append(r.data, []driver.Value{"u1"})
		}
		return r, nil
	case strings.Contains(s.q, "COUNT(*)"):
		return &txRows{cols: []string{"count"}, data: [][]driver.Value{{s.f.count}}}, nil
	case strings.Contains(s.q, "RETURNING user_id"):
		return &txRows{cols: []string{"user_id"}, data: [][]driver.Value{{"u1"}}}, nil
	}
	return &txRows{}, nil
}
func (r *txRows) Columns() []string { return r.cols }
func (r *txRows) Close() error      { return nil }
func (r *txRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.i])
	r.i++
	return nil
}

type txConnector struct{ f *txFakeDB }

func (c txConnector) Connect(context.Context) (driver.Conn, error) { return txConn(c), nil }
func (c txConnector) Driver() driver.Driver                        { return nil }

func newTxFake(t *testing.T) (Repository, *txFakeDB) {
	t.Helper()
	f := &txFakeDB{lockRows: 1}
	db := sqlx.NewDb(sql.OpenDB(txConnector{f}), "postgres")
	t.Cleanup(func() { _ = db.Close() })
	return NewRepository(db), f
}

func TestIssueResetToken_LocksUserRowBeforeCountingInOneTransaction(t *testing.T) {
	repo, f := newTxFake(t)
	f.count = 1
	now := time.Now()

	ok, err := repo.IssueResetToken(context.Background(), "u1", "h", now.Add(time.Hour), now, now.Add(-time.Hour), 3)

	require.NoError(t, err)
	assert.True(t, ok)
	require.Len(t, f.stmts, 4)
	assert.Contains(t, f.stmts[0], "FOR UPDATE", "row lock must come first")
	assert.Contains(t, f.stmts[0], "FROM users")
	assert.Contains(t, f.stmts[1], "COUNT(*)")
	assert.Contains(t, f.stmts[2], "UPDATE password_reset_tokens")
	assert.Contains(t, f.stmts[3], "INSERT INTO password_reset_tokens")
	assert.Equal(t, 1, f.begun, "lock, count and insert share one transaction")
	assert.Equal(t, sql.LevelReadCommitted, f.isolation, "isolation must not depend on server defaults")
	assert.Equal(t, 1, f.commits)
}

func TestIssueResetToken_AtLimit_InsertsNothing(t *testing.T) {
	repo, f := newTxFake(t)
	f.count = 3
	now := time.Now()
	ok, err := repo.IssueResetToken(context.Background(), "u1", "h", now, now, now, 3)
	require.NoError(t, err)
	assert.False(t, ok)
	for _, s := range f.stmts {
		assert.NotContains(t, s, "INSERT")
	}
	assert.Equal(t, 0, f.commits)
}

func TestIssueResetToken_NoActiveUser_StopsAfterLock(t *testing.T) {
	repo, f := newTxFake(t)
	f.lockRows = 0
	now := time.Now()
	ok, err := repo.IssueResetToken(context.Background(), noUserID, "h", now, now, now, 3)
	require.NoError(t, err)
	assert.False(t, ok)
	require.Len(t, f.stmts, 1)
	assert.Contains(t, f.stmts[0], "FOR UPDATE")
}

func TestCompleteReset_StampsPasswordChangedAt(t *testing.T) {
	repo, f := newTxFake(t)
	uid, err := repo.CompleteReset(context.Background(), "h", "pw", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "u1", uid)
	found := false
	for _, s := range f.stmts {
		if strings.Contains(s, "UPDATE users") && strings.Contains(s, "password_changed_at") {
			found = true
		}
	}
	assert.True(t, found, "reset must stamp users.password_changed_at")
}

// ---------------------------------------------------------------------------
// ISS-105 (4): access tokens issued before a password reset stop working.
// ---------------------------------------------------------------------------

func epochToken(t *testing.T, iat time.Time, withIAT bool) string {
	t.Helper()
	c := jwt.MapClaims{"sub": "u1", "role": "employee", "exp": time.Now().Add(15 * time.Minute).Unix()}
	if withIAT {
		c["iat"] = iat.Unix()
	}
	return makeToken(t, testSecret, c)
}

func runEpoch(t *testing.T, lookup AccountStateLookup, tok string) *httptest.ResponseRecorder {
	t.Helper()
	h := Authenticate(testSecret, WithAccountState(lookup))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAuthenticate_TokenIssuedBeforePasswordReset_Rejected(t *testing.T) {
	reset := time.Now()
	lookup := func(context.Context, string) (AccountState, error) { return AccountState{PasswordChangedAt: reset}, nil }

	old := runEpoch(t, lookup, epochToken(t, reset.Add(-5*time.Minute), true))
	assert.Equal(t, http.StatusUnauthorized, old.Code)
	assert.Equal(t, "TOKEN_REVOKED", decodeErrorEnvelope(t, old))

	fresh := runEpoch(t, lookup, epochToken(t, reset.Add(time.Second), true))
	assert.Equal(t, http.StatusOK, fresh.Code, "token issued after the reset is valid")
}

func TestAuthenticate_NeverResetUser_Accepted(t *testing.T) {
	lookup := func(context.Context, string) (AccountState, error) { return AccountState{}, nil }
	assert.Equal(t, http.StatusOK, runEpoch(t, lookup, epochToken(t, time.Now(), true)).Code)
}

func TestAuthenticate_Epoch_MissingIATRejected(t *testing.T) {
	lookup := func(context.Context, string) (AccountState, error) { return AccountState{}, nil }
	rec := runEpoch(t, lookup, epochToken(t, time.Now(), false))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthenticate_Epoch_UnknownUserRejected_LookupErrorFailsClosed(t *testing.T) {
	gone := func(context.Context, string) (AccountState, error) { return AccountState{}, ErrNotFound }
	assert.Equal(t, http.StatusUnauthorized, runEpoch(t, gone, epochToken(t, time.Now(), true)).Code)

	broken := func(context.Context, string) (AccountState, error) { return AccountState{}, errors.New("db down") }
	assert.Equal(t, http.StatusInternalServerError, runEpoch(t, broken, epochToken(t, time.Now(), true)).Code)
}

func TestAccountStateCache_CachesPerUserWithinTTL(t *testing.T) {
	var fetches int
	clock := testNow(0)
	c := NewAccountStateCache(func(context.Context, string) (AccountState, error) {
		fetches++
		return AccountState{PasswordChangedAt: clock}, nil
	}, func() time.Time { return clock })

	for i := 0; i < 5; i++ {
		_, err := c.Lookup(context.Background(), "u1")
		require.NoError(t, err)
	}
	assert.Equal(t, 1, fetches, "repeat requests within the TTL hit the cache")

	_, _ = c.Lookup(context.Background(), "u2")
	assert.Equal(t, 2, fetches, "cache is per user")

	clock = clock.Add(accountStateTTL + time.Second)
	_, _ = c.Lookup(context.Background(), "u1")
	assert.Equal(t, 3, fetches, "entry refreshed after TTL")
}

func TestTokenPredatesPasswordChange(t *testing.T) {
	assert.True(t, tokenPredatesPasswordChange(100, time.Unix(200, 0)))
	assert.False(t, tokenPredatesPasswordChange(200, time.Unix(200, 0)))
	assert.False(t, tokenPredatesPasswordChange(1, time.Time{}))
	assert.Contains(t, accountStateSQL, "password_changed_at")
	assert.Contains(t, accountStateSQL, "force_password_change")
}

// BeginTx lets the fake accept (and record) a non-default isolation level.
func (c txConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.f.mu.Lock()
	c.f.isolation = sql.IsolationLevel(opts.Isolation)
	c.f.mu.Unlock()
	return c.Begin()
}
