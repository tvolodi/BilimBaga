package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
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
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
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

	// AC-2: exam must be active.
	cfg, err := s.repo.GetExamConfig(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("sessions: CreateSession: get config: %w", err)
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
		}
	}

	// Collect type info alongside IDs so we can build poolQuestion slices for shuffle.
	type qWithType struct {
		id  string
		typ string
	}
	allQs := make([]qWithType, len(resolvedIDs))
	for i, id := range resolvedIDs {
		allQs[i] = qWithType{id: id}
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
		if cfg.ShuffleOptions && detail.Type != "short_text" {
			rng.Shuffle(len(optIDs), func(a, b int) { optIDs[a], optIDs[b] = optIDs[b], optIDs[a] })
		}

		resolved[i] = resolvedWithOptions{
			questionID:   q.id,
			sortOrder:    i,
			optionsOrder: optIDs,
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
		rqs[i] = resolvedQuestion{
			QuestionID:   r.questionID,
			SortOrder:    r.sortOrder,
			OptionsOrder: r.optionsOrder,
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
		if detail.Type != "short_text" {
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
	case "short_text":
		if len(input.SelectedOptionIDs) > 0 {
			return nil, ErrInvalidAnswerFormat
		}
	case "single_choice", "true_false":
		if len(input.SelectedOptionIDs) > 1 {
			return nil, ErrInvalidAnswerFormat
		}
	}

	// AC-5: validate all selected_option_ids belong to the question.
	if len(input.SelectedOptionIDs) > 0 {
		validOpts, err := s.repo.GetValidOptionIDs(ctx, questionID)
		if err != nil {
			return nil, fmt.Errorf("sessions: SaveAnswer: get valid options: %w", err)
		}
		for _, optID := range input.SelectedOptionIDs {
			if _, ok := validOpts[optID]; !ok {
				return nil, ErrInvalidOption
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
		if detail.Type != "short_text" {
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
		SessionID:        sess.ID,
		ExamID:           sess.ExamID,
		Status:           sess.Status,
		StartedAt:        sess.StartedAt,
		ExpiresAt:        sess.ExpiresAt,
		RemainingSeconds: remaining,
		Questions:        questions,
		Answers:          answers,
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
	return result, nil
}

// shufflePoolQuestions shuffles a slice of poolQuestion in-place using the provided rng.
func shufflePoolQuestions(pool []poolQuestion, rng *rand.Rand) {
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
}

// parseTagIDs is used by tests to verify tag parsing behaviour.
func parseTagIDs(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var ids []string
	return ids, json.Unmarshal(raw, &ids)
}
