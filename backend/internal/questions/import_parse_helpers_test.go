package questions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #514: the JSON import parser maps the file into import rows, and the tag helper keeps each tag once.

func TestParseJSONImport_MapsRowsTranslationsOptionsAndNumbersThem(t *testing.T) {
	src := `[
	  {"type":"single","difficulty":"easy","category_path":"Safety","default_locale":"en",
	   "translations":{"en":{"stem":"Q1?","explanation":"Because"}},
	   "answer_options":[
	     {"sort_order":1,"is_correct":true,"translations":{"en":{"text":"Yes"}}},
	     {"sort_order":2,"is_correct":false,"likert_weight":2.5,"likert_polarity":"positive",
	      "translations":{"en":{"text":"No"}}}],
	   "tags":["Safety"]},
	  {"type":"likert","difficulty":"hard","category_path":"HR","default_locale":"kk"}
	]`

	rows, err := parseJSONImport(strings.NewReader(src))
	require.NoError(t, err)
	require.Len(t, rows, 2)

	first := rows[0]
	assert.Equal(t, 1, first.RowNumber)
	assert.Equal(t, "single", first.Type)
	assert.Equal(t, "Safety", first.CategoryPath)
	assert.Equal(t, "Q1?", first.Translations["en"].Stem)
	if assert.NotNil(t, first.Translations["en"].Explanation) {
		assert.Equal(t, "Because", *first.Translations["en"].Explanation)
	}
	require.Len(t, first.AnswerOptions, 2)
	assert.Equal(t, 1, first.AnswerOptions[0].SortOrder)
	assert.True(t, first.AnswerOptions[0].IsCorrect)
	assert.Equal(t, "Yes", first.AnswerOptions[0].Translations["en"].Text)
	if assert.NotNil(t, first.AnswerOptions[1].LikertWeight) {
		assert.InDelta(t, 2.5, *first.AnswerOptions[1].LikertWeight, 1e-9)
	}
	if assert.NotNil(t, first.AnswerOptions[1].LikertPolarity) {
		assert.Equal(t, "positive", *first.AnswerOptions[1].LikertPolarity)
	}
	assert.Equal(t, []string{"Safety"}, first.Tags)

	second := rows[1]
	assert.Equal(t, 2, second.RowNumber, "row numbers are 1-based and follow the file order")
	assert.Empty(t, second.AnswerOptions)
	assert.NotNil(t, second.Tags, "a row without tags gets an empty list, not nil")
	assert.Empty(t, second.Tags)
}

func TestParseJSONImport_MalformedInputIsAnErrorNamingJSON(t *testing.T) {
	_, err := parseJSONImport(strings.NewReader(`[{"type":`))
	assert.ErrorContains(t, err, "JSON:")
}

func TestParseJSONImport_EmptyArrayGivesNoRows(t *testing.T) {
	rows, err := parseJSONImport(strings.NewReader(`[]`))
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestCollectTagNames_KeepsEachTagOnceFirstSpellingInOrder(t *testing.T) {
	rows := []ImportRow{
		{Tags: []string{"Math", "algebra"}},
		{Tags: []string{"MATH", "Geometry", "Algebra"}},
	}
	assert.Equal(t, []string{"Math", "algebra", "Geometry"}, collectTagNames(rows))
}

func TestCollectTagNames_NoTagsGivesNothing(t *testing.T) {
	assert.Empty(t, collectTagNames(nil))
	assert.Empty(t, collectTagNames([]ImportRow{{Tags: []string{}}}))
}
