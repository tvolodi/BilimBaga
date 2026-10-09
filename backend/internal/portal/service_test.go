package portal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock repository ──────────────────────────────────────────────────────────

type mockRepo struct {
	listAssignedExamsFn func(ctx context.Context, userID, deptID string) ([]*portalExamRow, error)
	getAssignedExamFn   func(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error)
	listUserSessionsFn  func(ctx context.Context, examID, userID string) ([]sessionRow, error)
}

func (m *mockRepo) ListAssignedExams(ctx context.Context, userID, deptID string) ([]*portalExamRow, error) {
	if m.listAssignedExamsFn != nil {
		return m.listAssignedExamsFn(ctx, userID, deptID)
	}
	return []*portalExamRow{}, nil
}

func (m *mockRepo) GetAssignedExam(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error) {
	if m.getAssignedExamFn != nil {
		return m.getAssignedExamFn(ctx, examID, userID, deptID)
	}
	return nil, ErrNotAssigned
}

func (m *mockRepo) ListUserSessions(ctx context.Context, examID, userID string) ([]sessionRow, error) {
	if m.listUserSessionsFn != nil {
		return m.listUserSessionsFn(ctx, examID, userID)
	}
	return []sessionRow{}, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func baseExamRow(id string) *portalExamRow {
	return &portalExamRow{
		ID:               id,
		Title:            "Test Exam",
		TimeLimitMinutes: 60,
		PassingScorePct:  70.0,
		MaxAttempts:      3,
		ShowAnswers:      "never",
	}
}

// fixNow pins the service's now() to a deterministic time for tests.
func fixNow(t *testing.T, fixed time.Time) {
	t.Helper()
	orig := now
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = orig })
}

// ── computeStatus unit tests ─────────────────────────────────────────────────

func TestComputeStatus_NotStarted(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	status, openID, used := computeStatus(3, nil, nil)
	assert.Equal(t, UserStatusNotStarted, status)
	assert.Nil(t, openID)
	assert.Equal(t, 0, used)
}

func TestComputeStatus_Passed(t *testing.T) {
	// AC-3: passed takes priority over everything else.
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	deadline := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // already passed
	sessions := []sessionRow{
		{SessionID: "s1", Status: "submitted", Passed: true},
	}
	status, _, used := computeStatus(1, &deadline, sessions)
	assert.Equal(t, UserStatusPassed, status)
	assert.Equal(t, 1, used)
}

func TestComputeStatus_InProgress(t *testing.T) {
	// AC-4: open session (not expired).
	fixedTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	fixNow(t, fixedTime)
	expires := fixedTime.Add(30 * time.Minute)
	sessions := []sessionRow{
		{SessionID: "s1", Status: "in_progress", ExpiresAt: &expires},
	}
	status, openID, used := computeStatus(3, nil, sessions)
	assert.Equal(t, UserStatusInProgress, status)
	require.NotNil(t, openID)
	assert.Equal(t, "s1", *openID)
	assert.Equal(t, 0, used)
}

func TestComputeStatus_InProgress_ExpiredSession_IsNotOpen(t *testing.T) {
	fixedTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	fixNow(t, fixedTime)
	expires := fixedTime.Add(-1 * time.Minute) // expired
	sessions := []sessionRow{
		{SessionID: "s1", Status: "in_progress", ExpiresAt: &expires},
	}
	status, openID, _ := computeStatus(3, nil, sessions)
	assert.Equal(t, UserStatusNotStarted, status)
	assert.Nil(t, openID)
}

func TestComputeStatus_Expired(t *testing.T) {
	// AC-2: expired takes priority over failed.
	fixedTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	fixNow(t, fixedTime)
	deadline := fixedTime.Add(-1 * time.Hour) // passed
	// All attempts used but deadline also passed → expired wins.
	sessions := []sessionRow{
		{SessionID: "s1", Status: "submitted", Passed: false},
		{SessionID: "s2", Status: "submitted", Passed: false},
		{SessionID: "s3", Status: "submitted", Passed: false},
	}
	status, _, used := computeStatus(3, &deadline, sessions)
	assert.Equal(t, UserStatusExpired, status)
	assert.Equal(t, 3, used)
}

func TestComputeStatus_Failed(t *testing.T) {
	// AC-5: all attempts exhausted, no pass, deadline in future.
	fixedTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	fixNow(t, fixedTime)
	deadline := fixedTime.Add(24 * time.Hour)
	sessions := []sessionRow{
		{SessionID: "s1", Status: "submitted", Passed: false},
		{SessionID: "s2", Status: "auto_submitted", Passed: false},
	}
	status, _, used := computeStatus(2, &deadline, sessions)
	assert.Equal(t, UserStatusFailed, status)
	assert.Equal(t, 2, used)
}

func TestComputeStatus_PassedPriority_OverExpired(t *testing.T) {
	// AC-3: passed ignores expired deadline.
	fixedTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	fixNow(t, fixedTime)
	deadline := fixedTime.Add(-1 * time.Hour)
	sessions := []sessionRow{
		{SessionID: "s1", Status: "submitted", Passed: true},
	}
	status, _, _ := computeStatus(2, &deadline, sessions)
	assert.Equal(t, UserStatusPassed, status)
}

func TestComputeStatus_AttemptsUsed_OnlyFinished(t *testing.T) {
	// AC-8: in_progress sessions do not count as attempts_used.
	fixedTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	fixNow(t, fixedTime)
	expires := fixedTime.Add(30 * time.Minute)
	sessions := []sessionRow{
		{SessionID: "s1", Status: "submitted", Passed: false},
		{SessionID: "s2", Status: "in_progress", ExpiresAt: &expires},
	}
	_, _, used := computeStatus(3, nil, sessions)
	assert.Equal(t, 1, used)
}

// ── ListMyExams service tests ────────────────────────────────────────────────

func TestListMyExams_Empty(t *testing.T) {
	svc := NewService(&mockRepo{})
	items, err := svc.ListMyExams(context.Background(), "user1", "dept1")
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestListMyExams_ReturnsItems(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		listAssignedExamsFn: func(ctx context.Context, userID, deptID string) ([]*portalExamRow, error) {
			return []*portalExamRow{baseExamRow("exam1"), baseExamRow("exam2")}, nil
		},
	}
	svc := NewService(repo)
	items, err := svc.ListMyExams(context.Background(), "user1", "dept1")
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, UserStatusNotStarted, items[0].UserStatus)
	assert.Nil(t, items[0].OpenSessionID)
	assert.Equal(t, 0, items[0].AttemptsUsed)
}

// ISS-132: the availability window is exposed so the UI can disable Start while the exam is closed.
func TestListMyExams_ExposesAvailabilityWindow(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		listAssignedExamsFn: func(ctx context.Context, userID, deptID string) ([]*portalExamRow, error) {
			row := baseExamRow("exam1")
			row.AvailableFrom = &from
			row.AvailableUntil = &until
			return []*portalExamRow{row}, nil
		},
	}
	items, err := NewService(repo).ListMyExams(context.Background(), "user1", "dept1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].AvailableFrom)
	assert.True(t, items[0].AvailableFrom.Equal(from))
	require.NotNil(t, items[0].AvailableUntil)
	assert.True(t, items[0].AvailableUntil.Equal(until))
}

func TestListMyExams_RepoError(t *testing.T) {
	repo := &mockRepo{
		listAssignedExamsFn: func(ctx context.Context, userID, deptID string) ([]*portalExamRow, error) {
			return nil, errors.New("db down")
		},
	}
	svc := NewService(repo)
	_, err := svc.ListMyExams(context.Background(), "user1", "dept1")
	require.Error(t, err)
}

func TestListMyExams_SessionError(t *testing.T) {
	repo := &mockRepo{
		listAssignedExamsFn: func(ctx context.Context, userID, deptID string) ([]*portalExamRow, error) {
			return []*portalExamRow{baseExamRow("exam1")}, nil
		},
		listUserSessionsFn: func(ctx context.Context, examID, userID string) ([]sessionRow, error) {
			return nil, errors.New("session db down")
		},
	}
	svc := NewService(repo)
	_, err := svc.ListMyExams(context.Background(), "user1", "dept1")
	require.Error(t, err)
}

func TestListMyExams_WithOpenSession(t *testing.T) {
	fixedTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	fixNow(t, fixedTime)
	expires := fixedTime.Add(30 * time.Minute)
	repo := &mockRepo{
		listAssignedExamsFn: func(ctx context.Context, userID, deptID string) ([]*portalExamRow, error) {
			return []*portalExamRow{baseExamRow("exam1")}, nil
		},
		listUserSessionsFn: func(ctx context.Context, examID, userID string) ([]sessionRow, error) {
			return []sessionRow{
				{SessionID: "sess42", Status: "in_progress", ExpiresAt: &expires},
			}, nil
		},
	}
	svc := NewService(repo)
	items, err := svc.ListMyExams(context.Background(), "user1", "dept1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, UserStatusInProgress, items[0].UserStatus)
	require.NotNil(t, items[0].OpenSessionID)
	assert.Equal(t, "sess42", *items[0].OpenSessionID)
}

// ── GetMyExam service tests ──────────────────────────────────────────────────

func TestGetMyExam_NotAssigned(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.GetMyExam(context.Background(), "exam1", "user1", "dept1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAssigned)
}

func TestGetMyExam_RepoError(t *testing.T) {
	repo := &mockRepo{
		getAssignedExamFn: func(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error) {
			return nil, errors.New("db down")
		},
	}
	svc := NewService(repo)
	_, err := svc.GetMyExam(context.Background(), "exam1", "user1", "dept1")
	require.Error(t, err)
}

func TestGetMyExam_NoSessions(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getAssignedExamFn: func(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error) {
			return baseExamRow("exam1"), nil
		},
	}
	svc := NewService(repo)
	detail, err := svc.GetMyExam(context.Background(), "exam1", "user1", "dept1")
	require.NoError(t, err)
	assert.Equal(t, "exam1", detail.ID)
	assert.Equal(t, UserStatusNotStarted, detail.UserStatus)
	assert.Empty(t, detail.AttemptHistory)
	assert.Equal(t, 0, detail.AttemptsUsed)
}

func TestGetMyExam_AttemptHistory_OnlyFinished(t *testing.T) {
	fixedTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	fixNow(t, fixedTime)
	expires := fixedTime.Add(30 * time.Minute)
	submittedAt := fixedTime.Add(-1 * time.Hour)
	score := 65.0
	sessions := []sessionRow{
		{
			SessionID:   "s1",
			Status:      "submitted",
			Passed:      false,
			SubmittedAt: &submittedAt,
			ScorePct:    &score,
			StartedAt:   fixedTime.Add(-2 * time.Hour),
		},
		{
			SessionID: "s2",
			Status:    "in_progress",
			ExpiresAt: &expires,
			StartedAt: fixedTime.Add(-10 * time.Minute),
		},
	}
	repo := &mockRepo{
		getAssignedExamFn: func(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error) {
			return baseExamRow("exam1"), nil
		},
		listUserSessionsFn: func(ctx context.Context, examID, userID string) ([]sessionRow, error) {
			return sessions, nil
		},
	}
	svc := NewService(repo)
	detail, err := svc.GetMyExam(context.Background(), "exam1", "user1", "dept1")
	require.NoError(t, err)
	assert.Len(t, detail.AttemptHistory, 1) // in_progress not included
	assert.Equal(t, "s1", detail.AttemptHistory[0].SessionID)
	assert.Equal(t, 1, detail.AttemptsUsed)
	assert.Equal(t, UserStatusInProgress, detail.UserStatus)
}

func TestGetMyExam_Passed_IncludesHistory(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	submittedAt := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	score := 90.0
	sessions := []sessionRow{
		{
			SessionID:   "s1",
			Status:      "submitted",
			Passed:      true,
			SubmittedAt: &submittedAt,
			ScorePct:    &score,
			StartedAt:   time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
		},
	}
	repo := &mockRepo{
		getAssignedExamFn: func(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error) {
			return baseExamRow("exam1"), nil
		},
		listUserSessionsFn: func(ctx context.Context, examID, userID string) ([]sessionRow, error) {
			return sessions, nil
		},
	}
	svc := NewService(repo)
	detail, err := svc.GetMyExam(context.Background(), "exam1", "user1", "dept1")
	require.NoError(t, err)
	assert.Equal(t, UserStatusPassed, detail.UserStatus)
	require.Len(t, detail.AttemptHistory, 1)
	assert.True(t, detail.AttemptHistory[0].Passed)
	assert.Equal(t, &score, detail.AttemptHistory[0].ScorePct)
}

func TestGetMyExam_Deadline_AC9(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	deadline := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	row := baseExamRow("exam1")
	row.Deadline = &deadline
	repo := &mockRepo{
		getAssignedExamFn: func(ctx context.Context, examID, userID, deptID string) (*portalExamRow, error) {
			return row, nil
		},
	}
	svc := NewService(repo)
	detail, err := svc.GetMyExam(context.Background(), "exam1", "user1", "dept1")
	require.NoError(t, err)
	require.NotNil(t, detail.Deadline)
	assert.Equal(t, deadline, *detail.Deadline)
}
