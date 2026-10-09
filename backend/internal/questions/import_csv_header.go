package questions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// csvLayout is the normalised view of an import CSV header (ISS-199).
//
// Two column dialects are accepted side by side:
//
//	legacy : stem, explanation, option_N          (text is in the row's default_locale)
//	export : stem_<loc>, explanation_<loc>, option_N_<loc>   (what GET /questions/export writes)
//
// A locale of "" in the maps below stands for the legacy (locale-less) column.
type csvLayout struct {
	// idx maps a plain, lower-cased column name to its position.
	idx map[string]int
	// stem / expl map locale -> column position.
	stem map[string]int
	expl map[string]int
	// opts maps option number (1-based) -> locale -> column position.
	opts   map[int]map[string]int
	maxOpt int
}

// bom is the UTF-8 byte order mark some spreadsheet tools prepend to CSV files.
const bom = "\xef\xbb\xbf"

var (
	reStemCol = regexp.MustCompile(`(?i)^stem(?:_(.+))?$`)
	reExplCol = regexp.MustCompile(`(?i)^explanation(?:_(.+))?$`)
	reOptCol  = regexp.MustCompile(`(?i)^option_(\d+)(?:_(.+))?$`)
)

// normaliseCSVHeader is a pure function that turns a raw header row into a
// csvLayout, or returns an error naming every missing required column.
// Column names are matched case-insensitively and after trimming whitespace
// (and a UTF-8 BOM on the first cell).
func normaliseCSVHeader(header []string) (*csvLayout, error) {
	l := &csvLayout{
		idx:  make(map[string]int, len(header)),
		stem: map[string]int{},
		expl: map[string]int{},
		opts: map[int]map[string]int{},
	}
	for i, h := range header {
		// Keep the original spelling for locale suffixes (e.g. "pt-BR") so they
		// match default_locale; only the keywords are case-insensitive.
		raw := strings.TrimSpace(strings.TrimPrefix(h, bom))
		name := strings.ToLower(raw)
		if name == "" {
			continue
		}
		l.idx[name] = i
		if m := reStemCol.FindStringSubmatch(raw); m != nil {
			l.stem[m[1]] = i
		} else if m := reExplCol.FindStringSubmatch(raw); m != nil {
			l.expl[m[1]] = i
		} else if m := reOptCol.FindStringSubmatch(raw); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil || n < 1 {
				continue
			}
			if l.opts[n] == nil {
				l.opts[n] = map[string]int{}
			}
			l.opts[n][m[2]] = i
			if n > l.maxOpt {
				l.maxOpt = n
			}
		}
	}

	var missing []string
	for _, col := range []string{"type", "difficulty", "category_path", "default_locale"} {
		if _, ok := l.idx[col]; !ok {
			missing = append(missing, fmt.Sprintf("%q", col))
		}
	}
	if len(l.stem) == 0 {
		missing = append(missing, `"stem" (or "stem_<locale>", e.g. "stem_en")`)
	}
	if len(missing) > 0 {
		noun := "column"
		if len(missing) > 1 {
			noun = "columns"
		}
		return nil, fmt.Errorf("CSV: missing required %s: %s", noun, strings.Join(missing, ", "))
	}
	return l, nil
}

// buildImportRowFromRecord maps one CSV record onto an ImportRow using the
// layout. get returns the trimmed, formula-guard-stripped cell at a position
// ("" if the record is short).
//
// Legacy (locale-less) columns bind to the row's default_locale; an explicit
// <name>_<locale> cell for the same locale takes precedence when non-blank.
// Translations for non-default locales are created only when their stem is
// non-blank (a translation cannot exist without a stem); blank option texts
// in non-default locales are skipped. The default-locale rules (#173) are
// enforced later by the service: missing stem and blank option text there are
// row errors.
func (l *csvLayout) buildRow(rowNum int, get func(pos int) string) ImportRow {
	cell := func(col string) string {
		if i, ok := l.idx[col]; ok {
			return get(i)
		}
		return ""
	}
	locale := cell("default_locale")

	// resolve picks the text for one locale from a locale->column map.
	// Locale "" (legacy) is folded into the default locale.
	texts := func(cols map[string]int) map[string]string {
		out := map[string]string{}
		for loc, pos := range cols {
			v := get(pos)
			target := loc
			if loc == "" || strings.EqualFold(loc, locale) {
				target = locale
			}
			if v == "" {
				if _, seen := out[target]; !seen {
					out[target] = ""
				}
				continue
			}
			// Explicit locale column wins over the legacy one.
			if prev, seen := out[target]; seen && prev != "" && loc == "" {
				continue
			}
			out[target] = v
		}
		return out
	}

	stems := texts(l.stem)
	expls := texts(l.expl)
	translations := map[string]TranslationInput{}
	for loc, stem := range stems {
		// Blank stem: no translation. Legacy column with no default_locale:
		// no translation either (the service reports default_locale: required).
		if stem == "" || loc == "" {
			continue
		}
		ti := TranslationInput{Stem: stem}
		if e := expls[loc]; e != "" {
			e := e
			ti.Explanation = &e
		}
		translations[loc] = ti
	}

	correctIdxs := parseCorrectIndices(cell("correct"))
	var options []AnswerOptionInput
	for n := 1; n <= l.maxOpt; n++ {
		cols, ok := l.opts[n]
		if !ok {
			break
		}
		byLoc := texts(cols)
		hasText := false
		tr := map[string]AnswerTranslationInput{}
		for loc, v := range byLoc {
			if v == "" {
				continue
			}
			hasText = true
			tr[loc] = AnswerTranslationInput{Text: v}
		}
		if !hasText {
			break // first fully blank option ends the list (legacy behaviour)
		}
		options = append(options, AnswerOptionInput{
			SortOrder:    n,
			IsCorrect:    correctIdxs[n],
			Translations: tr,
		})
	}

	var tags []string
	for _, t := range strings.Split(cell("tags"), ";") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}

	return ImportRow{
		RowNumber:     rowNum,
		Type:          cell("type"),
		Difficulty:    cell("difficulty"),
		CategoryPath:  cell("category_path"),
		DefaultLocale: locale,
		Translations:  translations,
		AnswerOptions: options,
		Tags:          tags,
	}
}
