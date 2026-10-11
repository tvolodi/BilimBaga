package sessions

import (
	"strings"
	"unicode/utf8"
)

// AI grading flags (FR-BB73 AC-15). A flag routes an answer to a person: a flagged answer is stored as a suggestion,
// never as a grade, whatever the mode of its exam. The rules are plain code in this file, never a question put to the
// model, and they are the same in any order of grading.

// Flag codes. They are stored in session_question_scores.ai_flags and shown to the examiner as chips.
const (
	aiFlagInstructionLike = "instruction_like"
	aiFlagShortHigh       = "short_high"
	aiFlagDecisivePass    = "decisive_pass"
)

// Thresholds of the short_high rule.
const (
	aiFlagShortAnswerMaxRunes = 20  // the answer is shorter than this
	aiFlagShortHighMinScore   = 80  // and the suggested score is at least this (percent)
	aiFlagModelAnswerMinRunes = 100 // and the model answer is longer than this
)

// aiInstructionPhrases is the phrase table of instruction_like, in lower case. A phrase is matched after the answer is
// lower-cased and its whitespace collapsed. The content owner adds Kazakh phrases as one-line additions here.
var aiInstructionPhrases = []string{
	"ignore previous",
	"ignore all",
	"ignore the above",
	"disregard",
	"system prompt",
	"you are now",
	"as the grader",
	"give full",
	"full marks",
	"maximum score",
	"score is",
	"score of 100",
	"100%",
	"игнорируй",
	"забудь",
	"поставь 100",
	"максимальный балл",
	"полный балл",
}

// flagContext is what the write transaction knows about the session when it routes an AI score. The caller reads all
// of it under the session lock (AC-15).
type flagContext struct {
	// FinalPoints (S0) is the sum of the points of the rows that are graded, that is final without AI.
	FinalPoints float64
	// OpenPoints (Open) is the sum of the maximum points of every row that is not graded (pending or ai_graded), this
	// row included.
	OpenPoints float64
	// TotalPoints is the sum of the maximum points of all rows of the session.
	TotalPoints float64
	// PassingScorePct is the pass mark of the exam, in percent.
	PassingScorePct float64
}

// flagAIGrade returns the flag codes of one AI score for one answer. It never returns nil, so that an unflagged
// suggestion is stored as the empty array '{}' (AC-14).
func flagAIGrade(answerText string, suggestedScorePct float64, modelAnswer string, ctx flagContext) []string {
	flags := []string{}
	if hasInstructionPhrase(answerText) {
		flags = append(flags, aiFlagInstructionLike)
	}
	if isShortHigh(answerText, suggestedScorePct, modelAnswer) {
		flags = append(flags, aiFlagShortHigh)
	}
	if isDecisivePass(ctx) {
		flags = append(flags, aiFlagDecisivePass)
	}
	return flags
}

// hasInstructionPhrase reports whether the answer contains a phrase of the table, after lower-casing and collapsing
// whitespace.
func hasInstructionPhrase(answerText string) bool {
	normalized := strings.Join(strings.Fields(strings.ToLower(answerText)), " ")
	for _, phrase := range aiInstructionPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

// isShortHigh reports a short answer that the model scored high, against a model answer with substance.
func isShortHigh(answerText string, suggestedScorePct float64, modelAnswer string) bool {
	return utf8.RuneCountInString(strings.TrimSpace(answerText)) < aiFlagShortAnswerMaxRunes &&
		suggestedScorePct >= aiFlagShortHighMinScore &&
		utf8.RuneCountInString(modelAnswer) > aiFlagModelAnswerMinRunes
}

// isDecisivePass reports whether an AI score could decide pass or fail. The score is applied only when it cannot: the
// session has already passed on the graded rows (S0 reaches the pass mark), or it can no longer pass whatever the open
// rows get ((S0 + Open) falls below it). The AI score itself is not part of the test, so the outcome is the same for
// any score and in any order of grading. Without points there is no basis for the test, so the answer is flagged.
func isDecisivePass(ctx flagContext) bool {
	if ctx.TotalPoints <= 0 {
		return true
	}
	alreadyPassed := ctx.FinalPoints*100/ctx.TotalPoints >= ctx.PassingScorePct
	cannotPass := (ctx.FinalPoints+ctx.OpenPoints)*100/ctx.TotalPoints < ctx.PassingScorePct
	return !alreadyPassed && !cannotPass
}
