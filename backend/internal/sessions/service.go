package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/email"
	"github.com/bilimbaga/bilimbaga/internal/exams"
)

// Service defines the business logic for session creation and answer saving.
type Service interface {
	// CreateSession runs all pre-flight checks, resolves questions, and creates
	// an exam session for the authenticated user. Returns the full session response
	// including the resolved question set (no correct-answer flags).
	CreateSession(ctx context.Context, examID, userID, deptID string) (*CreateSessionResponse, error)

	// SaveAnswer upserts the answer for one question in a session (FR-BB37).
	SaveAnswer(ctx context.Context, sessionID, questionID, userID string, input SaveAnswerInput) (*SaveAnswerResponse, error)

	// GetSessionState returns full session state for resume/review (FR-BB37).
	GetSessionState(ctx context.Context, sessionID, userID string) (*ResumeSessionResponse, error)

	// ReportEvent records a tab-switch/blur/fullscreen-exit event and enforces
	// the exam-level on_tab_switch policy (FR-BB38).
	ReportEvent(ctx context.Context, sessionID, userID string, input ReportEventInput) (*ReportEventResponse, error)

	// SubmitSession explicitly submits an in_progress session (FR-BB39).
	// tenantID and actorIP are forwarded for the in-transaction audit log entries.
	SubmitSession(ctx context.Context, sessionID, userID, tenantID, actorIP string) (*SubmitSessionResponse, error)

	// GetSessionResult returns the result for a completed session, scoped to the owning user (FR-BB41).
	// Returns ErrSessionInProgress if the session is still in_progress.
	// Returns ErrSessionForbidden if userID does not own the session.
	GetSessionResult(ctx context.Context, sessionID, userID string) (*SessionResultResponse, error)

	// GetAdminSessionResult returns the result for any session without user scoping (FR-BB41 AC-5).
	// Returns ErrSessionInProgress if the session is still in_progress.
	GetAdminSessionResult(ctx context.Context, sessionID string) (*SessionResultResponse, error)

	// GetExamHistory returns paginated session history for a user+exam pair (FR-BB41 AC-6/AC-10).
	// Returns ErrExamNotFound if the exam does not exist.
	GetExamHistory(ctx context.Context, examID, userID string, page, perPage int) (*ExamHistoryResponse, error)

	// GetMyResults returns paginated session history across all exams for the calling user (FR-BB46).
	// sort must be 'date' or 'score'; dir must be 'asc' or 'desc'.
	GetMyResults(ctx context.Context, userID string, page, perPage int, sort, dir string) (*MyResultsResponse, error)

	// ListGradingQueue returns sessions pending manual grading (FR-BB42 AC-1/AC-2/AC-3).
	ListGradingQueue(ctx context.Context, examID *string, dateFrom, dateTo *time.Time, page, perPage int) (*GradingQueueResponse, error)

	// GetGradingDetail returns session header and short-text questions for grading (FR-BB42 AC-4).
	GetGradingDetail(ctx context.Context, sessionID string) (*GradingDetailResponse, error)

	// GradeAnswer validates and persists a manual grade for one short-text answer (FR-BB42 AC-5/AC-6/AC-7/AC-8/AC-10).
	// Returns ErrInvalidScore if score_pct is not in [0, 100].
	GradeAnswer(ctx context.Context, sessionID, questionID, graderID, tenantID, actorIP string, req GradeAnswerRequest) (*GradeAnswerResponse, error)

	// SelectNextAdaptiveQuestion picks the next question for an adaptive session (FR-BB72 AC-3/AC-4/AC-5).
	// Returns ErrNotAdaptive if the session's exam is not adaptive.
	// Returns ErrSessionForbidden if userID does not own the session.
	SelectNextAdaptiveQuestion(ctx context.Context, sessionID, userID string) (*NextQuestionResponse, error)

	// RecordAdaptiveAnswer updates adaptive_state after an answer is saved (FR-BB72 AC-6).
	// Appends correctness to recent_results and clears current_question_id.
	// No-op (nil error) if the session is not adaptive.
	RecordAdaptiveAnswer(ctx context.Context, sessionID, questionID string, correct bool) error
}

type service struct {
	repo     Repository
	emailSvc *email.EmailService
}

// NewService returns a Service backed by the given Repository.
// An optional EmailService may be passed as the second argument to enable
// transactional email delivery; pass nil or omit to disable.
func NewService(repo Repository, emailSvc ...*email.EmailService) Service {
	s := &service{repo: repo}
	if len(emailSvc) > 0 {
		s.emailSvc = emailSvc[0]
	}
	return s
}

// nowFn is overridable in tests.
var nowFn = func() time.Time { return time.Now().UTC() }

func (s *service) CreateSession(ctx context.Context, examID, userID, deptID string) (*CreateSessionResponse, error) {
	// AC-1: assignment check.
	assigned, err := s.repo.IsAssigned(ctx, examID, userID, deptID)
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: assignment check: %w", err)
	}
	if !assigned {
		return nil, ErrNotAssigned
	}

	// AC-2: exam must be active (FR-BB32 AC-5: archived exam returns ErrExamArchived → 403).
	cfg, err := s.repo.GetExamConfig(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: get config: %w", err)
	}
	if cfg.Status == "archived" {
		return nil, ErrExamArchived
	}
	if cfg.Status != "active" {
		return nil, ErrExamNotActive
	}

	// AC-3: availability window.
	current := nowFn()
	if cfg.AvailableFrom != nil && current.Before(*cfg.AvailableFrom) {
		return nil, ErrExamOutsideWindow
	}
	if cfg.AvailableUntil != nil && current.After(*cfg.AvailableUntil) {
		return nil, ErrExamOutsideWindow
	}

	// AC-4: attempt limit.
	finished, err := s.repo.CountFinishedSessions(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: count sessions: %w", err)
	}
	if finished >= cfg.MaxAttempts {
		return nil, ErrAttemptsExhausted
	}

	// AC-5: no concurrent open session.
	hasOpen, err := s.repo.HasOpenSession(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: open session check: %w", err)
	}
	if hasOpen {
		return nil, ErrSessionAlreadyOpen
	}

	// AC-6/AC-7: resolve questions using seeded PRNG.
	seed := current.UnixNano()
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec

	rules, err := s.repo.GetRules(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: get rules: %w", err)
	}

	var resolvedIDs []string // question IDs in rule order before shuffle
	type resolvedWithRuleID struct {
		id     string
		ruleID string
	}
	var resolvedWithRules []resolvedWithRuleID
	for _, rule := range rules {
		var pool []poolQuestion
		if rule.Mode == "manual" {
			pool, err = s.repo.GetManualQuestions(ctx, rule.ID)
		} else {
			pool, err = s.repo.GetEligibleQuestions(ctx, rule)
			if err == nil {
				if len(pool) < rule.Count {
					return nil, fmt.Errorf("sessions: CreateSession: rule %s: %w", rule.ID, ErrInsufficientQuestions)
				}
				shufflePoolQuestions(pool, rng)
				pool = pool[:rule.Count]
			}
		}
		if err != nil {
			return nil, fmt.Errorf("sessions: CreateSession: resolve rule %s: %w", rule.ID, err)
		}
		for _, q := range pool {
			resolvedIDs = append(resolvedIDs, q.ID)
			resolvedWithRules = append(resolvedWithRules, resolvedWithRuleID{id: q.ID, ruleID: rule.ID})
		}
	}

	// ISS-132: a fixed-form exam that resolves to zero questions (no rules, empty manual rule, or
	// all manual questions archived) must not start: the employee would land on an empty session.
	// Adaptive exams pick questions at runtime, so they legitimately have none up front.
	if len(resolvedIDs) == 0 && !cfg.Adaptive {
		return nil, fmt.Errorf("sessions: CreateSession: exam %s resolved to zero questions: %w", examID, ErrInsufficientQuestions)
	}

	// Collect type info alongside IDs so we can build poolQuestion slices for shuffle.
	type qWithType struct {
		id     string
		ruleID string // rule that produced this question (empty for questions outside any rule)
	}
	allQs := make([]qWithType, len(resolvedIDs))
	for i, rwi := range resolvedWithRules {
		allQs[i] = qWithType(rwi)
	}

	// AC-7: apply shuffle_questions.
	if cfg.ShuffleQuestions {
		rng.Shuffle(len(allQs), func(i, j int) { allQs[i], allQs[j] = allQs[j], allQs[i] })
	}

	// Fetch question details (stem, type) for all resolved questions.
	ids := make([]string, len(allQs))
	for i, q := range allQs {
		ids[i] = q.id
	}
	details, err := s.repo.GetQuestionDetails(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: get question details: %w", err)
	}

	// AC-7/AC-9: resolve option order per question; collect all option IDs for text fetch.
	type resolvedWithOptions struct {
		questionID   string
		sortOrder    int
		optionsOrder []string // final display order
		ruleID       string
	}

	resolved := make([]resolvedWithOptions, len(allQs))
	var allOptionIDs []string

	for i, q := range allQs {
		opts, err := s.repo.GetOptionIDs(ctx, q.id)
		if err != nil {
			return nil, fmt.Errorf("sessions: CreateSession: get options for %s: %w", q.id, err)
		}

		optIDs := make([]string, len(opts))
		for j, o := range opts {
			optIDs[j] = o.OptionID
		}

		// AC-7: shuffle_options — use same rng, keyed by question index.
		detail := details[q.id]
		if cfg.ShuffleOptions && !isShortText(detail.Type) {
			rng.Shuffle(len(optIDs), func(a, b int) { optIDs[a], optIDs[b] = optIDs[b], optIDs[a] })
		}

		resolved[i] = resolvedWithOptions{
			questionID:   q.id,
			sortOrder:    i,
			optionsOrder: optIDs,
			ruleID:       q.ruleID,
		}
		allOptionIDs = append(allOptionIDs, optIDs...)
	}

	optTexts, err := s.repo.GetOptionTexts(ctx, allOptionIDs)
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: get option texts: %w", err)
	}

	// Build the createSessionInput for the transactional insert.
	expiresAt := current.Add(time.Duration(cfg.TimeLimitMinutes) * time.Minute)
	rqs := make([]resolvedQuestion, len(resolved))
	for i, r := range resolved {
		var ruleID *string
		if r.ruleID != "" {
			ruleID = &r.ruleID
		}
		rqs[i] = resolvedQuestion{
			QuestionID:   r.questionID,
			SortOrder:    r.sortOrder,
			OptionsOrder: r.optionsOrder,
			RuleID:       ruleID,
		}
	}

	// AC-10: single transaction.
	sessionID, startedAt, returnedExpiresAt, err := s.repo.CreateSession(ctx, createSessionInput{
		ExamID:    examID,
		UserID:    userID,
		Seed:      seed,
		ExpiresAt: expiresAt,
		Questions: rqs,
	})
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: persist: %w", err)
	}

	remainingSecs := returnedExpiresAt.Sub(current).Seconds()

	// Build response questions — AC-9: no is_correct flag, no options for short_text.
	responseQs := make([]SessionQuestionResponse, len(resolved))
	for i, r := range resolved {
		detail := details[r.questionID]
		var opts []SessionOptionResponse
		if !isShortText(detail.Type) {
			opts = make([]SessionOptionResponse, len(r.optionsOrder))
			for j, optID := range r.optionsOrder {
				opts[j] = SessionOptionResponse{ID: optID, Text: optTexts[optID]}
			}
		} else {
			opts = []SessionOptionResponse{}
		}
		responseQs[i] = SessionQuestionResponse{
			ID:        r.questionID,
			SortOrder: r.sortOrder,
			Stem:      detail.Stem,
			Type:      detail.Type,
			Options:   opts,
		}
	}

	return &CreateSessionResponse{
		SessionID:        sessionID,
		ExamID:           examID,
		StartedAt:        startedAt,
		ExpiresAt:        returnedExpiresAt,
		RemainingSeconds: remainingSecs,
		Questions:        responseQs,
	}, nil
}

func (s *service) SaveAnswer(ctx context.Context, sessionID, questionID, userID string, input SaveAnswerInput) (*SaveAnswerResponse, error) {
	// AC-9: reject negative time_spent_seconds.
	if input.TimeSpentSeconds < 0 {
		return nil, ErrNegativeTimeSpent
	}

	// AC-7/AC-10: fetch session and verify ownership.
	sess, err := s.repo.GetSessionForUser(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("sessions: SaveAnswer: get session: %w", err)
	}

	// AC-3: status must be in_progress.
	if sess.Status != "in_progress" {
		return nil, ErrSessionNotActive
	}

	// AC-2: check expiry server-side.
	current := nowFn()
	if !sess.ExpiresAt.After(current) {
		return nil, ErrSessionExpired
	}

	// AC-4: verify questionID belongs to session.
	sqRows, err := s.repo.GetSessionQuestions(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: SaveAnswer: get session questions: %w", err)
	}
	var questionType string
	found := false
	for _, sq := range sqRows {
		if sq.QuestionID == questionID {
			found = true
			questionType = sq.QuestionType
			break
		}
	}
	if !found {
		return nil, ErrQuestionNotInSession
	}

	// Validate answer format per question type (Notes section).
	switch questionType {
	case "short_text", "shorttext":
		if len(input.SelectedOptionIDs) > 0 {
			return nil, ErrInvalidAnswerFormat
		}
	case "single_choice", "single", "true_false", "truefalse":
		if len(input.SelectedOptionIDs) > 1 {
			return nil, ErrInvalidAnswerFormat
		}
	}

	// AC-5 / FR-BB64 AC-4: validate all selected_option_ids belong to the question.
	if len(input.SelectedOptionIDs) > 0 {
		validOpts, err := s.repo.GetValidOptionIDs(ctx, questionID)
		if err != nil {
			return nil, fmt.Errorf("sessions: SaveAnswer: get valid options: %w", err)
		}
		for _, optID := range input.SelectedOptionIDs {
			if _, ok := validOpts[optID]; !ok {
				return nil, ErrInvalidAnswerOption
			}
		}
	}

	// AC-1: upsert the answer row.
	savedAt, err := s.repo.UpsertAnswer(ctx, upsertAnswerInput{
		SessionID:         sessionID,
		QuestionID:        questionID,
		SelectedOptionIDs: input.SelectedOptionIDs,
		TextAnswer:        input.TextAnswer,
		TimeSpentSeconds:  input.TimeSpentSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("sessions: SaveAnswer: upsert: %w", err)
	}

	// AC-6: remaining_seconds clamped to >= 0.
	remaining := sess.ExpiresAt.Sub(nowFn()).Seconds()
	if remaining < 0 {
		remaining = 0
	}

	return &SaveAnswerResponse{
		QuestionID:       questionID,
		SavedAt:          savedAt,
		RemainingSeconds: remaining,
	}, nil
}

func (s *service) GetSessionState(ctx context.Context, sessionID, userID string) (*ResumeSessionResponse, error) {
	// AC-7/AC-10: fetch and verify ownership.
	sess, err := s.repo.GetSessionForUser(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSessionState: get session: %w", err)
	}

	// Fetch session questions (in shuffled sort_order).
	sqRows, err := s.repo.GetSessionQuestions(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSessionState: get questions: %w", err)
	}

	// Collect all question IDs and all option IDs.
	qIDs := make([]string, len(sqRows))
	for i, sq := range sqRows {
		qIDs[i] = sq.QuestionID
	}

	details, err := s.repo.GetQuestionDetails(ctx, qIDs)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSessionState: get question details: %w", err)
	}

	// Collect all option IDs from stored options_order.
	var allOptionIDs []string
	type optOrderEntry struct {
		questionID string
		optIDs     []string
	}
	optOrders := make([]optOrderEntry, len(sqRows))
	for i, sq := range sqRows {
		var optIDs []string
		if len(sq.OptionsOrder) > 0 {
			if err := json.Unmarshal(sq.OptionsOrder, &optIDs); err != nil {
				return nil, fmt.Errorf("sessions: GetSessionState: unmarshal options_order: %w", err)
			}
		}
		optOrders[i] = optOrderEntry{questionID: sq.QuestionID, optIDs: optIDs}
		allOptionIDs = append(allOptionIDs, optIDs...)
	}

	optTexts, err := s.repo.GetOptionTexts(ctx, allOptionIDs)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSessionState: get option texts: %w", err)
	}

	// Build questions slice.
	questions := make([]SessionQuestionResponse, len(sqRows))
	for i, sq := range sqRows {
		detail := details[sq.QuestionID]
		var opts []SessionOptionResponse
		if !isShortText(detail.Type) {
			opts = make([]SessionOptionResponse, len(optOrders[i].optIDs))
			for j, optID := range optOrders[i].optIDs {
				opts[j] = SessionOptionResponse{ID: optID, Text: optTexts[optID]}
			}
		} else {
			opts = []SessionOptionResponse{}
		}
		questions[i] = SessionQuestionResponse{
			ID:        sq.QuestionID,
			SortOrder: sq.SortOrder,
			Stem:      detail.Stem,
			Type:      detail.Type,
			Options:   opts,
		}
	}

	// Fetch all saved answers.
	answerRows, err := s.repo.GetSessionAnswers(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSessionState: get answers: %w", err)
	}

	answers := make(map[string]SavedAnswer, len(answerRows))
	for qID, ar := range answerRows {
		var selectedIDs []string
		if len(ar.SelectedOptionIDs) > 0 {
			if err := json.Unmarshal(ar.SelectedOptionIDs, &selectedIDs); err != nil {
				return nil, fmt.Errorf("sessions: GetSessionState: unmarshal selected_option_ids: %w", err)
			}
		}
		if selectedIDs == nil {
			selectedIDs = []string{}
		}
		answers[qID] = SavedAnswer{
			SelectedOptionIDs: selectedIDs,
			TextAnswer:        ar.TextAnswer,
			TimeSpentSeconds:  ar.TimeSpentSeconds,
			SavedAt:           ar.SavedAt,
		}
	}

	// AC-8: remaining_seconds = 0 for finished sessions (Notes section).
	current := nowFn()
	var remaining float64
	switch sess.Status {
	case "submitted", "auto_submitted":
		remaining = 0
	default:
		remaining = sess.ExpiresAt.Sub(current).Seconds()
		if remaining < 0 {
			remaining = 0
		}
	}

	return &ResumeSessionResponse{
		SessionID:          sess.ID,
		ExamID:             sess.ExamID,
		ExamTitle:          sess.ExamTitle,
		CertificateEnabled: sess.CertificateEnabled,
		Status:             sess.Status,
		StartedAt:          sess.StartedAt,
		ExpiresAt:          sess.ExpiresAt,
		RemainingSeconds:   remaining,
		Questions:          questions,
		Answers:            answers,
		Adaptive:           sess.Adaptive,
	}, nil
}

var validEventTypes = map[string]bool{
	"tab_switch":      true,
	"blur":            true,
	"fullscreen_exit": true,
}

func (s *service) ReportEvent(ctx context.Context, sessionID, userID string, input ReportEventInput) (*ReportEventResponse, error) {
	if !validEventTypes[input.Type] {
		return nil, ErrInvalidEventType
	}

	sess, err := s.repo.GetSessionForUser(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("sessions: ReportEvent: get session: %w", err)
	}

	if sess.Status != "in_progress" {
		return nil, ErrSessionNotActive
	}

	current := nowFn()
	if !sess.ExpiresAt.After(current) {
		return nil, ErrSessionExpired
	}

	// Fetch exam config to determine the on_tab_switch policy.
	cfg, err := s.repo.GetExamConfig(ctx, sess.ExamID)
	if err != nil {
		return nil, fmt.Errorf("sessions: ReportEvent: get exam config: %w", err)
	}

	// AC-10: action_taken is derived from the exam config, not the request body.
	if err := s.repo.InsertTabSwitchEvent(ctx, sessionID, input.Type, cfg.OnTabSwitch); err != nil {
		return nil, fmt.Errorf("sessions: ReportEvent: insert event: %w", err)
	}

	eventCount, err := s.repo.CountTabSwitchEvents(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: ReportEvent: count events: %w", err)
	}

	switch cfg.OnTabSwitch {
	case "submit":
		result, err := s.repo.AutoSubmitSession(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("sessions: ReportEvent: auto submit: %w", err)
		}
		sid := result.SessionID
		status := result.Status
		return &ReportEventResponse{
			Warn:      false,
			SessionID: &sid,
			Status:    &status,
			ScorePct:  result.ScorePct,
			Passed:    result.Passed,
		}, nil
	case "warn":
		return &ReportEventResponse{Warn: true, EventCount: eventCount}, nil
	default: // "log"
		return &ReportEventResponse{Warn: false, EventCount: eventCount}, nil
	}
}

func (s *service) SubmitSession(ctx context.Context, sessionID, userID, tenantID, actorIP string) (*SubmitSessionResponse, error) {
	result, err := s.repo.SubmitSession(ctx, sessionID, userID, tenantID, actorIP)
	if err != nil {
		return nil, fmt.Errorf("sessions: SubmitSession: %w", err)
	}
	if s.emailSvc != nil && result.Passed != nil {
		s.emailSvc.TriggerSessionResult(result.SessionID)
	}
	return result, nil
}

// shufflePoolQuestions shuffles a slice of poolQuestion in-place using the provided rng.
func shufflePoolQuestions(pool []poolQuestion, rng *rand.Rand) {
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
}

// buildSessionResult converts a sessionResultRow into a SessionResultResponse,
// fetching per-section and per-question data as determined by show_answers_mode.
func (s *service) buildSessionResult(ctx context.Context, row *sessionResultRow, alwaysBreakdown bool) (*SessionResultResponse, error) {
	sections, err := s.repo.GetSectionScores(ctx, row.SessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: buildSessionResult: section scores: %w", err)
	}

	resp := &SessionResultResponse{
		SessionID:        row.SessionID,
		ExamID:           row.ExamID,
		ExamTitle:        row.ExamTitle,
		Status:           row.Status,
		ScorePct:         row.ScorePct,
		Passed:           row.Passed,
		TimeTakenSeconds: row.TimeTakenSeconds,
		AttemptNumber:    row.AttemptNumber,
		SubmittedAt:      row.SubmittedAt,
		ShowAnswersMode:  row.ShowAnswersMode,
		PerSectionScores: sections,
	}

	// AC-3: include breakdown when show_answers_mode != 'never', or always for admin (AC-5).
	includeBreakdown := alwaysBreakdown || row.ShowAnswersMode != "never"
	if includeBreakdown {
		bRows, err := s.repo.GetQuestionBreakdown(ctx, row.SessionID, "")
		if err != nil {
			return nil, fmt.Errorf("sessions: buildSessionResult: breakdown: %w", err)
		}

		// Gather question IDs for correct-answer lookup.
		qIDs := make([]string, len(bRows))
		for i, b := range bRows {
			qIDs[i] = b.QuestionID
		}

		// Fetch employee answers.
		answerRows, err := s.repo.GetSessionAnswers(ctx, row.SessionID)
		if err != nil {
			return nil, fmt.Errorf("sessions: buildSessionResult: answers: %w", err)
		}

		// Collect all selected option IDs for text resolution.
		var allOptIDs []string
		for _, ar := range answerRows {
			var ids []string
			if len(ar.SelectedOptionIDs) > 0 {
				if err := json.Unmarshal(ar.SelectedOptionIDs, &ids); err == nil {
					allOptIDs = append(allOptIDs, ids...)
				}
			}
		}

		optTexts, err := s.repo.GetOptionTexts(ctx, allOptIDs)
		if err != nil {
			return nil, fmt.Errorf("sessions: buildSessionResult: option texts: %w", err)
		}

		correctMap, err := s.repo.GetCorrectAnswerTexts(ctx, qIDs, "")
		if err != nil {
			return nil, fmt.Errorf("sessions: buildSessionResult: correct answers: %w", err)
		}

		breakdown := make([]QuestionBreakdownItem, len(bRows))
		for i, b := range bRows {
			// Build employee answer texts.
			var empAnswer []string
			if ar, ok := answerRows[b.QuestionID]; ok {
				if isShortText(b.QuestionType) {
					if ar.TextAnswer != nil {
						empAnswer = []string{*ar.TextAnswer}
					}
				} else {
					var ids []string
					if len(ar.SelectedOptionIDs) > 0 {
						_ = json.Unmarshal(ar.SelectedOptionIDs, &ids)
					}
					empAnswer = make([]string, len(ids))
					for j, id := range ids {
						empAnswer[j] = optTexts[id]
					}
				}
			}
			if empAnswer == nil {
				empAnswer = []string{}
			}

			correct := correctMap[b.QuestionID]
			if correct == nil {
				correct = []string{}
			}

			breakdown[i] = QuestionBreakdownItem{
				QuestionID:     b.QuestionID,
				Stem:           b.Stem,
				EmployeeAnswer: empAnswer,
				CorrectAnswer:  correct,
				PointsEarned:   b.PointsEarned,
				MaxPoints:      b.MaxPoints,
				Explanation:    b.Explanation,
				ManualFeedback: b.ManualFeedback,
			}
		}
		resp.PerQuestionBreakdown = &breakdown
	}

	return resp, nil
}

func (s *service) GetSessionResult(ctx context.Context, sessionID, userID string) (*SessionResultResponse, error) {
	row, err := s.repo.GetSessionResult(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetSessionResult: %w", err)
	}
	// AC-2: reject in_progress sessions.
	if row.Status == "in_progress" {
		return nil, ErrSessionInProgress
	}
	return s.buildSessionResult(ctx, row, false)
}

func (s *service) GetAdminSessionResult(ctx context.Context, sessionID string) (*SessionResultResponse, error) {
	row, err := s.repo.GetAdminSessionResult(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetAdminSessionResult: %w", err)
	}
	// AC-2: reject in_progress sessions.
	if row.Status == "in_progress" {
		return nil, ErrSessionInProgress
	}
	// AC-5: always show breakdown for admin.
	return s.buildSessionResult(ctx, row, true)
}

func (s *service) GetExamHistory(ctx context.Context, examID, userID string, page, perPage int) (*ExamHistoryResponse, error) {
	title, err := s.repo.GetExamTitleByID(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetExamHistory: %w", err)
	}

	rows, total, err := s.repo.GetExamHistory(ctx, examID, userID, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetExamHistory: %w", err)
	}

	sessions := make([]HistorySession, len(rows))
	for i, r := range rows {
		sessions[i] = HistorySession(r)
	}

	return &ExamHistoryResponse{
		ExamID:    examID,
		ExamTitle: title,
		Sessions:  sessions,
		Meta: ExamHistoryMeta{
			Page:    page,
			PerPage: perPage,
			Total:   total,
		},
	}, nil
}

// GetMyResults returns paginated session history across all exams for the calling user (FR-BB46).
func (s *service) GetMyResults(ctx context.Context, userID string, page, perPage int, sort, dir string) (*MyResultsResponse, error) {
	items, total, err := s.repo.GetMyResults(ctx, userID, page, perPage, sort, dir)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetMyResults: %w", err)
	}
	return &MyResultsResponse{
		Sessions: items,
		Meta: MyResultsMeta{
			Page:    page,
			PerPage: perPage,
			Total:   total,
		},
	}, nil
}

// ListGradingQueue returns sessions pending manual grading (FR-BB42 AC-1/AC-2/AC-3).
func (s *service) ListGradingQueue(ctx context.Context, examID *string, dateFrom, dateTo *time.Time, page, perPage int) (*GradingQueueResponse, error) {
	items, total, err := s.repo.ListGradingQueue(ctx, examID, dateFrom, dateTo, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("sessions: ListGradingQueue: %w", err)
	}
	return &GradingQueueResponse{
		Items: items,
		Meta: GradingQueueMeta{
			Page:    page,
			PerPage: perPage,
			Total:   total,
		},
	}, nil
}

// GetGradingDetail returns session header and short-text questions (FR-BB42 AC-4).
func (s *service) GetGradingDetail(ctx context.Context, sessionID string) (*GradingDetailResponse, error) {
	detail, err := s.repo.GetGradingDetail(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GetGradingDetail: %w", err)
	}
	return detail, nil
}

// SelectNextAdaptiveQuestion picks the next question for an adaptive session (FR-BB72 AC-3/AC-4/AC-5).
func (s *service) SelectNextAdaptiveQuestion(ctx context.Context, sessionID, userID string) (*NextQuestionResponse, error) {
	// Verify ownership and get adaptive flag.
	adaptive, err := s.repo.GetSessionAdaptive(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: %w", err)
	}
	if !adaptive {
		return nil, ErrNotAdaptive
	}

	state, err := s.repo.GetAdaptiveState(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: get state: %w", err)
	}

	// AC-5: if current_question_id is set, return same question (idempotent).
	if state.CurrentQuestionID != nil {
		// Fetch that question's details.
		qID := *state.CurrentQuestionID
		details, err := s.repo.GetQuestionDetails(ctx, []string{qID})
		if err != nil {
			return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: get current q details: %w", err)
		}
		optIDs, err := s.repo.GetOptionIDs(ctx, qID)
		if err != nil {
			return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: get current q options: %w", err)
		}
		allOptIDs := make([]string, len(optIDs))
		for i, o := range optIDs {
			allOptIDs[i] = o.OptionID
		}
		optTexts, err := s.repo.GetOptionTexts(ctx, allOptIDs)
		if err != nil {
			return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: get option texts: %w", err)
		}
		answered, err := s.repo.CountAnsweredForSession(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: count answered: %w", err)
		}
		detail := details[qID]
		opts := buildOptions(detail.Type, allOptIDs, optTexts)
		return &NextQuestionResponse{
			Question: &SessionQuestionResponse{
				ID:        qID,
				SortOrder: answered,
				Stem:      detail.Stem,
				Type:      detail.Type,
				Options:   opts,
			},
			QuestionsAnswered: answered,
			Done:              false,
		}, nil
	}

	// Count already answered — if all served IDs are answered, check done condition.
	answered, err := s.repo.CountAnsweredForSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: count answered: %w", err)
	}

	// Compute next difficulty using the adaptive algorithm.
	nextDiff := exams.NextDifficulty(
		exams.Difficulty(state.CurrentDifficulty),
		state.RecentResults,
	)

	// Fetch a question at the target difficulty, excluding already-served IDs.
	q, err := s.repo.GetRandomByDifficultyExcluding(ctx, "", string(nextDiff), state.ServedQuestionIDs)
	if errors.Is(err, ErrNoQuestionsAvailable) {
		// Fall back to any difficulty.
		q, err = s.repo.GetRandomByDifficultyExcluding(ctx, "", string(exams.AnyDifficulty), state.ServedQuestionIDs)
	}
	if errors.Is(err, ErrNoQuestionsAvailable) {
		// All questions have been served — session is done.
		return &NextQuestionResponse{
			QuestionsAnswered: answered,
			Done:              true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: pick question: %w", err)
	}

	// Update state: record selected question, update difficulty, add to served list.
	state.CurrentDifficulty = q.Difficulty
	qIDStr := q.ID
	state.CurrentQuestionID = &qIDStr
	state.ServedQuestionIDs = append(state.ServedQuestionIDs, q.ID)
	if err := s.repo.UpdateAdaptiveState(ctx, sessionID, state); err != nil {
		return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: update state: %w", err)
	}

	// Build response question.
	optIDs, err := s.repo.GetOptionIDs(ctx, q.ID)
	if err != nil {
		return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: get option ids: %w", err)
	}
	allOptIDs := make([]string, len(optIDs))
	for i, o := range optIDs {
		allOptIDs[i] = o.OptionID
	}
	optTexts, err := s.repo.GetOptionTexts(ctx, allOptIDs)
	if err != nil {
		return nil, fmt.Errorf("sessions: SelectNextAdaptiveQuestion: get option texts: %w", err)
	}
	opts := buildOptions(q.Type, allOptIDs, optTexts)
	return &NextQuestionResponse{
		Question: &SessionQuestionResponse{
			ID:        q.ID,
			SortOrder: answered,
			Stem:      q.Stem,
			Type:      q.Type,
			Options:   opts,
		},
		QuestionsAnswered: answered,
		Done:              false,
	}, nil
}

// isShortText returns true for both DB type "shorttext" and canonical "short_text".
func isShortText(qType string) bool {
	return qType == "short_text" || qType == "shorttext"
}

// buildOptions constructs the option list for a question response, omitting options for short_text.
func buildOptions(qType string, optIDs []string, optTexts map[string]string) []SessionOptionResponse {
	if isShortText(qType) {
		return []SessionOptionResponse{}
	}
	opts := make([]SessionOptionResponse, len(optIDs))
	for i, id := range optIDs {
		opts[i] = SessionOptionResponse{ID: id, Text: optTexts[id]}
	}
	return opts
}

// RecordAdaptiveAnswer updates adaptive_state after an answer is saved (FR-BB72 AC-6).
func (s *service) RecordAdaptiveAnswer(ctx context.Context, sessionID, questionID string, correct bool) error {
	state, err := s.repo.GetAdaptiveState(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("sessions: RecordAdaptiveAnswer: get state: %w", err)
	}
	state.RecentResults = append(state.RecentResults, correct)
	state.CurrentQuestionID = nil // cleared; next call to next-question picks new q
	if err := s.repo.UpdateAdaptiveState(ctx, sessionID, state); err != nil {
		return fmt.Errorf("sessions: RecordAdaptiveAnswer: update state: %w", err)
	}
	return nil
}

// GradeAnswer validates and stores a manual grade for a short-text answer (FR-BB42).
func (s *service) GradeAnswer(ctx context.Context, sessionID, questionID, graderID, tenantID, actorIP string, req GradeAnswerRequest) (*GradeAnswerResponse, error) {
	// AC-5: validate score range.
	if req.ScorePct < 0 || req.ScorePct > 100 {
		return nil, ErrInvalidScore
	}

	result, err := s.repo.GradeAnswer(ctx, sessionID, questionID, graderID, tenantID, actorIP, req.ScorePct, req.Feedback)
	if err != nil {
		return nil, fmt.Errorf("sessions: GradeAnswer: %w", err)
	}

	resp := &GradeAnswerResponse{
		QuestionID:    questionID,
		GradingStatus: "graded",
		ScorePct:      req.ScorePct,
		SessionStatus: result.sessionStatus,
		AllGraded:     result.allGraded,
		FinalScorePct: result.finalScorePct,
		Passed:        result.passed,
	}
	if s.emailSvc != nil && resp.AllGraded {
		s.emailSvc.TriggerSessionResult(sessionID)
	}
	return resp, nil
}
