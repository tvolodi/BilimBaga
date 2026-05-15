package certificates

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock Repository ───────────────────────────────────────────────────────────

type mockRepo struct {
	getSessionFn        func(ctx context.Context, sessionID string) (*certSessionRow, error)
	getBySessionIDFn    func(ctx context.Context, sessionID string) (*Certificate, error)
	createFn            func(ctx context.Context, input certInsert) (*Certificate, error)
	getByVerificationFn func(ctx context.Context, code string) (*Certificate, error)
}

func (m *mockRepo) GetSessionForCertificate(ctx context.Context, sessionID string) (*certSessionRow, error) {
	if m.getSessionFn != nil {
		return m.getSessionFn(ctx, sessionID)
	}
	return defaultSessionRow(), nil
}

func (m *mockRepo) GetBySessionID(ctx context.Context, sessionID string) (*Certificate, error) {
	if m.getBySessionIDFn != nil {
		return m.getBySessionIDFn(ctx, sessionID)
	}
	return nil, ErrNotFound
}

func (m *mockRepo) Create(ctx context.Context, input certInsert) (*Certificate, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	return makeCert(input.SessionID), nil
}

func (m *mockRepo) GetByVerificationCode(ctx context.Context, code string) (*Certificate, error) {
	if m.getByVerificationFn != nil {
		return m.getByVerificationFn(ctx, code)
	}
	return nil, ErrNotFound
}

// ── Mock TenantConfigProvider ─────────────────────────────────────────────────

type mockTenant struct {
	cfg map[string]json.RawMessage
}

func (m *mockTenant) GetAllConfig() map[string]json.RawMessage {
	if m.cfg != nil {
		return m.cfg
	}
	return map[string]json.RawMessage{
		"app_name":      json.RawMessage(`"TestCorp"`),
		"primary_color": json.RawMessage(`"#0ea5e9"`),
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

const (
	testSessionID = "session-uuid-1"
	testUserID    = "user-uuid-1"
	testCertID    = "cert-uuid-1"
	testCode      = "verify-code-1"
)

func defaultSessionRow() *certSessionRow {
	score := 85.0
	return &certSessionRow{
		SessionID:          testSessionID,
		UserID:             testUserID,
		Status:             "submitted",
		Passed:             true,
		ScorePct:           &score,
		CertificateEnabled: true,
		ExamTitle:          "Fire Safety",
		EmployeeName:       "Alice Smith",
	}
}

func makeCert(sessionID string) *Certificate {
	snap, _ := json.Marshal(TemplateSnapshot{CompanyName: "TestCorp"})
	return &Certificate{
		ID:               testCertID,
		SessionID:        sessionID,
		VerificationCode: testCode,
		IssuedAt:         time.Now().UTC(),
		EmployeeName:     "Alice Smith",
		ExamTitle:        "Fire Safety",
		ScorePct:         85.0,
		TemplateSnapshot: snap,
	}
}

func newService(repo Repository) Service {
	return NewService(repo, &mockTenant{})
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// AC-1: portal endpoint must enforce session ownership.
func TestGetOrCreate_ErrNotOwner_WhenCallerDiffersFromOwner(t *testing.T) {
	svc := newService(&mockRepo{})
	_, _, err := svc.GetOrCreate(context.Background(), testSessionID, "other-user", false)
	assert.ErrorIs(t, err, ErrNotOwner)
}

// AC-1: admin bypass skips ownership check.
func TestGetOrCreate_AdminBypassesOwnerCheck(t *testing.T) {
	repo := &mockRepo{
		getBySessionIDFn: func(_ context.Context, _ string) (*Certificate, error) { return nil, ErrNotFound },
		createFn:         func(_ context.Context, input certInsert) (*Certificate, error) { return makeCert(input.SessionID), nil },
	}
	svc := newService(repo)
	cert, _, err := svc.GetOrCreate(context.Background(), testSessionID, "totally-different-user", true)
	require.NoError(t, err)
	assert.Equal(t, testCertID, cert.ID)
}

// AC-4: must return ErrNotSubmitted when status is not "submitted".
func TestGetOrCreate_ErrNotSubmitted_WhenStatusIsInProgress(t *testing.T) {
	repo := &mockRepo{
		getSessionFn: func(_ context.Context, _ string) (*certSessionRow, error) {
			row := defaultSessionRow()
			row.Status = "in_progress"
			return row, nil
		},
	}
	svc := newService(repo)
	_, _, err := svc.GetOrCreate(context.Background(), testSessionID, testUserID, false)
	assert.ErrorIs(t, err, ErrNotSubmitted)
}

// AC-4: grading_pending is also not valid.
func TestGetOrCreate_ErrNotSubmitted_WhenStatusIsGradingPending(t *testing.T) {
	repo := &mockRepo{
		getSessionFn: func(_ context.Context, _ string) (*certSessionRow, error) {
			row := defaultSessionRow()
			row.Status = "grading_pending"
			return row, nil
		},
	}
	svc := newService(repo)
	_, _, err := svc.GetOrCreate(context.Background(), testSessionID, testUserID, false)
	assert.ErrorIs(t, err, ErrNotSubmitted)
}

// AC-2: must return ErrNotCertifiable when certificate_enabled = false.
func TestGetOrCreate_ErrNotCertifiable_WhenCertDisabled(t *testing.T) {
	repo := &mockRepo{
		getSessionFn: func(_ context.Context, _ string) (*certSessionRow, error) {
			row := defaultSessionRow()
			row.CertificateEnabled = false
			return row, nil
		},
	}
	svc := newService(repo)
	_, _, err := svc.GetOrCreate(context.Background(), testSessionID, testUserID, false)
	assert.ErrorIs(t, err, ErrNotCertifiable)
}

// AC-3: must return ErrNotPassed when passed = false.
func TestGetOrCreate_ErrNotPassed_WhenSessionFailed(t *testing.T) {
	repo := &mockRepo{
		getSessionFn: func(_ context.Context, _ string) (*certSessionRow, error) {
			row := defaultSessionRow()
			row.Passed = false
			return row, nil
		},
	}
	svc := newService(repo)
	_, _, err := svc.GetOrCreate(context.Background(), testSessionID, testUserID, false)
	assert.ErrorIs(t, err, ErrNotPassed)
}

// AC-5 / AC-10: second call returns the existing cert without calling Create.
func TestGetOrCreate_Idempotent_ReturnsExistingCert(t *testing.T) {
	existing := makeCert(testSessionID)
	createCalled := false
	repo := &mockRepo{
		getBySessionIDFn: func(_ context.Context, _ string) (*Certificate, error) {
			return existing, nil
		},
		createFn: func(_ context.Context, _ certInsert) (*Certificate, error) {
			createCalled = true
			return nil, errors.New("should not be called")
		},
	}
	svc := newService(repo)
	cert, _, err := svc.GetOrCreate(context.Background(), testSessionID, testUserID, false)
	require.NoError(t, err)
	assert.Equal(t, existing.ID, cert.ID)
	assert.False(t, createCalled, "Create should not be called when cert already exists")
}

// AC-5: first call creates and returns new cert.
func TestGetOrCreate_CreatesNewCert_OnFirstCall(t *testing.T) {
	repo := &mockRepo{
		getBySessionIDFn: func(_ context.Context, _ string) (*Certificate, error) {
			return nil, ErrNotFound
		},
		createFn: func(_ context.Context, input certInsert) (*Certificate, error) {
			return makeCert(input.SessionID), nil
		},
	}
	svc := newService(repo)
	cert, snap, err := svc.GetOrCreate(context.Background(), testSessionID, testUserID, false)
	require.NoError(t, err)
	assert.Equal(t, testCertID, cert.ID)
	assert.Equal(t, "TestCorp", snap.CompanyName)
}

// AC-9: template snapshot captures tenant config at issuance.
func TestGetOrCreate_SnapshotCapturesTenantConfig(t *testing.T) {
	tenantCfg := &mockTenant{cfg: map[string]json.RawMessage{
		"company_name":   json.RawMessage(`"AcmeCorp"`),
		"primary_color":  json.RawMessage(`"#ff0000"`),
		"signatory_name": json.RawMessage(`"Jane Doe"`),
	}}
	var capturedInsert certInsert
	repo := &mockRepo{
		getBySessionIDFn: func(_ context.Context, _ string) (*Certificate, error) {
			return nil, ErrNotFound
		},
		createFn: func(_ context.Context, input certInsert) (*Certificate, error) {
			capturedInsert = input
			return makeCert(input.SessionID), nil
		},
	}
	svc := NewService(repo, tenantCfg)
	_, _, err := svc.GetOrCreate(context.Background(), testSessionID, testUserID, false)
	require.NoError(t, err)
	assert.Equal(t, "AcmeCorp", capturedInsert.TemplateSnapshot.CompanyName)
	assert.Equal(t, "#ff0000", capturedInsert.TemplateSnapshot.PrimaryColor)
	assert.Equal(t, "Jane Doe", capturedInsert.TemplateSnapshot.SignatoryName)
}

// AC-7: GetByVerificationCode returns valid=true for known code.
func TestGetByVerificationCode_ReturnsValidTrue_ForKnownCode(t *testing.T) {
	cert := makeCert(testSessionID)
	repo := &mockRepo{
		getByVerificationFn: func(_ context.Context, code string) (*Certificate, error) {
			if code == testCode {
				return cert, nil
			}
			return nil, ErrNotFound
		},
	}
	svc := newService(repo)
	resp, err := svc.GetByVerificationCode(context.Background(), testCode)
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, cert.EmployeeName, resp.EmployeeName)
	assert.Equal(t, cert.ExamTitle, resp.ExamTitle)
	require.NotNil(t, resp.ScorePct)
	assert.Equal(t, cert.ScorePct, *resp.ScorePct)
	require.NotNil(t, resp.IssuedAt)
}

// AC-7: GetByVerificationCode returns valid=false for unknown code — no error.
func TestGetByVerificationCode_ReturnsValidFalse_ForUnknownCode(t *testing.T) {
	svc := newService(&mockRepo{})
	resp, err := svc.GetByVerificationCode(context.Background(), "does-not-exist")
	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Empty(t, resp.EmployeeName)
	assert.Nil(t, resp.ScorePct)
	assert.Nil(t, resp.IssuedAt)
}

// Session not found propagates as ErrNotFound.
func TestGetOrCreate_ErrNotFound_WhenSessionMissing(t *testing.T) {
	repo := &mockRepo{
		getSessionFn: func(_ context.Context, _ string) (*certSessionRow, error) {
			return nil, ErrNotFound
		},
	}
	svc := newService(repo)
	_, _, err := svc.GetOrCreate(context.Background(), "missing-session", testUserID, false)
	assert.ErrorIs(t, err, ErrNotFound)
}
