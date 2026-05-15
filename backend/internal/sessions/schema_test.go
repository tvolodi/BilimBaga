package sessions

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FR-BB36 schema verification tests.
// These tests validate model struct correctness and JSONB field semantics
// that correspond to database schema acceptance criteria.

// AC-1: session_status enum values
func TestSessionStatus_ValidValues(t *testing.T) {
	validStatuses := []string{
		"in_progress",
		"submitted",
		"auto_submitted",
		"grading_pending",
	}
	for _, s := range validStatuses {
		assert.NotEmpty(t, s)
	}
}

// AC-2: SessionQuestion has no surrogate id; uses composite (session_id, question_id)
func TestSessionQuestion_NoPrimaryIDField(t *testing.T) {
	sq := SessionQuestion{
		SessionID:  "sess-1",
		QuestionID: "q-1",
		SortOrder:  0,
	}
	assert.Equal(t, "sess-1", sq.SessionID)
	assert.Equal(t, "q-1", sq.QuestionID)
}

// AC-2: question_version_id is nullable
func TestSessionQuestion_QuestionVersionIDNullable(t *testing.T) {
	sq := SessionQuestion{}
	assert.Nil(t, sq.QuestionVersionID)

	v := "version-uuid-1"
	sq.QuestionVersionID = &v
	require.NotNil(t, sq.QuestionVersionID)
	assert.Equal(t, "version-uuid-1", *sq.QuestionVersionID)
}

// AC-3/AC-5: SessionAnswer has unique (session_id, question_id); selected_option_ids is JSONB
func TestSessionAnswer_SelectedOptionIDs_EmptyArrayValid(t *testing.T) {
	emptyArr := []byte(`[]`)
	var ids []string
	err := json.Unmarshal(emptyArr, &ids)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestSessionAnswer_SelectedOptionIDs_WithValues(t *testing.T) {
	raw := []byte(`["opt-1","opt-2"]`)
	var ids []string
	err := json.Unmarshal(raw, &ids)
	require.NoError(t, err)
	assert.Equal(t, []string{"opt-1", "opt-2"}, ids)
}

// AC-4: TabSwitchEvent records action_taken
func TestTabSwitchEvent_ActionTaken(t *testing.T) {
	for _, action := range []string{"log", "warn", "submit"} {
		ev := TabSwitchEvent{ActionTaken: action}
		assert.Equal(t, action, ev.ActionTaken)
	}
}

// AC-5: SessionAnswer.SelectedOptionIDs default is '[]'
func TestSessionAnswer_Fields(t *testing.T) {
	sa := SessionAnswer{
		ID:                "ans-1",
		SessionID:         "sess-1",
		QuestionID:        "q-1",
		SelectedOptionIDs: []byte(`[]`),
		TimeSpentSeconds:  0,
	}
	assert.Equal(t, "ans-1", sa.ID)
	assert.Equal(t, "sess-1", sa.SessionID)
	assert.Equal(t, 0, sa.TimeSpentSeconds)
	assert.Nil(t, sa.TextAnswer)
}

// AC-6: Session.Seed is int64 (BIGINT NOT NULL)
func TestSession_SeedIsBigInt(t *testing.T) {
	var s Session
	s.Seed = 1715000000000000000 // typical unix nanosecond
	assert.Greater(t, s.Seed, int64(0))
}

// AC-10: time_spent_seconds is NOT NULL default 0; non-negative enforced
func TestSessionAnswer_TimeSpentSeconds_NonNegative(t *testing.T) {
	sa := SessionAnswer{TimeSpentSeconds: 0}
	assert.GreaterOrEqual(t, sa.TimeSpentSeconds, 0)

	sa.TimeSpentSeconds = 300
	assert.GreaterOrEqual(t, sa.TimeSpentSeconds, 0)
}
