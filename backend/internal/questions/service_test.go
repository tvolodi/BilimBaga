package questions

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// --- Mock Repository ---

type mockRepository struct {
	questions        map[string]*Question
	createFn         func(ctx context.Context, q *Question) error
	createVerFn      func(ctx context.Context, newQ *Question, previousID string) error
	createOptFn      func(ctx context.Context, opt *AnswerOption) error
	tagExistsFn      func(ctx context.Context, tagID string) (bool, error)
	GetTranslationFn func(ctx context.Context, questionID, locale string) (*QuestionTranslation, error)
}

func newMockRepo() *mockRepository {
	return &mockRepository{questions: make(map[string]*Question)}
}

func (m *mockRepository) Create(ctx context.Context, q *Question) error {
	if m.createFn != nil {
		return m.createFn(ctx, q)
	}
	q.ID = fmt.Sprintf("q-%d", len(m.questions)+1)
	q.CreatedAt = time.Now()
	q.UpdatedAt = time.Now()
	m.questions[q.ID] = q
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id string) (*Question, error) {
	q, ok := m.questions[id]
	if !ok {
		return nil, ErrQuestionNotFound
	}
	cp := *q
	return &cp, nil
}

func (m *mockRepository) ListByCategory(_ context.Context, categoryID string) ([]*Question, error) {
	var out []*Question
	for _, q := range m.questions {
		if q.CategoryID == categoryID {
			cp := *q
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *mockRepository) Update(_ context.Context, q *Question) error {
	if _, ok := m.questions[q.ID]; !ok {
		return ErrQuestionNotFound
	}
	m.questions[q.ID] = q
	return nil
}

func (m *mockRepository) CreateVersion(ctx context.Context, newQ *Question, previousID string) error {
	if m.createVerFn != nil {
		return m.createVerFn(ctx, newQ, previousID)
	}
	prev, ok := m.questions[previousID]
	if !ok {
		return ErrQuestionNotFound
	}
	prev.Status = "archived"
	m.questions[previousID] = prev

	newQ.ID = fmt.Sprintf("q-%d", len(m.questions)+1)
	newQ.CreatedAt = time.Now()
	newQ.UpdatedAt = time.Now()
	m.questions[newQ.ID] = newQ
	return nil
}

func (m *mockRepository) CreateTranslation(_ context.Context, t *QuestionTranslation) error {
	t.UpdatedAt = time.Now()
	return nil
}

func (m *mockRepository) GetTranslation(ctx context.Context, questionID, locale string) (*QuestionTranslation, error) {
	if m.GetTranslationFn != nil {
		return m.GetTranslationFn(ctx, questionID, locale)
	}
	if _, ok := m.questions[questionID]; ok {
		return &QuestionTranslation{QuestionID: questionID, Locale: locale, Stem: "mock stem"}, nil
	}
	return nil, ErrNotFound
}

func (m *mockRepository) CreateAnswerOption(ctx context.Context, opt *AnswerOption) error {
	if m.createOptFn != nil {
		return m.createOptFn(ctx, opt)
	}
	opt.ID = fmt.Sprintf("opt-%d", 1)
	opt.CreatedAt = time.Now()
	return nil
}

func (m *mockRepository) GetAnswerOptions(_ context.Context, questionID string) ([]*AnswerOption, error) {
	return nil, nil
}

func (m *mockRepository) CreateAnswerTranslation(_ context.Context, t *AnswerTranslation) error {
	return nil
}

func (m *mockRepository) AddTag(_ context.Context, questionID, tagID string) error {
	return nil
}

func (m *mockRepository) RemoveTag(_ context.Context, questionID, tagID string) error {
	return nil
}

func (m *mockRepository) GetTags(_ context.Context, questionID string) ([]string, error) {
	return nil, nil
}

// ── FR-BB23 mock stubs ────────────────────────────────────────────────────────

func (m *mockRepository) CreateFull(_ context.Context, input CreateQuestionFullInput) (*QuestionDetail, error) {
	id := fmt.Sprintf("q-%d", len(m.questions)+1)
	detail := &QuestionDetail{
		ID:            id,
		Type:          input.Type,
		Difficulty:    input.Difficulty,
		Status:        "draft",
		CategoryID:    input.CategoryID,
		DefaultLocale: input.DefaultLocale,
		Version:       1,
		Translations:  map[string]TranslationDetail{},
		AnswerOptions: []AnswerOptionDetail{},
		Tags:          input.TagIDs,
	}
	return detail, nil
}

func (m *mockRepository) ListFiltered(_ context.Context, _ QuestionFilter) ([]*QuestionListItem, int, error) {
	return nil, 0, nil
}

func (m *mockRepository) GetWithDetails(_ context.Context, id string) (*QuestionDetail, error) {
	q, ok := m.questions[id]
	if !ok {
		return nil, ErrQuestionNotFound
	}
	return &QuestionDetail{
		ID:            q.ID,
		Type:          q.Type,
		Difficulty:    q.Difficulty,
		Status:        q.Status,
		CategoryID:    q.CategoryID,
		DefaultLocale: q.DefaultLocale,
		Version:       q.Version,
		ParentID:      q.ParentID,
		Translations:  map[string]TranslationDetail{},
		AnswerOptions: []AnswerOptionDetail{},
		Tags:          []string{},
	}, nil
}

func (m *mockRepository) UpdateInPlace(_ context.Context, id string, input UpdateQuestionInput) (*Question, error) {
	q, ok := m.questions[id]
	if !ok {
		return nil, ErrQuestionNotFound
	}
	q.CategoryID = input.CategoryID
	q.Difficulty = input.Difficulty
	m.questions[id] = q
	return q, nil
}

func (m *mockRepository) CreateVersionFull(_ context.Context, previousID string, input UpdateQuestionInput) (*Question, error) {
	prev, ok := m.questions[previousID]
	if !ok {
		return nil, ErrQuestionNotFound
	}
	prev.Status = "archived"
	m.questions[previousID] = prev

	newQ := &Question{
		ID:            fmt.Sprintf("q-%d", len(m.questions)+1),
		CategoryID:    input.CategoryID,
		Difficulty:    input.Difficulty,
		Type:          prev.Type,
		DefaultLocale: prev.DefaultLocale,
		Status:        "draft",
		CreatedBy:     input.UpdatedBy,
		Version:       prev.Version + 1,
		ParentID:      &previousID,
	}
	m.questions[newQ.ID] = newQ
	return newQ, nil
}

func (m *mockRepository) DeleteByID(_ context.Context, id string) error {
	if _, ok := m.questions[id]; !ok {
		return ErrQuestionNotFound
	}
	delete(m.questions, id)
	return nil
}

func (m *mockRepository) GetVersionChain(_ context.Context, id string) ([]*VersionEntry, error) {
	q, ok := m.questions[id]
	if !ok {
		return nil, nil
	}
	return []*VersionEntry{{ID: q.ID, Version: q.Version, Status: q.Status}}, nil
}

func (m *mockRepository) TagExists(ctx context.Context, tagID string) (bool, error) {
	if m.tagExistsFn != nil {
		return m.tagExistsFn(ctx, tagID)
	}
	return true, nil
}

// ── FR-BB25 mock stubs ────────────────────────────────────────────────────────

func (m *mockRepository) ResolveCategoryPath(_ context.Context, path string) (string, error) {
	return "cat-id-" + path, nil
}

func (m *mockRepository) FindSimilarStems(_ context.Context, _ []string, _ string) (map[string]StemSimilarityResult, error) {
	return map[string]StemSimilarityResult{}, nil
}

func (m *mockRepository) ImportBatch(_ context.Context, rows []ImportRow, _ string) ([]string, error) {
	ids := make([]string, len(rows))
	for i := range rows {
		ids[i] = fmt.Sprintf("imported-q-%d", i+1)
	}
	return ids, nil
}

func (m *mockRepository) StreamExport(_ context.Context, _ ExportFilter, _ func(*ExportRow) error) error {
	return nil
}

func (m *mockRepository) TagNameToID(_ context.Context, name string) (string, error) {
	return "tag-id-" + name, nil
}

func (m *mockRepository) TagIDByName(_ context.Context, names []string) (map[string]string, error) {
	result := make(map[string]string, len(names))
	for _, n := range names {
		result[n] = "tag-id-" + n
	}
	return result, nil
}

// --- Tests ---

func TestCreateQuestion_DefaultsVersionAndStatus(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	input := CreateQuestionInput{
		CategoryID:    "cat-1",
		Difficulty:    "easy",
		Type:          "single",
		DefaultLocale: "kk",
		CreatedBy:     "user-1",
	}

	q, err := svc.CreateQuestion(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Version != 1 {
		t.Errorf("expected version=1, got %d", q.Version)
	}
	if q.Status != "draft" {
		t.Errorf("expected status=draft, got %q", q.Status)
	}
	if q.ParentID != nil {
		t.Errorf("expected parentID=nil for new question, got %v", q.ParentID)
	}
}

func TestPublishNewVersion_IncrementsVersionAndSetsParent(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	// Seed an existing question.
	prev := &Question{
		ID:            "prev-1",
		CategoryID:    "cat-1",
		Difficulty:    "medium",
		Type:          "multiple",
		DefaultLocale: "kk",
		Status:        "active",
		CreatedBy:     "user-1",
		Version:       2,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.questions["prev-1"] = prev

	input := CreateQuestionInput{
		CategoryID:    "cat-1",
		Difficulty:    "medium",
		Type:          "multiple",
		DefaultLocale: "kk",
		CreatedBy:     "user-1",
	}

	newQ, err := svc.PublishNewVersion(context.Background(), "prev-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// New version must be prev.Version + 1.
	if newQ.Version != 3 {
		t.Errorf("expected version=3, got %d", newQ.Version)
	}

	// Parent must point to the previous question.
	if newQ.ParentID == nil || *newQ.ParentID != "prev-1" {
		t.Errorf("expected parentID=prev-1, got %v", newQ.ParentID)
	}

	// New question must be draft.
	if newQ.Status != "draft" {
		t.Errorf("expected status=draft, got %q", newQ.Status)
	}

	// Previous question must be archived.
	archived := repo.questions["prev-1"]
	if archived.Status != "archived" {
		t.Errorf("expected previous question to be archived, got %q", archived.Status)
	}
}

func TestPublishNewVersion_PreviousNotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	_, err := svc.PublishNewVersion(context.Background(), "nonexistent", CreateQuestionInput{})
	if err == nil {
		t.Fatal("expected error for missing previous question, got nil")
	}
}

func TestAddAnswerOption_PassesSortOrderThrough(t *testing.T) {
	var capturedOpt *AnswerOption
	repo := newMockRepo()
	repo.createOptFn = func(_ context.Context, opt *AnswerOption) error {
		capturedOpt = opt
		opt.ID = "opt-captured"
		opt.CreatedAt = time.Now()
		return nil
	}
	svc := NewService(repo)

	weight := 1.5
	polarity := "positive"
	opt, err := svc.AddAnswerOption(context.Background(), "q-1", 3, true, &weight, &polarity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedOpt == nil {
		t.Fatal("expected CreateAnswerOption to be called")
	}
	if opt.SortOrder != 3 {
		t.Errorf("expected sort_order=3, got %d", opt.SortOrder)
	}
	if !opt.IsCorrect {
		t.Errorf("expected is_correct=true")
	}
}

// ── FR-BB23 service tests ─────────────────────────────────────────────────────

func TestTransitionStatus_ValidTransitions(t *testing.T) {
	cases := []struct {
		from string
		to   string
	}{
		{"draft", "review"},
		{"review", "active"},
		{"active", "archived"},
	}
	for _, tc := range cases {
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			repo := newMockRepo()
			svc := NewService(repo)

			q := &Question{
				ID:            "q-1",
				CategoryID:    "cat-1",
				Difficulty:    "easy",
				Type:          "single",
				DefaultLocale: "kk",
				Status:        tc.from,
				CreatedBy:     "user-1",
				Version:       1,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			repo.questions["q-1"] = q

			updated, err := svc.TransitionStatus(context.Background(), "q-1", tc.to)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated.Status != tc.to {
				t.Errorf("expected status=%s, got %s", tc.to, updated.Status)
			}
		})
	}
}

func TestTransitionStatus_InvalidTransitions(t *testing.T) {
	invalidCases := []struct {
		from string
		to   string
	}{
		{"draft", "active"},
		{"draft", "archived"},
		{"review", "draft"},
		{"active", "draft"},
		{"archived", "active"},
		{"archived", "draft"},
	}
	for _, tc := range invalidCases {
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			repo := newMockRepo()
			svc := NewService(repo)

			q := &Question{
				ID:            "q-1",
				Status:        tc.from,
				DefaultLocale: "kk",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			repo.questions["q-1"] = q

			_, err := svc.TransitionStatus(context.Background(), "q-1", tc.to)
			if err == nil {
				t.Fatalf("expected error for transition %s->%s, got nil", tc.from, tc.to)
			}
			if !errors.Is(err, ErrInvalidTransition) {
				t.Errorf("expected ErrInvalidTransition, got %v", err)
			}
		})
	}
}

func TestDeleteQuestion_DraftSucceeds(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	q := &Question{
		ID:        "q-1",
		Status:    "draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.questions["q-1"] = q

	if err := svc.DeleteQuestion(context.Background(), "q-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.questions["q-1"]; ok {
		t.Error("expected question to be removed from store")
	}
}

func TestDeleteQuestion_NonDraftFails(t *testing.T) {
	for _, status := range []string{"review", "active", "archived"} {
		t.Run(status, func(t *testing.T) {
			repo := newMockRepo()
			svc := NewService(repo)

			q := &Question{ID: "q-1", Status: status, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			repo.questions["q-1"] = q

			err := svc.DeleteQuestion(context.Background(), "q-1")
			if err == nil {
				t.Fatalf("expected error for status %s, got nil", status)
			}
			if !errors.Is(err, ErrNotDraft) {
				t.Errorf("expected ErrNotDraft, got %v", err)
			}
		})
	}
}

func TestUpdateQuestion_ActiveCreatesNewVersion(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	q := &Question{
		ID:            "q-active",
		CategoryID:    "cat-1",
		Difficulty:    "medium",
		Type:          "single",
		DefaultLocale: "kk",
		Status:        "active",
		CreatedBy:     "user-1",
		Version:       1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.questions["q-active"] = q

	newQ, err := svc.UpdateQuestion(context.Background(), "q-active", UpdateQuestionInput{
		CategoryID: "cat-1",
		Difficulty: "hard",
		UpdatedBy:  "user-2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newQ.Version != 2 {
		t.Errorf("expected version=2, got %d", newQ.Version)
	}
	if newQ.Status != "draft" {
		t.Errorf("expected status=draft, got %s", newQ.Status)
	}
	if newQ.ParentID == nil || *newQ.ParentID != "q-active" {
		t.Errorf("expected parent_id=q-active")
	}
	// Previous should be archived.
	if repo.questions["q-active"].Status != "archived" {
		t.Errorf("expected previous question to be archived")
	}
}

func TestUpdateQuestion_DraftUpdatesInPlace(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	q := &Question{
		ID:            "q-draft",
		CategoryID:    "cat-1",
		Difficulty:    "easy",
		Type:          "single",
		DefaultLocale: "kk",
		Status:        "draft",
		CreatedBy:     "user-1",
		Version:       1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.questions["q-draft"] = q

	updated, err := svc.UpdateQuestion(context.Background(), "q-draft", UpdateQuestionInput{
		CategoryID: "cat-1",
		Difficulty: "hard",
		UpdatedBy:  "user-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ID != "q-draft" {
		t.Errorf("expected same ID for in-place update, got %s", updated.ID)
	}
	if updated.Difficulty != "hard" {
		t.Errorf("expected difficulty=hard, got %s", updated.Difficulty)
	}
}

func TestListVersions_ReturnsChain(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	q := &Question{
		ID:        "q-1",
		Status:    "active",
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.questions["q-1"] = q

	versions, err := svc.ListVersions(context.Background(), "q-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) == 0 {
		t.Error("expected at least one version entry")
	}
}

func TestTagQuestion_NonExistentTagReturnsErrTagNotFound(t *testing.T) {
	repo := newMockRepo()
	// Override TagExists to return false.
	repo.tagExistsFn = func(_ context.Context, _ string) (bool, error) { return false, nil }
	svc := NewService(repo)

	err := svc.TagQuestion(context.Background(), "q-1", "nonexistent-tag")
	if !errors.Is(err, ErrTagNotFound) {
		t.Errorf("expected ErrTagNotFound, got %v", err)
	}
}

// ── FR-BB22 AC-specific tests ─────────────────────────────────────────────────

// AC-7: version starts at 1, set by application (not DB trigger).
func TestCreateQuestion_VersionStartsAtOne(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	q, err := svc.CreateQuestion(context.Background(), CreateQuestionInput{
		CategoryID:    "cat-1",
		Difficulty:    "easy",
		Type:          "single",
		DefaultLocale: "kk",
		CreatedBy:     "user-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Version != 1 {
		t.Errorf("AC-7: version must start at 1, got %d", q.Version)
	}
}

// AC-7: version increments by exactly 1 on each new version.
func TestPublishNewVersion_VersionIncrementsBy1(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	// Seed a version-3 question to ensure increment is always +1.
	prev := &Question{
		ID:            "q-v3",
		CategoryID:    "cat-1",
		Difficulty:    "easy",
		Type:          "single",
		DefaultLocale: "kk",
		Status:        "active",
		CreatedBy:     "user-1",
		Version:       3,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.questions["q-v3"] = prev

	newQ, err := svc.PublishNewVersion(context.Background(), "q-v3", CreateQuestionInput{
		CategoryID: "cat-1",
		Difficulty: "easy",
		Type:       "single",
		CreatedBy:  "user-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newQ.Version != 4 {
		t.Errorf("AC-7: expected version=4 (3+1), got %d", newQ.Version)
	}
}

// AC-6: only one active question per version chain at any time.
// Creating a new version archives the previous one atomically.
func TestPublishNewVersion_OnlyOneActiveInChain(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	// Seed an active question (v1) and its already-archived parent (v1's predecessor).
	prev := &Question{
		ID:            "q-active",
		CategoryID:    "cat-1",
		Difficulty:    "easy",
		Type:          "single",
		DefaultLocale: "kk",
		Status:        "active",
		CreatedBy:     "user-1",
		Version:       1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.questions["q-active"] = prev

	newQ, err := svc.PublishNewVersion(context.Background(), "q-active", CreateQuestionInput{
		CategoryID: "cat-1",
		Difficulty: "easy",
		Type:       "single",
		CreatedBy:  "user-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The new question must not be active (it starts as draft).
	if newQ.Status == "active" {
		t.Errorf("AC-6: new version must not be active, got %q", newQ.Status)
	}

	// The previous active question must now be archived.
	archived := repo.questions["q-active"]
	if archived.Status != "archived" {
		t.Errorf("AC-6: previous active question must be archived atomically, got %q", archived.Status)
	}
}

// AC-8: TransitionStatus to 'review' requires default_locale stem to exist.
func TestTransitionStatus_RequiresStemForReview(t *testing.T) {
	repo := newMockRepo()

	// Override GetTranslation to return ErrNotFound — no stem exists.
	repo.GetTranslationFn = func(_ context.Context, _, _ string) (*QuestionTranslation, error) {
		return nil, ErrNotFound
	}
	svc := NewService(repo)

	q := &Question{
		ID:            "q-nostem",
		Status:        "draft",
		DefaultLocale: "kk",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.questions["q-nostem"] = q

	_, err := svc.TransitionStatus(context.Background(), "q-nostem", "review")
	if err == nil {
		t.Fatal("AC-8: expected ErrStemRequired when default_locale translation is missing, got nil")
	}
	if !errors.Is(err, ErrStemRequired) {
		t.Errorf("AC-8: expected ErrStemRequired, got %v", err)
	}
}

// AC-8: TransitionStatus to 'active' also requires default_locale stem.
func TestTransitionStatus_RequiresStemForActive(t *testing.T) {
	repo := newMockRepo()

	repo.GetTranslationFn = func(_ context.Context, _, _ string) (*QuestionTranslation, error) {
		return nil, ErrNotFound
	}
	svc := NewService(repo)

	q := &Question{
		ID:            "q-nostem",
		Status:        "review",
		DefaultLocale: "kk",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.questions["q-nostem"] = q

	_, err := svc.TransitionStatus(context.Background(), "q-nostem", "active")
	if err == nil {
		t.Fatal("AC-8: expected ErrStemRequired when default_locale translation is missing, got nil")
	}
	if !errors.Is(err, ErrStemRequired) {
		t.Errorf("AC-8: expected ErrStemRequired, got %v", err)
	}
}
