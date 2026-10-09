package questions

import (
	"context"
	"fmt"
	"strings"
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
	GetQuestionTags(ctx context.Context, questionID string) ([]string, error)

	// FR-BB23 additions.
	CreateQuestionFull(ctx context.Context, input CreateQuestionFullInput) (*QuestionDetail, error)
	ListFiltered(ctx context.Context, filter QuestionFilter) ([]*QuestionListItem, int, error)
	GetQuestionWithDetails(ctx context.Context, id string) (*QuestionDetail, error)
	UpdateQuestion(ctx context.Context, id string, input UpdateQuestionInput) (*Question, error)
	TransitionStatus(ctx context.Context, id, newStatus string) (*Question, error)
	DeleteQuestion(ctx context.Context, id string) error
	ListVersions(ctx context.Context, id string) ([]*VersionEntry, error)

	// FR-BB25 additions.
	ValidateAndImport(ctx context.Context, rows []ImportRow, dryRun bool, createdBy string) (*DryRunReport, *CommitResult, error)
	StreamExport(ctx context.Context, filter ExportFilter, fn func(*ExportRow) error) error
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
	exists, err := s.repo.TagExists(ctx, tagID)
	if err != nil {
		return fmt.Errorf("questions: TagQuestion: %w", err)
	}
	if !exists {
		return ErrTagNotFound
	}
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

func (s *service) GetQuestionTags(ctx context.Context, questionID string) ([]string, error) {
	tags, err := s.repo.GetTags(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("questions: GetQuestionTags: %w", err)
	}
	return tags, nil
}

// ── FR-BB23 additions ────────────────────────────────────────────────────────

func (s *service) CreateQuestionFull(ctx context.Context, input CreateQuestionFullInput) (*QuestionDetail, error) {
	if err := validateOptionTextsWithStems(input.Type, input.DefaultLocale, input.Translations, input.AnswerOptions); err != nil {
		return nil, fmt.Errorf("questions: CreateQuestionFull: %w", err)
	}
	detail, err := s.repo.CreateFull(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("questions: CreateQuestionFull: %w", err)
	}
	return detail, nil
}

func (s *service) ListFiltered(ctx context.Context, filter QuestionFilter) ([]*QuestionListItem, int, error) {
	items, total, err := s.repo.ListFiltered(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("questions: ListFiltered: %w", err)
	}
	return items, total, nil
}

func (s *service) GetQuestionWithDetails(ctx context.Context, id string) (*QuestionDetail, error) {
	detail, err := s.repo.GetWithDetails(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("questions: GetQuestionWithDetails: %w", err)
	}
	return detail, nil
}

// UpdateQuestion updates a question in place (draft/review) or creates a new version (active).
func (s *service) UpdateQuestion(ctx context.Context, id string, input UpdateQuestionInput) (*Question, error) {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("questions: UpdateQuestion: %w", err)
	}

	// Validate auto_grade / model_answer against the immutable question type.
	if input.AutoGrade || (input.ModelAnswer != nil && strings.TrimSpace(*input.ModelAnswer) != "") {
		if q.Type != "shorttext" {
			return nil, fmt.Errorf("questions: UpdateQuestion: %w", ErrInvalidFieldForType)
		}
	}
	if input.AutoGrade && q.Type == "shorttext" {
		if input.ModelAnswer == nil || strings.TrimSpace(*input.ModelAnswer) == "" {
			return nil, fmt.Errorf("questions: UpdateQuestion: %w", ErrMissingModelAnswer)
		}
	}

	// Choice options must carry non-blank text in the question's default locale.
	if err := validateOptionTextsWithStems(q.Type, q.DefaultLocale, input.Translations, input.AnswerOptions); err != nil {
		return nil, fmt.Errorf("questions: UpdateQuestion: %w", err)
	}

	switch q.Status {
	case "active":
		newQ, err := s.repo.CreateVersionFull(ctx, id, input)
		if err != nil {
			return nil, fmt.Errorf("questions: UpdateQuestion: create version: %w", err)
		}
		return newQ, nil
	case "draft", "review":
		updated, err := s.repo.UpdateInPlace(ctx, id, input)
		if err != nil {
			return nil, fmt.Errorf("questions: UpdateQuestion: update in place: %w", err)
		}
		return updated, nil
	default:
		return nil, fmt.Errorf("questions: UpdateQuestion: %w: question is %s", ErrInvalidInput, q.Status)
	}
}

// validTransitions defines the allowed status state machine.
// Each entry maps a current status to the set of statuses it may transition to.
var validTransitions = map[string][]string{
	"draft":    {"review"},
	"review":   {"active"},
	"active":   {"archived"},
	"archived": {"draft"},
}

// TransitionStatus enforces the status state machine and updates the question.
func (s *service) TransitionStatus(ctx context.Context, id, newStatus string) (*Question, error) {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("questions: TransitionStatus: %w", err)
	}

	allowedTargets, ok := validTransitions[q.Status]
	if !ok {
		return nil, fmt.Errorf("%w: cannot transition from %q to %q", ErrInvalidTransition, q.Status, newStatus)
	}
	found := false
	for _, t := range allowedTargets {
		if t == newStatus {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("%w: cannot transition from %q to %q", ErrInvalidTransition, q.Status, newStatus)
	}

	// For review and active, the default locale stem must be non-empty.
	if newStatus == "review" || newStatus == "active" {
		t, err := s.repo.GetTranslation(ctx, id, q.DefaultLocale)
		if err != nil || t.Stem == "" {
			return nil, ErrStemRequired
		}
	}

	// Moving a choice question to review or active requires non-blank default-locale
	// option text, so legacy rows with blank options are caught early and cannot be
	// presented to examinees (ISS-173b, ISS-228). Other transitions are untouched.
	if newStatus == "review" || newStatus == "active" {
		detail, err := s.repo.GetWithDetails(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("questions: TransitionStatus: load options: %w", err)
		}
		inputs := make([]AnswerOptionInput, 0, len(detail.AnswerOptions))
		for _, o := range detail.AnswerOptions {
			tr := make(map[string]AnswerTranslationInput, len(o.Translations))
			for loc, at := range o.Translations {
				tr[loc] = AnswerTranslationInput(at)
			}
			inputs = append(inputs, AnswerOptionInput{SortOrder: o.SortOrder, Translations: tr})
		}
		if err := validateOptionTexts(q.Type, q.DefaultLocale, inputs); err != nil {
			return nil, fmt.Errorf("questions: TransitionStatus: %w", err)
		}
	}

	q.Status = newStatus
	if err := s.repo.Update(ctx, q); err != nil {
		return nil, fmt.Errorf("questions: TransitionStatus: update: %w", err)
	}
	return q, nil
}

// DeleteQuestion hard-deletes a question only if it is in draft status.
func (s *service) DeleteQuestion(ctx context.Context, id string) error {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("questions: DeleteQuestion: %w", err)
	}
	if q.Status != "draft" {
		return ErrNotDraft
	}
	if err := s.repo.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("questions: DeleteQuestion: %w", err)
	}
	return nil
}

// ListVersions returns the full version chain for any question in the chain.
func (s *service) ListVersions(ctx context.Context, id string) ([]*VersionEntry, error) {
	// Verify the question exists first.
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, fmt.Errorf("questions: ListVersions: %w", err)
	}
	versions, err := s.repo.GetVersionChain(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("questions: ListVersions: %w", err)
	}
	return versions, nil
}
