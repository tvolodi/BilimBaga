package questions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository defines all persistence operations for the questions domain.
type Repository interface {
	Create(ctx context.Context, q *Question) error
	GetByID(ctx context.Context, id string) (*Question, error)
	ListByCategory(ctx context.Context, categoryID string) ([]*Question, error)
	Update(ctx context.Context, q *Question) error
	// CreateVersion atomically archives the previous question and inserts the new version.
	CreateVersion(ctx context.Context, newQ *Question, previousID string) error

	CreateTranslation(ctx context.Context, t *QuestionTranslation) error
	GetTranslation(ctx context.Context, questionID string, locale string) (*QuestionTranslation, error)

	CreateAnswerOption(ctx context.Context, opt *AnswerOption) error
	GetAnswerOptions(ctx context.Context, questionID string) ([]*AnswerOption, error)

	CreateAnswerTranslation(ctx context.Context, t *AnswerTranslation) error

	AddTag(ctx context.Context, questionID, tagID string) error
	RemoveTag(ctx context.Context, questionID, tagID string) error
	GetTags(ctx context.Context, questionID string) ([]string, error)

	// FR-BB23 additions.
	CreateFull(ctx context.Context, input CreateQuestionFullInput) (*QuestionDetail, error)
	ListFiltered(ctx context.Context, filter QuestionFilter) ([]*QuestionListItem, int, error)
	GetWithDetails(ctx context.Context, id string) (*QuestionDetail, error)
	UpdateInPlace(ctx context.Context, id string, input UpdateQuestionInput) (*Question, error)
	CreateVersionFull(ctx context.Context, previousID string, input UpdateQuestionInput) (*Question, error)
	DeleteByID(ctx context.Context, id string) error
	GetVersionChain(ctx context.Context, id string) ([]*VersionEntry, error)
	TagExists(ctx context.Context, tagID string) (bool, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewRepository returns a PostgreSQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, q *Question) error {
	const query = `
		INSERT INTO questions
			(id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at)
		VALUES
			(gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		RETURNING id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at`
	return r.db.GetContext(ctx, q, query,
		q.CategoryID, q.Difficulty, q.Type, q.DefaultLocale,
		q.Status, q.CreatedBy, q.Version, q.ParentID,
	)
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*Question, error) {
	const query = `
		SELECT id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at
		FROM questions WHERE id = $1`
	var q Question
	if err := r.db.GetContext(ctx, &q, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("questions.GetByID: %w", err)
	}
	return &q, nil
}

func (r *postgresRepository) ListByCategory(ctx context.Context, categoryID string) ([]*Question, error) {
	const query = `
		SELECT id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at
		FROM questions WHERE category_id = $1 ORDER BY created_at DESC`
	var rows []*Question
	if err := r.db.SelectContext(ctx, &rows, query, categoryID); err != nil {
		return nil, fmt.Errorf("questions.ListByCategory: %w", err)
	}
	return rows, nil
}

func (r *postgresRepository) Update(ctx context.Context, q *Question) error {
	const query = `
		UPDATE questions
		SET category_id = $1, difficulty = $2, type = $3, default_locale = $4,
		    status = $5, version = $6, parent_id = $7, updated_at = now()
		WHERE id = $8
		RETURNING id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at`
	if err := r.db.GetContext(ctx, q, query,
		q.CategoryID, q.Difficulty, q.Type, q.DefaultLocale,
		q.Status, q.Version, q.ParentID, q.ID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrQuestionNotFound
		}
		return fmt.Errorf("questions.Update: %w", err)
	}
	return nil
}

// CreateVersion atomically archives the previous question and inserts the new
// version row in a single transaction.
func (r *postgresRepository) CreateVersion(ctx context.Context, newQ *Question, previousID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("questions.CreateVersion: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Archive the previous question.
	const archiveQ = `UPDATE questions SET status = 'archived', updated_at = now() WHERE id = $1`
	if _, err = tx.ExecContext(ctx, archiveQ, previousID); err != nil {
		return fmt.Errorf("questions.CreateVersion: archive previous: %w", err)
	}

	// Insert the new version.
	const insertQ = `
		INSERT INTO questions
			(id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at)
		VALUES
			(gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		RETURNING id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at`
	if err = tx.GetContext(ctx, newQ, insertQ,
		newQ.CategoryID, newQ.Difficulty, newQ.Type, newQ.DefaultLocale,
		newQ.Status, newQ.CreatedBy, newQ.Version, newQ.ParentID,
	); err != nil {
		return fmt.Errorf("questions.CreateVersion: insert new: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("questions.CreateVersion: commit: %w", err)
	}
	return nil
}

func (r *postgresRepository) CreateTranslation(ctx context.Context, t *QuestionTranslation) error {
	const query = `
		INSERT INTO question_translations (question_id, locale, stem, explanation, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (question_id, locale) DO UPDATE
		    SET stem = EXCLUDED.stem, explanation = EXCLUDED.explanation, updated_at = now()
		RETURNING question_id, locale, stem, explanation, updated_at`
	return r.db.GetContext(ctx, t, query, t.QuestionID, t.Locale, t.Stem, t.Explanation)
}

func (r *postgresRepository) GetTranslation(ctx context.Context, questionID string, locale string) (*QuestionTranslation, error) {
	const query = `
		SELECT question_id, locale, stem, explanation, updated_at
		FROM question_translations WHERE question_id = $1 AND locale = $2`
	var t QuestionTranslation
	if err := r.db.GetContext(ctx, &t, query, questionID, locale); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("questions.GetTranslation: %w", err)
	}
	return &t, nil
}

func (r *postgresRepository) CreateAnswerOption(ctx context.Context, opt *AnswerOption) error {
	const query = `
		INSERT INTO answer_options (id, question_id, sort_order, is_correct, likert_weight, likert_polarity, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, now())
		RETURNING id, question_id, sort_order, is_correct, likert_weight, likert_polarity, created_at`
	return r.db.GetContext(ctx, opt, query,
		opt.QuestionID, opt.SortOrder, opt.IsCorrect, opt.LikertWeight, opt.LikertPolarity,
	)
}

func (r *postgresRepository) GetAnswerOptions(ctx context.Context, questionID string) ([]*AnswerOption, error) {
	const query = `
		SELECT id, question_id, sort_order, is_correct, likert_weight, likert_polarity, created_at
		FROM answer_options WHERE question_id = $1 ORDER BY sort_order`
	var rows []*AnswerOption
	if err := r.db.SelectContext(ctx, &rows, query, questionID); err != nil {
		return nil, fmt.Errorf("questions.GetAnswerOptions: %w", err)
	}
	return rows, nil
}

func (r *postgresRepository) CreateAnswerTranslation(ctx context.Context, t *AnswerTranslation) error {
	const query = `
		INSERT INTO answer_translations (option_id, locale, text)
		VALUES ($1, $2, $3)
		ON CONFLICT (option_id, locale) DO UPDATE SET text = EXCLUDED.text
		RETURNING option_id, locale, text`
	return r.db.GetContext(ctx, t, query, t.OptionID, t.Locale, t.Text)
}

func (r *postgresRepository) AddTag(ctx context.Context, questionID, tagID string) error {
	const query = `
		INSERT INTO question_tags (question_id, tag_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`
	if _, err := r.db.ExecContext(ctx, query, questionID, tagID); err != nil {
		return fmt.Errorf("questions.AddTag: %w", err)
	}
	return nil
}

func (r *postgresRepository) RemoveTag(ctx context.Context, questionID, tagID string) error {
	const query = `DELETE FROM question_tags WHERE question_id = $1 AND tag_id = $2`
	if _, err := r.db.ExecContext(ctx, query, questionID, tagID); err != nil {
		return fmt.Errorf("questions.RemoveTag: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetTags(ctx context.Context, questionID string) ([]string, error) {
	const query = `SELECT tag_id::text FROM question_tags WHERE question_id = $1`
	var ids []string
	if err := r.db.SelectContext(ctx, &ids, query, questionID); err != nil {
		return nil, fmt.Errorf("questions.GetTags: %w", err)
	}
	return ids, nil
}

// ── FR-BB23 additions ────────────────────────────────────────────────────────

// TagExists checks whether a tag with the given ID exists in the tags table.
func (r *postgresRepository) TagExists(ctx context.Context, tagID string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, tagID); err != nil {
		return false, fmt.Errorf("questions.TagExists: %w", err)
	}
	return exists, nil
}

// CreateFull atomically creates a question along with its translations, answer options, and tags.
func (r *postgresRepository) CreateFull(ctx context.Context, input CreateQuestionFullInput) (*QuestionDetail, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("questions.CreateFull: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	q := &Question{
		CategoryID:    input.CategoryID,
		Difficulty:    input.Difficulty,
		Type:          input.Type,
		DefaultLocale: input.DefaultLocale,
		Status:        "draft",
		CreatedBy:     input.CreatedBy,
		Version:       1,
	}
	const insertQ = `
		INSERT INTO questions
			(id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at)
		VALUES
			(gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		RETURNING id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at`
	if err = tx.GetContext(ctx, q, insertQ,
		q.CategoryID, q.Difficulty, q.Type, q.DefaultLocale,
		q.Status, q.CreatedBy, q.Version, q.ParentID,
	); err != nil {
		return nil, fmt.Errorf("questions.CreateFull: insert question: %w", err)
	}

	detail, err := applySubObjects(ctx, tx, q.ID, input.Translations, input.AnswerOptions, input.TagIDs)
	if err != nil {
		return nil, fmt.Errorf("questions.CreateFull: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("questions.CreateFull: commit: %w", err)
	}

	localeCoverage := sortedKeys(detail.translations)
	return &QuestionDetail{
		ID:             q.ID,
		Type:           q.Type,
		Difficulty:     q.Difficulty,
		Status:         q.Status,
		CategoryID:     q.CategoryID,
		DefaultLocale:  q.DefaultLocale,
		Version:        q.Version,
		ParentID:       q.ParentID,
		LocaleCoverage: localeCoverage,
		Translations:   detail.translations,
		AnswerOptions:  detail.answerOptions,
		Tags:           detail.tags,
		CreatedBy:      q.CreatedBy,
		CreatedAt:      q.CreatedAt,
		UpdatedAt:      q.UpdatedAt,
	}, nil
}

// UpdateInPlace atomically replaces all mutable fields of a draft/review question.
func (r *postgresRepository) UpdateInPlace(ctx context.Context, id string, input UpdateQuestionInput) (*Question, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("questions.UpdateInPlace: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var q Question
	const updateQ = `
		UPDATE questions
		SET category_id = $1, difficulty = $2, updated_at = now()
		WHERE id = $3
		RETURNING id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at`
	if err = tx.GetContext(ctx, &q, updateQ, input.CategoryID, input.Difficulty, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("questions.UpdateInPlace: update: %w", err)
	}

	// Replace translations.
	if _, err = tx.ExecContext(ctx, `DELETE FROM question_translations WHERE question_id = $1`, id); err != nil {
		return nil, fmt.Errorf("questions.UpdateInPlace: delete translations: %w", err)
	}
	// Replace answer options (cascades to answer_translations).
	if _, err = tx.ExecContext(ctx, `DELETE FROM answer_options WHERE question_id = $1`, id); err != nil {
		return nil, fmt.Errorf("questions.UpdateInPlace: delete answer_options: %w", err)
	}
	// Replace tags.
	if _, err = tx.ExecContext(ctx, `DELETE FROM question_tags WHERE question_id = $1`, id); err != nil {
		return nil, fmt.Errorf("questions.UpdateInPlace: delete tags: %w", err)
	}

	if _, err = applySubObjects(ctx, tx, id, input.Translations, input.AnswerOptions, input.TagIDs); err != nil {
		return nil, fmt.Errorf("questions.UpdateInPlace: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("questions.UpdateInPlace: commit: %w", err)
	}
	return &q, nil
}

// CreateVersionFull atomically archives the previous active question and creates a new draft version.
func (r *postgresRepository) CreateVersionFull(ctx context.Context, previousID string, input UpdateQuestionInput) (*Question, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("questions.CreateVersionFull: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Lock and read the previous question.
	var prev Question
	const lockQ = `
		SELECT id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at
		FROM questions WHERE id = $1 FOR UPDATE`
	if err = tx.GetContext(ctx, &prev, lockQ, previousID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("questions.CreateVersionFull: lock previous: %w", err)
	}

	// Archive previous.
	if _, err = tx.ExecContext(ctx, `UPDATE questions SET status = 'archived', updated_at = now() WHERE id = $1`, previousID); err != nil {
		return nil, fmt.Errorf("questions.CreateVersionFull: archive: %w", err)
	}

	// Insert new version.
	newQ := &Question{
		CategoryID:    input.CategoryID,
		Difficulty:    input.Difficulty,
		Type:          prev.Type,
		DefaultLocale: prev.DefaultLocale,
		Status:        "draft",
		CreatedBy:     input.UpdatedBy,
		Version:       prev.Version + 1,
		ParentID:      &previousID,
	}
	const insertQ = `
		INSERT INTO questions
			(id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at)
		VALUES
			(gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		RETURNING id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at`
	if err = tx.GetContext(ctx, newQ, insertQ,
		newQ.CategoryID, newQ.Difficulty, newQ.Type, newQ.DefaultLocale,
		newQ.Status, newQ.CreatedBy, newQ.Version, newQ.ParentID,
	); err != nil {
		return nil, fmt.Errorf("questions.CreateVersionFull: insert new: %w", err)
	}

	if _, err = applySubObjects(ctx, tx, newQ.ID, input.Translations, input.AnswerOptions, input.TagIDs); err != nil {
		return nil, fmt.Errorf("questions.CreateVersionFull: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("questions.CreateVersionFull: commit: %w", err)
	}
	return newQ, nil
}

// applySubObjectsResult holds the built detail sub-objects after insertion.
type applySubObjectsResult struct {
	translations  map[string]TranslationDetail
	answerOptions []AnswerOptionDetail
	tags          []string
}

// applySubObjects inserts translations, answer options+translations, and tags for a question
// within the provided transaction. It is used by CreateFull, UpdateInPlace, and CreateVersionFull.
func applySubObjects(ctx context.Context, tx *sqlx.Tx, questionID string, translations map[string]TranslationInput, answerOptions []AnswerOptionInput, tagIDs []string) (*applySubObjectsResult, error) {
	result := &applySubObjectsResult{
		translations:  make(map[string]TranslationDetail),
		answerOptions: make([]AnswerOptionDetail, 0, len(answerOptions)),
		tags:          make([]string, 0, len(tagIDs)),
	}

	// Translations.
	const insertTr = `
		INSERT INTO question_translations (question_id, locale, stem, explanation, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (question_id, locale) DO UPDATE
		    SET stem = EXCLUDED.stem, explanation = EXCLUDED.explanation, updated_at = now()`
	for locale, t := range translations {
		if _, err := tx.ExecContext(ctx, insertTr, questionID, locale, t.Stem, t.Explanation); err != nil {
			return nil, fmt.Errorf("insert translation %s: %w", locale, err)
		}
		result.translations[locale] = TranslationDetail{Stem: t.Stem, Explanation: t.Explanation}
	}

	// Answer options + their translations.
	const insertOpt = `
		INSERT INTO answer_options (id, question_id, sort_order, is_correct, likert_weight, likert_polarity, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, now())
		RETURNING id, sort_order, is_correct, likert_weight, likert_polarity`
	type optRow struct {
		ID             string   `db:"id"`
		SortOrder      int      `db:"sort_order"`
		IsCorrect      bool     `db:"is_correct"`
		LikertWeight   *float64 `db:"likert_weight"`
		LikertPolarity *string  `db:"likert_polarity"`
	}
	const insertAt = `
		INSERT INTO answer_translations (option_id, locale, text)
		VALUES ($1, $2, $3)
		ON CONFLICT (option_id, locale) DO UPDATE SET text = EXCLUDED.text`
	for _, optIn := range answerOptions {
		var opt optRow
		if err := tx.GetContext(ctx, &opt, insertOpt,
			questionID, optIn.SortOrder, optIn.IsCorrect, optIn.LikertWeight, optIn.LikertPolarity,
		); err != nil {
			return nil, fmt.Errorf("insert answer_option: %w", err)
		}
		optTranslations := make(map[string]AnswerTranslationDetail, len(optIn.Translations))
		for locale, at := range optIn.Translations {
			if _, err := tx.ExecContext(ctx, insertAt, opt.ID, locale, at.Text); err != nil {
				return nil, fmt.Errorf("insert answer_translation: %w", err)
			}
			optTranslations[locale] = AnswerTranslationDetail{Text: at.Text}
		}
		result.answerOptions = append(result.answerOptions, AnswerOptionDetail{
			ID:             opt.ID,
			SortOrder:      opt.SortOrder,
			IsCorrect:      opt.IsCorrect,
			LikertWeight:   opt.LikertWeight,
			LikertPolarity: opt.LikertPolarity,
			Translations:   optTranslations,
		})
	}

	// Tags.
	const insertTag = `INSERT INTO question_tags (question_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	for _, tagID := range tagIDs {
		if _, err := tx.ExecContext(ctx, insertTag, questionID, tagID); err != nil {
			return nil, fmt.Errorf("insert tag %s: %w", tagID, err)
		}
		result.tags = append(result.tags, tagID)
	}

	return result, nil
}

// questionListRow is the SQL scan target for ListFiltered.
type questionListRow struct {
	ID             string         `db:"id"`
	Type           string         `db:"type"`
	Difficulty     string         `db:"difficulty"`
	Status         string         `db:"status"`
	CategoryID     string         `db:"category_id"`
	DefaultLocale  string         `db:"default_locale"`
	Version        int            `db:"version"`
	CreatedBy      string         `db:"created_by"`
	CreatedAt      string         `db:"created_at"`
	UpdatedAt      string         `db:"updated_at"`
	StemPreview    string         `db:"stem_preview"`
	LocaleCoverage pq.StringArray `db:"locale_coverage"`
	Tags           pq.StringArray `db:"tags"`
	CategoryName   string         `db:"category_name"`
	CreatedByName  string         `db:"created_by_name"`
}

// ListFiltered returns a paginated, filtered list of questions together with the total count.
func (r *postgresRepository) ListFiltered(ctx context.Context, filter QuestionFilter) ([]*QuestionListItem, int, error) {
	// Build a safe ORDER BY clause from an allowlist.
	allowedSortCols := map[string]string{
		"created_at": "q.created_at",
		"updated_at": "q.updated_at",
		"difficulty": "q.difficulty",
	}
	sortCol, ok := allowedSortCols[filter.Sort]
	if !ok {
		sortCol = "q.created_at"
	}
	direction := "DESC"
	if strings.ToLower(filter.Order) == "asc" {
		direction = "ASC"
	}

	baseSelectQ := `
		SELECT
			q.id, q.type, q.difficulty, q.status, q.category_id, q.default_locale,
			q.version, q.created_by,
			q.created_at::text, q.updated_at::text,
			COALESCE(
				LEFT((SELECT stem FROM question_translations WHERE question_id = q.id AND locale = q.default_locale LIMIT 1), 120),
				''
			) AS stem_preview,
			COALESCE(
				(SELECT ARRAY_AGG(DISTINCT locale ORDER BY locale) FROM question_translations WHERE question_id = q.id),
				'{}'::text[]
			) AS locale_coverage,
			COALESCE(
				(SELECT ARRAY_AGG(t.name ORDER BY t.name) FROM question_tags qt_sub JOIN tags t ON qt_sub.tag_id = t.id WHERE qt_sub.question_id = q.id),
				'{}'::text[]
			) AS tags,
			COALESCE(c.name, '') AS category_name,
			COALESCE(u.full_name, '') AS created_by_name
		FROM questions q
		LEFT JOIN categories c ON q.category_id = c.id
		LEFT JOIN users u ON q.created_by = u.id
		WHERE
			($1::uuid IS NULL OR q.category_id = $1::uuid)
			AND (array_length($2::text[], 1) IS NULL OR EXISTS (SELECT 1 FROM question_tags WHERE question_id = q.id AND tag_id::text = ANY($2::text[])))
			AND (array_length($3::text[], 1) IS NULL OR q.difficulty = ANY($3::text[]))
			AND ($4::text IS NULL OR q.type = $4)
			AND (array_length($5::text[], 1) IS NULL OR q.status = ANY($5::text[]))
			AND ($6::text IS NULL OR EXISTS (SELECT 1 FROM question_translations WHERE question_id = q.id AND locale = $6))
			AND ($7::text IS NULL OR NOT EXISTS (SELECT 1 FROM question_translations WHERE question_id = q.id AND locale = $7))
			AND ($8::text IS NULL OR EXISTS (SELECT 1 FROM question_translations WHERE question_id = q.id AND locale = q.default_locale AND stem ILIKE '%' || $8 || '%'))`

	listQ := baseSelectQ + "\n\t\tORDER BY " + sortCol + " " + direction + "\n\t\tLIMIT $9 OFFSET $10"

	const countQ = `
		SELECT COUNT(*)
		FROM questions q
		WHERE
			($1::uuid IS NULL OR q.category_id = $1::uuid)
			AND (array_length($2::text[], 1) IS NULL OR EXISTS (SELECT 1 FROM question_tags WHERE question_id = q.id AND tag_id::text = ANY($2::text[])))
			AND (array_length($3::text[], 1) IS NULL OR q.difficulty = ANY($3::text[]))
			AND ($4::text IS NULL OR q.type = $4)
			AND (array_length($5::text[], 1) IS NULL OR q.status = ANY($5::text[]))
			AND ($6::text IS NULL OR EXISTS (SELECT 1 FROM question_translations WHERE question_id = q.id AND locale = $6))
			AND ($7::text IS NULL OR NOT EXISTS (SELECT 1 FROM question_translations WHERE question_id = q.id AND locale = $7))
			AND ($8::text IS NULL OR EXISTS (SELECT 1 FROM question_translations WHERE question_id = q.id AND locale = q.default_locale AND stem ILIKE '%' || $8 || '%'))`

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}
	offset := (filter.Page - 1) * filter.PerPage

	var searchParam *string
	if filter.Search != "" {
		searchParam = &filter.Search
	}

	args := []any{
		filter.CategoryID,
		pq.Array(filter.TagIDs),
		pq.Array(filter.Difficulties),
		filter.Type,
		pq.Array(filter.Statuses),
		filter.Locale,
		filter.LocaleMissing,
		searchParam,
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, fmt.Errorf("questions.ListFiltered: count: %w", err)
	}

	listArgs := append(args, filter.PerPage, offset)
	var rows []questionListRow
	if err := r.db.SelectContext(ctx, &rows, listQ, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("questions.ListFiltered: select: %w", err)
	}

	items := make([]*QuestionListItem, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		lc := []string(row.LocaleCoverage)
		if lc == nil {
			lc = []string{}
		}
		tags := []string(row.Tags)
		if tags == nil {
			tags = []string{}
		}
		items = append(items, &QuestionListItem{
			ID:             row.ID,
			Type:           row.Type,
			Difficulty:     row.Difficulty,
			Status:         row.Status,
			CategoryID:     row.CategoryID,
			CategoryName:   row.CategoryName,
			DefaultLocale:  row.DefaultLocale,
			Version:        row.Version,
			CreatedBy:      row.CreatedBy,
			CreatedByName:  row.CreatedByName,
			StemPreview:    row.StemPreview,
			LocaleCoverage: lc,
			Tags:           tags,
		})
	}
	return items, total, nil
}

// GetWithDetails returns a full QuestionDetail including translations, answer options, and tags.
func (r *postgresRepository) GetWithDetails(ctx context.Context, id string) (*QuestionDetail, error) {
	q, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Translations.
	type trRow struct {
		Locale      string  `db:"locale"`
		Stem        string  `db:"stem"`
		Explanation *string `db:"explanation"`
	}
	var trRows []trRow
	if err = r.db.SelectContext(ctx, &trRows,
		`SELECT locale, stem, explanation FROM question_translations WHERE question_id = $1 ORDER BY locale`, id,
	); err != nil {
		return nil, fmt.Errorf("questions.GetWithDetails: translations: %w", err)
	}
	translations := make(map[string]TranslationDetail, len(trRows))
	for _, t := range trRows {
		translations[t.Locale] = TranslationDetail{Stem: t.Stem, Explanation: t.Explanation}
	}

	// Answer options.
	type optRow struct {
		ID             string   `db:"id"`
		SortOrder      int      `db:"sort_order"`
		IsCorrect      bool     `db:"is_correct"`
		LikertWeight   *float64 `db:"likert_weight"`
		LikertPolarity *string  `db:"likert_polarity"`
	}
	var optRows []optRow
	if err = r.db.SelectContext(ctx, &optRows,
		`SELECT id, sort_order, is_correct, likert_weight, likert_polarity FROM answer_options WHERE question_id = $1 ORDER BY sort_order`, id,
	); err != nil {
		return nil, fmt.Errorf("questions.GetWithDetails: answer_options: %w", err)
	}

	// Answer translations (bulk fetch, group in Go).
	type atRow struct {
		OptionID string `db:"option_id"`
		Locale   string `db:"locale"`
		Text     string `db:"text"`
	}
	var atRows []atRow
	if len(optRows) > 0 {
		if err = r.db.SelectContext(ctx, &atRows,
			`SELECT option_id, locale, text FROM answer_translations
			 WHERE option_id IN (SELECT id FROM answer_options WHERE question_id = $1)
			 ORDER BY option_id, locale`, id,
		); err != nil {
			return nil, fmt.Errorf("questions.GetWithDetails: answer_translations: %w", err)
		}
	}

	atByOpt := make(map[string]map[string]AnswerTranslationDetail)
	for _, at := range atRows {
		if atByOpt[at.OptionID] == nil {
			atByOpt[at.OptionID] = make(map[string]AnswerTranslationDetail)
		}
		atByOpt[at.OptionID][at.Locale] = AnswerTranslationDetail{Text: at.Text}
	}

	answerOptions := make([]AnswerOptionDetail, 0, len(optRows))
	for _, opt := range optRows {
		tr := atByOpt[opt.ID]
		if tr == nil {
			tr = map[string]AnswerTranslationDetail{}
		}
		answerOptions = append(answerOptions, AnswerOptionDetail{
			ID:             opt.ID,
			SortOrder:      opt.SortOrder,
			IsCorrect:      opt.IsCorrect,
			LikertWeight:   opt.LikertWeight,
			LikertPolarity: opt.LikertPolarity,
			Translations:   tr,
		})
	}

	// Tags.
	tags, err := r.GetTags(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("questions.GetWithDetails: tags: %w", err)
	}
	if tags == nil {
		tags = []string{}
	}

	localeCoverage := sortedKeys(translations)

	return &QuestionDetail{
		ID:             q.ID,
		Type:           q.Type,
		Difficulty:     q.Difficulty,
		Status:         q.Status,
		CategoryID:     q.CategoryID,
		DefaultLocale:  q.DefaultLocale,
		Version:        q.Version,
		ParentID:       q.ParentID,
		LocaleCoverage: localeCoverage,
		Translations:   translations,
		AnswerOptions:  answerOptions,
		Tags:           tags,
		CreatedBy:      q.CreatedBy,
		CreatedAt:      q.CreatedAt,
		UpdatedAt:      q.UpdatedAt,
	}, nil
}

// DeleteByID hard-deletes a question row; cascades to translations, options, and tags.
func (r *postgresRepository) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM questions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("questions.DeleteByID: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrQuestionNotFound
	}
	return nil
}

// GetVersionChain returns all questions in the version chain that includes the given ID,
// ordered oldest-first. It uses a recursive CTE to first walk up to the root, then down.
func (r *postgresRepository) GetVersionChain(ctx context.Context, id string) ([]*VersionEntry, error) {
	const query = `
		WITH RECURSIVE
		up AS (
			SELECT id, parent_id FROM questions WHERE id = $1
			UNION ALL
			SELECT q.id, q.parent_id FROM questions q INNER JOIN up ON q.id = up.parent_id
		),
		root_q AS (SELECT id FROM up WHERE parent_id IS NULL),
		chain AS (
			SELECT id, version, status, created_at, created_by, parent_id
			FROM questions WHERE id = (SELECT id FROM root_q)
			UNION ALL
			SELECT q.id, q.version, q.status, q.created_at, q.created_by, q.parent_id
			FROM questions q INNER JOIN chain c ON q.parent_id = c.id
		)
		SELECT id, version, status, created_at, created_by FROM chain ORDER BY version ASC`

	var rows []*VersionEntry
	if err := r.db.SelectContext(ctx, &rows, query, id); err != nil {
		return nil, fmt.Errorf("questions.GetVersionChain: %w", err)
	}
	return rows, nil
}

// sortedKeys returns the sorted keys of a map[string]TranslationDetail.
func sortedKeys(m map[string]TranslationDetail) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
