package questions

import (
	"context"
	"fmt"
	"strings"
)

var (
	validQuestionTypes  = map[string]bool{"single": true, "multiple": true, "truefalse": true, "likert": true, "shorttext": true}
	validDifficultyVals = map[string]bool{"easy": true, "medium": true, "hard": true}
)

// ValidateAndImport validates rows, runs duplicate detection, and either returns a dry-run
// report or commits the batch and returns the created question IDs.
func (s *service) ValidateAndImport(ctx context.Context, rows []ImportRow, dryRun bool, createdBy string) (*DryRunReport, *CommitResult, error) {
	var errorRows []ImportRowError
	var warningRows []ImportRowWarning
	var validRows []ImportRow

	// ── 1. Collect all stems for batch trigram search ────────────────────────
	type stemKey struct {
		stem   string
		locale string
		rowIdx int
	}
	var stemKeys []stemKey
	for i, row := range rows {
		if tr, ok := row.Translations[row.DefaultLocale]; ok && tr.Stem != "" {
			stemKeys = append(stemKeys, stemKey{stem: tr.Stem, locale: row.DefaultLocale, rowIdx: i})
		}
	}

	// Group stems by locale for batch similarity search.
	stemsByLocale := make(map[string][]string)
	for _, sk := range stemKeys {
		stemsByLocale[sk.locale] = append(stemsByLocale[sk.locale], sk.stem)
	}
	similarityByLocale := make(map[string]map[string]StemSimilarityResult)
	for locale, stems := range stemsByLocale {
		res, err := s.repo.FindSimilarStems(ctx, stems, locale)
		if err != nil {
			// Non-fatal — proceed without duplicate detection.
			res = map[string]StemSimilarityResult{}
		}
		similarityByLocale[locale] = res
	}

	// ── 2. Validate each row ─────────────────────────────────────────────────
	for i, row := range rows {
		var errs []string

		if !validQuestionTypes[row.Type] {
			errs = append(errs, fmt.Sprintf("type: invalid value %q (must be single|multiple|truefalse|likert|shorttext)", row.Type))
		}
		if !validDifficultyVals[row.Difficulty] {
			errs = append(errs, fmt.Sprintf("difficulty: invalid value %q (must be easy|medium|hard)", row.Difficulty))
		}
		if row.DefaultLocale == "" {
			errs = append(errs, "default_locale: required")
		}
		if row.CategoryPath == "" {
			errs = append(errs, "category_path: required")
		}

		// Validate stem present for default locale.
		var stem string
		if row.DefaultLocale != "" {
			tr, ok := row.Translations[row.DefaultLocale]
			if !ok || tr.Stem == "" {
				errs = append(errs, fmt.Sprintf("stem: required for default_locale %q", row.DefaultLocale))
			} else {
				stem = tr.Stem
			}
		}

		// Validate minimum option count.
		switch row.Type {
		case "single", "multiple", "truefalse":
			if len(row.AnswerOptions) < 2 {
				errs = append(errs, "answer_options: at least 2 required for this question type")
			}
		case "likert":
			if len(row.AnswerOptions) < 2 {
				errs = append(errs, "answer_options: at least 2 required for likert questions")
			}
		}

		// Resolve category_path to category_id (store resolved ID back in CategoryPath field).
		if row.CategoryPath != "" && len(errs) == 0 {
			catID, err := s.repo.ResolveCategoryPath(ctx, row.CategoryPath)
			if err != nil {
				errs = append(errs, fmt.Sprintf("category_path: %q not found", row.CategoryPath))
			} else {
				rows[i].CategoryPath = catID
			}
		}

		if len(errs) > 0 {
			errorRows = append(errorRows, ImportRowError{Row: row.RowNumber, Errors: errs})
			continue
		}

		// ── 3. Trigram similarity check ──────────────────────────────────────
		if stem != "" && row.DefaultLocale != "" {
			if simMap, ok := similarityByLocale[row.DefaultLocale]; ok {
				if match, found := simMap[stem]; found {
					warningRows = append(warningRows, ImportRowWarning{
						Row: row.RowNumber,
						SimilarityMatch: SimilarityMatch{
							QuestionID:  match.QuestionID,
							Score:       match.Score,
							StemPreview: match.Stem,
						},
					})
				}
			}
		}

		validRows = append(validRows, rows[i])
	}

	// Ensure nil slices become empty slices for clean JSON.
	if errorRows == nil {
		errorRows = []ImportRowError{}
	}
	if warningRows == nil {
		warningRows = []ImportRowWarning{}
	}

	report := &DryRunReport{
		DryRun:      true,
		ValidCount:  len(validRows),
		ErrorRows:   errorRows,
		WarningRows: warningRows,
	}

	if dryRun {
		return report, nil, nil
	}

	// ── 4. Commit mode — any validation errors abort the whole batch ─────────
	if len(errorRows) > 0 {
		return nil, nil, &importValidationError{
			validCount: len(validRows),
			errorRows:  errorRows,
			warningRows: warningRows,
		}
	}

	// Resolve tag names to IDs across all valid rows.
	allTagNames := collectTagNames(validRows)
	tagIDMap, err := s.repo.TagIDByName(ctx, allTagNames)
	if err != nil {
		return nil, nil, fmt.Errorf("questions: ValidateAndImport: resolve tags: %w", err)
	}
	for i, row := range validRows {
		resolved := make([]string, 0, len(row.Tags))
		for _, name := range row.Tags {
			id, ok := tagIDMap[strings.ToLower(name)]
			if ok {
				resolved = append(resolved, id)
			}
		}
		validRows[i].Tags = resolved
	}

	ids, err := s.repo.ImportBatch(ctx, validRows, createdBy)
	if err != nil {
		return nil, nil, fmt.Errorf("questions: ValidateAndImport: commit: %w", err)
	}

	return nil, &CommitResult{
		DryRun:        false,
		ImportedCount: len(ids),
		QuestionIDs:   ids,
	}, nil
}

// StreamExport delegates to the repository streaming export.
func (s *service) StreamExport(ctx context.Context, filter ExportFilter, fn func(*ExportRow) error) error {
	return s.repo.StreamExport(ctx, filter, fn)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func collectTagNames(rows []ImportRow) []string {
	seen := make(map[string]bool)
	var names []string
	for _, r := range rows {
		for _, t := range r.Tags {
			key := strings.ToLower(t)
			if !seen[key] {
				seen[key] = true
				names = append(names, t)
			}
		}
	}
	return names
}

// importValidationError is returned when a commit-mode import has validation errors.
type importValidationError struct {
	validCount  int
	errorRows   []ImportRowError
	warningRows []ImportRowWarning
}

func (e *importValidationError) Error() string {
	return fmt.Sprintf("%d rows failed validation; no questions were imported", len(e.errorRows))
}
