package questions

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/upload"
)

const maxImportBatch = 500

// Import handles POST /api/v1/questions/import[?dry_run=true].
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if err := upload.ParseImportMultipart(w, r); err != nil {
		if errors.Is(err, upload.ErrFileTooLarge) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, "ERR_FILE_TOO_LARGE", "import file must not exceed 10 MB")
			return
		}
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

	// AC-3 (FR-BB64): read all bytes for size and content-type validation
	// (magic bytes, not the client-supplied Content-Type) before parsing.
	// Bound the read so an oversize upload is not fully buffered.
	rawBytes, err := io.ReadAll(io.LimitReader(file, upload.MaxCSVBytes+1))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "failed to read uploaded file")
		return
	}
	if err := upload.ValidateCSVFile(rawBytes); err != nil {
		if errors.Is(err, upload.ErrFileTooLarge) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, "ERR_FILE_TOO_LARGE", "import file must not exceed 10 MB")
		} else {
			api.WriteError(w, http.StatusUnsupportedMediaType, "ERR_INVALID_FILE_TYPE", "uploaded file must be plain-text CSV or JSON")
		}
		return
	}

	var rows []ImportRow
	ct := strings.ToLower(fh.Filename)
	switch {
	case strings.HasSuffix(ct, ".json"):
		rows, err = parseJSONImport(bytes.NewReader(rawBytes))
	default:
		rows, err = parseCSVImport(bytes.NewReader(rawBytes))
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
		if !api.ValidateUUIDList(w, "ids", filter.IDs) {
			return
		}
	} else {
		categoryID, ok := api.UUIDQuery(w, r, "category_id")
		if !ok {
			return
		}
		if categoryID != "" {
			filter.CategoryID = &categoryID
		}
		if v := r.URL.Query().Get("tag_ids"); v != "" {
			filter.TagIDs = splitCommaSeparated(v)
			if !api.ValidateUUIDList(w, "tag_ids", filter.TagIDs) {
				return
			}
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

		// ISS-199b/#219: the header must describe EVERY row, so a first pass over the
		// stream collects the union of locales and the maximum option count (it keeps
		// only those two small summaries, not the rows); the second pass writes
		// records padded to the header width.
		scan := newCSVExportScan()
		exportErr := h.svc.StreamExport(r.Context(), filter, func(row *ExportRow) error {
			scan.add(row)
			return nil
		})
		if exportErr == nil && scan.rows > 0 { // empty export: no rows, no header (unchanged)
			cols := scan.columns()
			if err := cw.Write(cols.header()); err != nil {
				exportErr = err
			}
			if exportErr == nil {
				exportErr = h.svc.StreamExport(r.Context(), filter, func(row *ExportRow) error {
					return cw.Write(cols.record(row))
				})
			}
		}
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

// Accepted CSV layouts (see normaliseCSVHeader, ISS-199):
//   legacy : type, difficulty, category_path, default_locale, stem, explanation,
//            option_1…option_N, correct, tags
//   export : the same, with stem_<loc>, explanation_<loc>, option_N_<loc> per locale
//            (exactly what GET /questions/export writes).

func parseCSVImport(r io.Reader) ([]ImportRow, error) {
	cr := csv.NewReader(r)
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true
	// ISS-199b: tolerate ragged records (legacy exports); see the extra-cell check below.
	cr.FieldsPerRecord = -1

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("CSV: failed to read header: %w", err)
	}
	layout, err := normaliseCSVHeader(header)
	if err != nil {
		return nil, err
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

		get := func(pos int) string {
			if pos >= len(record) {
				return ""
			}
			// ISS-191: undo the export-side formula guard so exports re-import cleanly.
			// ISS-199: applies to every text column, legacy or <name>_<locale>.
			return api.CSVUnsafe(strings.TrimSpace(record[pos]))
		}
		row := layout.buildRow(rowNum, get)
		// Extra empty cells beyond the header are ignored; a non-empty one is data we
		// cannot place, so the row is reported as an error instead of silently dropped.
		for i := len(header); i < len(record); i++ {
			if strings.TrimSpace(record[i]) != "" {
				row.ParseErrors = append(row.ParseErrors,
					fmt.Sprintf("row has %d cells but the header has %d columns (unexpected value in column %d)", len(record), len(header), i+1))
				break
			}
		}
		rows = append(rows, row)
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
			translations[locale] = TranslationInput(td)
		}
		options := make([]AnswerOptionInput, 0, len(jr.AnswerOptions))
		for _, opt := range jr.AnswerOptions {
			optTr := make(map[string]AnswerTranslationInput, len(opt.Translations))
			for locale, at := range opt.Translations {
				optTr[locale] = AnswerTranslationInput(at)
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

// csvExportScan accumulates, across all rows, the locales and option count that
// determine the export header.
type csvExportScan struct {
	locales map[string]struct{}
	maxOpts int
	rows    int
}

func newCSVExportScan() *csvExportScan {
	return &csvExportScan{locales: map[string]struct{}{}}
}

func (s *csvExportScan) add(row *ExportRow) {
	s.rows++
	for loc := range row.Translations {
		s.locales[loc] = struct{}{}
	}
	for _, opt := range row.AnswerOptions {
		for loc := range opt.Translations {
			s.locales[loc] = struct{}{}
		}
	}
	if n := len(row.AnswerOptions); n > s.maxOpts {
		s.maxOpts = n
	}
}

// csvExportColumns is the fixed column layout of one export.
type csvExportColumns struct {
	locales []string // sorted alphabetically (stable, documented order)
	maxOpts int
}

func (s *csvExportScan) columns() *csvExportColumns {
	locs := make([]string, 0, len(s.locales))
	for l := range s.locales {
		locs = append(locs, l)
	}
	sort.Strings(locs)
	return &csvExportColumns{locales: locs, maxOpts: s.maxOpts}
}

// header: type, difficulty, category_path, default_locale, stem_<loc>…,
// explanation_<loc>…, option_N_<loc>… (N = 1..maxOpts over all rows), correct, tags.
// Locales are the sorted union over all exported questions.
func (c *csvExportColumns) header() []string {
	cols := []string{"type", "difficulty", "category_path", "default_locale"}
	for _, loc := range c.locales {
		cols = append(cols, "stem_"+loc)
	}
	for _, loc := range c.locales {
		cols = append(cols, "explanation_"+loc)
	}
	for n := 1; n <= c.maxOpts; n++ {
		for _, loc := range c.locales {
			cols = append(cols, fmt.Sprintf("option_%d_%s", n, loc))
		}
	}
	return append(cols, "correct", "tags")
}

// record renders one row at exactly header() width. Missing locales / options are
// empty cells. Text cells get the ISS-191 formula guard; padding stays empty.
// Anything outside the scanned layout (data that appeared between the two passes)
// is dropped rather than misaligning the row.
func (c *csvExportColumns) record(row *ExportRow) []string {
	rec := make([]string, 0, 6+len(c.locales)*(2+c.maxOpts))
	rec = append(rec, row.Type, row.Difficulty, row.CategoryPath, row.DefaultLocale)
	for _, loc := range c.locales {
		rec = append(rec, row.Translations[loc].Stem)
	}
	for _, loc := range c.locales {
		expl := ""
		if e := row.Translations[loc].Explanation; e != nil {
			expl = *e
		}
		rec = append(rec, expl)
	}
	for n := 1; n <= c.maxOpts; n++ {
		for _, loc := range c.locales {
			text := ""
			if n <= len(row.AnswerOptions) {
				text = row.AnswerOptions[n-1].Translations[loc].Text
			}
			rec = append(rec, text)
		}
	}
	var correctIdxs []string
	for n, opt := range row.AnswerOptions {
		if opt.IsCorrect && n < c.maxOpts {
			correctIdxs = append(correctIdxs, strconv.Itoa(n+1))
		}
	}
	rec = append(rec, strings.Join(correctIdxs, ","), strings.Join(row.Tags, ";"))
	return api.CSVSafeRecord(rec)
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
