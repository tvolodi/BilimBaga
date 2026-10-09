package questions

import (
	"errors"
	"fmt"
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

// validateOptionTexts returns an *OptionValidationError (or nil).
func validateOptionTexts(qType, defaultLocale string, opts []AnswerOptionInput) error {
	if errs := optionTextFieldErrors(qType, defaultLocale, opts); len(errs) > 0 {
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
