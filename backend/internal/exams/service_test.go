package exams

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
	exams       map[string]*Exam
	sections    map[string]*ExamSection
	rules       map[string]*ExamQuestionRule
	manual      map[string][]ManualQuestionInput
	assignments map[string]*ExamAssignment

	createFn                    func(ctx context.Context, e *Exam) error
	getByIDFn                   func(ctx context.Context, id string) (*Exam, error)
	listFilteredFn              func(ctx context.Context, f ExamFilter) ([]*ExamListItem, int, error)
	updateFn                    func(ctx context.Context, id string, input UpdateExamInput) (*Exam, error)
	updateStatusFn              func(ctx context.Context, id, status string) error
	deleteByIDFn                func(ctx context.Context, id string) error
	getWithDetailsFn            func(ctx context.Context, id string) (*ExamDetail, error)
	createSectionFn             func(ctx context.Context, examID string, input SectionInput) (*ExamSection, error)
	updateSectionFn             func(ctx context.Context, id string, input SectionInput) (*ExamSection, error)
	deleteSectionFn             func(ctx context.Context, id string) error
	createRuleFn                func(ctx context.Context, examID string, input QuestionRuleInput) (*ExamQuestionRule, error)
	updateRuleFn                func(ctx context.Context, id string, input QuestionRuleInput) (*ExamQuestionRule, error)
	deleteRuleFn                func(ctx context.Context, id string) error
	setManualQsFn               func(ctx context.Context, ruleID string, qs []ManualQuestionInput) error
	getRuleByIDFn               func(ctx context.Context, id string) (*ExamQuestionRule, error)
	listRulesForExamFn          func(ctx context.Context, examID string) ([]*ExamQuestionRule, error)
	countAvailableForRuleFn     func(ctx context.Context, rule *ExamQuestionRule) (int, error)
	createAssignmentFn          func(ctx context.Context, a *ExamAssignment) error
	getAssignmentByIDFn         func(ctx context.Context, id string) (*ExamAssignment, error)
	deleteAssignmentFn          func(ctx context.Context, id string) error
	listAssignmentsWithStatsFn  func(ctx context.Context, examID string) ([]*AssignmentDetail, error)
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		exams:       make(map[string]*Exam),
		sections:    make(map[string]*ExamSection),
		rules:       make(map[string]*ExamQuestionRule),
		manual:      make(map[string][]ManualQuestionInput),
		assignments: make(map[string]*ExamAssignment),
	}
}

func (m *mockRepo) Create(ctx context.Context, e *Exam) error {
	if m.createFn != nil {
		return m.createFn(ctx, e)
	}
	e.ID = "exam-1"
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	m.exams[e.ID] = e
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*Exam, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	e, ok := m.exams[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (m *mockRepo) ListFiltered(ctx context.Context, f ExamFilter) ([]*ExamListItem, int, error) {
	if m.listFilteredFn != nil {
		return m.listFilteredFn(ctx, f)
	}
	return []*ExamListItem{}, 0, nil
}

func (m *mockRepo) Update(ctx context.Context, id string, input UpdateExamInput) (*Exam, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, input)
	}
	e, ok := m.exams[id]
	if !ok {
		return nil, ErrNotFound
	}
	e.Title = input.Title
	e.Description = input.Description
	e.TimeLimitMinutes = input.TimeLimitMinutes
	e.PassingScorePct = input.PassingScorePct
	return e, nil
}

func (m *mockRepo) UpdateStatus(ctx context.Context, id, status string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	e, ok := m.exams[id]
	if !ok {
		return ErrNotFound
	}
	e.Status = status
	return nil
}

func (m *mockRepo) DeleteByID(ctx context.Context, id string) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, id)
	}
	if _, ok := m.exams[id]; !ok {
		return ErrNotFound
	}
	delete(m.exams, id)
	return nil
}

func (m *mockRepo) GetWithDetails(ctx context.Context, id string) (*ExamDetail, error) {
	if m.getWithDetailsFn != nil {
		return m.getWithDetailsFn(ctx, id)
	}
	e, ok := m.exams[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &ExamDetail{
		ID:               e.ID,
		Title:            e.Title,
		Status:           e.Status,
		TimeLimitMinutes: e.TimeLimitMinutes,
		PassingScorePct:  e.PassingScorePct,
		MaxAttempts:      e.MaxAttempts,
		ShowAnswers:      e.ShowAnswers,
		OnTabSwitch:      e.OnTabSwitch,
		CreatedBy:        e.CreatedBy,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
		Sections:         []SectionDetail{},
		Rules:            []QuestionRuleDetail{},
	}, nil
}

func (m *mockRepo) CreateSection(ctx context.Context, examID string, input SectionInput) (*ExamSection, error) {
	if m.createSectionFn != nil {
		return m.createSectionFn(ctx, examID, input)
	}
	s := &ExamSection{ID: "sec-1", ExamID: examID, Title: input.Title, SortOrder: input.SortOrder}
	m.sections[s.ID] = s
	return s, nil
}

func (m *mockRepo) UpdateSection(ctx context.Context, id string, input SectionInput) (*ExamSection, error) {
	if m.updateSectionFn != nil {
		return m.updateSectionFn(ctx, id, input)
	}
	s, ok := m.sections[id]
	if !ok {
		return nil, ErrNotFound
	}
	s.Title = input.Title
	s.SortOrder = input.SortOrder
	return s, nil
}

func (m *mockRepo) DeleteSection(ctx context.Context, id string) error {
	if m.deleteSectionFn != nil {
		return m.deleteSectionFn(ctx, id)
	}
	if _, ok := m.sections[id]; !ok {
		return ErrNotFound
	}
	delete(m.sections, id)
	return nil
}

func (m *mockRepo) CreateRule(ctx context.Context, examID string, input QuestionRuleInput) (*ExamQuestionRule, error) {
	if m.createRuleFn != nil {
		return m.createRuleFn(ctx, examID, input)
	}
	r := &ExamQuestionRule{ID: "rule-1", ExamID: examID, Mode: input.Mode, Count: input.Count, SortOrder: input.SortOrder}
	m.rules[r.ID] = r
	return r, nil
}

func (m *mockRepo) UpdateRule(ctx context.Context, id string, input QuestionRuleInput) (*ExamQuestionRule, error) {
	if m.updateRuleFn != nil {
		return m.updateRuleFn(ctx, id, input)
	}
	r, ok := m.rules[id]
	if !ok {
		return nil, ErrNotFound
	}
	r.Mode = input.Mode
	r.Count = input.Count
	return r, nil
}

func (m *mockRepo) DeleteRule(ctx context.Context, id string) error {
	if m.deleteRuleFn != nil {
		return m.deleteRuleFn(ctx, id)
	}
	if _, ok := m.rules[id]; !ok {
		return ErrNotFound
	}
	delete(m.rules, id)
	return nil
}

func (m *mockRepo) SetManualQuestions(ctx context.Context, ruleID string, qs []ManualQuestionInput) error {
	if m.setManualQsFn != nil {
		return m.setManualQsFn(ctx, ruleID, qs)
	}
	m.manual[ruleID] = qs
	return nil
}

func (m *mockRepo) GetRuleByID(ctx context.Context, id string) (*ExamQuestionRule, error) {
	if m.getRuleByIDFn != nil {
		return m.getRuleByIDFn(ctx, id)
	}
	r, ok := m.rules[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (m *mockRepo) ListRulesForExam(ctx context.Context, examID string) ([]*ExamQuestionRule, error) {
	if m.listRulesForExamFn != nil {
		return m.listRulesForExamFn(ctx, examID)
	}
	var out []*ExamQuestionRule
	for _, r := range m.rules {
		if r.ExamID == examID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *mockRepo) CountAvailableForRule(ctx context.Context, rule *ExamQuestionRule) (int, error) {
	if m.countAvailableForRuleFn != nil {
		return m.countAvailableForRuleFn(ctx, rule)
	}
	return 100, nil
}

func (m *mockRepo) CreateAssignment(ctx context.Context, a *ExamAssignment) error {
	if m.createAssignmentFn != nil {
		return m.createAssignmentFn(ctx, a)
	}
	a.ID = "assign-1"
	a.AssignedAt = time.Now()
	m.assignments[a.ID] = a
	return nil
}

func (m *mockRepo) GetAssignmentByID(ctx context.Context, id string) (*ExamAssignment, error) {
	if m.getAssignmentByIDFn != nil {
		return m.getAssignmentByIDFn(ctx, id)
	}
	a, ok := m.assignments[id]
	if !ok {
		return nil, ErrAssignmentNotFound
	}
	cp := *a
	return &cp, nil
}

func (m *mockRepo) DeleteAssignment(ctx context.Context, id string) error {
	if m.deleteAssignmentFn != nil {
		return m.deleteAssignmentFn(ctx, id)
	}
	if _, ok := m.assignments[id]; !ok {
		return ErrAssignmentNotFound
	}
	delete(m.assignments, id)
	return nil
}

func (m *mockRepo) ListAssignmentsWithStats(ctx context.Context, examID string) ([]*AssignmentDetail, error) {
	if m.listAssignmentsWithStatsFn != nil {
		return m.listAssignmentsWithStatsFn(ctx, examID)
	}
	var out []*AssignmentDetail
	for _, a := range m.assignments {
		if a.ExamID == examID {
			out = append(out, &AssignmentDetail{
				ID:           a.ID,
				AssigneeType: a.AssigneeType,
				AssigneeID:   a.AssigneeID,
				Deadline:     a.Deadline,
				AssignedAt:   a.AssignedAt,
				Stats:        AssignmentStats{TotalUsers: 1},
			})
		}
	}
	if out == nil {
		out = []*AssignmentDetail{}
	}
	return out, nil
}

// ── Helper ───────────────────────────────────────────────────────────────────

func seedExam(repo *mockRepo, id, status string) {
	repo.exams[id] = &Exam{
		ID:               id,
		Title:            "Test Exam",
		Status:           status,
		TimeLimitMinutes: 30,
		PassingScorePct:  70,
		MaxAttempts:      1,
		ShowAnswers:      "never",
		OnTabSwitch:      "log",
		CreatedBy:        "user-1",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// ── CreateExam ───────────────────────────────────────────────────────────────

func TestCreateExam_SetsStatusDraft(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	e, err := svc.CreateExam(context.Background(), CreateExamInput{
		Title:            "My Exam",
		TimeLimitMinutes: 60,
		PassingScorePct:  80,
		MaxAttempts:      2,
		ShowAnswers:      "never",
		OnTabSwitch:      "log",
		CreatedBy:        "user-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "draft", e.Status)
	assert.Equal(t, "My Exam", e.Title)
	assert.Equal(t, "user-1", e.CreatedBy)
}

func TestCreateExam_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.createFn = func(_ context.Context, _ *Exam) error { return errors.New("db error") }
	svc := NewService(repo)
	_, err := svc.CreateExam(context.Background(), CreateExamInput{Title: "X"})
	require.Error(t, err)
}

// ── GetExam ──────────────────────────────────────────────────────────────────

func TestGetExam_ReturnsDetail(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	detail, err := svc.GetExam(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Equal(t, "exam-1", detail.ID)
	assert.Equal(t, "Test Exam", detail.Title)
}

func TestGetExam_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.GetExam(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── ListExams ────────────────────────────────────────────────────────────────

func TestListExams_ReturnsItems(t *testing.T) {
	repo := newMockRepo()
	repo.listFilteredFn = func(_ context.Context, _ ExamFilter) ([]*ExamListItem, int, error) {
		return []*ExamListItem{{ID: "e1", Title: "A"}}, 1, nil
	}
	svc := NewService(repo)
	items, total, err := svc.ListExams(context.Background(), ExamFilter{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, "e1", items[0].ID)
}

// ── UpdateExam ───────────────────────────────────────────────────────────────

func TestUpdateExam_UpdatesDraftExam(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	e, err := svc.UpdateExam(context.Background(), "exam-1", UpdateExamInput{
		Title:            "Updated",
		TimeLimitMinutes: 45,
		PassingScorePct:  75,
		MaxAttempts:      1,
		ShowAnswers:      "never",
		OnTabSwitch:      "log",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", e.Title)
}

func TestUpdateExam_RejectsArchivedExam(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "archived")
	svc := NewService(repo)
	_, err := svc.UpdateExam(context.Background(), "exam-1", UpdateExamInput{Title: "X"})
	assert.ErrorIs(t, err, ErrNotDraft)
}

func TestUpdateExam_RejectsActiveExam(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	_, err := svc.UpdateExam(context.Background(), "exam-1", UpdateExamInput{Title: "X"})
	assert.ErrorIs(t, err, ErrNotDraft)
}

func TestUpdateExam_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.UpdateExam(context.Background(), "missing", UpdateExamInput{Title: "X"})
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── TransitionStatus ─────────────────────────────────────────────────────────

func TestTransitionStatus_DraftToActive(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	e, err := svc.TransitionStatus(context.Background(), "exam-1", "active")
	require.NoError(t, err)
	assert.Equal(t, "active", e.Status)
}

func TestTransitionStatus_ActiveToArchived(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	e, err := svc.TransitionStatus(context.Background(), "exam-1", "archived")
	require.NoError(t, err)
	assert.Equal(t, "archived", e.Status)
}

func TestTransitionStatus_SkipStateRejected(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	_, err := svc.TransitionStatus(context.Background(), "exam-1", "archived")
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestTransitionStatus_CannotGoBackward(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	_, err := svc.TransitionStatus(context.Background(), "exam-1", "draft")
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestTransitionStatus_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.TransitionStatus(context.Background(), "missing", "active")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── DeleteExam ───────────────────────────────────────────────────────────────

func TestDeleteExam_DeletesDraftExam(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	err := svc.DeleteExam(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Empty(t, repo.exams)
}

func TestDeleteExam_RejectsActiveExam(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	err := svc.DeleteExam(context.Background(), "exam-1")
	assert.ErrorIs(t, err, ErrNotDraft)
}

func TestDeleteExam_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	err := svc.DeleteExam(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── Sections ─────────────────────────────────────────────────────────────────

func TestCreateSection_Success(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	s, err := svc.CreateSection(context.Background(), "exam-1", SectionInput{Title: strPtr("Part A"), SortOrder: 0})
	require.NoError(t, err)
	assert.Equal(t, "exam-1", s.ExamID)
	assert.Equal(t, "Part A", *s.Title)
}

func TestCreateSection_ExamNotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.CreateSection(context.Background(), "missing", SectionInput{})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateSection_NotFound(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	_, err := svc.UpdateSection(context.Background(), "exam-1", "missing-sec", SectionInput{SortOrder: 1})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteSection_Success(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	repo.sections["sec-1"] = &ExamSection{ID: "sec-1", ExamID: "exam-1"}
	svc := NewService(repo)
	err := svc.DeleteSection(context.Background(), "exam-1", "sec-1")
	require.NoError(t, err)
	assert.Empty(t, repo.sections)
}

// ── Rules ────────────────────────────────────────────────────────────────────

func TestCreateRule_Success(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	rule, err := svc.CreateRule(context.Background(), "exam-1", QuestionRuleInput{
		Mode:      "random",
		Count:     5,
		SortOrder: 0,
		TagIDs:    []string{},
	})
	require.NoError(t, err)
	assert.Equal(t, "random", rule.Mode)
	assert.Equal(t, 5, rule.Count)
}

func TestCreateRule_ExamNotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.CreateRule(context.Background(), "missing", QuestionRuleInput{Mode: "random", Count: 1})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteRule_Success(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	repo.rules["rule-1"] = &ExamQuestionRule{ID: "rule-1", ExamID: "exam-1", Mode: "manual", Count: 3}
	svc := NewService(repo)
	err := svc.DeleteRule(context.Background(), "exam-1", "rule-1")
	require.NoError(t, err)
	assert.Empty(t, repo.rules)
}

// ── SetManualQuestions ───────────────────────────────────────────────────────

func TestSetManualQuestions_Stores(t *testing.T) {
	repo := newMockRepo()
	repo.rules["rule-1"] = &ExamQuestionRule{ID: "rule-1", ExamID: "exam-1", Mode: "manual", Count: 2}
	svc := NewService(repo)
	qs := []ManualQuestionInput{
		{QuestionID: "q-1", SortOrder: 0},
		{QuestionID: "q-2", SortOrder: 1},
	}
	err := svc.SetManualQuestions(context.Background(), "rule-1", qs)
	require.NoError(t, err)
	assert.Len(t, repo.manual["rule-1"], 2)
}

func TestSetManualQuestions_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.setManualQsFn = func(_ context.Context, _ string, _ []ManualQuestionInput) error {
		return errors.New("db error")
	}
	svc := NewService(repo)
	err := svc.SetManualQuestions(context.Background(), "rule-1", nil)
	require.Error(t, err)
}

func TestSetManualQuestions_RandomModeReturnsConflict(t *testing.T) {
	repo := newMockRepo()
	repo.rules["rule-1"] = &ExamQuestionRule{ID: "rule-1", ExamID: "exam-1", Mode: "random", Count: 5}
	svc := NewService(repo)
	err := svc.SetManualQuestions(context.Background(), "rule-1", nil)
	assert.ErrorIs(t, err, ErrRulesModeConflict)
}

func TestSetManualQuestions_RuleNotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	err := svc.SetManualQuestions(context.Background(), "missing", nil)
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── Publish ───────────────────────────────────────────────────────────────────

func TestPublish_DraftWithSatisfiedRules_Succeeds(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	repo.rules["rule-1"] = &ExamQuestionRule{ID: "rule-1", ExamID: "exam-1", Mode: "random", Count: 5}
	repo.countAvailableForRuleFn = func(_ context.Context, _ *ExamQuestionRule) (int, error) {
		return 10, nil
	}
	svc := NewService(repo)
	e, err := svc.Publish(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Equal(t, "active", e.Status)
}

func TestPublish_UnsatisfiedRule_Returns422Error(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	repo.rules["rule-1"] = &ExamQuestionRule{ID: "rule-1", ExamID: "exam-1", Mode: "random", Count: 10}
	repo.countAvailableForRuleFn = func(_ context.Context, _ *ExamQuestionRule) (int, error) {
		return 3, nil
	}
	svc := NewService(repo)
	_, err := svc.Publish(context.Background(), "exam-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRulesUnsatisfied)
	var pve *PublishValidationError
	require.ErrorAs(t, err, &pve)
	require.Len(t, pve.Details, 1)
	assert.Equal(t, "rule-1", pve.Details[0].RuleID)
	assert.Equal(t, 10, pve.Details[0].Required)
	assert.Equal(t, 3, pve.Details[0].Available)
}

func TestPublish_ManualRulesSkippedInValidation(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	repo.rules["rule-1"] = &ExamQuestionRule{ID: "rule-1", ExamID: "exam-1", Mode: "manual", Count: 10}
	repo.countAvailableForRuleFn = func(_ context.Context, _ *ExamQuestionRule) (int, error) {
		return 0, nil // would fail if called
	}
	svc := NewService(repo)
	e, err := svc.Publish(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Equal(t, "active", e.Status)
}

func TestPublish_NonDraftExam_ReturnsErrNotDraft(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	_, err := svc.Publish(context.Background(), "exam-1")
	assert.ErrorIs(t, err, ErrNotDraft)
}

func TestPublish_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.Publish(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── Archive ───────────────────────────────────────────────────────────────────

func TestArchive_DraftExam_Succeeds(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	e, err := svc.Archive(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Equal(t, "archived", e.Status)
}

func TestArchive_ActiveExam_Succeeds(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	e, err := svc.Archive(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Equal(t, "archived", e.Status)
}

func TestArchive_AlreadyArchived_ReturnsInvalidTransition(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "archived")
	svc := NewService(repo)
	_, err := svc.Archive(context.Background(), "exam-1")
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestArchive_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.Archive(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ── CreateAssignment ──────────────────────────────────────────────────────────

func TestCreateAssignment_ExamNotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "missing",
		AssigneeType: "all",
		AssignedBy:   "admin-1",
		CallerRole:   "super_admin",
	})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCreateAssignment_ExamNotActive_Returns422(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "draft")
	svc := NewService(repo)
	_, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "all",
		AssignedBy:   "admin-1",
		CallerRole:   "super_admin",
	})
	assert.ErrorIs(t, err, ErrNotActive)
}

func TestCreateAssignment_Success_User(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	uid := "user-99"
	svc := NewService(repo)
	a, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "user",
		AssigneeID:   &uid,
		AssignedBy:   "admin-1",
		CallerRole:   "super_admin",
	})
	require.NoError(t, err)
	assert.Equal(t, "exam-1", a.ExamID)
	assert.Equal(t, "user", a.AssigneeType)
	assert.Equal(t, &uid, a.AssigneeID)
}

func TestCreateAssignment_Success_All(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	a, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "all",
		AssignedBy:   "admin-1",
		CallerRole:   "super_admin",
	})
	require.NoError(t, err)
	assert.Equal(t, "all", a.AssigneeType)
}

func TestCreateAssignment_DuplicateReturnsConflict(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	repo.createAssignmentFn = func(_ context.Context, _ *ExamAssignment) error {
		return ErrAssignmentExists
	}
	svc := NewService(repo)
	_, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "all",
		AssignedBy:   "admin-1",
		CallerRole:   "super_admin",
	})
	assert.ErrorIs(t, err, ErrAssignmentExists)
}

func TestCreateAssignment_DepartmentAdmin_CannotAssignAll(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	_, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "all",
		AssignedBy:   "dept-admin-1",
		CallerRole:   "department_admin",
		CallerDeptID: "dept-1",
	})
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestCreateAssignment_DepartmentAdmin_CannotAssignOtherDept(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	otherDept := "dept-99"
	_, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "department",
		AssigneeID:   &otherDept,
		AssignedBy:   "dept-admin-1",
		CallerRole:   "department_admin",
		CallerDeptID: "dept-1",
	})
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestCreateAssignment_DepartmentAdmin_CanAssignOwnDept(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	ownDept := "dept-1"
	a, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "department",
		AssigneeID:   &ownDept,
		AssignedBy:   "dept-admin-1",
		CallerRole:   "department_admin",
		CallerDeptID: "dept-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "department", a.AssigneeType)
}

func TestCreateAssignment_DeadlineInPastReturnsError(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	past := time.Now().Add(-time.Hour)
	_, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "all",
		Deadline:     &past,
		AssignedBy:   "admin-1",
		CallerRole:   "super_admin",
	})
	assert.ErrorIs(t, err, ErrDeadlineInPast)
}

func TestCreateAssignment_FutureDeadlineSucceeds(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	future := time.Now().Add(24 * time.Hour)
	a, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		ExamID:       "exam-1",
		AssigneeType: "all",
		Deadline:     &future,
		AssignedBy:   "admin-1",
		CallerRole:   "super_admin",
	})
	require.NoError(t, err)
	require.NotNil(t, a.Deadline)
}

// ── DeleteAssignment ──────────────────────────────────────────────────────────

func TestDeleteAssignment_Success(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	repo.assignments["assign-1"] = &ExamAssignment{
		ID: "assign-1", ExamID: "exam-1", AssigneeType: "all", AssignedBy: "admin-1",
	}
	svc := NewService(repo)
	err := svc.DeleteAssignment(context.Background(), "exam-1", "assign-1", "super_admin", "")
	require.NoError(t, err)
	assert.Empty(t, repo.assignments)
}

func TestDeleteAssignment_ExamNotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	err := svc.DeleteAssignment(context.Background(), "missing", "assign-1", "super_admin", "")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteAssignment_AssignmentNotFound(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	err := svc.DeleteAssignment(context.Background(), "exam-1", "missing-assign", "super_admin", "")
	assert.ErrorIs(t, err, ErrAssignmentNotFound)
}

func TestDeleteAssignment_WrongExamID_ReturnsNotFound(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	seedExam(repo, "exam-2", "active")
	repo.assignments["assign-1"] = &ExamAssignment{
		ID: "assign-1", ExamID: "exam-2", AssigneeType: "all", AssignedBy: "admin-1",
	}
	svc := NewService(repo)
	err := svc.DeleteAssignment(context.Background(), "exam-1", "assign-1", "super_admin", "")
	assert.ErrorIs(t, err, ErrAssignmentNotFound)
}

func TestDeleteAssignment_DepartmentAdmin_CannotRemoveAllAssignment(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	repo.assignments["assign-1"] = &ExamAssignment{
		ID: "assign-1", ExamID: "exam-1", AssigneeType: "all", AssignedBy: "super-1",
	}
	svc := NewService(repo)
	err := svc.DeleteAssignment(context.Background(), "exam-1", "assign-1", "department_admin", "dept-1")
	assert.ErrorIs(t, err, ErrForbidden)
}

// ── ListAssignments ───────────────────────────────────────────────────────────

func TestListAssignments_ExamNotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.ListAssignments(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListAssignments_ReturnsEmpty(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	svc := NewService(repo)
	details, err := svc.ListAssignments(context.Background(), "exam-1")
	require.NoError(t, err)
	assert.Empty(t, details)
}

func TestListAssignments_ReturnsAssignments(t *testing.T) {
	repo := newMockRepo()
	seedExam(repo, "exam-1", "active")
	repo.assignments["assign-1"] = &ExamAssignment{
		ID: "assign-1", ExamID: "exam-1", AssigneeType: "all", AssignedBy: "admin-1",
	}
	svc := NewService(repo)
	details, err := svc.ListAssignments(context.Background(), "exam-1")
	require.NoError(t, err)
	require.Len(t, details, 1)
	assert.Equal(t, "all", details[0].AssigneeType)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }
