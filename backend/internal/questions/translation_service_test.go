package questions

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- Mock TranslationRepository ---

type mockTranslationRepo struct {
	questions    map[string]*Question
	options      map[string][]string                       // questionID -> option IDs in order
	translations map[string]map[string]*LocaleTranslation  // questionID -> locale -> translation
	upsertCalls  int
}

func newMockTranslationRepo() *mockTranslationRepo {
	return &mockTranslationRepo{
		questions:    map[string]*Question{},
		options:      map[string][]string{},
		translations: map[string]map[string]*LocaleTranslation{},
	}
}

func (m *mockTranslationRepo) GetQuestion(_ context.Context, id string) (*Question, error) {
	q, ok := m.questions[id]
	if !ok {
		return nil, ErrQuestionNotFound
	}
	cp := *q
	return &cp, nil
}

func (m *mockTranslationRepo) ListAnswerOptionIDs(_ context.Context, questionID string) ([]string, error) {
	return append([]string(nil), m.options[questionID]...), nil
}

func (m *mockTranslationRepo) LoadAllTranslations(_ context.Context, questionID string) (map[string]LocaleTranslation, error) {
	src, ok := m.translations[questionID]
	if !ok {
		return map[string]LocaleTranslation{}, nil
	}
	out := make(map[string]LocaleTranslation, len(src))
	for k, v := range src {
		out[k] = *v
	}
	return out, nil
}

func (m *mockTranslationRepo) LoadLocaleTranslation(_ context.Context, questionID, locale string) (*LocaleTranslation, error) {
	if locMap, ok := m.translations[questionID]; ok {
		if t, ok := locMap[locale]; ok {
			cp := *t
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockTranslationRepo) UpsertLocale(_ context.Context, questionID, locale string, input UpsertTranslationInput) (*LocaleTranslation, error) {
	m.upsertCalls++
	if _, ok := m.translations[questionID]; !ok {
		m.translations[questionID] = map[string]*LocaleTranslation{}
	}
	out := &LocaleTranslation{
		Locale:      locale,
		Stem:        input.Stem,
		Explanation: input.Explanation,
		Options:     append([]AnswerTextTranslation(nil), input.Options...),
		UpdatedAt:   time.Now(),
	}
	m.translations[questionID][locale] = out
	cp := *out
	return &cp, nil
}

func (m *mockTranslationRepo) DeleteLocale(_ context.Context, questionID, locale string) error {
	if locMap, ok := m.translations[questionID]; ok {
		delete(locMap, locale)
	}
	return nil
}

// --- Mock TenantLocaleProvider ---

type mockTenant struct {
	defaultLocale    string
	availableLocales []string
}

func (m *mockTenant) GetDefaultLocale() string      { return m.defaultLocale }
func (m *mockTenant) GetAvailableLocales() []string { return m.availableLocales }

// --- Fixtures ---

func seedQuestion(repo *mockTranslationRepo, id, defaultLocale string, optionIDs ...string) {
	repo.questions[id] = &Question{
		ID:            id,
		DefaultLocale: defaultLocale,
		Status:        "draft",
	}
	repo.options[id] = optionIDs
}

// --- Tests ---

func TestTranslationGetAll_ReportsCoverage(t *testing.T) {
	repo := newMockTranslationRepo()
	seedQuestion(repo, "q-1", "kk", "opt-1", "opt-2")
	repo.translations["q-1"] = map[string]*LocaleTranslation{
		"kk": {Locale: "kk", Stem: "kk stem"},
		"ru": {Locale: "ru", Stem: "ru stem"},
	}
	svc := NewTranslationService(repo, &mockTenant{defaultLocale: "kk", availableLocales: []string{"kk", "ru", "en"}})

	out, err := svc.GetAll(context.Background(), "q-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := out.DefaultLocale, "kk"; got != want {
		t.Errorf("default_locale: got %q, want %q", got, want)
	}
	if got, want := len(out.LocaleCoverage.Present), 2; got != want {
		t.Errorf("present len: got %d, want %d (have %v)", got, want, out.LocaleCoverage.Present)
	}
	if len(out.LocaleCoverage.Missing) != 1 || out.LocaleCoverage.Missing[0] != "en" {
		t.Errorf("missing: got %v, want [en]", out.LocaleCoverage.Missing)
	}
}

func TestTranslationGetAll_QuestionNotFound(t *testing.T) {
	repo := newMockTranslationRepo()
	svc := NewTranslationService(repo, &mockTenant{})
	_, err := svc.GetAll(context.Background(), "missing")
	if !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}
}

func TestTranslationUpsert_RejectsUnsupportedLocale(t *testing.T) {
	repo := newMockTranslationRepo()
	seedQuestion(repo, "q-1", "kk")
	svc := NewTranslationService(repo, &mockTenant{defaultLocale: "kk", availableLocales: []string{"kk", "ru"}})

	_, _, err := svc.Upsert(context.Background(), "q-1", "fr", UpsertTranslationInput{Stem: "x"})
	if !errors.Is(err, ErrUnsupportedLocale) {
		t.Fatalf("expected ErrUnsupportedLocale, got %v", err)
	}
}

func TestTranslationUpsert_MissingOptionTranslations(t *testing.T) {
	repo := newMockTranslationRepo()
	seedQuestion(repo, "q-1", "kk", "opt-1", "opt-2")
	svc := NewTranslationService(repo, &mockTenant{defaultLocale: "kk", availableLocales: []string{"kk", "en"}})

	_, _, err := svc.Upsert(context.Background(), "q-1", "en", UpsertTranslationInput{
		Stem: "What?",
		Options: []AnswerTextTranslation{
			{OptionID: "opt-1", Text: "yes"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var missing *MissingOptionsError
	if !errors.As(err, &missing) {
		t.Fatalf("expected *MissingOptionsError, got %T (%v)", err, err)
	}
	if !errors.Is(err, ErrMissingOptionTranslations) {
		t.Errorf("errors.Is should match ErrMissingOptionTranslations")
	}
	if len(missing.MissingIDs) != 1 || missing.MissingIDs[0] != "opt-2" {
		t.Errorf("missing IDs: got %v, want [opt-2]", missing.MissingIDs)
	}
}

func TestTranslationUpsert_HappyPath(t *testing.T) {
	repo := newMockTranslationRepo()
	seedQuestion(repo, "q-1", "kk", "opt-1")
	svc := NewTranslationService(repo, &mockTenant{defaultLocale: "kk", availableLocales: []string{"kk", "en"}})

	before, after, err := svc.Upsert(context.Background(), "q-1", "en", UpsertTranslationInput{
		Stem: "What is X?",
		Options: []AnswerTextTranslation{
			{OptionID: "opt-1", Text: "A thing"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if before != nil {
		t.Errorf("expected before=nil on first write, got %+v", before)
	}
	if after == nil || after.Stem != "What is X?" {
		t.Fatalf("after mismatch: %+v", after)
	}
}

func TestTranslationUpsert_IsIdempotent(t *testing.T) {
	repo := newMockTranslationRepo()
	seedQuestion(repo, "q-1", "kk", "opt-1")
	svc := NewTranslationService(repo, &mockTenant{defaultLocale: "kk", availableLocales: []string{"kk", "en"}})

	input := UpsertTranslationInput{
		Stem: "What is X?",
		Options: []AnswerTextTranslation{
			{OptionID: "opt-1", Text: "A thing"},
		},
	}
	if _, _, err := svc.Upsert(context.Background(), "q-1", "en", input); err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}
	before, after, err := svc.Upsert(context.Background(), "q-1", "en", input)
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	if before == nil || before.Stem != "What is X?" {
		t.Errorf("second upsert should see prior state as before, got %+v", before)
	}
	if after == nil || after.Stem != "What is X?" {
		t.Errorf("second upsert after-state wrong: %+v", after)
	}
	if repo.upsertCalls != 2 {
		t.Errorf("expected 2 repo upsert calls, got %d", repo.upsertCalls)
	}
}

func TestTranslationDelete_RejectsDefaultLocale(t *testing.T) {
	repo := newMockTranslationRepo()
	seedQuestion(repo, "q-1", "kk")
	svc := NewTranslationService(repo, &mockTenant{defaultLocale: "kk", availableLocales: []string{"kk", "en"}})

	_, err := svc.Delete(context.Background(), "q-1", "kk")
	if !errors.Is(err, ErrCannotDeleteDefaultLocale) {
		t.Fatalf("expected ErrCannotDeleteDefaultLocale, got %v", err)
	}
}

func TestTranslationDelete_RemovesNonDefaultLocale(t *testing.T) {
	repo := newMockTranslationRepo()
	seedQuestion(repo, "q-1", "kk")
	repo.translations["q-1"] = map[string]*LocaleTranslation{
		"en": {Locale: "en", Stem: "What?"},
	}
	svc := NewTranslationService(repo, &mockTenant{defaultLocale: "kk", availableLocales: []string{"kk", "en"}})

	before, err := svc.Delete(context.Background(), "q-1", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if before == nil || before.Stem != "What?" {
		t.Errorf("expected before to contain prior translation, got %+v", before)
	}
	if _, ok := repo.translations["q-1"]["en"]; ok {
		t.Error("expected en translation to be removed")
	}
}

func TestTranslationDelete_QuestionNotFound(t *testing.T) {
	repo := newMockTranslationRepo()
	svc := NewTranslationService(repo, &mockTenant{defaultLocale: "kk"})
	_, err := svc.Delete(context.Background(), "missing", "en")
	if !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}
}

func TestComputeCoverage_SortsAlphabetically(t *testing.T) {
	cov := computeCoverage(map[string]LocaleTranslation{
		"ru": {Locale: "ru"},
		"kk": {Locale: "kk"},
	}, []string{"en", "ru", "kk"})
	if len(cov.Present) != 2 || cov.Present[0] != "kk" || cov.Present[1] != "ru" {
		t.Errorf("present sort wrong: %v", cov.Present)
	}
	if len(cov.Missing) != 1 || cov.Missing[0] != "en" {
		t.Errorf("missing wrong: %v", cov.Missing)
	}
}
