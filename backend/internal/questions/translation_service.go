package questions

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// Sentinel errors for the translation API (FR-BB24).
var (
	ErrUnsupportedLocale         = errors.New("locale is not in tenant available_locales")
	ErrCannotDeleteDefaultLocale = errors.New("cannot delete the default-locale translation")
	ErrMissingOptionTranslations = errors.New("translation missing for one or more answer options")
)

// MissingOptionsError is returned when a locale upsert omits translations for some options.
// It carries the list of option IDs that must also be provided.
type MissingOptionsError struct {
	MissingIDs []string
}

func (e *MissingOptionsError) Error() string {
	return fmt.Sprintf("missing translations for option IDs: %v", e.MissingIDs)
}

func (e *MissingOptionsError) Is(target error) bool {
	return target == ErrMissingOptionTranslations
}

// LocaleCoverage summarises which locales are present vs. missing relative to
// the tenant's available_locales.
type LocaleCoverage struct {
	Present []string `json:"present"`
	Missing []string `json:"missing"`
}

// AnswerTextTranslation is the per-option translated text used in API responses.
type AnswerTextTranslation struct {
	OptionID string `json:"option_id"`
	Text     string `json:"text"`
}

// LocaleTranslation is the full set of translated text for a single locale.
type LocaleTranslation struct {
	Locale      string                  `json:"locale"`
	Stem        string                  `json:"stem"`
	Explanation *string                 `json:"explanation,omitempty"`
	Options     []AnswerTextTranslation `json:"options"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

// QuestionTranslations is the payload returned by GET .../translations.
type QuestionTranslations struct {
	QuestionID     string                       `json:"question_id"`
	DefaultLocale  string                       `json:"default_locale"`
	LocaleCoverage LocaleCoverage               `json:"locale_coverage"`
	Translations   map[string]LocaleTranslation `json:"translations"`
}

// UpsertTranslationInput carries the body of PUT .../translations/:locale.
type UpsertTranslationInput struct {
	Stem        string
	Explanation *string
	Options     []AnswerTextTranslation
}

// TenantLocaleProvider exposes only the locale-related parts of tenant config
// that the translation service needs. Implemented by *tenant.service.
type TenantLocaleProvider interface {
	GetDefaultLocale() string
	GetAvailableLocales() []string
}

// TranslationService is the business-logic interface for FR-BB24.
type TranslationService interface {
	GetAll(ctx context.Context, questionID string) (*QuestionTranslations, error)
	Upsert(ctx context.Context, questionID, locale string, input UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error)
	Delete(ctx context.Context, questionID, locale string) (*LocaleTranslation, error)
}

type translationService struct {
	repo   TranslationRepository
	tenant TenantLocaleProvider
}

// NewTranslationService returns a TranslationService backed by the given repository
// and tenant locale provider.
func NewTranslationService(repo TranslationRepository, tenant TenantLocaleProvider) TranslationService {
	return &translationService{repo: repo, tenant: tenant}
}

// GetAll returns all locale translations for a question, plus the locale_coverage
// summary computed against the tenant's available_locales.
func (s *translationService) GetAll(ctx context.Context, questionID string) (*QuestionTranslations, error) {
	q, err := s.repo.GetQuestion(ctx, questionID)
	if err != nil {
		return nil, err
	}

	translations, err := s.repo.LoadAllTranslations(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("translations.GetAll: %w", err)
	}
	if translations == nil {
		translations = map[string]LocaleTranslation{}
	}

	coverage := computeCoverage(translations, s.tenant.GetAvailableLocales())

	return &QuestionTranslations{
		QuestionID:     q.ID,
		DefaultLocale:  q.DefaultLocale,
		LocaleCoverage: coverage,
		Translations:   translations,
	}, nil
}

// Upsert atomically replaces the translation for a single locale.
// Returns (before, after, error). `before` is nil if no translation existed yet.
func (s *translationService) Upsert(ctx context.Context, questionID, locale string, input UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error) {
	if locale == "" {
		return nil, nil, fmt.Errorf("%w: locale is required", ErrInvalidInput)
	}

	q, err := s.repo.GetQuestion(ctx, questionID)
	if err != nil {
		return nil, nil, err
	}

	if !isLocaleAllowed(locale, s.tenant.GetAvailableLocales()) {
		return nil, nil, fmt.Errorf("%w: %q", ErrUnsupportedLocale, locale)
	}

	// Validate: every answer_option for this question must have a translation in the payload.
	optionIDs, err := s.repo.ListAnswerOptionIDs(ctx, questionID)
	if err != nil {
		return nil, nil, fmt.Errorf("translations.Upsert: list options: %w", err)
	}
	providedOpts := make(map[string]string, len(input.Options))
	for _, opt := range input.Options {
		providedOpts[opt.OptionID] = opt.Text
	}
	var missing []string
	for _, id := range optionIDs {
		if _, ok := providedOpts[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, nil, &MissingOptionsError{MissingIDs: missing}
	}

	// Default-locale option text must be non-blank for choice questions (ISS-173b).
	// Non-default locales may stay blank; short-text has no options to check.
	if locale == q.DefaultLocale {
		inputs := make([]AnswerOptionInput, 0, len(optionIDs))
		for _, id := range optionIDs {
			inputs = append(inputs, AnswerOptionInput{
				Translations: map[string]AnswerTranslationInput{locale: {Text: providedOpts[id]}},
			})
		}
		if err := validateOptionTexts(q.Type, q.DefaultLocale, inputs); err != nil {
			return nil, nil, fmt.Errorf("translations.Upsert: %w", err)
		}
	}

	// Capture before-state for the audit log (may be nil).
	before, err := s.repo.LoadLocaleTranslation(ctx, questionID, locale)
	if err != nil {
		return nil, nil, fmt.Errorf("translations.Upsert: load before: %w", err)
	}

	after, err := s.repo.UpsertLocale(ctx, questionID, locale, input)
	if err != nil {
		return nil, nil, fmt.Errorf("translations.Upsert: %w", err)
	}

	return before, after, nil
}

// Delete removes the translation for a non-default locale. Returns the deleted
// translation (for audit before-state). Returns ErrCannotDeleteDefaultLocale if
// the locale equals the question's default_locale.
func (s *translationService) Delete(ctx context.Context, questionID, locale string) (*LocaleTranslation, error) {
	if locale == "" {
		return nil, fmt.Errorf("%w: locale is required", ErrInvalidInput)
	}

	q, err := s.repo.GetQuestion(ctx, questionID)
	if err != nil {
		return nil, err
	}

	if locale == q.DefaultLocale {
		return nil, fmt.Errorf("%w: %q", ErrCannotDeleteDefaultLocale, locale)
	}

	before, err := s.repo.LoadLocaleTranslation(ctx, questionID, locale)
	if err != nil {
		return nil, fmt.Errorf("translations.Delete: load before: %w", err)
	}

	if err := s.repo.DeleteLocale(ctx, questionID, locale); err != nil {
		return nil, fmt.Errorf("translations.Delete: %w", err)
	}
	return before, nil
}

// computeCoverage returns present/missing locales sorted alphabetically.
// `available` may be nil/empty — in that case `missing` is empty and `present`
// reflects whatever rows exist.
func computeCoverage(translations map[string]LocaleTranslation, available []string) LocaleCoverage {
	present := make([]string, 0, len(translations))
	for k := range translations {
		present = append(present, k)
	}
	sort.Strings(present)

	presentSet := make(map[string]struct{}, len(present))
	for _, l := range present {
		presentSet[l] = struct{}{}
	}
	missing := make([]string, 0)
	for _, l := range available {
		if _, ok := presentSet[l]; !ok {
			missing = append(missing, l)
		}
	}
	sort.Strings(missing)
	return LocaleCoverage{Present: present, Missing: missing}
}

func isLocaleAllowed(locale string, available []string) bool {
	if len(available) == 0 {
		// No constraint configured — accept anything (matches existing FR-BB23 behaviour
		// for upserts via the question CRUD endpoints).
		return true
	}
	for _, l := range available {
		if l == locale {
			return true
		}
	}
	return false
}
