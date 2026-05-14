package questions

import (
	"context"
	"fmt"
)

// CreateQuestionInput carries the fields needed to create a new question.
type CreateQuestionInput struct {
	CategoryID    string
	Difficulty    string
	Type          string
	DefaultLocale string
	CreatedBy     string
}

// Service defines the business logic for the questions domain.
type Service interface {
	CreateQuestion(ctx context.Context, input CreateQuestionInput) (*Question, error)
	GetQuestion(ctx context.Context, id string) (*Question, error)
	ListQuestions(ctx context.Context, categoryID string) ([]*Question, error)
	// PublishNewVersion archives the previous question and creates a new version.
	PublishNewVersion(ctx context.Context, previousID string, input CreateQuestionInput) (*Question, error)

	AddTranslation(ctx context.Context, questionID string, locale, stem string, explanation *string) (*QuestionTranslation, error)
	AddAnswerOption(ctx context.Context, questionID string, sortOrder int, isCorrect bool, likertWeight *float64, likertPolarity *string) (*AnswerOption, error)
	AddAnswerTranslation(ctx context.Context, optionID string, locale, text string) (*AnswerTranslation, error)

	TagQuestion(ctx context.Context, questionID, tagID string) error
	UntagQuestion(ctx context.Context, questionID, tagID string) error
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateQuestion(ctx context.Context, input CreateQuestionInput) (*Question, error) {
	locale := input.DefaultLocale
	if locale == "" {
		locale = "kk"
	}
	q := &Question{
		CategoryID:    input.CategoryID,
		Difficulty:    input.Difficulty,
		Type:          input.Type,
		DefaultLocale: locale,
		Status:        "draft",
		CreatedBy:     input.CreatedBy,
		Version:       1,
	}
	if err := s.repo.Create(ctx, q); err != nil {
		return nil, fmt.Errorf("questions: CreateQuestion: %w", err)
	}
	return q, nil
}

func (s *service) GetQuestion(ctx context.Context, id string) (*Question, error) {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("questions: GetQuestion: %w", err)
	}
	return q, nil
}

func (s *service) ListQuestions(ctx context.Context, categoryID string) ([]*Question, error) {
	qs, err := s.repo.ListByCategory(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("questions: ListQuestions: %w", err)
	}
	return qs, nil
}

// PublishNewVersion fetches the previous question, increments the version,
// sets parentID, and delegates the atomic archive+insert to the repository.
func (s *service) PublishNewVersion(ctx context.Context, previousID string, input CreateQuestionInput) (*Question, error) {
	prev, err := s.repo.GetByID(ctx, previousID)
	if err != nil {
		return nil, fmt.Errorf("questions: PublishNewVersion: fetch previous: %w", err)
	}

	locale := input.DefaultLocale
	if locale == "" {
		locale = prev.DefaultLocale
	}

	newQ := &Question{
		CategoryID:    input.CategoryID,
		Difficulty:    input.Difficulty,
		Type:          input.Type,
		DefaultLocale: locale,
		Status:        "draft",
		CreatedBy:     input.CreatedBy,
		Version:       prev.Version + 1,
		ParentID:      &previousID,
	}

	if err := s.repo.CreateVersion(ctx, newQ, previousID); err != nil {
		return nil, fmt.Errorf("questions: PublishNewVersion: %w", err)
	}
	return newQ, nil
}

func (s *service) AddTranslation(ctx context.Context, questionID string, locale, stem string, explanation *string) (*QuestionTranslation, error) {
	t := &QuestionTranslation{
		QuestionID:  questionID,
		Locale:      locale,
		Stem:        stem,
		Explanation: explanation,
	}
	if err := s.repo.CreateTranslation(ctx, t); err != nil {
		return nil, fmt.Errorf("questions: AddTranslation: %w", err)
	}
	return t, nil
}

func (s *service) AddAnswerOption(ctx context.Context, questionID string, sortOrder int, isCorrect bool, likertWeight *float64, likertPolarity *string) (*AnswerOption, error) {
	opt := &AnswerOption{
		QuestionID:     questionID,
		SortOrder:      sortOrder,
		IsCorrect:      isCorrect,
		LikertWeight:   likertWeight,
		LikertPolarity: likertPolarity,
	}
	if err := s.repo.CreateAnswerOption(ctx, opt); err != nil {
		return nil, fmt.Errorf("questions: AddAnswerOption: %w", err)
	}
	return opt, nil
}

func (s *service) AddAnswerTranslation(ctx context.Context, optionID string, locale, text string) (*AnswerTranslation, error) {
	t := &AnswerTranslation{
		OptionID: optionID,
		Locale:   locale,
		Text:     text,
	}
	if err := s.repo.CreateAnswerTranslation(ctx, t); err != nil {
		return nil, fmt.Errorf("questions: AddAnswerTranslation: %w", err)
	}
	return t, nil
}

func (s *service) TagQuestion(ctx context.Context, questionID, tagID string) error {
	if err := s.repo.AddTag(ctx, questionID, tagID); err != nil {
		return fmt.Errorf("questions: TagQuestion: %w", err)
	}
	return nil
}

func (s *service) UntagQuestion(ctx context.Context, questionID, tagID string) error {
	if err := s.repo.RemoveTag(ctx, questionID, tagID); err != nil {
		return fmt.Errorf("questions: UntagQuestion: %w", err)
	}
	return nil
}
