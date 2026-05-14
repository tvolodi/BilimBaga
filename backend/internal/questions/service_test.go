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
	questions    map[string]*Question
	createFn     func(ctx context.Context, q *Question) error
	createVerFn  func(ctx context.Context, newQ *Question, previousID string) error
	createOptFn  func(ctx context.Context, opt *AnswerOption) error
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

func (m *mockRepository) GetTranslation(_ context.Context, questionID, locale string) (*QuestionTranslation, error) {
	return nil, errors.New("not implemented in mock")
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
