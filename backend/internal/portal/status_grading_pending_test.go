package portal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// FR-BB322 AC-2: a session waiting for grading shows the grading_pending status, counts as a used attempt,
// and never counts as passed.
func TestComputeStatus_GradingPending(t *testing.T) {
	t0 := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	past := t0.Add(-time.Hour)
	future := t0.Add(time.Hour)
	withNow(t, t0)

	cases := []struct {
		name        string
		maxAttempts int
		deadline    *time.Time
		sessions    []sessionRow
		want        UserStatus
		wantUsed    int
		wantOpen    bool
	}{
		{
			name:        "pending session with attempts left",
			maxAttempts: 2,
			sessions:    []sessionRow{{SessionID: "p", Status: "grading_pending"}},
			want:        UserStatusGradingPending,
			wantUsed:    1,
		},
		{
			name:        "pending session uses the only attempt",
			maxAttempts: 1,
			sessions:    []sessionRow{{SessionID: "p", Status: "grading_pending"}},
			want:        UserStatusGradingPending,
			wantUsed:    1,
		},
		{
			name:        "a passed session beside a pending one is passed",
			maxAttempts: 3,
			sessions: []sessionRow{
				{SessionID: "a", Status: "submitted", Passed: true},
				{SessionID: "p", Status: "grading_pending"},
			},
			want:     UserStatusPassed,
			wantUsed: 2,
		},
		{
			name:        "a pending session whose stored passed is true is not passed",
			maxAttempts: 2,
			sessions:    []sessionRow{{SessionID: "p", Status: "grading_pending", Passed: true}},
			want:        UserStatusGradingPending,
			wantUsed:    1,
		},
		{
			name:        "an expired deadline does not hide a pending session",
			maxAttempts: 2,
			deadline:    &past,
			sessions:    []sessionRow{{SessionID: "p", Status: "grading_pending"}},
			want:        UserStatusGradingPending,
			wantUsed:    1,
		},
		{
			name:        "an open session beats a pending one",
			maxAttempts: 3,
			sessions: []sessionRow{
				{SessionID: "p", Status: "grading_pending"},
				{SessionID: "o", Status: "in_progress", ExpiresAt: &future},
			},
			want:     UserStatusInProgress,
			wantUsed: 1,
			wantOpen: true,
		},
		{
			name:        "a pending session beats failed when the attempts are used up",
			maxAttempts: 2,
			sessions: []sessionRow{
				{SessionID: "a", Status: "submitted", Passed: false},
				{SessionID: "p", Status: "grading_pending"},
			},
			want:     UserStatusGradingPending,
			wantUsed: 2,
		},
		{
			name:        "finished failed attempts without a pending one stay failed",
			maxAttempts: 1,
			sessions:    []sessionRow{{SessionID: "a", Status: "submitted", Passed: false}},
			want:        UserStatusFailed,
			wantUsed:    1,
		},
		{
			name:        "an expired deadline without a pending session stays expired",
			maxAttempts: 2,
			deadline:    &past,
			want:        UserStatusExpired,
			wantUsed:    0,
		},
		{
			name:        "no sessions is not started",
			maxAttempts: 1,
			want:        UserStatusNotStarted,
			wantUsed:    0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, openID, used := computeStatus(tc.maxAttempts, tc.deadline, tc.sessions)
			assert.Equal(t, tc.want, status)
			assert.Equal(t, tc.wantUsed, used)
			assert.Equal(t, tc.wantOpen, openID != nil)
		})
	}
}

// AC-3: the pending session reported is the newest one waiting for grading, with its submit time.
func TestComputePending_NewestWaitingSession(t *testing.T) {
	older := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC)
	sessions := []sessionRow{
		{SessionID: "old", Status: "grading_pending", SubmittedAt: &older},
		{SessionID: "done", Status: "submitted"},
		{SessionID: "new", Status: "grading_pending", SubmittedAt: &newer},
	}

	id, submittedAt := computePending(sessions)
	if assert.NotNil(t, id) {
		assert.Equal(t, "new", *id)
	}
	if assert.NotNil(t, submittedAt) {
		assert.Equal(t, newer, *submittedAt)
	}
}

func TestComputePending_NoneWaiting(t *testing.T) {
	id, submittedAt := computePending([]sessionRow{{SessionID: "a", Status: "submitted"}})
	assert.Nil(t, id)
	assert.Nil(t, submittedAt)
}

func withNow(t *testing.T, at time.Time) {
	t.Helper()
	prev := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = prev })
}
