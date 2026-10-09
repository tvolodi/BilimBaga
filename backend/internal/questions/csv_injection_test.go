package questions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-191: questions CSV export guards stems / explanations / options / tags,
// and the CSV importer reverses the guard so exports round-trip.
func TestRowToCSV_FormulaInjectionGuard(t *testing.T) {
	expl := "@SUM(A1)"
	row := &ExportRow{
		Type: "single", Difficulty: "easy", CategoryPath: "=Cat", DefaultLocale: "en",
		Translations: map[string]TranslationDetail{"en": {Stem: `=HYPERLINK("http://evil","x")`, Explanation: &expl}},
		AnswerOptions: []AnswerOptionDetail{
			{IsCorrect: true, Translations: map[string]AnswerTranslationDetail{"en": {Text: "-5 degrees"}}},
			{Translations: map[string]AnswerTranslationDetail{"en": {Text: "Normal"}}},
		},
		Tags: []string{"+tag"},
	}
	sc := newCSVExportScan()
	sc.add(row)
	rec := sc.columns().record(row)
	assert.Equal(t, "'=Cat", rec[2])
	assert.Equal(t, `'=HYPERLINK("http://evil","x")`, rec[4])
	assert.Equal(t, "'@SUM(A1)", rec[5])
	assert.Equal(t, "'-5 degrees", rec[6])
	assert.Equal(t, "Normal", rec[7])
	assert.Equal(t, "1", rec[8]) // correct indices stay numeric
	assert.Equal(t, "'+tag", rec[9])
}

func TestParseCSVImport_UnescapesExportGuard(t *testing.T) {
	rows, err := parseCSVImport(strings.NewReader(
		"type,difficulty,category_path,default_locale,stem,explanation,option_1,option_2,correct,tags\n" +
			"single,easy,'=Cat,en,'=HYPERLINK(1),'@SUM(A1),'-5 degrees,Normal,1,'+tag\n"))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "=Cat", rows[0].CategoryPath)
	assert.Equal(t, "=HYPERLINK(1)", rows[0].Translations["en"].Stem)
	assert.Equal(t, "-5 degrees", rows[0].AnswerOptions[0].Translations["en"].Text)
	assert.Equal(t, []string{"+tag"}, rows[0].Tags)
}
