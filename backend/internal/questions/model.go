package questions

import (
	"errors"
	"time"
)

// Sentinel errors for the questions domain.
var (
	ErrNotFound         = errors.New("not found")
	ErrQuestionNotFound = errors.New("question not found")
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidInput     = errors.New("invalid input")
)

// Question is the core metadata row — no user-visible text.
type Question struct {
	ID            string    `db:"id"`
	CategoryID    string    `db:"category_id"`
	Difficulty    string    `db:"difficulty"`
	Type          string    `db:"type"`
	DefaultLocale string    `db:"default_locale"`
	Status        string    `db:"status"`
	CreatedBy     string    `db:"created_by"`
	Version       int       `db:"version"`
	ParentID      *string   `db:"parent_id"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// QuestionTranslation holds locale-specific text for a question.
type QuestionTranslation struct {
	QuestionID  string    `db:"question_id"`
	Locale      string    `db:"locale"`
	Stem        string    `db:"stem"`
	Explanation *string   `db:"explanation"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// AnswerOption holds structure-only data for a single answer choice.
type AnswerOption struct {
	ID             string    `db:"id"`
	QuestionID     string    `db:"question_id"`
	SortOrder      int       `db:"sort_order"`
	IsCorrect      bool      `db:"is_correct"`
	LikertWeight   *float64  `db:"likert_weight"`
	LikertPolarity *string   `db:"likert_polarity"`
	CreatedAt      time.Time `db:"created_at"`
}

// AnswerTranslation holds locale-specific text for an answer option.
type AnswerTranslation struct {
	OptionID string `db:"option_id"`
	Locale   string `db:"locale"`
	Text     string `db:"text"`
}

// QuestionTag is the join between a question and a tag.
type QuestionTag struct {
	QuestionID string `db:"question_id"`
	TagID      string `db:"tag_id"`
}
