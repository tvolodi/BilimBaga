package questions

import (
	"errors"
	"time"
)

// Sentinel errors for the questions domain.
var (
	ErrNotFound          = errors.New("not found")
	ErrQuestionNotFound  = errors.New("question not found")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrNotDraft          = errors.New("question is not in draft status")
	ErrStemRequired      = errors.New("default locale stem is required for this transition")
	ErrTagNotFound       = errors.New("tag not found")
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

// QuestionFilter holds all filter parameters for the paginated question list.
type QuestionFilter struct {
	CategoryID    *string
	TagIDs        []string // multi-value: question must have ANY of these tag IDs
	Difficulties  []string // multi-value: question difficulty must be ANY of these
	Type          *string
	Statuses      []string // multi-value: question status must be ANY of these
	Locale        *string  // filter: only questions that have a translation for this locale
	LocaleMissing *string  // filter: only questions that do NOT have a translation for this locale
	Search        string   // full-text search against the default-locale stem
	Sort          string   // one of: created_at, updated_at, difficulty
	Order         string   // asc or desc
	Page          int
	PerPage       int
}

// QuestionListItem is the summary row returned in the paginated list endpoint.
type QuestionListItem struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	Difficulty     string    `json:"difficulty"`
	Status         string    `json:"status"`
	CategoryID     string    `json:"category_id"`
	CategoryName   string    `json:"category_name,omitempty"`
	DefaultLocale  string    `json:"default_locale"`
	Version        int       `json:"version"`
	LocaleCoverage []string  `json:"locale_coverage"`
	StemPreview    string    `json:"stem_preview"`
	Tags           []string  `json:"tags"`
	CreatedBy      string    `json:"created_by"`
	CreatedByName  string    `json:"created_by_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TranslationDetail holds locale-specific text for the detail response.
type TranslationDetail struct {
	Stem        string  `json:"stem"`
	Explanation *string `json:"explanation"`
}

// AnswerTranslationDetail holds the translation text for a single locale in the detail response.
type AnswerTranslationDetail struct {
	Text string `json:"text"`
}

// AnswerOptionDetail holds full answer option data including translations.
type AnswerOptionDetail struct {
	ID             string                             `json:"id"`
	SortOrder      int                                `json:"sort_order"`
	IsCorrect      bool                               `json:"is_correct"`
	LikertWeight   *float64                           `json:"likert_weight"`
	LikertPolarity *string                            `json:"likert_polarity"`
	Translations   map[string]AnswerTranslationDetail `json:"translations"`
}

// QuestionDetail is the full question including translations, options, and tags.
type QuestionDetail struct {
	ID             string                       `json:"id"`
	Type           string                       `json:"type"`
	Difficulty     string                       `json:"difficulty"`
	Status         string                       `json:"status"`
	CategoryID     string                       `json:"category_id"`
	DefaultLocale  string                       `json:"default_locale"`
	Version        int                          `json:"version"`
	ParentID       *string                      `json:"parent_id"`
	LocaleCoverage []string                     `json:"locale_coverage"`
	Translations   map[string]TranslationDetail `json:"translations"`
	AnswerOptions  []AnswerOptionDetail         `json:"answer_options"`
	Tags           []string                     `json:"tags"`
	CreatedBy      string                       `json:"created_by"`
	CreatedAt      time.Time                    `json:"created_at"`
	UpdatedAt      time.Time                    `json:"updated_at"`
}

// VersionEntry is a single entry in a question's version history chain.
type VersionEntry struct {
	ID        string    `json:"id"         db:"id"`
	Version   int       `json:"version"    db:"version"`
	Status    string    `json:"status"     db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	CreatedBy string    `json:"created_by" db:"created_by"`
}

// AnswerTranslationInput is the locale-specific text for an answer option in create/update requests.
type AnswerTranslationInput struct {
	Text string `json:"text"`
}

// AnswerOptionInput holds input data for a single answer option in create/update requests.
type AnswerOptionInput struct {
	SortOrder      int                               `json:"sort_order"`
	IsCorrect      bool                              `json:"is_correct"`
	LikertWeight   *float64                          `json:"likert_weight"`
	LikertPolarity *string                           `json:"likert_polarity"`
	Translations   map[string]AnswerTranslationInput `json:"translations"`
}

// TranslationInput holds locale-specific stem and explanation for create/update requests.
type TranslationInput struct {
	Stem        string  `json:"stem"`
	Explanation *string `json:"explanation"`
}

// CreateQuestionFullInput bundles all fields needed to create a question and its sub-objects atomically.
type CreateQuestionFullInput struct {
	CategoryID    string
	Difficulty    string
	Type          string
	DefaultLocale string
	CreatedBy     string
	Translations  map[string]TranslationInput
	AnswerOptions []AnswerOptionInput
	TagIDs        []string
}

// UpdateQuestionInput bundles all updateable fields for a question (in-place update or auto-versioning).
type UpdateQuestionInput struct {
	CategoryID    string
	Difficulty    string
	UpdatedBy     string
	Translations  map[string]TranslationInput
	AnswerOptions []AnswerOptionInput
	TagIDs        []string
}

// ── FR-BB25: Bulk Import / Export ────────────────────────────────────────────

// ImportRow is a single parsed row from a CSV or JSON import file.
type ImportRow struct {
	// RowNumber is the 1-based row index in the source file (for error reporting).
	RowNumber     int
	Type          string
	Difficulty    string
	CategoryPath  string
	DefaultLocale string
	Translations  map[string]TranslationInput
	AnswerOptions []AnswerOptionInput
	Tags          []string
}

// ImportRowError describes validation errors for a single import row.
type ImportRowError struct {
	Row    int      `json:"row"`
	Errors []string `json:"errors"`
}

// SimilarityMatch describes a near-duplicate found during import duplicate detection.
type SimilarityMatch struct {
	QuestionID  string  `json:"question_id"`
	Score       float64 `json:"score"`
	StemPreview string  `json:"stem_preview"`
}

// ImportRowWarning describes a similarity warning for a single import row.
type ImportRowWarning struct {
	Row             int             `json:"row"`
	SimilarityMatch SimilarityMatch `json:"similarity_match"`
}

// DryRunReport is the response body for a dry-run import.
type DryRunReport struct {
	DryRun      bool               `json:"dry_run"`
	ValidCount  int                `json:"valid_count"`
	ErrorRows   []ImportRowError   `json:"error_rows"`
	WarningRows []ImportRowWarning `json:"warning_rows"`
}

// CommitResult is the response body for a successful committed import.
type CommitResult struct {
	DryRun        bool     `json:"dry_run"`
	ImportedCount int      `json:"imported_count"`
	QuestionIDs   []string `json:"question_ids"`
}

// ExportRow is a single question formatted for export (JSON array element or CSV row).
type ExportRow struct {
	Type          string                       `json:"type"`
	Difficulty    string                       `json:"difficulty"`
	CategoryPath  string                       `json:"category_path"`
	DefaultLocale string                       `json:"default_locale"`
	Translations  map[string]TranslationDetail `json:"translations"`
	AnswerOptions []AnswerOptionDetail         `json:"answer_options"`
	Tags          []string                     `json:"tags"`
}

// ExportFilter mirrors QuestionFilter plus an optional explicit ID list.
type ExportFilter struct {
	IDs          []string // when non-empty, only export these question IDs (max 100)
	CategoryID   *string
	TagIDs       []string
	Difficulties []string
	Type         *string
	Statuses     []string
	Locale       *string
}

// StemSimilarityResult is returned by the trigram batch search in the repository.
type StemSimilarityResult struct {
	QuestionID string
	Score      float64
	Stem       string
}
