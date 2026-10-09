package questions

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-199b / #219: the exporter must emit a rectangular CSV for mixed question
// shapes, and the importer must survive ragged legacy files.

func mixedExportRows() []*ExportRow {
	at := func(m map[string]string) map[string]AnswerTranslationDetail {
		out := map[string]AnswerTranslationDetail{}
		for k, v := range m {
			out[k] = AnswerTranslationDetail{Text: v}
		}
		return out
	}
	opt := func(n int, correct bool, en string) AnswerOptionDetail {
		return AnswerOptionDetail{SortOrder: n, IsCorrect: correct, Translations: at(map[string]string{"en": en})}
	}
	stem := func(s string) TranslationDetail { return TranslationDetail{Stem: s} }
	likert := func() []AnswerOptionDetail {
		var o []AnswerOptionDetail
		for _, l := range []string{"Strongly disagree", "Disagree", "Neutral", "Agree", "Strongly agree"} {
			o = append(o, opt(len(o)+1, false, l))
		}
		return o
	}
	return []*ExportRow{
		{Type: "single", Difficulty: "easy", CategoryPath: "Gen", DefaultLocale: "en",
			Translations:  map[string]TranslationDetail{"en": stem("Plain stem, with comma")},
			AnswerOptions: []AnswerOptionDetail{opt(1, true, "Yes"), opt(2, false, "No, never")}, Tags: []string{}},
		{Type: "truefalse", Difficulty: "easy", CategoryPath: "Gen", DefaultLocale: "en",
			Translations:  map[string]TranslationDetail{"en": stem("=Sky is blue")},
			AnswerOptions: []AnswerOptionDetail{opt(1, true, "True"), opt(2, false, "False")}, Tags: []string{}},
		{Type: "shorttext", Difficulty: "medium", CategoryPath: "Gen", DefaultLocale: "en",
			Translations: map[string]TranslationDetail{"en": stem("+Capital of France?")}, Tags: []string{}},
		{Type: "likert", Difficulty: "medium", CategoryPath: "Gen", DefaultLocale: "en",
			Translations: map[string]TranslationDetail{"en": stem("@I like tests")}, AnswerOptions: likert(), Tags: []string{}},
		{Type: "multiple", Difficulty: "hard", CategoryPath: "Gen", DefaultLocale: "en",
			Translations: map[string]TranslationDetail{"en": stem("Pick primes")},
			AnswerOptions: []AnswerOptionDetail{opt(1, true, "2"), opt(2, true, "3"), opt(3, false, "-4"),
				opt(4, true, "5")}, Tags: []string{"math", "-neg"}},
		// ru-default question that has no en translation.
		{Type: "single", Difficulty: "easy", CategoryPath: "Gen", DefaultLocale: "ru",
			Translations: map[string]TranslationDetail{"ru": stem("Russian only")},
			AnswerOptions: []AnswerOptionDetail{
				{SortOrder: 1, IsCorrect: true, Translations: at(map[string]string{"ru": "da"})},
				{SortOrder: 2, Translations: at(map[string]string{"ru": "net"})}}, Tags: []string{}},
		// en default with a kk translation too.
		{Type: "single", Difficulty: "easy", CategoryPath: "Gen", DefaultLocale: "en",
			Translations: map[string]TranslationDetail{"en": stem("Bilingual"), "kk": stem("Eki tilde")},
			AnswerOptions: []AnswerOptionDetail{
				{SortOrder: 1, IsCorrect: true, Translations: at(map[string]string{"en": "A", "kk": "A-kk"})},
				{SortOrder: 2, Translations: at(map[string]string{"en": "B", "kk": "B-kk"})}}, Tags: []string{}},
	}
}

func TestExport_MixedTypesAreRectangular_AndRoundTrip(t *testing.T) {
	rows := mixedExportRows()
	text := exportCSVBytes(t, rows)

	cr := csv.NewReader(strings.NewReader(text))
	cr.FieldsPerRecord = 0 // strict: every record must match the header width
	recs, err := cr.ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, len(rows)+1)
	header := recs[0]

	// Union of locales (sorted) and 5 option slots.
	assert.Contains(t, header, "stem_en")
	assert.Contains(t, header, "stem_kk")
	assert.Contains(t, header, "stem_ru")
	assert.Contains(t, header, "option_5_en")
	assert.Contains(t, header, "option_5_ru")
	assert.Equal(t, "correct", header[len(header)-2])
	assert.Equal(t, "tags", header[len(header)-1])

	// Padding cells are empty and never guarded.
	shortText := recs[3]
	for i, c := range shortText {
		if strings.HasPrefix(header[i], "option_") {
			assert.Equal(t, "", c, header[i])
		}
	}

	code, report, raw := importDryRun(t, text)
	require.Equal(t, http.StatusOK, code, raw)
	require.NotNil(t, report)
	assert.Empty(t, report.ErrorRows, raw)
	assert.Equal(t, len(rows), report.ValidCount)

	// Guard survives the round trip on every text cell kind.
	parsed, err := parseCSVImport(strings.NewReader(text))
	require.NoError(t, err)
	require.Len(t, parsed, len(rows))
	assert.Equal(t, "=Sky is blue", parsed[1].Translations["en"].Stem)
	assert.Equal(t, "+Capital of France?", parsed[2].Translations["en"].Stem)
	assert.Equal(t, "@I like tests", parsed[3].Translations["en"].Stem)
	assert.Equal(t, "-4", parsed[4].AnswerOptions[2].Translations["en"].Text)
	assert.Equal(t, []string{"math", "-neg"}, parsed[4].Tags)
	assert.Len(t, parsed[3].AnswerOptions, 5)
	assert.Len(t, parsed[2].AnswerOptions, 0)
	assert.True(t, parsed[4].AnswerOptions[0].IsCorrect && parsed[4].AnswerOptions[1].IsCorrect && !parsed[4].AnswerOptions[2].IsCorrect && parsed[4].AnswerOptions[3].IsCorrect)
	assert.Equal(t, "Russian only", parsed[5].Translations["ru"].Stem)
	assert.Equal(t, "Eki tilde", parsed[6].Translations["kk"].Stem)
}

func TestParseCSVImport_RaggedLegacyFileAccepted(t *testing.T) {
	// Short rows (as written by the pre-fix exporter) and a row padded with empty cells.
	in := "type,difficulty,category_path,default_locale,stem_en,explanation_en,option_1_en,option_2_en,option_3_en,correct,tags\n" +
		"single,easy,Gen,en,Q1,,Yes,No,1,\n" +
		"shorttext,easy,Gen,en,Q2,,,\n" +
		"single,easy,Gen,en,Q3,,A,B,C,2,t,,,\n"
	rows, err := parseCSVImport(strings.NewReader(in))
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Empty(t, rows[0].ParseErrors)
	assert.Empty(t, rows[1].ParseErrors)
	assert.Empty(t, rows[2].ParseErrors, "extra EMPTY cells are ignored")
	assert.Equal(t, "Q1", rows[0].Translations["en"].Stem)
}

func TestImportDryRun_ExtraNonEmptyCellIsRowError(t *testing.T) {
	in := "type,difficulty,category_path,default_locale,stem_en,option_1_en,option_2_en,correct,tags\n" +
		"single,easy,Gen,en,Good,A,B,1,t\n" +
		"single,easy,Gen,en,Bad,A,B,1,t,stray\n"
	code, report, raw := importDryRun(t, in)
	require.Equal(t, http.StatusOK, code, raw)
	require.NotNil(t, report)
	require.Len(t, report.ErrorRows, 1, raw)
	assert.Equal(t, 3, report.ErrorRows[0].Row)
	assert.Contains(t, strings.Join(report.ErrorRows[0].Errors, " "), "unexpected value in column 10")
	assert.Equal(t, 1, report.ValidCount)
}

// ISS-268 / FR-BB24 AC-11: an exported legacy row that is only partially
// translated (kk stem + one kk option, other option blank) is not silently
// accepted on re-import; it is reported as a row error naming the option index.
func TestExport_PartialLocaleRow_ReimportReportsRowError(t *testing.T) {
	rows := mixedExportRows()[:1]
	rows[0].Translations["kk"] = TranslationDetail{Stem: "Zhai soz"}
	rows[0].AnswerOptions[0].Translations["kk"] = AnswerTranslationDetail{Text: "Ia"}

	code, report, raw := importDryRun(t, exportCSVBytes(t, rows))
	require.Equal(t, http.StatusOK, code, raw)
	require.NotNil(t, report)
	assert.Equal(t, 0, report.ValidCount, raw)
	require.Len(t, report.ErrorRows, 1, raw)
	assert.Contains(t, strings.Join(report.ErrorRows[0].Errors, ";"), "answer_options[1].translations.kk.text")
}
