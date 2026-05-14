package questions

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// ResolveCategoryPath resolves a "/"-delimited category path (e.g. "Security Awareness/Phishing")
// to a category UUID. The lookup is case-insensitive and whitespace-trimmed at each level.
func (r *postgresRepository) ResolveCategoryPath(ctx context.Context, path string) (string, error) {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}

	// Walk the tree level by level.
	var parentID *string
	var currentID string
	for i, name := range parts {
		if name == "" {
			return "", fmt.Errorf("empty segment at position %d in category_path", i)
		}
		var id string
		var err error
		if parentID == nil {
			err = r.db.GetContext(ctx, &id,
				`SELECT id FROM categories WHERE LOWER(TRIM(name)) = LOWER(TRIM($1)) AND parent_id IS NULL LIMIT 1`,
				name,
			)
		} else {
			err = r.db.GetContext(ctx, &id,
				`SELECT id FROM categories WHERE LOWER(TRIM(name)) = LOWER(TRIM($1)) AND parent_id = $2 LIMIT 1`,
				name, *parentID,
			)
		}
		if err != nil {
			return "", fmt.Errorf("category_path segment %q not found", name)
		}
		currentID = id
		parentID = &currentID
	}
	return currentID, nil
}

// FindSimilarStems runs a batch trigram similarity search against all active question stems
// for the given locale. It returns a map keyed by input stem → best matching result (score ≥ 0.8).
// If pg_trgm is not installed the function returns an empty map and logs a warning — it does not fail.
func (r *postgresRepository) FindSimilarStems(ctx context.Context, stems []string, locale string) (map[string]StemSimilarityResult, error) {
	if len(stems) == 0 {
		return map[string]StemSimilarityResult{}, nil
	}

	// Check that pg_trgm is available. If not, return empty map.
	var trgmAvailable bool
	if err := r.db.GetContext(ctx, &trgmAvailable,
		`SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm')`,
	); err != nil || !trgmAvailable {
		return map[string]StemSimilarityResult{}, nil
	}

	// For each input stem, find the best matching existing active question stem.
	// We do this in a single query using unnest to avoid N round-trips.
	const query = `
		SELECT
			input_stem,
			question_id,
			similarity_score,
			stem
		FROM (
			SELECT
				s.input_stem,
				qt.question_id,
				similarity(qt.stem, s.input_stem) AS similarity_score,
				qt.stem,
				ROW_NUMBER() OVER (PARTITION BY s.input_stem ORDER BY similarity(qt.stem, s.input_stem) DESC) AS rn
			FROM
				unnest($1::text[]) AS s(input_stem)
				JOIN question_translations qt ON qt.locale = $2
				JOIN questions q ON q.id = qt.question_id AND q.status = 'active'
			WHERE
				similarity(qt.stem, s.input_stem) >= 0.8
		) ranked
		WHERE rn = 1`

	type row struct {
		InputStem       string  `db:"input_stem"`
		QuestionID      string  `db:"question_id"`
		SimilarityScore float64 `db:"similarity_score"`
		Stem            string  `db:"stem"`
	}

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, pq.Array(stems), locale); err != nil {
		// Gracefully degrade — trigram queries are optional.
		return map[string]StemSimilarityResult{}, nil
	}

	result := make(map[string]StemSimilarityResult, len(rows))
	for _, r := range rows {
		result[r.InputStem] = StemSimilarityResult{
			QuestionID: r.QuestionID,
			Score:      r.SimilarityScore,
			Stem:       r.Stem,
		}
	}
	return result, nil
}

// ImportBatch inserts all rows in a single transaction. On any error the entire batch is rolled back.
func (r *postgresRepository) ImportBatch(ctx context.Context, rows []ImportRow, createdBy string) ([]string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("questions.ImportBatch: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	ids := make([]string, 0, len(rows))

	const insertQ = `
		INSERT INTO questions
			(id, category_id, difficulty, type, default_locale, status, created_by, version, parent_id, created_at, updated_at)
		VALUES
			(gen_random_uuid(), $1, $2, $3, $4, 'draft', $5, 1, NULL, now(), now())
		RETURNING id`

	for _, row := range rows {
		var qid string
		if err = tx.GetContext(ctx, &qid, insertQ,
			row.CategoryPath, // already resolved to category_id by the service
			row.Difficulty, row.Type, row.DefaultLocale, createdBy,
		); err != nil {
			return nil, fmt.Errorf("questions.ImportBatch: insert question row %d: %w", row.RowNumber, err)
		}

		_, err = applySubObjects(ctx, tx, qid, row.Translations, row.AnswerOptions, nil)
		if err != nil {
			return nil, fmt.Errorf("questions.ImportBatch: sub-objects row %d: %w", row.RowNumber, err)
		}

		// Insert tag IDs (stored in Tags field after service resolution).
		const insertTag = `INSERT INTO question_tags (question_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
		for _, tagID := range row.Tags {
			if _, err = tx.ExecContext(ctx, insertTag, qid, tagID); err != nil {
				return nil, fmt.Errorf("questions.ImportBatch: tag row %d: %w", row.RowNumber, err)
			}
		}

		ids = append(ids, qid)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("questions.ImportBatch: commit: %w", err)
	}
	return ids, nil
}

// StreamExport calls fn for each ExportRow matching the filter without loading all rows into memory.
func (r *postgresRepository) StreamExport(ctx context.Context, filter ExportFilter, fn func(*ExportRow) error) error {
	var (
		whereClause string
		args        []any
	)

	if len(filter.IDs) > 0 {
		whereClause = `q.id = ANY($1::uuid[])`
		args = []any{pq.Array(filter.IDs)}
	} else {
		clauses := []string{"1=1"}
		args = []any{}
		n := 1
		if filter.CategoryID != nil {
			n++
			clauses = append(clauses, fmt.Sprintf("q.category_id = $%d::uuid", n))
			args = append(args, *filter.CategoryID)
		}
		if len(filter.TagIDs) > 0 {
			n++
			clauses = append(clauses, fmt.Sprintf("EXISTS (SELECT 1 FROM question_tags WHERE question_id = q.id AND tag_id::text = ANY($%d::text[]))", n))
			args = append(args, pq.Array(filter.TagIDs))
		}
		if len(filter.Difficulties) > 0 {
			n++
			clauses = append(clauses, fmt.Sprintf("q.difficulty = ANY($%d::text[])", n))
			args = append(args, pq.Array(filter.Difficulties))
		}
		if filter.Type != nil {
			n++
			clauses = append(clauses, fmt.Sprintf("q.type = $%d", n))
			args = append(args, *filter.Type)
		}
		if len(filter.Statuses) > 0 {
			n++
			clauses = append(clauses, fmt.Sprintf("q.status = ANY($%d::text[])", n))
			args = append(args, pq.Array(filter.Statuses))
		}
		if filter.Locale != nil {
			n++
			clauses = append(clauses, fmt.Sprintf("EXISTS (SELECT 1 FROM question_translations WHERE question_id = q.id AND locale = $%d)", n))
			args = append(args, *filter.Locale)
		}
		whereClause = strings.Join(clauses, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT q.id, q.type, q.difficulty, q.default_locale,
		       COALESCE(
		           (SELECT name FROM categories c WHERE c.id = q.category_id),
		           ''
		       ) AS category_name,
		       COALESCE(
		           (SELECT STRING_AGG(parent_name || '/' || c2.name, '/' ORDER BY parent_name)
		            FROM (
		                SELECT c2.name, COALESCE(cp.name, '') AS parent_name
		                FROM categories c2
		                LEFT JOIN categories cp ON cp.id = c2.parent_id
		                WHERE c2.id = q.category_id
		            ) c2
		           ),
		           (SELECT name FROM categories WHERE id = q.category_id)
		       ) AS category_path_raw
		FROM questions q
		WHERE %s
		ORDER BY q.created_at`, whereClause)

	sqlRows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("questions.StreamExport: query: %w", err)
	}
	defer sqlRows.Close()

	type baseRow struct {
		ID              string `db:"id"`
		Type            string `db:"type"`
		Difficulty      string `db:"difficulty"`
		DefaultLocale   string `db:"default_locale"`
		CategoryName    string `db:"category_name"`
		CategoryPathRaw string `db:"category_path_raw"`
	}

	for sqlRows.Next() {
		var base baseRow
		if err := sqlRows.StructScan(&base); err != nil {
			return fmt.Errorf("questions.StreamExport: scan: %w", err)
		}

		// Resolve full category path via recursive CTE.
		categoryPath, pathErr := r.resolveCategoryPathByID(ctx, base.ID)
		if pathErr != nil {
			categoryPath = base.CategoryName
		}

		// Fetch translations.
		type trRow struct {
			Locale      string  `db:"locale"`
			Stem        string  `db:"stem"`
			Explanation *string `db:"explanation"`
		}
		var trRows []trRow
		if err2 := r.db.SelectContext(ctx, &trRows,
			`SELECT locale, stem, explanation FROM question_translations WHERE question_id = $1 ORDER BY locale`,
			base.ID,
		); err2 != nil {
			return fmt.Errorf("questions.StreamExport: translations %s: %w", base.ID, err2)
		}
		translations := make(map[string]TranslationDetail, len(trRows))
		for _, t := range trRows {
			translations[t.Locale] = TranslationDetail{Stem: t.Stem, Explanation: t.Explanation}
		}

		// Fetch answer options + translations.
		type optRow struct {
			ID             string   `db:"id"`
			SortOrder      int      `db:"sort_order"`
			IsCorrect      bool     `db:"is_correct"`
			LikertWeight   *float64 `db:"likert_weight"`
			LikertPolarity *string  `db:"likert_polarity"`
		}
		var optRows []optRow
		if err2 := r.db.SelectContext(ctx, &optRows,
			`SELECT id, sort_order, is_correct, likert_weight, likert_polarity FROM answer_options WHERE question_id = $1 ORDER BY sort_order`,
			base.ID,
		); err2 != nil {
			return fmt.Errorf("questions.StreamExport: answer_options %s: %w", base.ID, err2)
		}

		type atRow struct {
			OptionID string `db:"option_id"`
			Locale   string `db:"locale"`
			Text     string `db:"text"`
		}
		var atRows []atRow
		if len(optRows) > 0 {
			if err2 := r.db.SelectContext(ctx, &atRows,
				`SELECT option_id, locale, text FROM answer_translations
				 WHERE option_id IN (SELECT id FROM answer_options WHERE question_id = $1)
				 ORDER BY option_id, locale`,
				base.ID,
			); err2 != nil {
				return fmt.Errorf("questions.StreamExport: answer_translations %s: %w", base.ID, err2)
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

		// Fetch tag names.
		var tagNames []string
		if err2 := r.db.SelectContext(ctx, &tagNames,
			`SELECT t.name FROM question_tags qt JOIN tags t ON t.id = qt.tag_id WHERE qt.question_id = $1 ORDER BY t.name`,
			base.ID,
		); err2 != nil {
			return fmt.Errorf("questions.StreamExport: tags %s: %w", base.ID, err2)
		}
		if tagNames == nil {
			tagNames = []string{}
		}

		row := &ExportRow{
			Type:          base.Type,
			Difficulty:    base.Difficulty,
			CategoryPath:  categoryPath,
			DefaultLocale: base.DefaultLocale,
			Translations:  translations,
			AnswerOptions: answerOptions,
			Tags:          tagNames,
		}
		if err := fn(row); err != nil {
			return err
		}
	}
	return sqlRows.Err()
}

// resolveCategoryPathByID builds a "/" delimited path string for a question's category.
func (r *postgresRepository) resolveCategoryPathByID(ctx context.Context, questionID string) (string, error) {
	const query = `
		WITH RECURSIVE cat_path AS (
			SELECT c.id, c.name, c.parent_id
			FROM categories c
			JOIN questions q ON q.category_id = c.id
			WHERE q.id = $1
			UNION ALL
			SELECT p.id, p.name, p.parent_id
			FROM categories p
			JOIN cat_path cp ON p.id = cp.parent_id
		)
		SELECT name FROM cat_path ORDER BY (
			WITH RECURSIVE d AS (
				SELECT id, 0 AS depth FROM cat_path WHERE parent_id IS NULL
				UNION ALL
				SELECT cp.id, d.depth + 1 FROM cat_path cp JOIN d ON cp.parent_id = d.id
			)
			SELECT depth FROM d WHERE d.id = cat_path.id
		)`

	var names []string
	if err := r.db.SelectContext(ctx, &names, query, questionID); err != nil {
		return "", fmt.Errorf("resolveCategoryPathByID: %w", err)
	}
	return strings.Join(names, "/"), nil
}

// TagNameToID looks up a tag UUID by its name (case-insensitive).
func (r *postgresRepository) TagNameToID(ctx context.Context, name string) (string, error) {
	var id string
	err := r.db.GetContext(ctx, &id,
		`SELECT id FROM tags WHERE LOWER(name) = LOWER($1) LIMIT 1`, name)
	if err != nil {
		return "", fmt.Errorf("tag %q not found", name)
	}
	return id, nil
}

// TagIDByName resolves a slice of tag names to a name→UUID map.
func (r *postgresRepository) TagIDByName(ctx context.Context, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return map[string]string{}, nil
	}
	type tagRow struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	var rows []tagRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT id, name FROM tags WHERE LOWER(name) = ANY($1::text[])`,
		pq.Array(lowercaseAll(names)),
	); err != nil {
		return nil, fmt.Errorf("questions.TagIDByName: %w", err)
	}
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[strings.ToLower(row.Name)] = row.ID
	}
	return result, nil
}

func lowercaseAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToLower(s)
	}
	return out
}
