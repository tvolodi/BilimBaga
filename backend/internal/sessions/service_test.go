package sessions

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
	isAssignedFn              func(ctx context.Context, examID, userID, deptID string) (bool, error)
	getExamConfigFn           func(ctx context.Context, examID string) (*examConfig, error)
	countFinishedSessionsFn   func(ctx context.Context, examID, userID string) (int, error)
	hasOpenSessionFn          func(ctx context.Context, examID, userID string) (bool, error)
	getRulesFn                func(ctx context.Context, examID string) ([]questionRule, error)
	getManualQuestionsFn      func(ctx context.Context, ruleID string) ([]poolQuestion, error)
	getEligibleQuestionsFn    func(ctx context.Context, rule questionRule) ([]poolQuestion, error)
	getOptionIDsFn            func(ctx context.Context, questionID string) ([]poolOption, error)
	getQuestionDetailsFn      func(ctx context.Context, ids []string) (map[string]questionDetail, error)
	getOptionTextsFn          func(ctx context.Context, optionIDs []string) (map[string]string, error)
	createSessionFn           func(ctx context.Context, input createSessionInput) (string, time.Time, time.Time, error)
	getSessionForUserFn       func(ctx context.Context, sessionID, userID string) (*sessionStateRow, error)
	getSessionQuestionsFn     func(ctx context.Context, sessionID string) ([]sessionQuestionRow, error)
	getSessionAnswersFn       func(ctx context.Context, sessionID string) (map[string]savedAnswerRow, error)
	getValidOptionIDsFn       func(ctx context.Context, questionID string) (map[string]struct{}, error)
	upsertAnswerFn            func(ctx context.Context, input upsertAnswerInput) (time.Time, error)
	insertTabSwitchEventFn    func(ctx context.Context, sessionID, eventType, actionTaken string) error
	countTabSwitchEventsFn    func(ctx context.Context, sessionID string) (int, error)
	autoSubmitSessionFn       func(ctx context.Context, sessionID string) (*autoSubmitResult, error)
	submitSessionFn           func(ctx context.Context, sessionID, userID, tenantID, actorIP string) (*SubmitSessionResponse, error)
}

func (m *mockRepo) IsAssigned(ctx context.Context, examID, userID, deptID string) (bool, error) {
	if m.isAssignedFn != nil {
		return m.isAssignedFn(ctx, examID, userID, deptID)
	}
	return true, nil
}
func (m *mockRepo) GetExamConfig(ctx context.Context, examID string) (*examConfig, error) {
	if m.getExamConfigFn != nil {
		return m.getExamConfigFn(ctx, examID)
	}
	return defaultConfig(), nil
}
func (m *mockRepo) CountFinishedSessions(ctx context.Context, examID, userID string) (int, error) {
	if m.countFinishedSessionsFn != nil {
		return m.countFinishedSessionsFn(ctx, examID, userID)
	}
	return 0, nil
}
func (m *mockRepo) HasOpenSession(ctx context.Context, examID, userID string) (bool, error) {
	if m.hasOpenSessionFn != nil {
		return m.hasOpenSessionFn(ctx, examID, userID)
	}
	return false, nil
}
func (m *mockRepo) GetRules(ctx context.Context, examID string) ([]questionRule, error) {
	if m.getRulesFn != nil {
		return m.getRulesFn(ctx, examID)
	}
	return []questionRule{}, nil
}
func (m *mockRepo) GetManualQuestions(ctx context.Context, ruleID string) ([]poolQuestion, error) {
	if m.getManualQuestionsFn != nil {
		return m.getManualQuestionsFn(ctx, ruleID)
	}
	return []poolQuestion{}, nil
}
func (m *mockRepo) GetEligibleQuestions(ctx context.Context, rule questionRule) ([]poolQuestion, error) {
	if m.getEligibleQuestionsFn != nil {
		return m.getEligibleQuestionsFn(ctx, rule)
	}
	return []poolQuestion{}, nil
}
func (m *mockRepo) GetOptionIDs(ctx context.Context, questionID string) ([]poolOption, error) {
	if m.getOptionIDsFn != nil {
		return m.getOptionIDsFn(ctx, questionID)
	}
	return []poolOption{}, nil
}
func (m *mockRepo) GetQuestionDetails(ctx context.Context, ids []string) (map[string]questionDetail, error) {
	if m.getQuestionDetailsFn != nil {
		return m.getQuestionDetailsFn(ctx, ids)
	}
	result := make(map[string]questionDetail, len(ids))
	for _, id := range ids {
		result[id] = questionDetail{Stem: "Q: " + id, Type: "single_choice"}
	}
	return result, nil
}
func (m *mockRepo) GetOptionTexts(ctx context.Context, optionIDs []string) (map[string]string, error) {
	if m.getOptionTextsFn != nil {
		return m.getOptionTextsFn(ctx, optionIDs)
	}
	result := make(map[string]string, len(optionIDs))
	for _, id := range optionIDs {
		result[id] = "Option " + id
	}
	return result, nil
}
func (m *mockRepo) CreateSession(ctx context.Context, input createSessionInput) (string, time.Time, time.Time, error) {
	if m.createSessionFn != nil {
		return m.createSessionFn(ctx, input)
	}
	now := time.Now().UTC()
	return "sess-1", now, now.Add(time.Duration(60) * time.Minute), nil
}
func (m *mockRepo) GetSessionForUser(ctx context.Context, sessionID, userID string) (*sessionStateRow, error) {
	if m.getSessionForUserFn != nil {
		return m.getSessionForUserFn(ctx, sessionID, userID)
	}
	now := time.Now().UTC()
	return &sessionStateRow{
		ID:        sessionID,
		ExamID:    "exam-1",
		UserID:    userID,
		Status:    "in_progress",
		StartedAt: now,
		ExpiresAt: now.Add(90 * time.Minute),
	}, nil
}
func (m *mockRepo) GetSessionQuestions(ctx context.Context, sessionID string) ([]sessionQuestionRow, error) {
	if m.getSessionQuestionsFn != nil {
		return m.getSessionQuestionsFn(ctx, sessionID)
	}
	return []sessionQuestionRow{
		{QuestionID: "q-1", SortOrder: 0, OptionsOrder: []byte(`["opt-a","opt-b"]`), QuestionType: "single_choice"},
	}, nil
}
func (m *mockRepo) GetSessionAnswers(ctx context.Context, sessionID string) (map[string]savedAnswerRow, error) {
	if m.getSessionAnswersFn != nil {
		return m.getSessionAnswersFn(ctx, sessionID)
	}
	return map[string]savedAnswerRow{}, nil
}
func (m *mockRepo) GetValidOptionIDs(ctx context.Context, questionID string) (map[string]struct{}, error) {
	if m.getValidOptionIDsFn != nil {
		return m.getValidOptionIDsFn(ctx, questionID)
	}
	return map[string]struct{}{"opt-a": {}, "opt-b": {}}, nil
}
func (m *mockRepo) UpsertAnswer(ctx context.Context, input upsertAnswerInput) (time.Time, error) {
	if m.upsertAnswerFn != nil {
		return m.upsertAnswerFn(ctx, input)
	}
	return time.Now().UTC(), nil
}
func (m *mockRepo) InsertTabSwitchEvent(ctx context.Context, sessionID, eventType, actionTaken string) error {
	if m.insertTabSwitchEventFn != nil {
		return m.insertTabSwitchEventFn(ctx, sessionID, eventType, actionTaken)
	}
	return nil
}
func (m *mockRepo) CountTabSwitchEvents(ctx context.Context, sessionID string) (int, error) {
	if m.countTabSwitchEventsFn != nil {
		return m.countTabSwitchEventsFn(ctx, sessionID)
	}
	return 1, nil
}
func (m *mockRepo) AutoSubmitSession(ctx context.Context, sessionID string) (*autoSubmitResult, error) {
	if m.autoSubmitSessionFn != nil {
		return m.autoSubmitSessionFn(ctx, sessionID)
	}
	sid := sessionID
	status := "auto_submitted"
	return &autoSubmitResult{SessionID: sid, Status: status}, nil
}
func (m *mockRepo) SubmitSession(ctx context.Context, sessionID, userID, tenantID, actorIP string) (*SubmitSessionResponse, error) {
	if m.submitSessionFn != nil {
		return m.submitSessionFn(ctx, sessionID, userID, tenantID, actorIP)
	}
	now := time.Now().UTC()
	score := 80.0
	passed := true
	return &SubmitSessionResponse{
		SessionID:   sessionID,
		Status:      "submitted",
		SubmittedAt: now,
		ScorePct:    &score,
		Passed:      &passed,
	}, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func defaultConfig() *examConfig {
	return &examConfig{
		ID:               "exam-1",
		Status:           "active",
		TimeLimitMinutes: 60,
		MaxAttempts:      3,
		OnTabSwitch:      "log",
	}
}

func fixNow(t *testing.T, fixed time.Time) {
	t.Helper()
	orig := nowFn
	nowFn = func() time.Time { return fixed }
	t.Cleanup(func() { nowFn = orig })
}

func twoManualQuestions() []poolQuestion {
	return []poolQuestion{
		{ID: "q1", Type: "single_choice"},
		{ID: "q2", Type: "single_choice"},
	}
}

func twoOptions(qID string) []poolOption {
	return []poolOption{
		{OptionID: qID + "-opt-a", SortOrder: 0},
		{OptionID: qID + "-opt-b", SortOrder: 1},
	}
}

// ── AC-1: not assigned ───────────────────────────────────────────────────────

func TestCreateSession_NotAssigned(t *testing.T) {
	repo := &mockRepo{
		isAssignedFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAssigned)
}

func TestCreateSession_IsAssignedRepoError(t *testing.T) {
	repo := &mockRepo{
		isAssignedFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, errors.New("db down")
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
}

// ── AC-2: exam not active ────────────────────────────────────────────────────

func TestCreateSession_ExamNotActive(t *testing.T) {
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.Status = "draft"
			return cfg, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExamNotActive)
}

func TestCreateSession_ExamArchived(t *testing.T) {
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.Status = "archived"
			return cfg, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExamNotActive)
}

// ── AC-3: outside availability window ────────────────────────────────────────

func TestCreateSession_BeforeAvailableFrom(t *testing.T) {
	current := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	fixNow(t, current)
	future := current.Add(1 * time.Hour)
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.AvailableFrom = &future
			return cfg, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExamOutsideWindow)
}

func TestCreateSession_AfterAvailableUntil(t *testing.T) {
	current := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	fixNow(t, current)
	past := current.Add(-1 * time.Hour)
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.AvailableUntil = &past
			return cfg, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExamOutsideWindow)
}

func TestCreateSession_WithinWindow(t *testing.T) {
	current := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	fixNow(t, current)
	from := current.Add(-1 * time.Hour)
	until := current.Add(1 * time.Hour)
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.AvailableFrom = &from
			cfg.AvailableUntil = &until
			return cfg, nil
		},
	}
	svc := NewService(repo)
	// No rules → session creates with zero questions.
	resp, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ── AC-4: attempts exhausted ─────────────────────────────────────────────────

func TestCreateSession_AttemptsExhausted(t *testing.T) {
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.MaxAttempts = 2
			return cfg, nil
		},
		countFinishedSessionsFn: func(_ context.Context, _, _ string) (int, error) {
			return 2, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAttemptsExhausted)
}

func TestCreateSession_AttemptsNotYetExhausted(t *testing.T) {
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.MaxAttempts = 3
			return cfg, nil
		},
		countFinishedSessionsFn: func(_ context.Context, _, _ string) (int, error) {
			return 2, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ── AC-5: concurrent session ─────────────────────────────────────────────────

func TestCreateSession_SessionAlreadyOpen(t *testing.T) {
	repo := &mockRepo{
		hasOpenSessionFn: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionAlreadyOpen)
}

// ── AC-6: manual rule question resolution ────────────────────────────────────

func TestCreateSession_ManualRule_ResolvedQuestions(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getRulesFn: func(_ context.Context, _ string) ([]questionRule, error) {
			return []questionRule{
				{ID: "rule-1", Mode: "manual", Count: 2, SortOrder: 0},
			}, nil
		},
		getManualQuestionsFn: func(_ context.Context, _ string) ([]poolQuestion, error) {
			return twoManualQuestions(), nil
		},
		getOptionIDsFn: func(_ context.Context, qID string) ([]poolOption, error) {
			return twoOptions(qID), nil
		},
		createSessionFn: func(_ context.Context, input createSessionInput) (string, time.Time, time.Time, error) {
			assert.Len(t, input.Questions, 2)
			assert.Equal(t, "q1", input.Questions[0].QuestionID)
			assert.Equal(t, "q2", input.Questions[1].QuestionID)
			now := time.Now().UTC()
			return "sess-1", now, now.Add(60 * time.Minute), nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.Len(t, resp.Questions, 2)
}

// ── AC-6: random rule — insufficient questions ────────────────────────────────

func TestCreateSession_RandomRule_InsufficientQuestions(t *testing.T) {
	repo := &mockRepo{
		getRulesFn: func(_ context.Context, _ string) ([]questionRule, error) {
			return []questionRule{
				{ID: "rule-1", Mode: "random", Count: 5, SortOrder: 0},
			}, nil
		},
		getEligibleQuestionsFn: func(_ context.Context, _ questionRule) ([]poolQuestion, error) {
			return []poolQuestion{{ID: "q1", Type: "single_choice"}}, nil // only 1, need 5
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInsufficientQuestions)
}

// ── AC-6: random rule — exactly enough questions ──────────────────────────────

func TestCreateSession_RandomRule_ExactCount(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getRulesFn: func(_ context.Context, _ string) ([]questionRule, error) {
			return []questionRule{
				{ID: "rule-1", Mode: "random", Count: 2, SortOrder: 0},
			}, nil
		},
		getEligibleQuestionsFn: func(_ context.Context, _ questionRule) ([]poolQuestion, error) {
			return twoManualQuestions(), nil
		},
		getOptionIDsFn: func(_ context.Context, qID string) ([]poolOption, error) {
			return twoOptions(qID), nil
		},
		createSessionFn: func(_ context.Context, input createSessionInput) (string, time.Time, time.Time, error) {
			assert.Len(t, input.Questions, 2)
			now := time.Now().UTC()
			return "sess-1", now, now.Add(60 * time.Minute), nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.Len(t, resp.Questions, 2)
}

// ── AC-8: expires_at and remaining_seconds ───────────────────────────────────

func TestCreateSession_RemainingSecondsComputed(t *testing.T) {
	current := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	fixNow(t, current)
	expiresAt := current.Add(90 * time.Minute)
	repo := &mockRepo{
		createSessionFn: func(_ context.Context, input createSessionInput) (string, time.Time, time.Time, error) {
			return "sess-1", current, expiresAt, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.Equal(t, expiresAt, resp.ExpiresAt)
	assert.InDelta(t, 5400.0, resp.RemainingSeconds, 1.0)
}

// ── AC-9: correct-answer flags not in response ───────────────────────────────

func TestCreateSession_NoCorrectAnswerFlag(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getRulesFn: func(_ context.Context, _ string) ([]questionRule, error) {
			return []questionRule{
				{ID: "rule-1", Mode: "manual", Count: 1, SortOrder: 0},
			}, nil
		},
		getManualQuestionsFn: func(_ context.Context, _ string) ([]poolQuestion, error) {
			return []poolQuestion{{ID: "q1", Type: "single_choice"}}, nil
		},
		getOptionIDsFn: func(_ context.Context, _ string) ([]poolOption, error) {
			return []poolOption{{OptionID: "opt-1", SortOrder: 0}}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	require.Len(t, resp.Questions, 1)
	// SessionOptionResponse has no is_correct field — verified by the type definition.
	assert.Len(t, resp.Questions[0].Options, 1)
	assert.Equal(t, "opt-1", resp.Questions[0].Options[0].ID)
}

// ── AC-9: short_text has empty options ───────────────────────────────────────

func TestCreateSession_ShortText_EmptyOptions(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getRulesFn: func(_ context.Context, _ string) ([]questionRule, error) {
			return []questionRule{
				{ID: "rule-1", Mode: "manual", Count: 1, SortOrder: 0},
			}, nil
		},
		getManualQuestionsFn: func(_ context.Context, _ string) ([]poolQuestion, error) {
			return []poolQuestion{{ID: "q1", Type: "short_text"}}, nil
		},
		getOptionIDsFn: func(_ context.Context, _ string) ([]poolOption, error) {
			return []poolOption{}, nil
		},
		getQuestionDetailsFn: func(_ context.Context, ids []string) (map[string]questionDetail, error) {
			return map[string]questionDetail{"q1": {Stem: "Describe X", Type: "short_text"}}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	require.Len(t, resp.Questions, 1)
	assert.Empty(t, resp.Questions[0].Options)
}

// ── AC-10: seed stored in createSessionInput ─────────────────────────────────

func TestCreateSession_SeedStoredInInput(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	var capturedSeed int64
	repo := &mockRepo{
		createSessionFn: func(_ context.Context, input createSessionInput) (string, time.Time, time.Time, error) {
			capturedSeed = input.Seed
			now := time.Now().UTC()
			return "sess-1", now, now.Add(60 * time.Minute), nil
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.NotZero(t, capturedSeed)
}

// ── Sort order assigned correctly ────────────────────────────────────────────

func TestCreateSession_SortOrderAssigned(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getRulesFn: func(_ context.Context, _ string) ([]questionRule, error) {
			return []questionRule{
				{ID: "rule-1", Mode: "manual", Count: 2, SortOrder: 0},
			}, nil
		},
		getManualQuestionsFn: func(_ context.Context, _ string) ([]poolQuestion, error) {
			return twoManualQuestions(), nil
		},
		getOptionIDsFn: func(_ context.Context, qID string) ([]poolOption, error) {
			return twoOptions(qID), nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Questions[0].SortOrder)
	assert.Equal(t, 1, resp.Questions[1].SortOrder)
}

// ── Repo errors propagate ────────────────────────────────────────────────────

func TestCreateSession_GetRulesError(t *testing.T) {
	repo := &mockRepo{
		getRulesFn: func(_ context.Context, _ string) ([]questionRule, error) {
			return nil, errors.New("db down")
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
}

func TestCreateSession_CreateSessionError(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		createSessionFn: func(_ context.Context, _ createSessionInput) (string, time.Time, time.Time, error) {
			return "", time.Time{}, time.Time{}, errors.New("tx failed")
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	require.Error(t, err)
}

// ── FR-BB37: SaveAnswer ───────────────────────────────────────────────────────

func activeSession(userID string) *sessionStateRow {
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	return &sessionStateRow{
		ID:        "sess-1",
		ExamID:    "exam-1",
		UserID:    userID,
		Status:    "in_progress",
		StartedAt: now,
		ExpiresAt: now.Add(90 * time.Minute),
	}
}

func TestSaveAnswer_HappyPath(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	savedAt := time.Date(2026, 5, 1, 10, 1, 0, 0, time.UTC)
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		upsertAnswerFn: func(_ context.Context, _ upsertAnswerInput) (time.Time, error) {
			return savedAt, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{
		SelectedOptionIDs: []string{"opt-a"},
		TimeSpentSeconds:  30,
	})
	require.NoError(t, err)
	assert.Equal(t, "q-1", resp.QuestionID)
	assert.Equal(t, savedAt, resp.SavedAt)
	assert.InDelta(t, 5400.0, resp.RemainingSeconds, 1.0)
}

// AC-9: negative time_spent_seconds → ErrNegativeTimeSpent.
func TestSaveAnswer_NegativeTimeSpent(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{
		SelectedOptionIDs: []string{},
		TimeSpentSeconds:  -1,
	})
	require.ErrorIs(t, err, ErrNegativeTimeSpent)
}

// AC-10: different user → ErrSessionForbidden.
func TestSaveAnswer_Forbidden(t *testing.T) {
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, _ string) (*sessionStateRow, error) {
			return nil, ErrSessionForbidden
		},
	}
	svc := NewService(repo)
	_, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "other-user", SaveAnswerInput{})
	require.ErrorIs(t, err, ErrSessionForbidden)
}

// AC-3: status not in_progress → ErrSessionNotActive.
func TestSaveAnswer_SessionNotActive(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			s := activeSession(userID)
			s.Status = "submitted"
			return s, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{})
	require.ErrorIs(t, err, ErrSessionNotActive)
}

// AC-2: expires_at <= NOW() → ErrSessionExpired.
func TestSaveAnswer_SessionExpired(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			s := activeSession(userID)
			s.ExpiresAt = time.Date(2026, 5, 1, 11, 30, 0, 0, time.UTC)
			return s, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{})
	require.ErrorIs(t, err, ErrSessionExpired)
}

// AC-4: questionId not in session → ErrQuestionNotInSession.
func TestSaveAnswer_QuestionNotInSession(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getSessionQuestionsFn: func(_ context.Context, _ string) ([]sessionQuestionRow, error) {
			return []sessionQuestionRow{
				{QuestionID: "q-other", SortOrder: 0, QuestionType: "single_choice"},
			}, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{})
	require.ErrorIs(t, err, ErrQuestionNotInSession)
}

// AC-5: invalid option ID → ErrInvalidOption.
func TestSaveAnswer_InvalidOption(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getValidOptionIDsFn: func(_ context.Context, _ string) (map[string]struct{}, error) {
			return map[string]struct{}{"opt-a": {}, "opt-b": {}}, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{
		SelectedOptionIDs: []string{"opt-z"},
	})
	require.ErrorIs(t, err, ErrInvalidOption)
}

// Notes: short_text with non-empty selected_option_ids → ErrInvalidAnswerFormat.
func TestSaveAnswer_ShortText_NonEmptyOptions(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getSessionQuestionsFn: func(_ context.Context, _ string) ([]sessionQuestionRow, error) {
			return []sessionQuestionRow{
				{QuestionID: "q-1", SortOrder: 0, QuestionType: "short_text"},
			}, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{
		SelectedOptionIDs: []string{"opt-a"},
	})
	require.ErrorIs(t, err, ErrInvalidAnswerFormat)
}

// Notes: single_choice with >1 selected → ErrInvalidAnswerFormat.
func TestSaveAnswer_SingleChoice_TooManyOptions(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getSessionQuestionsFn: func(_ context.Context, _ string) ([]sessionQuestionRow, error) {
			return []sessionQuestionRow{
				{QuestionID: "q-1", SortOrder: 0, QuestionType: "single_choice"},
			}, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{
		SelectedOptionIDs: []string{"opt-a", "opt-b"},
	})
	require.ErrorIs(t, err, ErrInvalidAnswerFormat)
}

// AC-6: remaining_seconds is positive when session is near expiry.
func TestSaveAnswer_RemainingSecondsNearExpiry(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 11, 29, 59, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			s := activeSession(userID)
			s.ExpiresAt = time.Date(2026, 5, 1, 11, 30, 0, 0, time.UTC)
			return s, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.SaveAnswer(context.Background(), "sess-1", "q-1", "user-1", SaveAnswerInput{
		SelectedOptionIDs: []string{"opt-a"},
	})
	require.NoError(t, err)
	assert.InDelta(t, 1.0, resp.RemainingSeconds, 1.0)
}

// ── FR-BB37: GetSessionState ──────────────────────────────────────────────────

func TestGetSessionState_HappyPath(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	savedAt := time.Date(2026, 5, 1, 10, 1, 0, 0, time.UTC)
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getSessionAnswersFn: func(_ context.Context, _ string) (map[string]savedAnswerRow, error) {
			return map[string]savedAnswerRow{
				"q-1": {
					SelectedOptionIDs: []byte(`["opt-a"]`),
					TimeSpentSeconds:  45,
					SavedAt:           savedAt,
				},
			}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.GetSessionState(context.Background(), "sess-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "sess-1", resp.SessionID)
	assert.Equal(t, "in_progress", resp.Status)
	assert.Len(t, resp.Questions, 1)
	assert.InDelta(t, 5400.0, resp.RemainingSeconds, 1.0)
	ans, ok := resp.Answers["q-1"]
	require.True(t, ok)
	assert.Equal(t, []string{"opt-a"}, ans.SelectedOptionIDs)
	assert.Equal(t, 45, ans.TimeSpentSeconds)
}

// AC-7: different user → ErrSessionForbidden.
func TestGetSessionState_Forbidden(t *testing.T) {
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, _ string) (*sessionStateRow, error) {
			return nil, ErrSessionForbidden
		},
	}
	svc := NewService(repo)
	_, err := svc.GetSessionState(context.Background(), "sess-1", "other-user")
	require.ErrorIs(t, err, ErrSessionForbidden)
}

// AC-8: remaining_seconds = 0 for submitted sessions.
func TestGetSessionState_SubmittedSession_ZeroRemaining(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			s := activeSession(userID)
			s.Status = "submitted"
			return s, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.GetSessionState(context.Background(), "sess-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.RemainingSeconds)
}

// AC-8: answers map includes all saved answers.
func TestGetSessionState_AnswersMap(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	savedAt := time.Date(2026, 5, 1, 10, 5, 0, 0, time.UTC)
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getSessionQuestionsFn: func(_ context.Context, _ string) ([]sessionQuestionRow, error) {
			return []sessionQuestionRow{
				{QuestionID: "q-1", SortOrder: 0, OptionsOrder: []byte(`["opt-a","opt-b"]`), QuestionType: "single_choice"},
				{QuestionID: "q-2", SortOrder: 1, OptionsOrder: []byte(`[]`), QuestionType: "short_text"},
			}, nil
		},
		getSessionAnswersFn: func(_ context.Context, _ string) (map[string]savedAnswerRow, error) {
			return map[string]savedAnswerRow{
				"q-1": {SelectedOptionIDs: []byte(`["opt-b"]`), TimeSpentSeconds: 20, SavedAt: savedAt},
			}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.GetSessionState(context.Background(), "sess-1", "user-1")
	require.NoError(t, err)
	assert.Len(t, resp.Questions, 2)
	assert.Len(t, resp.Answers, 1)
	ans := resp.Answers["q-1"]
	assert.Equal(t, []string{"opt-b"}, ans.SelectedOptionIDs)
}

// GetSessionState: session not found → ErrSessionNotFound.
func TestGetSessionState_NotFound(t *testing.T) {
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, _ string) (*sessionStateRow, error) {
			return nil, ErrSessionNotFound
		},
	}
	svc := NewService(repo)
	_, err := svc.GetSessionState(context.Background(), "sess-missing", "user-1")
	require.ErrorIs(t, err, ErrSessionNotFound)
}

// ── FR-BB38: ReportEvent ──────────────────────────────────────────────────────

// AC-8: invalid event type → ErrInvalidEventType.
func TestReportEvent_InvalidEventType(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "keyboard"})
	require.ErrorIs(t, err, ErrInvalidEventType)
}

// AC-7: different user → ErrSessionForbidden.
func TestReportEvent_Forbidden(t *testing.T) {
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, _ string) (*sessionStateRow, error) {
			return nil, ErrSessionForbidden
		},
	}
	svc := NewService(repo)
	_, err := svc.ReportEvent(context.Background(), "sess-1", "other", ReportEventInput{Type: "tab_switch"})
	require.ErrorIs(t, err, ErrSessionForbidden)
}

// AC-5: session not in_progress → ErrSessionNotActive.
func TestReportEvent_SessionNotActive(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			s := activeSession(userID)
			s.Status = "submitted"
			return s, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "blur"})
	require.ErrorIs(t, err, ErrSessionNotActive)
}

// AC-6: session expired → ErrSessionExpired.
func TestReportEvent_SessionExpired(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			s := activeSession(userID)
			s.ExpiresAt = time.Date(2026, 5, 1, 11, 0, 0, 0, time.UTC)
			return s, nil
		},
	}
	svc := NewService(repo)
	_, err := svc.ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "fullscreen_exit"})
	require.ErrorIs(t, err, ErrSessionExpired)
}

// AC-4: on_tab_switch='log' → warn=false + event_count.
func TestReportEvent_PolicyLog(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	var capturedAction string
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.OnTabSwitch = "log"
			return cfg, nil
		},
		insertTabSwitchEventFn: func(_ context.Context, _, _, actionTaken string) error {
			capturedAction = actionTaken
			return nil
		},
		countTabSwitchEventsFn: func(_ context.Context, _ string) (int, error) {
			return 3, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "tab_switch"})
	require.NoError(t, err)
	assert.False(t, resp.Warn)
	assert.Equal(t, 3, resp.EventCount)
	assert.Equal(t, "log", capturedAction)
}

// AC-3: on_tab_switch='warn' → warn=true + event_count.
func TestReportEvent_PolicyWarn(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.OnTabSwitch = "warn"
			return cfg, nil
		},
		countTabSwitchEventsFn: func(_ context.Context, _ string) (int, error) {
			return 2, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "blur"})
	require.NoError(t, err)
	assert.True(t, resp.Warn)
	assert.Equal(t, 2, resp.EventCount)
}

// AC-2: on_tab_switch='submit' → auto_submitted response.
func TestReportEvent_PolicySubmit(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	score := 80.0
	passed := true
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.OnTabSwitch = "submit"
			return cfg, nil
		},
		autoSubmitSessionFn: func(_ context.Context, sessionID string) (*autoSubmitResult, error) {
			status := "auto_submitted"
			return &autoSubmitResult{
				SessionID: sessionID,
				Status:    status,
				ScorePct:  &score,
				Passed:    &passed,
			}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "fullscreen_exit"})
	require.NoError(t, err)
	assert.False(t, resp.Warn)
	require.NotNil(t, resp.Status)
	assert.Equal(t, "auto_submitted", *resp.Status)
	require.NotNil(t, resp.ScorePct)
	assert.InDelta(t, 80.0, *resp.ScorePct, 0.01)
	require.NotNil(t, resp.Passed)
	assert.True(t, *resp.Passed)
}

// AC-10: action_taken in event matches exam policy, not request type.
func TestReportEvent_ActionTakenMatchesPolicy(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	var storedAction string
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.OnTabSwitch = "warn"
			return cfg, nil
		},
		insertTabSwitchEventFn: func(_ context.Context, _, _, actionTaken string) error {
			storedAction = actionTaken
			return nil
		},
	}
	svc := NewService(repo)
	_, err := svc.ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "blur"})
	require.NoError(t, err)
	assert.Equal(t, "warn", storedAction)
}

// AC-9: each call inserts a new event (no dedup).
func TestReportEvent_EachCallInsertsEvent(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	insertCount := 0
	repo := &mockRepo{
		getSessionForUserFn: func(_ context.Context, _, userID string) (*sessionStateRow, error) {
			return activeSession(userID), nil
		},
		insertTabSwitchEventFn: func(_ context.Context, _, _, _ string) error {
			insertCount++
			return nil
		},
	}
	svc := NewService(repo)
	for i := 0; i < 3; i++ {
		_, err := svc.ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "tab_switch"})
		require.NoError(t, err)
	}
	assert.Equal(t, 3, insertCount)
}

// ── FR-BB39: SubmitSession ────────────────────────────────────────────────────

// AC-5: auto-graded session → status=submitted + score populated.
func TestSubmitSession_AutoGraded(t *testing.T) {
	score := 82.5
	passed := true
	now := time.Date(2026, 5, 14, 11, 15, 0, 0, time.UTC)
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{
				SessionID:   sessionID,
				Status:      "submitted",
				SubmittedAt: now,
				ScorePct:    &score,
				Passed:      &passed,
			}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.SubmitSession(context.Background(), "sess-42", "user-1", "tenant-1", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "sess-42", resp.SessionID)
	assert.Equal(t, "submitted", resp.Status)
	require.NotNil(t, resp.ScorePct)
	assert.InDelta(t, 82.5, *resp.ScorePct, 0.01)
	require.NotNil(t, resp.Passed)
	assert.True(t, *resp.Passed)
}

// AC-4: session with short_text → status=grading_pending + no score.
func TestSubmitSession_PendingManualGrade(t *testing.T) {
	now := time.Date(2026, 5, 14, 11, 15, 0, 0, time.UTC)
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{
				SessionID:   sessionID,
				Status:      "grading_pending",
				SubmittedAt: now,
				ScorePct:    nil,
				Passed:      nil,
			}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.SubmitSession(context.Background(), "sess-43", "user-1", "tenant-1", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "grading_pending", resp.Status)
	assert.Nil(t, resp.ScorePct)
	assert.Nil(t, resp.Passed)
}

// AC-1: wrong user → ErrSessionForbidden.
func TestSubmitSession_Forbidden(t *testing.T) {
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, _, _, _, _ string) (*SubmitSessionResponse, error) {
			return nil, ErrSessionForbidden
		},
	}
	svc := NewService(repo)
	_, err := svc.SubmitSession(context.Background(), "sess-1", "other-user", "tenant-1", "127.0.0.1")
	require.ErrorIs(t, err, ErrSessionForbidden)
}

// AC-2: session not in_progress (but not forbidden) → still returns 200 with current state.
func TestSubmitSession_AlreadySubmitted_Idempotent(t *testing.T) {
	score := 90.0
	passed := true
	now := time.Date(2026, 5, 14, 11, 15, 0, 0, time.UTC)
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{
				SessionID:   sessionID,
				Status:      "submitted",
				SubmittedAt: now,
				ScorePct:    &score,
				Passed:      &passed,
			}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.SubmitSession(context.Background(), "sess-42", "user-1", "tenant-1", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "submitted", resp.Status)
	require.NotNil(t, resp.ScorePct)
	assert.InDelta(t, 90.0, *resp.ScorePct, 0.01)
}

// Session not found → ErrSessionNotFound propagated.
func TestSubmitSession_NotFound(t *testing.T) {
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, _, _, _, _ string) (*SubmitSessionResponse, error) {
			return nil, ErrSessionNotFound
		},
	}
	svc := NewService(repo)
	_, err := svc.SubmitSession(context.Background(), "sess-missing", "user-1", "tenant-1", "127.0.0.1")
	require.ErrorIs(t, err, ErrSessionNotFound)
}

// AC-7: response includes session_id and status always; score/passed only when submitted.
func TestSubmitSession_ResponseShape_GradingPending(t *testing.T) {
	now := time.Now().UTC()
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{
				SessionID:   sessionID,
				Status:      "grading_pending",
				SubmittedAt: now,
				ScorePct:    nil,
				Passed:      nil,
			}, nil
		},
	}
	svc := NewService(repo)
	resp, err := svc.SubmitSession(context.Background(), "sess-43", "user-1", "tenant-1", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "sess-43", resp.SessionID)
	assert.Equal(t, "grading_pending", resp.Status)
	assert.Nil(t, resp.ScorePct)
	assert.Nil(t, resp.Passed)
}
