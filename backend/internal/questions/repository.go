package questions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
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
	const query = `SELECT tag_id FROM question_tags WHERE question_id = $1`
	var ids []string
	if err := r.db.SelectContext(ctx, &ids, query, questionID); err != nil {
		return nil, fmt.Errorf("questions.GetTags: %w", err)
	}
	return ids, nil
}
