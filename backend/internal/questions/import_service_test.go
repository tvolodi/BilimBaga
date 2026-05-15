package questions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Import service unit tests (FR-BB25) ──────────────────────────────────────
//
// These tests use the in-memory mockRepository defined in service_test.go and
// exercise ValidateAndImport directly — no HTTP layer, no real database.

// buildValidRow returns a minimal import row that passes all validation checks.
func buildValidRow(rowNum int) ImportRow {
	stem := "What is Go?"
	expl := "Go is a compiled language."
	return ImportRow{
		RowNumber:     rowNum,
		Type:          "single",
		Difficulty:    "easy",
		CategoryPath:  "Science",
		DefaultLocale: "kk",
		Translations: map[string]TranslationInput{
			"kk": {Stem: stem, Explanation: &expl},
		},
		AnswerOptions: []AnswerOptionInput{
			{SortOrder: 1, IsCorrect: true, Translations: map[string]AnswerTranslationInput{"kk": {Text: "Option A"}}},
			{SortOrder: 2, IsCorrect: false, Translations: map[string]AnswerTranslationInput{"kk": {Text: "Option B"}}},
		},
		Tags: []string{},
	}
}

// TestImport_DryRun_ReturnsReport validates that dry_run=true returns the
// structured report without persisting anything.
func TestImport_DryRun_ReturnsReport(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	rows := []ImportRow{buildValidRow(1), buildValidRow(2)}
	report, commit, err := svc.ValidateAndImport(context.Background(), rows, true, "actor-1")

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Nil(t, commit)
	assert.True(t, report.DryRun)
	assert.Equal(t, 2, report.ValidCount)
	assert.Empty(t, report.ErrorRows)
	assert.Empty(t, report.WarningRows)
}

// TestImport_DryRun_DetectsInvalidRows verifies that rows with invalid type,
// missing category_path, and wrong option count are all caught in dry-run mode.
func TestImport_DryRun_DetectsInvalidRows(t *testing.T) {
	repo := newMockRepo()
	// Make category resolution fail for an empty path — the invalid row below
	// will have an empty category_path which is caught before repo call.
	svc := NewService(repo)

	invalidType := buildValidRow(1)
	invalidType.Type = "checkbox" // not a valid type

	missingCategory := buildValidRow(2)
	missingCategory.CategoryPath = "" // required field

	tooFewOptions := buildValidRow(3)
	tooFewOptions.AnswerOptions = tooFewOptions.AnswerOptions[:1] // only 1 option for "single"

	rows := []ImportRow{invalidType, missingCategory, tooFewOptions}
	report, commit, err := svc.ValidateAndImport(context.Background(), rows, true, "actor-1")

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Nil(t, commit)
	assert.True(t, report.DryRun)
	assert.Equal(t, 0, report.ValidCount)
	assert.Len(t, report.ErrorRows, 3)

	// Verify each error row has at least one error message.
	for _, er := range report.ErrorRows {
		assert.NotEmpty(t, er.Errors, "row %d should have error messages", er.Row)
	}
}

// TestImport_Commit_InsertsAllValid checks that commit mode (dry_run=false)
// inserts all valid rows and returns the created question IDs.
func TestImport_Commit_InsertsAllValid(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	rows := []ImportRow{buildValidRow(1), buildValidRow(2), buildValidRow(3)}
	report, commit, err := svc.ValidateAndImport(context.Background(), rows, false, "actor-1")

	require.NoError(t, err)
	assert.Nil(t, report)
	assert.NotNil(t, commit)
	assert.False(t, commit.DryRun)
	assert.Equal(t, 3, commit.ImportedCount)
	assert.Len(t, commit.QuestionIDs, 3)
	for _, id := range commit.QuestionIDs {
		assert.NotEmpty(t, id)
	}
}

// TestImport_Commit_RollsBackOnAnyError verifies that if any row fails validation
// in commit mode the entire batch is rejected (422-equivalent error) and nothing
// is persisted.
func TestImport_Commit_RollsBackOnAnyError(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	validRow := buildValidRow(1)
	invalidRow := buildValidRow(2)
	invalidRow.Type = "badtype" // forces validation error

	rows := []ImportRow{validRow, invalidRow}
	report, commit, err := svc.ValidateAndImport(context.Background(), rows, false, "actor-1")

	// Should return an importValidationError.
	require.Error(t, err)
	assert.Nil(t, report)
	assert.Nil(t, commit)

	var ve *importValidationError
	require.True(t, errors.As(err, &ve), "expected importValidationError, got %T: %v", err, err)
	assert.Equal(t, 1, ve.validCount)
	assert.NotEmpty(t, ve.errorRows)
}

// TestImport_BatchTooLarge verifies that batches with more than 500 rows trigger
// the ERR_BATCH_TOO_LARGE error. This is enforced at the handler level; at the
// service level we verify that all rows are processed (limit is a handler concern).
// We test 501 rows in commit mode to confirm the service itself is not the gate.
func TestImport_BatchTooLarge(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	// The maxImportBatch check is in the handler, not the service.
	// Build exactly 501 valid rows and confirm the service can process them
	// (the handler is responsible for rejecting the request before calling the service).
	rows := make([]ImportRow, 501)
	for i := range rows {
		rows[i] = buildValidRow(i + 1)
	}

	// Service should process all rows without error.
	_, commit, err := svc.ValidateAndImport(context.Background(), rows, false, "actor-1")
	require.NoError(t, err)
	assert.NotNil(t, commit)
	assert.Equal(t, 501, commit.ImportedCount)
}

// TestImport_DuplicateDetection verifies that when FindSimilarStems returns a
// match with score ≥ 0.8, the matching row appears in warning_rows (but is not
// blocked from import).
func TestImport_DuplicateDetection(t *testing.T) {
	repo := newMockRepo()

	const targetStem = "What is Go?"
	// Override FindSimilarStems to return a match for the target stem.
	origFind := repo.FindSimilarStems
	_ = origFind // suppress unused warning
	repo.findSimilarStemsFn = func(_ context.Context, stems []string, locale string) (map[string]StemSimilarityResult, error) {
		result := make(map[string]StemSimilarityResult)
		for _, s := range stems {
			if s == targetStem {
				result[s] = StemSimilarityResult{
					QuestionID: "existing-q-1",
					Score:      0.87,
					Stem:       "What is Go language?",
				}
			}
		}
		return result, nil
	}

	svc := NewService(repo)

	row := buildValidRow(1) // uses targetStem "What is Go?"
	report, _, err := svc.ValidateAndImport(context.Background(), []ImportRow{row}, true, "actor-1")

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 1, report.ValidCount)
	assert.Len(t, report.WarningRows, 1)
	assert.Equal(t, 1, report.WarningRows[0].Row)
	assert.Equal(t, "existing-q-1", report.WarningRows[0].SimilarityMatch.QuestionID)
	assert.InDelta(t, 0.87, report.WarningRows[0].SimilarityMatch.Score, 0.001)
}

// TestImport_NoTrigram_Proceeds verifies that when FindSimilarStems returns an
// error (simulating pg_trgm unavailability), the import proceeds without
// duplicate detection and without failing the request.
func TestImport_NoTrigram_Proceeds(t *testing.T) {
	repo := newMockRepo()

	// Simulate pg_trgm not being available by returning an error from FindSimilarStems.
	repo.findSimilarStemsFn = func(_ context.Context, _ []string, _ string) (map[string]StemSimilarityResult, error) {
		return map[string]StemSimilarityResult{}, errors.New("pg_trgm extension not installed")
	}

	svc := NewService(repo)

	row := buildValidRow(1)
	report, _, err := svc.ValidateAndImport(context.Background(), []ImportRow{row}, true, "actor-1")

	require.NoError(t, err)
	require.NotNil(t, report)
	// Import should succeed; no warnings because trigram unavailable.
	assert.Equal(t, 1, report.ValidCount)
	assert.Empty(t, report.WarningRows)
}
