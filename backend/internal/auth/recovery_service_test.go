package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// --- in-memory repository modelling the recovery tables ---

type resetRow struct {
	userID    string
	hash      string
	expiresAt time.Time
	usedAt    *time.Time
	createdAt time.Time
}

type recoveryRepo struct {
	users         map[string]*User // by email
	byID          map[string]*User
	resets        []*resetRow
	refreshActive map[string]int // userID -> active refresh tokens
	now           func() time.Time
	createErr     error
	completeErr   error
	purgedCutoff  time.Time
	mu            sync.Mutex
	calls         []string // sequence of repository calls made by ForgotPassword
}

func newRecoveryRepo(now func() time.Time) *recoveryRepo {
	return &recoveryRepo{users: map[string]*User{}, byID: map[string]*User{}, refreshActive: map[string]int{}, now: now}
}

func (r *recoveryRepo) addUser(u *User) {
	r.users[u.Email] = u
	r.byID[u.ID] = u
}

func (r *recoveryRepo) GetUserByEmail(_ context.Context, email string) (*User, error) {
	r.mu.Lock()
	r.calls = append(r.calls, "GetUserByEmail")
	r.mu.Unlock()
	if u, ok := r.users[email]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}
func (r *recoveryRepo) GetUserByID(_ context.Context, id string) (*User, error) {
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}
func (r *recoveryRepo) UpdateFailedAttempts(context.Context, string, int) error     { return nil }
func (r *recoveryRepo) LockAccount(context.Context, string, time.Time) error        { return nil }
func (r *recoveryRepo) ResetFailedAttempts(context.Context, string) error            { return nil }
func (r *recoveryRepo) CreateRefreshToken(context.Context, *RefreshToken) error      { return nil }
func (r *recoveryRepo) RevokeRefreshToken(context.Context, string) error             { return nil }
func (r *recoveryRepo) RevokeAllUserRefreshTokens(context.Context, string) error     { return nil }
func (r *recoveryRepo) UpdatePassword(context.Context, string, string) error         { return nil }
func (r *recoveryRepo) GetRefreshTokenByHash(context.Context, string) (*RefreshToken, error) {
	return nil, ErrNotFound
}

// IssueResetToken models the atomic repository contract: the whole check-and-insert runs
// under the mutex, exactly as the user-row lock serialises it in Postgres.
func (r *recoveryRepo) IssueResetToken(_ context.Context, userID, hash string, exp, now, since time.Time, max int) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "IssueResetToken")
	if r.createErr != nil {
		return false, r.createErr
	}
	u := r.byID[userID]
	if u == nil || u.Status != "active" {
		return false, nil
	}
	n := 0
	for _, t := range r.resets {
		if t.userID == userID && t.createdAt.After(since) {
			n++
		}
	}
	if n >= max {
		return false, nil
	}
	for _, t := range r.resets {
		if t.userID == userID && t.usedAt == nil {
			t.usedAt = &now
		}
	}
	r.resets = append(r.resets, &resetRow{userID: userID, hash: hash, expiresAt: exp, createdAt: now})
	return true, nil
}

func (r *recoveryRepo) PurgeExpiredResetTokens(_ context.Context, cutoff time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "PurgeExpiredResetTokens")
	r.purgedCutoff = cutoff
	kept := r.resets[:0]
	for _, t := range r.resets {
		if !t.expiresAt.Before(cutoff) {
			kept = append(kept, t)
		}
	}
	r.resets = kept
	return nil
}

func (r *recoveryRepo) CompleteReset(_ context.Context, hash, pwHash string, now time.Time) (string, error) {
	if r.completeErr != nil {
		return "", r.completeErr
	}
	for _, t := range r.resets {
		if t.hash != hash || t.usedAt != nil || !t.expiresAt.After(now) {
			continue
		}
		u := r.byID[t.userID]
		if u == nil || u.Status != "active" {
			return "", ErrNotFound
		}
		t.usedAt = &now
		u.PasswordHash = pwHash
		u.ForcePasswordChange = false
		u.FailedAttempts = 0
		u.LockedUntil = nil
		r.refreshActive[u.ID] = 0
		for _, o := range r.resets {
			if o.userID == u.ID && o.usedAt == nil {
				o.usedAt = &now
			}
		}
		return u.ID, nil
	}
	return "", ErrNotFound
}

type fakeMailer struct {
	calls []struct{ userID, token string }
}

func (m *fakeMailer) TriggerPasswordResetLink(userID, token string) {
	m.calls = append(m.calls, struct{ userID, token string }{userID, token})
}

type recoveryFixture struct {
	svc    Service
	repo   *recoveryRepo
	mailer *fakeMailer
	clock  *time.Time
}

func newRecoveryFixture(t *testing.T) *recoveryFixture {
	t.Helper()
	clock := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	f := &recoveryFixture{clock: &clock, mailer: &fakeMailer{}}
	f.repo = newRecoveryRepo(func() time.Time { return *f.clock })
	locked := clock.Add(20 * time.Minute)
	f.repo.addUser(&User{ID: "u1", Email: "alice@example.com", Status: "active", PasswordHash: "old", FailedAttempts: 5, LockedUntil: &locked, ForcePasswordChange: true})
	f.repo.addUser(&User{ID: "u2", Email: "gone@example.com", Status: "inactive"})
	f.repo.refreshActive["u1"] = 2
	s := NewService(ServiceConfig{BcryptCost: bcrypt.MinCost}, f.repo, f.mailer).(*service)
	s.now = func() time.Time { return *f.clock }
	f.svc = s
	return f
}

func serviceErrCode(t *testing.T, err error) string {
	t.Helper()
	var se *ServiceError
	require.True(t, errors.As(err, &se), "expected ServiceError, got %v", err)
	return se.Code
}

// --- ForgotPassword (AC-1, AC-2, AC-3, AC-9) ---

func TestForgotPassword_ActiveUser_StoresHashedTokenWith60MinExpiry(t *testing.T) {
	f := newRecoveryFixture(t)

	id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "1.2.3.4")

	require.NoError(t, err)
	assert.Equal(t, "u1", id)
	require.Len(t, f.mailer.calls, 1)
	raw := f.mailer.calls[0].token
	assert.GreaterOrEqual(t, len(raw), 43, "32 random bytes => >= 43 base64url chars")
	assert.NotContains(t, raw, "=")
	require.Len(t, f.repo.resets, 1)
	row := f.repo.resets[0]
	assert.Equal(t, hashToken(raw), row.hash, "only the SHA-256 hash is stored")
	assert.NotEqual(t, raw, row.hash)
	assert.Equal(t, f.clock.Add(60*time.Minute), row.expiresAt)
	assert.Nil(t, row.usedAt)
}

func TestForgotPassword_TokensAreUnique(t *testing.T) {
	f := newRecoveryFixture(t)
	for i := 0; i < 2; i++ {
		_, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")
		require.NoError(t, err)
	}
	assert.NotEqual(t, f.mailer.calls[0].token, f.mailer.calls[1].token)
}

func TestForgotPassword_UnknownEmail_NoTokenNoMailNoError(t *testing.T) {
	f := newRecoveryFixture(t)

	id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "nobody@example.com"}, "")

	require.NoError(t, err)
	assert.Empty(t, id)
	assert.Empty(t, f.mailer.calls)
	assert.Empty(t, f.repo.resets)
}

func TestForgotPassword_InactiveUser_NoTokenNoMail(t *testing.T) {
	f := newRecoveryFixture(t)

	id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "gone@example.com"}, "")

	require.NoError(t, err)
	assert.Empty(t, id)
	assert.Empty(t, f.mailer.calls)
	assert.Empty(t, f.repo.resets)
}

func TestForgotPassword_MalformedEmail_ReturnsValidationError(t *testing.T) {
	f := newRecoveryFixture(t)
	for _, bad := range []string{"", "   ", "not-an-email", "Alice <alice@example.com>"} {
		_, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: bad}, "")
		require.Error(t, err, bad)
		assert.Equal(t, "VALIDATION_ERROR", serviceErrCode(t, err), bad)
		var se *ServiceError
		require.True(t, errors.As(err, &se))
		assert.Equal(t, http.StatusUnprocessableEntity, se.HTTPStatus, bad)
	}
	assert.Empty(t, f.mailer.calls)
}

func TestForgotPassword_NewRequestInvalidatesEarlierTokens(t *testing.T) {
	f := newRecoveryFixture(t)
	_, _ = f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")
	first := f.mailer.calls[0].token
	_, _ = f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")
	second := f.mailer.calls[1].token

	_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: first, NewPassword: "Str0ngPass"}, "")
	require.Error(t, err)
	assert.Equal(t, "INVALID_TOKEN", serviceErrCode(t, err))

	_, err = f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: second, NewPassword: "Str0ngPass"}, "")
	require.NoError(t, err)
}

func TestForgotPassword_ThrottledAfterThreePerHour_StillSucceedsSilently(t *testing.T) {
	f := newRecoveryFixture(t)
	for i := 0; i < 3; i++ {
		id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")
		require.NoError(t, err)
		assert.Equal(t, "u1", id)
	}

	id, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")

	require.NoError(t, err)
	assert.Empty(t, id, "4th request in the hour issues nothing")
	assert.Len(t, f.mailer.calls, 3)

	// After the window passes, requests work again.
	*f.clock = f.clock.Add(61 * time.Minute)
	id, err = f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")
	require.NoError(t, err)
	assert.Equal(t, "u1", id)
}

func TestForgotPassword_PurgesLongExpiredTokens(t *testing.T) {
	f := newRecoveryFixture(t)
	f.repo.resets = append(f.repo.resets, &resetRow{userID: "u1", hash: "old", expiresAt: f.clock.Add(-48 * time.Hour), createdAt: f.clock.Add(-49 * time.Hour)})

	_, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")

	require.NoError(t, err)
	assert.Len(t, f.repo.resets, 1)
	assert.Equal(t, f.clock.Add(-24*time.Hour), f.repo.purgedCutoff)
}

func TestForgotPassword_StoreFailure_ReturnsWrappedErrorWithoutMail(t *testing.T) {
	f := newRecoveryFixture(t)
	f.repo.createErr = errors.New("boom")

	_, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.service.ForgotPassword")
	assert.Empty(t, f.mailer.calls)
}

// --- ResetPassword (AC-4, AC-9) ---

func issueToken(t *testing.T, f *recoveryFixture) string {
	t.Helper()
	_, err := f.svc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "alice@example.com"}, "")
	require.NoError(t, err)
	return f.mailer.calls[len(f.mailer.calls)-1].token
}

func TestResetPassword_ValidToken_SetsPasswordClearsLockRevokesSessions(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)

	id, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "Str0ngPass"}, "")

	require.NoError(t, err)
	assert.Equal(t, "u1", id)
	u := f.repo.byID["u1"]
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("Str0ngPass")))
	assert.False(t, u.ForcePasswordChange)
	assert.Equal(t, 0, u.FailedAttempts)
	assert.Nil(t, u.LockedUntil)
	assert.Equal(t, 0, f.repo.refreshActive["u1"])
	require.NotNil(t, f.repo.resets[0].usedAt)
}

func TestResetPassword_TokenIsSingleUse(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)
	_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "Str0ngPass"}, "")
	require.NoError(t, err)

	_, err = f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "An0therPass"}, "")

	require.Error(t, err)
	assert.Equal(t, "INVALID_TOKEN", serviceErrCode(t, err))
}

func TestResetPassword_ExpiredToken_ReturnsInvalidToken(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)
	*f.clock = f.clock.Add(60*time.Minute + time.Second)

	_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "Str0ngPass"}, "")

	require.Error(t, err)
	assert.Equal(t, "INVALID_TOKEN", serviceErrCode(t, err))
	assert.Equal(t, "old", f.repo.byID["u1"].PasswordHash)
}

func TestResetPassword_TokenStillValidJustBeforeExpiry(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)
	*f.clock = f.clock.Add(59 * time.Minute)

	_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "Str0ngPass"}, "")
	require.NoError(t, err)
}

func TestResetPassword_WrongOrEmptyToken_AllFailuresIdentical(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)
	// Used + expired + unknown + empty must be indistinguishable.
	_, _ = f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "Str0ngPass"}, "")

	var msgs []string
	for _, bad := range []string{tok, "wrong-token", "", "../../etc"} {
		_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: bad, NewPassword: "Str0ngPass"}, "")
		require.Error(t, err)
		var se *ServiceError
		require.True(t, errors.As(err, &se))
		assert.Equal(t, "INVALID_TOKEN", se.Code)
		assert.Equal(t, http.StatusBadRequest, se.HTTPStatus)
		msgs = append(msgs, se.Message)
	}
	for _, m := range msgs {
		assert.Equal(t, msgs[0], m)
	}
}

func TestResetPassword_WeakPassword_ReturnsValidationAndDoesNotConsumeToken(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)

	for _, weak := range []string{"short1A", "alllowercase1", "ALLUPPERCASE1", "NoDigitsHere", ""} {
		_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: weak}, "")
		require.Error(t, err, weak)
		assert.Equal(t, "VALIDATION_ERROR", serviceErrCode(t, err), weak)
		var se *ServiceError
		require.True(t, errors.As(err, &se))
		assert.Equal(t, http.StatusUnprocessableEntity, se.HTTPStatus, weak)
	}
	assert.Nil(t, f.repo.resets[0].usedAt, "token must survive policy failures")

	_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "Str0ngPass"}, "")
	require.NoError(t, err)
}

func TestResetPassword_InactiveUser_ReturnsInvalidToken(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)
	f.repo.byID["u1"].Status = "inactive"

	_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "Str0ngPass"}, "")

	require.Error(t, err)
	assert.Equal(t, "INVALID_TOKEN", serviceErrCode(t, err))
}

func TestResetPassword_RepoFailure_ReturnsWrappedError(t *testing.T) {
	f := newRecoveryFixture(t)
	f.repo.completeErr = errors.New("db down")

	_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: "x", NewPassword: "Str0ngPass"}, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth.service.ResetPassword")
	var se *ServiceError
	assert.False(t, errors.As(err, &se))
}

func TestResetPassword_LoginWorksAfterLockedAccountReset(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)
	_, err := f.svc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: tok, NewPassword: "Str0ngPass"}, "")
	require.NoError(t, err)

	// A locked-until in the future would make Login return 423; the reset must have cleared it.
	resp, _, err := f.svc.Login(context.Background(), &LoginRequest{Email: "alice@example.com", Password: "Str0ngPass"}, "")
	require.NoError(t, err)
	assert.False(t, resp.User.ForcePasswordChange)
}

func TestTokenNeverLeaksIntoStoredRows(t *testing.T) {
	f := newRecoveryFixture(t)
	tok := issueToken(t, f)
	for _, r := range f.repo.resets {
		assert.False(t, strings.Contains(r.hash, tok))
	}
}
