package questions

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
)

const maxImportBatch = 500

// Import handles POST /api/v1/questions/import[?dry_run=true].
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "multipart/form-data required")
		return
	}

	file, fh, err := r.FormFile("file")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "field 'file' is required")
		return
	}
	defer file.Close()

	dryRun := r.URL.Query().Get("dry_run") == "true"
	actorID := actorFromCtx(r)

	var rows []ImportRow
	ct := strings.ToLower(fh.Filename)
	switch {
	case strings.HasSuffix(ct, ".json"):
		rows, err = parseJSONImport(file)
	default:
		rows, err = parseCSVImport(file)
	}
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", err.Error())
		return
	}

	if len(rows) > maxImportBatch {
		api.WriteJSON(w, http.StatusRequestEntityTooLarge, map[string]any{
			"data": nil,
			"error": map[string]any{
				"code":    "ERR_BATCH_TOO_LARGE",
				"message": fmt.Sprintf("Import batch exceeds maximum of %d questions", maxImportBatch),
			},
		})
		return
	}

	dryReport, commitResult, err := h.svc.ValidateAndImport(r.Context(), rows, dryRun, actorID)
	if err != nil {
		var ve *importValidationError
		if errors.As(err, &ve) {
			api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"data": nil,
				"error": map[string]any{
					"code":    "ERR_IMPORT_VALIDATION",
					"message": ve.Error(),
					"details": map[string]any{
						"valid_count": ve.validCount,
						"error_rows":  ve.errorRows,
						"warning_rows": ve.warningRows,
					},
				},
			})
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "import failed")
		return
	}

	if dryRun {
		h.writer.Write(r.Context(), r, "question.import_dry_run", "question_import", nil, map[string]any{
			"valid_count": dryReport.ValidCount,
			"error_count": len(dryReport.ErrorRows),
			"format":      formatFromFilename(fh.Filename),
		})
		api.WriteJSON(w, http.StatusOK, map[string]any{"data": dryReport, "error": nil})
		return
	}

	h.writer.Write(r.Context(), r, "question.import", "question_import", nil, map[string]any{
		"imported_count": commitResult.ImportedCount,
		"format":         formatFromFilename(fh.Filename),
	})
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": commitResult, "error": nil})
}

// Export handles GET /api/v1/questions/export.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	filter := ExportFilter{}

	if ids := r.URL.Query().Get("ids"); ids != "" {
		filter.IDs = splitCommaSeparated(ids)
		if len(filter.IDs) > 100 {
			api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_PARAM", "ids: maximum 100 IDs allowed")
			return
		}
	} else {
		if v := r.URL.Query().Get("category_id"); v != "" {
			filter.CategoryID = &v
		}
		if v := r.URL.Query().Get("tag_ids"); v != "" {
			filter.TagIDs = splitCommaSeparated(v)
		}
		if v := r.URL.Query().Get("difficulties"); v != "" {
			filter.Difficulties = splitCommaSeparated(v)
		}
		if v := r.URL.Query().Get("type"); v != "" {
			filter.Type = &v
		}
		if v := r.URL.Query().Get("statuses"); v != "" {
			filter.Statuses = splitCommaSeparated(v)
		}
		if v := r.URL.Query().Get("locale"); v != "" {
			filter.Locale = &v
		}
	}

	accept := r.Header.Get("Accept")
	date := time.Now().UTC().Format("2006-01-02")

	var errs []error
	switch {
	case strings.Contains(accept, "text/csv"):
		filename := fmt.Sprintf("questions-export-%s.csv", date)
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

		cw := csv.NewWriter(w)
		headerWritten := false

		exportErr := h.svc.StreamExport(r.Context(), filter, func(row *ExportRow) error {
			if !headerWritten {
				cw.Write(buildCSVHeader(row))
				headerWritten = true
			}
			cw.Write(rowToCSV(row))
			return nil
		})
		cw.Flush()
		if exportErr != nil {
			errs = append(errs, exportErr)
		}

	default: // application/json
		filename := fmt.Sprintf("questions-export-%s.json", date)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

		enc := json.NewEncoder(w)
		first := true
		fmt.Fprint(w, "[")
		exportErr := h.svc.StreamExport(r.Context(), filter, func(row *ExportRow) error {
			if !first {
				fmt.Fprint(w, ",")
			}
			first = false
			return enc.Encode(row)
		})
		fmt.Fprint(w, "]")
		if exportErr != nil {
			errs = append(errs, exportErr)
		}
	}

	if len(errs) == 0 {
		h.writer.Write(r.Context(), r, "question.export", "question_export", nil, map[string]any{
			"format":  formatFromAccept(r.Header.Get("Accept")),
			"filter":  r.URL.RawQuery,
		})
	}
}

// ── CSV parsing ───────────────────────────────────────────────────────────────

// CSV column order: type, difficulty, category_path, default_locale, stem, explanation,
//   option_1…option_N, correct, tags
// Multi-locale columns not supported on import (only single-locale stem/explanation per row).

func parseCSVImport(r io.Reader) ([]ImportRow, error) {
	cr := csv.NewReader(r)
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("CSV: failed to read header: %w", err)
	}
	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	required := []string{"type", "difficulty", "category_path", "default_locale", "stem"}
	for _, col := range required {
		if _, ok := colIdx[col]; !ok {
			return nil, fmt.Errorf("CSV: missing required column %q", col)
		}
	}

	var rows []ImportRow
	rowNum := 1
	for {
		record, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV row %d: parse error: %w", rowNum+1, err)
		}
		rowNum++

		get := func(col string) string {
			idx, ok := colIdx[col]
			if !ok || idx >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[idx])
		}

		locale := get("default_locale")
		stem := get("stem")
		expl := get("explanation")

		translations := map[string]TranslationInput{}
		if locale != "" && stem != "" {
			ti := TranslationInput{Stem: stem}
			if expl != "" {
				ti.Explanation = &expl
			}
			translations[locale] = ti
		}

		// Parse options: option_1, option_2, … up to whatever columns exist.
		var options []AnswerOptionInput
		correctRaw := get("correct")
		correctIdxs := parseCorrectIndices(correctRaw)

		for n := 1; ; n++ {
			colName := fmt.Sprintf("option_%d", n)
			text := get(colName)
			if text == "" {
				break
			}
			isCorrect := correctIdxs[n]
			options = append(options, AnswerOptionInput{
				SortOrder: n,
				IsCorrect: isCorrect,
				Translations: map[string]AnswerTranslationInput{
					locale: {Text: text},
				},
			})
		}

		// Tags: semicolon-separated in the tags column.
		var tags []string
		if raw := get("tags"); raw != "" {
			for _, t := range strings.Split(raw, ";") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		rows = append(rows, ImportRow{
			RowNumber:     rowNum,
			Type:          get("type"),
			Difficulty:    get("difficulty"),
			CategoryPath:  get("category_path"),
			DefaultLocale: locale,
			Translations:  translations,
			AnswerOptions: options,
			Tags:          tags,
		})
	}
	return rows, nil
}

// parseCorrectIndices converts a comma-separated list of 1-based option indices (e.g. "1,3")
// to a set of ints for fast lookup.
func parseCorrectIndices(raw string) map[int]bool {
	result := make(map[int]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err == nil {
			result[n] = true
		}
	}
	return result
}

// ── JSON parsing ──────────────────────────────────────────────────────────────

type jsonImportRow struct {
	Type          string                       `json:"type"`
	Difficulty    string                       `json:"difficulty"`
	CategoryPath  string                       `json:"category_path"`
	DefaultLocale string                       `json:"default_locale"`
	Translations  map[string]TranslationDetail `json:"translations"`
	AnswerOptions []jsonAnswerOption           `json:"answer_options"`
	Tags          []string                     `json:"tags"`
}

type jsonAnswerOption struct {
	SortOrder      int                                `json:"sort_order"`
	IsCorrect      bool                               `json:"is_correct"`
	LikertWeight   *float64                           `json:"likert_weight"`
	LikertPolarity *string                            `json:"likert_polarity"`
	Translations   map[string]AnswerTranslationDetail `json:"translations"`
}

func parseJSONImport(r io.Reader) ([]ImportRow, error) {
	var raw []jsonImportRow
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("JSON: %w", err)
	}
	rows := make([]ImportRow, 0, len(raw))
	for i, jr := range raw {
		translations := make(map[string]TranslationInput, len(jr.Translations))
		for locale, td := range jr.Translations {
			translations[locale] = TranslationInput{Stem: td.Stem, Explanation: td.Explanation}
		}
		options := make([]AnswerOptionInput, 0, len(jr.AnswerOptions))
		for _, opt := range jr.AnswerOptions {
			optTr := make(map[string]AnswerTranslationInput, len(opt.Translations))
			for locale, at := range opt.Translations {
				optTr[locale] = AnswerTranslationInput{Text: at.Text}
			}
			options = append(options, AnswerOptionInput{
				SortOrder:      opt.SortOrder,
				IsCorrect:      opt.IsCorrect,
				LikertWeight:   opt.LikertWeight,
				LikertPolarity: opt.LikertPolarity,
				Translations:   optTr,
			})
		}
		tags := jr.Tags
		if tags == nil {
			tags = []string{}
		}
		rows = append(rows, ImportRow{
			RowNumber:     i + 1,
			Type:          jr.Type,
			Difficulty:    jr.Difficulty,
			CategoryPath:  jr.CategoryPath,
			DefaultLocale: jr.DefaultLocale,
			Translations:  translations,
			AnswerOptions: options,
			Tags:          tags,
		})
	}
	return rows, nil
}

// ── CSV export helpers ────────────────────────────────────────────────────────

// buildCSVHeader produces the header row for exported CSV.
// The column order matches AC-5/AC-8:
// type, difficulty, category_path, default_locale, stem_{locale}…, explanation_{locale}…,
// option_N_{locale}…, correct, tags
func buildCSVHeader(sample *ExportRow) []string {
	locales := sortedLocales(sample.Translations)
	maxOpts := len(sample.AnswerOptions)

	var cols []string
	cols = append(cols, "type", "difficulty", "category_path", "default_locale")
	for _, loc := range locales {
		cols = append(cols, "stem_"+loc)
	}
	for _, loc := range locales {
		cols = append(cols, "explanation_"+loc)
	}
	for n := 1; n <= maxOpts; n++ {
		for _, loc := range locales {
			cols = append(cols, fmt.Sprintf("option_%d_%s", n, loc))
		}
	}
	cols = append(cols, "correct", "tags")
	return cols
}

func rowToCSV(row *ExportRow) []string {
	locales := sortedLocales(row.Translations)
	maxOpts := len(row.AnswerOptions)

	var rec []string
	rec = append(rec, row.Type, row.Difficulty, row.CategoryPath, row.DefaultLocale)

	for _, loc := range locales {
		rec = append(rec, row.Translations[loc].Stem)
	}
	for _, loc := range locales {
		expl := ""
		if e := row.Translations[loc].Explanation; e != nil {
			expl = *e
		}
		rec = append(rec, expl)
	}
	for n := 1; n <= maxOpts; n++ {
		opt := row.AnswerOptions[n-1]
		for _, loc := range locales {
			text := ""
			if at, ok := opt.Translations[loc]; ok {
				text = at.Text
			}
			rec = append(rec, text)
		}
	}

	// correct: 1-based indices of correct options.
	var correctIdxs []string
	for n, opt := range row.AnswerOptions {
		if opt.IsCorrect {
			correctIdxs = append(correctIdxs, strconv.Itoa(n+1))
		}
	}
	rec = append(rec, strings.Join(correctIdxs, ","))
	rec = append(rec, strings.Join(row.Tags, ";"))
	return rec
}

func sortedLocales(m map[string]TranslationDetail) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — locale count is tiny (2-3).
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func formatFromFilename(name string) string {
	if strings.HasSuffix(strings.ToLower(name), ".json") {
		return "json"
	}
	return "csv"
}

func formatFromAccept(accept string) string {
	if strings.Contains(accept, "text/csv") {
		return "csv"
	}
	return "json"
}
