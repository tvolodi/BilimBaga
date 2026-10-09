package questions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TranslationRepository defines the persistence operations needed by the
// translation service (FR-BB24). It deliberately operates only on the question's
// translation tables, not on the question row itself.
type TranslationRepository interface {
	// GetQuestion returns the minimal Question row used for existence + default_locale checks.
	GetQuestion(ctx context.Context, id string) (*Question, error)
	// ListAnswerOptionIDs returns the answer-option IDs for a question, sorted by sort_order.
	ListAnswerOptionIDs(ctx context.Context, questionID string) ([]string, error)
	// LoadAllTranslations returns a map keyed by locale of every translation row for the question.
	LoadAllTranslations(ctx context.Context, questionID string) (map[string]LocaleTranslation, error)
	// LoadLocaleTranslation returns the translation for a single locale, or nil if absent.
	LoadLocaleTranslation(ctx context.Context, questionID, locale string) (*LocaleTranslation, error)
	// UpsertLocale atomically replaces question_translations + answer_translations rows for the locale.
	UpsertLocale(ctx context.Context, questionID, locale string, input UpsertTranslationInput) (*LocaleTranslation, error)
	// DeleteLocale removes the question_translations + answer_translations rows for the locale.
	DeleteLocale(ctx context.Context, questionID, locale string) error
}

type translationRepository struct {
	db *sqlx.DB
}

// NewTranslationRepository returns a PostgreSQL-backed TranslationRepository.
func NewTranslationRepository(db *sqlx.DB) TranslationRepository {
	return &translationRepository{db: db}
}

func (r *translationRepository) GetQuestion(ctx context.Context, id string) (*Question, error) {
	const q = `
		SELECT id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at
		FROM questions WHERE id = $1`
	var out Question
	if err := r.db.GetContext(ctx, &out, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("translations.GetQuestion: %w", err)
	}
	return &out, nil
}

func (r *translationRepository) ListAnswerOptionIDs(ctx context.Context, questionID string) ([]string, error) {
	const q = `SELECT id::text FROM answer_options WHERE question_id = $1 ORDER BY sort_order, id`
	var ids []string
	if err := r.db.SelectContext(ctx, &ids, q, questionID); err != nil {
		return nil, fmt.Errorf("translations.ListAnswerOptionIDs: %w", err)
	}
	return ids, nil
}

// LoadAllTranslations issues two queries (one for stem/explanation, one for option text)
// and merges them in memory.
func (r *translationRepository) LoadAllTranslations(ctx context.Context, questionID string) (map[string]LocaleTranslation, error) {
	type qtRow struct {
		Locale      string    `db:"locale"`
		Stem        string    `db:"stem"`
		Explanation *string   `db:"explanation"`
		UpdatedAt   sql.NullTime `db:"updated_at"`
	}
	var qtRows []qtRow
	if err := r.db.SelectContext(ctx, &qtRows,
		`SELECT locale, stem, explanation, updated_at FROM question_translations WHERE question_id = $1`,
		questionID,
	); err != nil {
		return nil, fmt.Errorf("translations.LoadAllTranslations: question_translations: %w", err)
	}

	type atRow struct {
		OptionID string `db:"option_id"`
		Locale   string `db:"locale"`
		Text     string `db:"text"`
	}
	var atRows []atRow
	if err := r.db.SelectContext(ctx, &atRows,
		`SELECT at.option_id::text, at.locale, at.text
		 FROM answer_translations at
		 JOIN answer_options ao ON ao.id = at.option_id
		 WHERE ao.question_id = $1
		 ORDER BY ao.sort_order, ao.id`,
		questionID,
	); err != nil {
		return nil, fmt.Errorf("translations.LoadAllTranslations: answer_translations: %w", err)
	}

	out := make(map[string]LocaleTranslation, len(qtRows))
	for _, qt := range qtRows {
		entry := LocaleTranslation{
			Locale:      qt.Locale,
			Stem:        qt.Stem,
			Explanation: qt.Explanation,
			Options:     []AnswerTextTranslation{},
		}
		if qt.UpdatedAt.Valid {
			entry.UpdatedAt = qt.UpdatedAt.Time
		}
		out[qt.Locale] = entry
	}
	for _, at := range atRows {
		entry, ok := out[at.Locale]
		if !ok {
			// Orphan option text for a locale with no question_translations row —
			// surface it anyway so locale_coverage reflects reality.
			entry = LocaleTranslation{Locale: at.Locale, Options: []AnswerTextTranslation{}}
		}
		entry.Options = append(entry.Options, AnswerTextTranslation{OptionID: at.OptionID, Text: at.Text})
		out[at.Locale] = entry
	}
	return out, nil
}

func (r *translationRepository) LoadLocaleTranslation(ctx context.Context, questionID, locale string) (*LocaleTranslation, error) {
	const q = `SELECT locale, stem, explanation, updated_at FROM question_translations WHERE question_id = $1 AND locale = $2`
	var row struct {
		Locale      string       `db:"locale"`
		Stem        string       `db:"stem"`
		Explanation *string      `db:"explanation"`
		UpdatedAt   sql.NullTime `db:"updated_at"`
	}
	if err := r.db.GetContext(ctx, &row, q, questionID, locale); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("translations.LoadLocaleTranslation: %w", err)
	}
	out := &LocaleTranslation{
		Locale:      row.Locale,
		Stem:        row.Stem,
		Explanation: row.Explanation,
		Options:     []AnswerTextTranslation{},
	}
	if row.UpdatedAt.Valid {
		out.UpdatedAt = row.UpdatedAt.Time
	}

	const optQ = `
		SELECT at.option_id::text, at.text
		FROM answer_translations at
		JOIN answer_options ao ON ao.id = at.option_id
		WHERE ao.question_id = $1 AND at.locale = $2
		ORDER BY ao.sort_order, ao.id`
	type optRow struct {
		OptionID string `db:"option_id"`
		Text     string `db:"text"`
	}
	var optRows []optRow
	if err := r.db.SelectContext(ctx, &optRows, optQ, questionID, locale); err != nil {
		return nil, fmt.Errorf("translations.LoadLocaleTranslation: options: %w", err)
	}
	for _, o := range optRows {
		out.Options = append(out.Options, AnswerTextTranslation(o))
	}
	return out, nil
}

// UpsertLocale wraps the writes in a single transaction and uses ON CONFLICT
// upserts to avoid touching FK references mid-transaction.
func (r *translationRepository) UpsertLocale(ctx context.Context, questionID, locale string, input UpsertTranslationInput) (*LocaleTranslation, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("translations.UpsertLocale: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const upsertQt = `
		INSERT INTO question_translations (question_id, locale, stem, explanation, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (question_id, locale) DO UPDATE
		    SET stem = EXCLUDED.stem,
		        explanation = EXCLUDED.explanation,
		        updated_at = now()
		RETURNING locale, stem, explanation, updated_at`
	var row struct {
		Locale      string       `db:"locale"`
		Stem        string       `db:"stem"`
		Explanation *string      `db:"explanation"`
		UpdatedAt   sql.NullTime `db:"updated_at"`
	}
	if err = tx.GetContext(ctx, &row, upsertQt, questionID, locale, input.Stem, input.Explanation); err != nil {
		return nil, fmt.Errorf("translations.UpsertLocale: upsert question_translations: %w", err)
	}

	const upsertAt = `
		INSERT INTO answer_translations (option_id, locale, text)
		VALUES ($1, $2, $3)
		ON CONFLICT (option_id, locale) DO UPDATE SET text = EXCLUDED.text`
	for _, opt := range input.Options {
		if _, err = tx.ExecContext(ctx, upsertAt, opt.OptionID, locale, opt.Text); err != nil {
			return nil, fmt.Errorf("translations.UpsertLocale: upsert option %s: %w", opt.OptionID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("translations.UpsertLocale: commit: %w", err)
	}

	out := &LocaleTranslation{
		Locale:      row.Locale,
		Stem:        row.Stem,
		Explanation: row.Explanation,
		Options:     append([]AnswerTextTranslation(nil), input.Options...),
	}
	if row.UpdatedAt.Valid {
		out.UpdatedAt = row.UpdatedAt.Time
	}
	return out, nil
}

func (r *translationRepository) DeleteLocale(ctx context.Context, questionID, locale string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("translations.DeleteLocale: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Delete answer_translations first (FK from answer_options).
	if _, err = tx.ExecContext(ctx,
		`DELETE FROM answer_translations
		 WHERE locale = $1
		   AND option_id IN (SELECT id FROM answer_options WHERE question_id = $2)`,
		locale, questionID,
	); err != nil {
		return fmt.Errorf("translations.DeleteLocale: answer_translations: %w", err)
	}

	if _, err = tx.ExecContext(ctx,
		`DELETE FROM question_translations WHERE question_id = $1 AND locale = $2`,
		questionID, locale,
	); err != nil {
		return fmt.Errorf("translations.DeleteLocale: question_translations: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("translations.DeleteLocale: commit: %w", err)
	}
	return nil
}
