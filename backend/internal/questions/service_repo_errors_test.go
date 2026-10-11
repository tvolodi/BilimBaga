package questions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #514: the service wraps each repository failure with its own method name, and keeps the sentinel reachable with
// errors.Is. The handlers depend on that: RemoveTag answers 404 only when ErrQuestionNotFound survives the wrap.

// failingRepo overrides one repository method to return err. Every other method is the shared mock.
type failingRepo struct {
	mockRepository
	err error
}

func (r *failingRepo) GetByID(context.Context, string) (*Question, error) { return nil, r.err }
func (r *failingRepo) ListByCategory(context.Context, string) ([]*Question, error) {
	return nil, r.err
}
func (r *failingRepo) CreateTranslation(context.Context, *QuestionTranslation) error { return r.err }
func (r *failingRepo) CreateAnswerTranslation(context.Context, *AnswerTranslation) error {
	return r.err
}
func (r *failingRepo) RemoveTag(context.Context, string, string) error { return r.err }
func (r *failingRepo) GetTags(context.Context, string) ([]string, error) {
	return nil, r.err
}
func (r *failingRepo) ListFiltered(context.Context, QuestionFilter) ([]*QuestionListItem, int, error) {
	return nil, 0, r.err
}
func (r *failingRepo) GetWithDetails(context.Context, string) (*QuestionDetail, error) {
	return nil, r.err
}

func TestService_RepositoryFailuresAreWrappedWithTheMethodAndKeepTheSentinel(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("db down")

	cases := []struct {
		name     string
		err      error
		call     func(Service) error
		sentinel error
		method   string
	}{
		{"GetQuestion", ErrQuestionNotFound, func(s Service) error { _, err := s.GetQuestion(ctx, "q-1"); return err }, ErrQuestionNotFound, "GetQuestion"},
		{"ListQuestions", boom, func(s Service) error { _, err := s.ListQuestions(ctx, "cat-1"); return err }, boom, "ListQuestions"},
		{"AddTranslation", boom, func(s Service) error { _, err := s.AddTranslation(ctx, "q-1", "en", "stem", nil); return err }, boom, "AddTranslation"},
		{"AddAnswerTranslation", boom, func(s Service) error { _, err := s.AddAnswerTranslation(ctx, "o-1", "en", "text"); return err }, boom, "AddAnswerTranslation"},
		{"UntagQuestion", boom, func(s Service) error { return s.UntagQuestion(ctx, "q-1", "t-1") }, boom, "UntagQuestion"},
		{"GetQuestionTags", boom, func(s Service) error { _, err := s.GetQuestionTags(ctx, "q-1"); return err }, boom, "GetQuestionTags"},
		{"ListFiltered", boom, func(s Service) error { _, _, err := s.ListFiltered(ctx, QuestionFilter{}); return err }, boom, "ListFiltered"},
		{"GetQuestionWithDetails", ErrQuestionNotFound, func(s Service) error { _, err := s.GetQuestionWithDetails(ctx, "q-1"); return err }, ErrQuestionNotFound, "GetQuestionWithDetails"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(&failingRepo{err: tc.err})
			err := tc.call(svc)
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.sentinel)
			assert.ErrorContains(t, err, "questions: "+tc.method+":")
		})
	}
}

func TestService_GetQuestionReturnsTheStoredQuestion(t *testing.T) {
	repo := &mockRepository{questions: map[string]*Question{"q-1": {ID: "q-1", Type: "single"}}}
	got, err := NewService(repo).GetQuestion(context.Background(), "q-1")
	require.NoError(t, err)
	assert.Equal(t, "q-1", got.ID)
}

func TestMissingOptionsError_NamesTheIDsAndMatchesTheSentinel(t *testing.T) {
	var err error = &MissingOptionsError{MissingIDs: []string{"opt-1", "opt-2"}}
	assert.ErrorContains(t, err, "opt-1")
	assert.ErrorContains(t, err, "opt-2")
	assert.ErrorIs(t, err, ErrMissingOptionTranslations)
	assert.NotErrorIs(t, err, ErrUnsupportedLocale)
}
