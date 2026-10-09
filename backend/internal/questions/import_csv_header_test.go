package questions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── normaliseCSVHeader (pure) ─────────────────────────────────────────────────

func TestNormaliseCSVHeader_Table(t *testing.T) {
	base := []string{"type", "difficulty", "category_path", "default_locale"}
	with := func(extra ...string) []string { return append(append([]string{}, base...), extra...) }

	tests := []struct {
		name       string
		header     []string
		wantErr    []string // substrings that must all appear
		wantNoErr  bool
		stemLocs   []string
		maxOpt     int
		optLocs    map[int][]string
		notInError []string
	}{
		{name: "legacy", header: with("stem", "explanation", "option_1", "option_2", "correct", "tags"),
			wantNoErr: true, stemLocs: []string{""}, maxOpt: 2, optLocs: map[int][]string{1: {""}, 2: {""}}},
		{name: "export layout", header: with("stem_en", "stem_ru", "explanation_en", "explanation_ru", "option_1_en", "option_1_ru", "option_2_en", "option_2_ru", "correct", "tags"),
			wantNoErr: true, stemLocs: []string{"en", "ru"}, maxOpt: 2, optLocs: map[int][]string{1: {"en", "ru"}, 2: {"en", "ru"}}},
		{name: "mixed legacy and locale", header: with("stem", "stem_kk", "option_1", "option_1_kk"),
			wantNoErr: true, stemLocs: []string{"", "kk"}, maxOpt: 1, optLocs: map[int][]string{1: {"", "kk"}}},
		{name: "case, spaces and BOM tolerated", header: []string{"\xef\xbb\xbfType", " Difficulty ", "CATEGORY_PATH", "default_locale", "STEM_EN"},
			wantNoErr: true, stemLocs: []string{"EN"}},
		{name: "regional locale", header: with("stem_pt-BR"), wantNoErr: true, stemLocs: []string{"pt-BR"}},
		{name: "no stem column names stem", header: with("option_1"),
			wantErr: []string{"missing required column", `"stem"`, "stem_<locale>"}, notInError: []string{`"type"`, `"difficulty"`}},
		{name: "only stem missing is named alone", header: with("explanation_en"), wantErr: []string{`"stem"`}, notInError: []string{`"category_path"`}},
		{name: "several missing all named", header: []string{"difficulty", "stem_en"},
			wantErr: []string{"columns", `"type"`, `"category_path"`, `"default_locale"`}, notInError: []string{`"difficulty"`, `"stem"`}},
		{name: "garbage header", header: []string{"garbage", "data"},
			wantErr: []string{`"type"`, `"difficulty"`, `"category_path"`, `"default_locale"`, `"stem"`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, err := normaliseCSVHeader(tc.header)
			if !tc.wantNoErr {
				require.Error(t, err)
				for _, s := range tc.wantErr {
					assert.Contains(t, err.Error(), s)
				}
				for _, s := range tc.notInError {
					assert.NotContains(t, err.Error(), s)
				}
				return
			}
			require.NoError(t, err)
			var got []string
			for loc := range l.stem {
				got = append(got, loc)
			}
			assert.ElementsMatch(t, tc.stemLocs, got)
			assert.Equal(t, tc.maxOpt, l.maxOpt)
			for n, locs := range tc.optLocs {
				var g []string
				for loc := range l.opts[n] {
					g = append(g, loc)
				}
				assert.ElementsMatch(t, locs, g, "option %d", n)
			}
		})
	}
}

// ── parseCSVImport ────────────────────────────────────────────────────────────

func parseCSV(t *testing.T, s string) []ImportRow {
	t.Helper()
	rows, err := parseCSVImport(strings.NewReader(s))
	require.NoError(t, err)
	return rows
}

func TestParseCSVImport_LegacyLayoutStillWorks(t *testing.T) {
	rows := parseCSV(t, "type,difficulty,category_path,default_locale,stem,explanation,option_1,option_2,correct,tags\n"+
		"single,easy,Science,kk,What is H2O?,Because,Water,Fire,1,chem;bio\n")
	require.Len(t, rows, 1)
	r := rows[0]
	assert.Equal(t, 2, r.RowNumber)
	assert.Equal(t, "kk", r.DefaultLocale)
	assert.Equal(t, "What is H2O?", r.Translations["kk"].Stem)
	require.NotNil(t, r.Translations["kk"].Explanation)
	assert.Equal(t, "Because", *r.Translations["kk"].Explanation)
	require.Len(t, r.AnswerOptions, 2)
	assert.True(t, r.AnswerOptions[0].IsCorrect)
	assert.False(t, r.AnswerOptions[1].IsCorrect)
	assert.Equal(t, "Water", r.AnswerOptions[0].Translations["kk"].Text)
	assert.Equal(t, []string{"chem", "bio"}, r.Tags)
}

func TestParseCSVImport_ExportLayoutMultiLocale(t *testing.T) {
	rows := parseCSV(t, "type,difficulty,category_path,default_locale,stem_en,stem_ru,explanation_en,explanation_ru,option_1_en,option_1_ru,option_2_en,option_2_ru,correct,tags\n"+
		"single,easy,Science,en,Water?,Voda?,why,,A,Ay,B,,2,\n")
	require.Len(t, rows, 1)
	r := rows[0]
	assert.Equal(t, "Water?", r.Translations["en"].Stem)
	assert.Equal(t, "Voda?", r.Translations["ru"].Stem)
	require.NotNil(t, r.Translations["en"].Explanation)
	assert.Nil(t, r.Translations["ru"].Explanation)
	require.Len(t, r.AnswerOptions, 2)
	assert.Equal(t, "A", r.AnswerOptions[0].Translations["en"].Text)
	assert.Equal(t, "Ay", r.AnswerOptions[0].Translations["ru"].Text)
	_, hasRu := r.AnswerOptions[1].Translations["ru"]
	assert.False(t, hasRu, "blank non-default option text is skipped")
	assert.True(t, r.AnswerOptions[1].IsCorrect)
	assert.False(t, r.AnswerOptions[0].IsCorrect)
}

func TestParseCSVImport_NonDefaultBlankStemCreatesNoTranslation(t *testing.T) {
	rows := parseCSV(t, "type,difficulty,category_path,default_locale,stem_en,stem_ru,explanation_ru,option_1_en,option_2_en,correct,tags\n"+
		"single,easy,Science,en,Q?,,orphan expl,A,B,1,\n")
	_, hasRu := rows[0].Translations["ru"]
	assert.False(t, hasRu)
}

func TestParseCSVImport_FormulaGuardStrippedOnAllColumns(t *testing.T) {
	// Guarded cells in the export layout; a legit leading apostrophe stays intact.
	rows := parseCSV(t, "type,difficulty,category_path,default_locale,stem_en,explanation_en,option_1_en,option_2_en,correct,tags\n"+
		"single,easy,Science,en,'=1+1,'+x,'-5,'@home,1,'=tag\n"+
		"single,easy,Science,en,'Tis the season,'plain,'quoted,it's,1,\n")
	require.Len(t, rows, 2)
	assert.Equal(t, "=1+1", rows[0].Translations["en"].Stem)
	assert.Equal(t, "+x", *rows[0].Translations["en"].Explanation)
	assert.Equal(t, "-5", rows[0].AnswerOptions[0].Translations["en"].Text)
	assert.Equal(t, "@home", rows[0].AnswerOptions[1].Translations["en"].Text)
	assert.Equal(t, []string{"=tag"}, rows[0].Tags)

	assert.Equal(t, "'Tis the season", rows[1].Translations["en"].Stem)
	assert.Equal(t, "'plain", *rows[1].Translations["en"].Explanation)
	assert.Equal(t, "'quoted", rows[1].AnswerOptions[0].Translations["en"].Text)
	assert.Equal(t, "it's", rows[1].AnswerOptions[1].Translations["en"].Text)
}

func TestParseCSVImport_LegacyFormulaGuardStripped(t *testing.T) {
	rows := parseCSV(t, "type,difficulty,category_path,default_locale,stem,option_1,option_2,correct,tags\n"+
		"single,easy,Science,kk,'=SUM(A1),'+1,'-1,1,\n")
	assert.Equal(t, "=SUM(A1)", rows[0].Translations["kk"].Stem)
	assert.Equal(t, "+1", rows[0].AnswerOptions[0].Translations["kk"].Text)
}

func TestParseCSVImport_MissingColumnsAreNamed(t *testing.T) {
	_, err := parseCSVImport(strings.NewReader("type,difficulty,default_locale,stem_en\nsingle,easy,en,Q\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"category_path"`)
	assert.NotContains(t, err.Error(), `"stem"`)
}

// ── round trip through the real handlers + service ───────────────────────────

func roundTripRows() []*ExportRow {
	ex := func(s string) *string { return &s }
	opt := func(n int, correct bool, en, ru string) AnswerOptionDetail {
		return AnswerOptionDetail{SortOrder: n, IsCorrect: correct,
			Translations: map[string]AnswerTranslationDetail{"en": {Text: en}, "ru": {Text: ru}}}
	}
	return []*ExportRow{
		{Type: "single", Difficulty: "easy", CategoryPath: "Science", DefaultLocale: "en",
			Translations: map[string]TranslationDetail{
				"en": {Stem: "=1+1 equals?", Explanation: ex("+explained")},
				"ru": {Stem: "@ru stem", Explanation: ex("-ru expl")}},
			AnswerOptions: []AnswerOptionDetail{opt(1, true, "=two", "dva"), opt(2, false, "+three", "tri"), opt(3, false, "-four", "'chetyre")},
			Tags:          []string{"math", "=weird"}},
		{Type: "multiple", Difficulty: "hard", CategoryPath: "Science/Physics", DefaultLocale: "en",
			Translations: map[string]TranslationDetail{
				"en": {Stem: "@mention, \"quoted\", comma"},
				"ru": {Stem: "'Plain apostrophe stem"}},
			AnswerOptions: []AnswerOptionDetail{opt(1, true, "@a", "a"), opt(2, true, "b", "b"), opt(3, false, "c", "c")},
			Tags:          []string{}},
	}
}

func exportCSVBytes(t *testing.T, rows []*ExportRow) string {
	t.Helper()
	h := NewHandler(&mockQService{
		streamExportFn: func(_ context.Context, _ ExportFilter, fn func(*ExportRow) error) error {
			for _, r := range rows {
				if err := fn(r); err != nil {
					return err
				}
			}
			return nil
		},
	}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/export", nil)
	req.Header.Set("Accept", "text/csv")
	w := httptest.NewRecorder()
	h.Export(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	return w.Body.String()
}

func importDryRun(t *testing.T, csvText string) (int, *DryRunReport, string) {
	t.Helper()
	h := NewHandler(NewService(newMockRepo()), nil)
	body, ct := buildQCSVMultipart(t, csvText, "export.csv")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/import?dry_run=true", body)
	req.Header.Set("Content-Type", ct)
	req = withQImportAuthCtx(req)
	w := httptest.NewRecorder()
	h.Import(w, req)
	var env struct {
		Data *DryRunReport `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	return w.Code, env.Data, w.Body.String()
}

func TestExportThenImportDryRun_RoundTrip(t *testing.T) {
	rows := roundTripRows()
	csvText := exportCSVBytes(t, rows)
	require.Contains(t, csvText, "stem_en", "export layout expected")
	require.Contains(t, csvText, "'=1+1", "formula guard expected in export")

	code, report, raw := importDryRun(t, csvText)
	require.Equal(t, http.StatusOK, code, raw)
	require.NotNil(t, report)
	assert.Empty(t, report.ErrorRows)
	assert.Equal(t, len(rows), report.ValidCount)

	// Parsed values equal the originals (guard fully reversed, nothing else touched).
	parsed, err := parseCSVImport(strings.NewReader(csvText))
	require.NoError(t, err)
	require.Len(t, parsed, len(rows))
	for i, src := range rows {
		p := parsed[i]
		assert.Equal(t, src.DefaultLocale, p.DefaultLocale)
		assert.Equal(t, src.Tags, append([]string{}, p.Tags...))
		for loc, tr := range src.Translations {
			assert.Equal(t, tr.Stem, p.Translations[loc].Stem)
			if tr.Explanation != nil {
				require.NotNil(t, p.Translations[loc].Explanation)
				assert.Equal(t, *tr.Explanation, *p.Translations[loc].Explanation)
			}
		}
		require.Len(t, p.AnswerOptions, len(src.AnswerOptions))
		for j, so := range src.AnswerOptions {
			assert.Equal(t, so.IsCorrect, p.AnswerOptions[j].IsCorrect)
			for loc, at := range so.Translations {
				assert.Equal(t, at.Text, p.AnswerOptions[j].Translations[loc].Text)
			}
		}
	}
}

func TestImportDryRun_MissingDefaultLocaleStem_ClearRowError(t *testing.T) {
	// default_locale is kk but only en columns exist.
	csvText := "type,difficulty,category_path,default_locale,stem_en,option_1_en,option_2_en,correct,tags\n" +
		"single,easy,Science,kk,Q?,A,B,1,\n"
	code, report, raw := importDryRun(t, csvText)
	require.Equal(t, http.StatusOK, code, raw)
	require.Len(t, report.ErrorRows, 1)
	assert.Equal(t, 2, report.ErrorRows[0].Row)
	joined := strings.Join(report.ErrorRows[0].Errors, "|")
	assert.Contains(t, joined, `stem: required for default_locale "kk"`)
}

func TestImportDryRun_BlankDefaultLocaleOptionIsRowError(t *testing.T) {
	// #173: option 2 only has a ru text; default locale en is blank -> row error.
	csvText := "type,difficulty,category_path,default_locale,stem_en,option_1_en,option_1_ru,option_2_en,option_2_ru,correct,tags\n" +
		"single,easy,Science,en,Q?,A,a,,b,1,\n"
	code, report, raw := importDryRun(t, csvText)
	require.Equal(t, http.StatusOK, code, raw)
	require.Len(t, report.ErrorRows, 1)
	assert.Contains(t, strings.Join(report.ErrorRows[0].Errors, "|"), "option 2 must have non-empty text for the default locale")
}

func TestImportDryRun_LegacyLayoutStillValid(t *testing.T) {
	code, report, raw := importDryRun(t, validCSV)
	require.Equal(t, http.StatusOK, code, raw)
	assert.Empty(t, report.ErrorRows)
	assert.Equal(t, 1, report.ValidCount)
}

func TestImportDryRun_MissingColumnsReturns400WithNames(t *testing.T) {
	code, _, raw := importDryRun(t, "type,difficulty,default_locale,stem_en\nsingle,easy,en,Q\n")
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, raw, "category_path")
}

func TestParseCSVImport_LocaleCaseAndPrecedence(t *testing.T) {
	rows := parseCSV(t, "type,difficulty,category_path,default_locale,stem,stem_pt-BR,STEM_EN,option_1,option_1_pt-BR,option_2_pt-BR,correct,tags\n"+
		"single,easy,Science,pt-BR,legacy,explicit,en text,la,lb,lc,1,\n")
	r := rows[0]
	assert.Equal(t, "explicit", r.Translations["pt-BR"].Stem, "explicit locale column beats legacy")
	assert.Equal(t, "en text", r.Translations["EN"].Stem)
	require.Len(t, r.AnswerOptions, 2)
	assert.Equal(t, "lb", r.AnswerOptions[0].Translations["pt-BR"].Text)
	assert.Equal(t, "lc", r.AnswerOptions[1].Translations["pt-BR"].Text)

	// A different-case default_locale still resolves to the column locale.
	rows = parseCSV(t, "type,difficulty,category_path,default_locale,stem_pt-BR,option_1_pt-BR,option_2_pt-BR,correct,tags\n"+
		"single,easy,Science,pt-br,s,a,b,1,\n")
	assert.Equal(t, "s", rows[0].Translations["pt-br"].Stem)
}
