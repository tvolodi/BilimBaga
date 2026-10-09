package questions

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrInvalidOptionText is returned when a choice question has an answer option
// whose text for the question's default locale is empty or whitespace-only.
var ErrInvalidOptionText = errors.New("INVALID_OPTION_TEXT")

// OptionValidationError carries per-option field errors and unwraps to ErrInvalidOptionText.
type OptionValidationError struct {
	Fields []fieldError
}

func (e *OptionValidationError) Error() string {
	msgs := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		msgs = append(msgs, f.Field+": "+f.Message)
	}
	return ErrInvalidOptionText.Error() + ": " + strings.Join(msgs, "; ")
}

func (e *OptionValidationError) Unwrap() error { return ErrInvalidOptionText }

// optionTextFieldErrors checks that every answer option of a choice question
// (single, multiple, truefalse, likert) has non-blank text for defaultLocale.
// Other locales may be empty or absent. Short-text questions have no options to check.
func optionTextFieldErrors(qType, defaultLocale string, opts []AnswerOptionInput) []fieldError {
	if qType == "shorttext" || defaultLocale == "" {
		return nil
	}
	var errs []fieldError
	for i, opt := range opts {
		if strings.TrimSpace(opt.Translations[defaultLocale].Text) == "" {
			errs = append(errs, fieldError{
				fmt.Sprintf("answer_options[%d].translations.%s.text", i, defaultLocale),
				fmt.Sprintf("option %d must have non-empty text for the default locale %q", i+1, defaultLocale),
			})
		}
	}
	return errs
}

// partialLocaleFieldErrors enforces FR-BB24 AC-11 for create/update payloads: a
// non-default locale that is "present" (stem non-blank or any option text
// non-blank) must have non-blank text for every option. A locale with no option
// text at all is untranslated and falls back to the default locale.
func partialLocaleFieldErrors(qType, defaultLocale string, stems map[string]TranslationInput, opts []AnswerOptionInput) []fieldError {
	if qType == "shorttext" || len(opts) == 0 {
		return nil
	}
	locales := map[string]bool{}
	for loc := range stems {
		locales[loc] = true
	}
	for _, opt := range opts {
		for loc := range opt.Translations {
			locales[loc] = true
		}
	}
	names := make([]string, 0, len(locales))
	for loc := range locales {
		if loc != defaultLocale {
			names = append(names, loc)
		}
	}
	sort.Strings(names)

	var errs []fieldError
	for _, loc := range names {
		texts := make([]string, len(opts))
		for i, opt := range opts {
			texts[i] = opt.Translations[loc].Text
		}
		errs = append(errs, partialLocaleErrors(loc, strings.TrimSpace(stems[loc].Stem) != "", texts)...)
	}
	return errs
}

// partialLocaleErrors returns one field error per blank option text when the
// locale is present (hasStem or any option text non-blank).
func partialLocaleErrors(locale string, hasStem bool, texts []string) []fieldError {
	present := hasStem
	for _, t := range texts {
		if strings.TrimSpace(t) != "" {
			present = true
		}
	}
	if !present {
		return nil
	}
	var errs []fieldError
	for i, t := range texts {
		if strings.TrimSpace(t) == "" {
			errs = append(errs, fieldError{
				fmt.Sprintf("answer_options[%d].translations.%s.text", i, locale),
				fmt.Sprintf("option %d must have non-empty text for locale %q once that locale is translated", i+1, locale),
			})
		}
	}
	return errs
}

// validateOptionTexts returns an *OptionValidationError (or nil) for blank
// default-locale option text only (used by status transitions and the default-locale PUT).
func validateOptionTexts(qType, defaultLocale string, opts []AnswerOptionInput) error {
	if errs := optionTextFieldErrors(qType, defaultLocale, opts); len(errs) > 0 {
		return &OptionValidationError{Fields: errs}
	}
	return nil
}

// validateOptionTextsWithStems also applies the partial non-default locale rule.
func validateOptionTextsWithStems(qType, defaultLocale string, stems map[string]TranslationInput, opts []AnswerOptionInput) error {
	errs := optionTextFieldErrors(qType, defaultLocale, opts)
	errs = append(errs, partialLocaleFieldErrors(qType, defaultLocale, stems, opts)...)
	if len(errs) > 0 {
		return &OptionValidationError{Fields: errs}
	}
	return nil
}

func answerOptionInputsFromReq(reqs []answerOptionReq) []AnswerOptionInput {
	options := make([]AnswerOptionInput, 0, len(reqs))
	for _, opt := range reqs {
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
	return options
}
