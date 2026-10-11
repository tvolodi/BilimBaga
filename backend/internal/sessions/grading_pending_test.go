package sessions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mailRecorder counts the session mails a service asks for (FR-BB322 AC-9, AC-10).
type mailRecorder struct {
	submitted []string
	results   []string
	corrected []string
}

func (m *mailRecorder) TriggerSessionResult(sessionID string) {
	m.results = append(m.results, sessionID)
}
func (m *mailRecorder) TriggerExamSubmitted(sessionID string) {
	m.submitted = append(m.submitted, sessionID)
}
func (m *mailRecorder) TriggerResultCorrected(sessionID string) {
	m.corrected = append(m.corrected, sessionID)
}

// ── FR-BB322 AC-1/AC-3: the start check order ───────────────────────────────

func TestCreateSession_GradingPendingIsCheckedBeforeAttempts(t *testing.T) {
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.MaxAttempts = 1
			return cfg, nil
		},
		hasGradingPendingFn:     func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		countFinishedSessionsFn: func(_ context.Context, _, _ string) (int, error) { return 1, nil },
	}
	_, err := NewService(repo).CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	assert.ErrorIs(t, err, ErrGradingPending, "a waiting session answers first, even when the attempts are used up")
}

func TestCreateSession_OnlyPendingWithOneAttemptIsGradingPending(t *testing.T) {
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.MaxAttempts = 1
			return cfg, nil
		},
		hasGradingPendingFn:     func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		countFinishedSessionsFn: func(_ context.Context, _, _ string) (int, error) { return 0, nil },
	}
	_, err := NewService(repo).CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	assert.ErrorIs(t, err, ErrGradingPending)
}

func TestCreateSession_OnlyFinishedAttemptsIsExhausted(t *testing.T) {
	repo := &mockRepo{
		getExamConfigFn: func(_ context.Context, _ string) (*examConfig, error) {
			cfg := defaultConfig()
			cfg.MaxAttempts = 1
			return cfg, nil
		},
		hasGradingPendingFn:     func(_ context.Context, _, _ string) (bool, error) { return false, nil },
		countFinishedSessionsFn: func(_ context.Context, _, _ string) (int, error) { return 1, nil },
	}
	_, err := NewService(repo).CreateSession(context.Background(), "exam-1", "user-1", "dept-1")
	assert.ErrorIs(t, err, ErrAttemptsExhausted)
	assert.NotErrorIs(t, err, ErrGradingPending)
}

// ── FR-BB322 AC-9/AC-10: mails follow the transition, once ──────────────────

func TestSubmitSession_PendingSendsExamSubmittedOnce(t *testing.T) {
	mailer := &mailRecorder{}
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{SessionID: sessionID, Status: "grading_pending", Transitioned: true}, nil
		},
	}
	_, err := NewService(repo, WithResultMailer(mailer)).SubmitSession(context.Background(), "sess-1", "user-1", "t", "ip")
	require.NoError(t, err)
	assert.Equal(t, []string{"sess-1"}, mailer.submitted)
	assert.Empty(t, mailer.results, "a waiting session never gets a result mail (AC-10)")
}

func TestSubmitSession_ReplayResendsNoMail(t *testing.T) {
	passed := true
	score := 80.0
	cases := []struct {
		name   string
		status string
		passed *bool
	}{
		{"replayed waiting session", "grading_pending", nil},
		{"replayed final session", "submitted", &passed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mailer := &mailRecorder{}
			repo := &mockRepo{
				submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
					return &SubmitSessionResponse{SessionID: sessionID, Status: tc.status, ScorePct: &score, Passed: tc.passed}, nil
				},
			}
			_, err := NewService(repo, WithResultMailer(mailer)).SubmitSession(context.Background(), "sess-1", "user-1", "t", "ip")
			require.NoError(t, err)
			assert.Empty(t, mailer.submitted)
			assert.Empty(t, mailer.results)
		})
	}
}

func TestSubmitSession_FinalSendsResultOnce(t *testing.T) {
	mailer := &mailRecorder{}
	passed := true
	score := 80.0
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{SessionID: sessionID, Status: "submitted", ScorePct: &score, Passed: &passed, Transitioned: true}, nil
		},
	}
	_, err := NewService(repo, WithResultMailer(mailer)).SubmitSession(context.Background(), "sess-1", "user-1", "t", "ip")
	require.NoError(t, err)
	assert.Equal(t, []string{"sess-1"}, mailer.results)
	assert.Empty(t, mailer.submitted)
}

// FR-BB322 AC-13: the submit response of a waiting session carries no score and no pass, even though the repository
// stores and returns the partial values. Both the transition and a replayed submit are covered.
func TestSubmitSession_WaitingSessionReturnsNoScoreOnTransition(t *testing.T) {
	score := 40.0
	passed := true
	mailer := &mailRecorder{}
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{SessionID: sessionID, Status: "grading_pending", ScorePct: &score, Passed: &passed, Transitioned: true}, nil
		},
	}
	resp, err := NewService(repo, WithResultMailer(mailer)).SubmitSession(context.Background(), "sess-1", "user-1", "t", "ip")
	require.NoError(t, err)
	assert.Nil(t, resp.ScorePct)
	assert.Nil(t, resp.Passed)
	assert.Equal(t, []string{"sess-1"}, mailer.submitted, "the mail decision still sees the transition")
}

func TestSubmitSession_WaitingSessionReturnsNoScoreOnReplay(t *testing.T) {
	score := 40.0
	passed := true
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{SessionID: sessionID, Status: "grading_pending", ScorePct: &score, Passed: &passed}, nil
		},
	}
	resp, err := NewService(repo).SubmitSession(context.Background(), "sess-1", "user-1", "t", "ip")
	require.NoError(t, err)
	assert.Nil(t, resp.ScorePct)
	assert.Nil(t, resp.Passed)
}

// FR-BB322 AC-13 through the HTTP handler and the real service: the JSON body carries null, not the partial values.
func TestSubmitSession_Handler_WaitingSessionBodyHasNullScoreAndPass(t *testing.T) {
	score := 40.0
	passed := true
	repo := &mockRepo{
		submitSessionFn: func(_ context.Context, sessionID, _, _, _ string) (*SubmitSessionResponse, error) {
			return &SubmitSessionResponse{SessionID: sessionID, Status: "grading_pending", SubmittedAt: time.Now().UTC(), ScorePct: &score, Passed: &passed, Transitioned: true}, nil
		},
	}
	h := NewHandler(NewService(repo))
	w := httptest.NewRecorder()
	h.SubmitSession(w, submitRequest("sess-7"))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"score_pct":null,"passed":null`)
	body := decodeBody(t, w.Body.Bytes())
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Nil(t, data["score_pct"])
	assert.Nil(t, data["passed"])
}

// FR-BB73 AC-19: the mail decision of a grade. A pass or fail that flips on a final session sends one corrected mail;
// the transition from grading_pending sends the result mail; an override that does not flip sends nothing.
func TestGradeAnswer_MailDecisionFollowsTheTransition(t *testing.T) {
	passed := false
	cases := []struct {
		name          string
		result        gradeAnswerResult
		wantResult    int
		wantCorrected int
	}{
		{"finalising grade on a waiting session sends the result mail", gradeAnswerResult{allGraded: true, sessionStatus: "submitted", passed: &passed, previousStatus: "grading_pending"}, 1, 0},
		{"a grade that leaves the session waiting sends nothing", gradeAnswerResult{allGraded: false, sessionStatus: "grading_pending", previousStatus: "grading_pending"}, 0, 0},
		{"an override that flips a final pass sends one corrected mail", gradeAnswerResult{allGraded: true, sessionStatus: "submitted", passed: &passed, previousStatus: "submitted", passedFlipped: true}, 0, 1},
		{"an override that does not flip sends nothing", gradeAnswerResult{allGraded: true, sessionStatus: "submitted", passed: &passed, previousStatus: "submitted"}, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &mailRecorder{}
			result := tc.result
			repo := &mockRepo{
				gradeAnswerFn: func(_ context.Context, _, _, _, _, _ string, _ float64, _ string) (*gradeAnswerResult, error) {
					return &result, nil
				},
			}
			_, err := NewService(repo, WithResultMailer(rec)).GradeAnswer(context.Background(), "sess-1", "q1", "grader-1", "t", "ip", GradeAnswerRequest{ScorePct: 40})
			require.NoError(t, err)
			assert.Len(t, rec.results, tc.wantResult)
			assert.Len(t, rec.corrected, tc.wantCorrected)
		})
	}
}

func TestReportEvent_AutoSubmitWaitingSendsExamSubmittedAndHidesScore(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	mailer := &mailRecorder{}
	score := 40.0
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
			return &autoSubmitResult{SessionID: sessionID, Status: "grading_pending", ScorePct: &score, Passed: &passed, Transitioned: true}, nil
		},
	}
	resp, err := NewService(repo, WithResultMailer(mailer)).ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "tab_switch"})
	require.NoError(t, err)
	assert.Equal(t, []string{"sess-1"}, mailer.submitted)
	assert.Empty(t, mailer.results)
	assert.Nil(t, resp.ScorePct, "a waiting session shows no score (AC-13)")
	assert.Nil(t, resp.Passed)
}

func TestReportEvent_AutoSubmitReplayResendsNoMail(t *testing.T) {
	fixNow(t, time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	mailer := &mailRecorder{}
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
			return &autoSubmitResult{SessionID: sessionID, Status: "grading_pending"}, nil
		},
	}
	_, err := NewService(repo, WithResultMailer(mailer)).ReportEvent(context.Background(), "sess-1", "user-1", ReportEventInput{Type: "tab_switch"})
	require.NoError(t, err)
	assert.Empty(t, mailer.submitted, "Transitioned is false on a replay")
}

// ── FR-BB322 AC-4/AC-13: the result of a waiting session ────────────────────

// waitingRow is a session that is waiting for grading and carries a stored partial score and pass.
func waitingRow() *sessionResultRow {
	score := 40.0
	now := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	return &sessionResultRow{
		SessionID: "sess-1", ExamID: "exam-1", ExamTitle: "Safety", UserID: "user-1",
		Status: "grading_pending", ScorePct: &score, Passed: true,
		ShowAnswersMode: "never", AttemptNumber: 1, SubmittedAt: &now,
	}
}

func waitingRepo(pending, total int) *mockRepo {
	return &mockRepo{
		getSessionResultFn: func(_ context.Context, _, _ string) (*sessionResultRow, error) {
			return waitingRow(), nil
		},
		getAdminSessionResultFn: func(_ context.Context, _ string) (*sessionResultRow, error) {
			return waitingRow(), nil
		},
		getSectionScoresFn: func(_ context.Context, _ string) ([]SectionScore, error) {
			return []SectionScore{{SectionID: "s-1", Title: "A", ScorePct: 40}}, nil
		},
		gradableCountsFn: func(_ context.Context, _ string) (int, int, error) {
			return pending, total, nil
		},
	}
}

func TestGetSessionResult_WaitingHidesScoreAndSectionsForEmployee(t *testing.T) {
	svc := NewService(waitingRepo(2, 3))
	resp, err := svc.GetSessionResult(context.Background(), "sess-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "grading_pending", resp.Status)
	assert.Nil(t, resp.ScorePct)
	assert.False(t, resp.Passed)
	assert.Empty(t, resp.PerSectionScores)
	assert.Equal(t, 2, resp.PendingQuestionCount)
	assert.Equal(t, 3, resp.TotalGradableCount)
	require.NotNil(t, resp.GradingExpectedBy)
	assert.Equal(t, "examiner", *resp.GradingExpectedBy, "AI is off, so an examiner grades")
}

func TestGetAdminSessionResult_WaitingKeepsStoredScore(t *testing.T) {
	resp, err := NewService(waitingRepo(2, 3)).GetAdminSessionResult(context.Background(), "sess-1")
	require.NoError(t, err)
	if assert.NotNil(t, resp.ScorePct) {
		assert.InDelta(t, 40.0, *resp.ScorePct, 0.01)
	}
	assert.True(t, resp.Passed, "examiners see the stored values (AC-4, AC-13)")
	assert.Equal(t, 2, resp.PendingQuestionCount)
}

func TestGetSessionResult_FinalSessionHasNoPendingFields(t *testing.T) {
	score := 80.0
	now := time.Now().UTC()
	repo := &mockRepo{
		getSessionResultFn: func(_ context.Context, _, _ string) (*sessionResultRow, error) {
			return &sessionResultRow{SessionID: "sess-1", ExamID: "exam-1", ExamTitle: "T", UserID: "user-1",
				Status: "submitted", ScorePct: &score, Passed: true, ShowAnswersMode: "never", AttemptNumber: 1, SubmittedAt: &now}, nil
		},
	}
	resp, err := NewService(repo).GetSessionResult(context.Background(), "sess-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, 0, resp.PendingQuestionCount)
	assert.Equal(t, 0, resp.TotalGradableCount)
	assert.Nil(t, resp.GradingExpectedBy, "a final session has no grader to name")
	if assert.NotNil(t, resp.ScorePct) {
		assert.InDelta(t, 80.0, *resp.ScorePct, 0.01)
	}
}

// AC-4: grading_expected_by is "ai" only when the job would grade a pending answer, with the daily cap not reached.
func TestGradingExpectedBy(t *testing.T) {
	cases := []struct {
		name       string
		enabled    bool
		limit      int
		eligible   int
		used       int
		wantExpect string
	}{
		{"eligible answers and quota left", true, 10, 2, 3, "ai"},
		{"daily cap reached", true, 10, 2, 10, "examiner"},
		{"no eligible answer", true, 10, 0, 3, "examiner"},
		{"provider not configured", false, 10, 2, 0, "examiner"},
		{"daily limit off", true, 0, 2, 0, "examiner"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := waitingRepo(2, 2)
			repo.aiCandidateCountFn = func(_ context.Context, _ string) (int, error) { return tc.eligible, nil }
			repo.aiDailyUsageFn = func(_ context.Context) (int, error) { return tc.used, nil }
			resp, err := NewService(repo, WithAIGrading(tc.enabled, tc.limit)).GetSessionResult(context.Background(), "sess-1", "user-1")
			require.NoError(t, err)
			require.NotNil(t, resp.GradingExpectedBy)
			assert.Equal(t, tc.wantExpect, *resp.GradingExpectedBy)
		})
	}
}

// AC-8: a question still waiting for grading shows no correct answer and no explanation to the employee.
func TestGetSessionResult_PendingQuestionHidesAnswerForEmployee(t *testing.T) {
	score := 50.0
	now := time.Now().UTC()
	explanation := "Because B"
	repo := &mockRepo{
		getSessionResultFn: func(_ context.Context, _, _ string) (*sessionResultRow, error) {
			return &sessionResultRow{SessionID: "sess-1", ExamID: "exam-1", ExamTitle: "T", UserID: "user-1",
				Status: "grading_pending", ScorePct: &score, ShowAnswersMode: "after_completion", AttemptNumber: 1, SubmittedAt: &now}, nil
		},
		getQuestionBreakdownFn: func(_ context.Context, _, _ string) ([]questionBreakdownRow, error) {
			return []questionBreakdownRow{
				{QuestionID: "q-1", Stem: "Open", GradingStatus: "pending_manual", Explanation: &explanation},
				{QuestionID: "q-2", Stem: "Closed", GradingStatus: "graded", PointsEarned: 1, MaxPoints: 1, Explanation: &explanation},
			}, nil
		},
		getCorrectAnswerTextsFn: func(_ context.Context, _ []string, _ string) (map[string][]string, error) {
			return map[string][]string{"q-1": {"Answer A"}, "q-2": {"Answer B"}}, nil
		},
	}
	resp, err := NewService(repo).GetSessionResult(context.Background(), "sess-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, resp.PerQuestionBreakdown)
	items := *resp.PerQuestionBreakdown
	require.Len(t, items, 2)

	assert.Equal(t, "pending_manual", items[0].GradingStatus)
	assert.Empty(t, items[0].CorrectAnswer)
	assert.Nil(t, items[0].Explanation)

	assert.Equal(t, "graded", items[1].GradingStatus)
	assert.Equal(t, []string{"Answer B"}, items[1].CorrectAnswer, "graded questions render as before")
	assert.Equal(t, &explanation, items[1].Explanation)
}
