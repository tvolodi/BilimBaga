package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/bilimbaga/bilimbaga/internal/ai"
	"github.com/jmoiron/sqlx"
)

// GradingStatus represents the grading state of a session_question_scores row.
type GradingStatus string

const (
	GradingStatusGraded        GradingStatus = "graded"
	GradingStatusPendingManual GradingStatus = "pending_manual"
	GradingStatusAIGraded      GradingStatus = "ai_graded"
)

// GradingEngine grades a submitted exam session within a transaction.
// All writes happen within tx; the caller is responsible for commit/rollback.
type GradingEngine interface {
	Grade(tx *sqlx.Tx, sessionID string) (scorePct float64, passed bool, err error)
}

// DefaultGradingEngine is the standard Go implementation of GradingEngine.
type DefaultGradingEngine struct{}

// NewGradingEngine returns a GradingEngine backed by DefaultGradingEngine.
func NewGradingEngine() GradingEngine {
	return &DefaultGradingEngine{}
}

// gradingExamRow holds exam-level data needed for pass/fail determination.
type gradingExamRow struct {
	PassingScorePct float64 `db:"passing_score_pct"`
}

// gradingQuestionRow holds one session question row joined with its saved answer.
type gradingQuestionRow struct {
	QuestionID        string `db:"question_id"`
	QuestionType      string `db:"question_type"`
	SelectedOptionIDs []byte `db:"selected_option_ids"` // JSONB: []string
}

// gradingOptionRow holds one answer_options row needed for grading.
type gradingOptionRow struct {
	ID             string   `db:"id"`
	IsCorrect      bool     `db:"is_correct"`
	LikertWeight   *float64 `db:"likert_weight"`
	LikertPolarity *string  `db:"likert_polarity"`
}

// Grade runs the full grading cycle for a submitted session:
//  1. Fetches exam passing threshold.
//  2. Fetches all session questions and their saved answers.
//  3. Grades each question and inserts a session_question_scores row.
//  4. Computes ROUND(SUM(score)/SUM(max_score)*100, 2) and updates exam_sessions.
//
// tx must be an active transaction; the caller commits or rolls back.
func (e *DefaultGradingEngine) Grade(tx *sqlx.Tx, sessionID string) (float64, bool, error) {
	// 1. Fetch exam passing score via the session.
	const examQ = `
SELECT e.passing_score_pct
FROM exams e
JOIN exam_sessions es ON es.exam_id = e.id
WHERE es.id = $1`

	var exam gradingExamRow
	if err := tx.QueryRowx(examQ, sessionID).StructScan(&exam); err != nil {
		return 0, false, fmt.Errorf("gradeSession: fetch exam: %w", err)
	}

	// 2. Fetch all session questions with saved answers (LEFT JOIN — unanswered → empty array).
	const questionsQ = `
SELECT
    sq.question_id,
    q.type AS question_type,
    COALESCE(sa.selected_option_ids, '[]'::jsonb) AS selected_option_ids
FROM session_questions sq
JOIN questions q ON q.id = sq.question_id
LEFT JOIN session_answers sa
    ON sa.session_id = sq.session_id AND sa.question_id = sq.question_id
WHERE sq.session_id = $1`

	qRows, err := tx.Queryx(questionsQ, sessionID)
	if err != nil {
		return 0, false, fmt.Errorf("gradeSession: fetch questions: %w", err)
	}
	var questions []gradingQuestionRow
	for qRows.Next() {
		var q gradingQuestionRow
		if err := qRows.StructScan(&q); err != nil {
			qRows.Close()
			return 0, false, fmt.Errorf("gradeSession: scan question: %w", err)
		}
		questions = append(questions, q)
	}
	qRows.Close()
	if err := qRows.Err(); err != nil {
		return 0, false, fmt.Errorf("gradeSession: iterate questions: %w", err)
	}

	// 3. Grade each question.
	var totalScore, totalMax float64

	for _, q := range questions {
		// Fetch answer options for this question.
		const optionsQ = `
SELECT id, is_correct, likert_weight, likert_polarity
FROM answer_options
WHERE question_id = $1`

		optRows, err := tx.Queryx(optionsQ, q.QuestionID)
		if err != nil {
			return 0, false, fmt.Errorf("gradeSession: fetch options for %s: %w", q.QuestionID, err)
		}
		var options []gradingOptionRow
		for optRows.Next() {
			var opt gradingOptionRow
			if err := optRows.StructScan(&opt); err != nil {
				optRows.Close()
				return 0, false, fmt.Errorf("gradeSession: scan option for %s: %w", q.QuestionID, err)
			}
			options = append(options, opt)
		}
		optRows.Close()
		if err := optRows.Err(); err != nil {
			return 0, false, fmt.Errorf("gradeSession: iterate options for %s: %w", q.QuestionID, err)
		}

		// Parse selected option IDs from JSONB.
		var selectedIDs []string
		if len(q.SelectedOptionIDs) > 0 {
			if err := json.Unmarshal(q.SelectedOptionIDs, &selectedIDs); err != nil {
				return 0, false, fmt.Errorf("gradeSession: parse selected_option_ids for %s: %w", q.QuestionID, err)
			}
		}

		// Compute per-question score.
		score, maxScore, status, err := gradeQuestion(q.QuestionType, options, selectedIDs)
		if err != nil {
			return 0, false, fmt.Errorf("gradeSession: grade question %s: %w", q.QuestionID, err)
		}

		// Insert session_question_scores row (AC-5).
		const insertQ = `
INSERT INTO session_question_scores (session_id, question_id, score, max_score, grading_status)
VALUES ($1, $2, $3, $4, $5)`
		if _, err := tx.Exec(insertQ, sessionID, q.QuestionID, score, maxScore, string(status)); err != nil {
			return 0, false, fmt.Errorf("gradeSession: insert score for %s: %w", q.QuestionID, err)
		}

		totalScore += score
		totalMax += maxScore
	}

	// 4. Compute aggregate score_pct (AC-6).
	var scorePct float64
	if totalMax > 0 {
		scorePct = math.Round(totalScore/totalMax*100*100) / 100
	}

	// AC-8: passed if score_pct >= passing_score_pct.
	passed := scorePct >= exam.PassingScorePct

	// 5. Update exam_sessions aggregate (AC-7).
	const updateQ = `UPDATE exam_sessions SET score_pct = $2, passed = $3 WHERE id = $1`
	if _, err := tx.Exec(updateQ, sessionID, scorePct, passed); err != nil {
		return 0, false, fmt.Errorf("gradeSession: update session: %w", err)
	}

	return scorePct, passed, nil
}

// gradeQuestion routes to the appropriate grading helper based on question type.
// Question type values match the DB CHECK constraint: 'single','multiple','truefalse','likert','shorttext'.
func gradeQuestion(questionType string, options []gradingOptionRow, selectedIDs []string) (score, maxScore float64, status GradingStatus, err error) {
	switch questionType {
	case "single", "truefalse":
		return gradeSingleOrTrueFalse(options, selectedIDs)
	case "multiple":
		return gradeMultipleChoice(options, selectedIDs)
	case "likert":
		return gradeLikert(options, selectedIDs)
	case "shorttext":
		// AC-4: short-text always deferred to manual grading.
		return 0.0, 1.0, GradingStatusPendingManual, nil
	default:
		// Unknown type: treat as graded with zero score to avoid blocking.
		return 0.0, 1.0, GradingStatusGraded, nil
	}
}

// gradeSingleOrTrueFalse grades a 'single' or 'truefalse' question (AC-1).
// score = 1.0 iff exactly one option is selected and that option has is_correct = true.
// Returns an error if no correct option exists in answer_options (AC-10).
func gradeSingleOrTrueFalse(options []gradingOptionRow, selectedIDs []string) (float64, float64, GradingStatus, error) {
	hasCorrect := false
	for _, o := range options {
		if o.IsCorrect {
			hasCorrect = true
			break
		}
	}
	if !hasCorrect {
		return 0, 0, GradingStatusGraded, fmt.Errorf("no correct option found for single/truefalse question")
	}

	if len(selectedIDs) == 1 {
		selID := selectedIDs[0]
		for _, o := range options {
			if o.ID == selID && o.IsCorrect {
				return 1.0, 1.0, GradingStatusGraded, nil
			}
		}
	}
	return 0.0, 1.0, GradingStatusGraded, nil
}

// gradeMultipleChoice grades a 'multiple' question with partial credit (AC-2).
// score = max(0, correct_selected - incorrect_selected) / total_correct, clamped [0,1].
// Returns an error if total_correct == 0 (AC-10).
func gradeMultipleChoice(options []gradingOptionRow, selectedIDs []string) (float64, float64, GradingStatus, error) {
	correctSet := make(map[string]bool, len(options))
	for _, o := range options {
		if o.IsCorrect {
			correctSet[o.ID] = true
		}
	}
	totalCorrect := len(correctSet)
	if totalCorrect == 0 {
		return 0, 0, GradingStatusGraded, fmt.Errorf("no correct options found for multiple question")
	}

	selectedSet := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selectedSet[id] = true
	}

	var correctSelected, incorrectSelected int
	for id := range selectedSet {
		if correctSet[id] {
			correctSelected++
		} else {
			incorrectSelected++
		}
	}

	rawScore := correctSelected - incorrectSelected
	if rawScore < 0 {
		rawScore = 0
	}
	score := float64(rawScore) / float64(totalCorrect)
	if score > 1.0 {
		score = 1.0
	}
	return score, 1.0, GradingStatusGraded, nil
}

// gradeLikert grades a 'likert' question using weighted polarity normalisation (AC-3).
//
//	contribution = weight * (+1 if positive, -1 if negative)
//	max_q        = max likert_weight across positive-polarity options
//	min_q        = -max_q if any negative-polarity options exist, else 0
//	score        = (contribution - min_q) / (max_q - min_q), clamped [0,1]
//	             = 0.0 if max_q - min_q == 0
func gradeLikert(options []gradingOptionRow, selectedIDs []string) (float64, float64, GradingStatus, error) {
	if len(selectedIDs) == 0 {
		return 0.0, 1.0, GradingStatusGraded, nil
	}

	// Find the selected option.
	selID := selectedIDs[0]
	var selected *gradingOptionRow
	for i := range options {
		if options[i].ID == selID {
			selected = &options[i]
			break
		}
	}
	if selected == nil {
		// Selected option not found in DB — treat as no selection.
		return 0.0, 1.0, GradingStatusGraded, nil
	}

	// Compute contribution from the selected option.
	weight := 0.0
	if selected.LikertWeight != nil {
		weight = *selected.LikertWeight
	}
	polarity := "positive"
	if selected.LikertPolarity != nil {
		polarity = *selected.LikertPolarity
	}
	sign := 1.0
	if polarity == "negative" {
		sign = -1.0
	}
	contribution := weight * sign

	// Compute max_q (max weight of positive options) and detect negative options.
	maxQ := 0.0
	hasNegative := false
	for _, o := range options {
		w := 0.0
		if o.LikertWeight != nil {
			w = *o.LikertWeight
		}
		p := "positive"
		if o.LikertPolarity != nil {
			p = *o.LikertPolarity
		}
		if p == "positive" && w > maxQ {
			maxQ = w
		}
		if p == "negative" {
			hasNegative = true
		}
	}

	minQ := 0.0
	if hasNegative {
		minQ = -maxQ
	}

	rangeQ := maxQ - minQ
	if rangeQ == 0 {
		return 0.0, 1.0, GradingStatusGraded, nil
	}

	score := (contribution - minQ) / rangeQ
	// Clamp to [0, 1].
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score, 1.0, GradingStatusGraded, nil
}

// ---------------------------------------------------------------------------
// AIGradingEngine — Anthropic-backed grading for short-text questions
// ---------------------------------------------------------------------------

// aiGradingQuestionRow extends gradingQuestionRow with auto_grade / model_answer / text_answer.
type aiGradingQuestionRow struct {
	QuestionID        string  `db:"question_id"`
	QuestionType      string  `db:"question_type"`
	SelectedOptionIDs []byte  `db:"selected_option_ids"`
	AutoGrade         bool    `db:"auto_grade"`
	ModelAnswer       *string `db:"model_answer"`
	TextAnswer        *string `db:"text_answer"`
}

// aiGradingResult is the JSON shape returned by the LLM for short-text grading.
type aiGradingResult struct {
	ScorePct  int    `json:"score_pct"`
	Reasoning string `json:"reasoning"`
}

// AIGradingEngine extends DefaultGradingEngine with Anthropic-backed short-text scoring.
type AIGradingEngine struct {
	aiClient ai.AnthropicClient
	model    string
	db       *sqlx.DB
	logger   *slog.Logger
}

// NewAIGradingEngine constructs an AIGradingEngine.
func NewAIGradingEngine(aiClient ai.AnthropicClient, model string, db *sqlx.DB, logger *slog.Logger) GradingEngine {
	return &AIGradingEngine{aiClient: aiClient, model: model, db: db, logger: logger}
}

// Grade runs the full grading cycle, using AI for auto-graded short-text questions.
func (e *AIGradingEngine) Grade(tx *sqlx.Tx, sessionID string) (float64, bool, error) {
	// 1. Fetch exam passing score via the session.
	const examQ = `
SELECT e.passing_score_pct
FROM exams e
JOIN exam_sessions es ON es.exam_id = e.id
WHERE es.id = $1`

	var exam gradingExamRow
	if err := tx.QueryRowx(examQ, sessionID).StructScan(&exam); err != nil {
		return 0, false, fmt.Errorf("aiGradeSession: fetch exam: %w", err)
	}

	// 2. Fetch all session questions with answers, plus auto_grade / model_answer / text_answer.
	const questionsQ = `
SELECT
    sq.question_id,
    q.type AS question_type,
    COALESCE(sa.selected_option_ids, '[]'::jsonb) AS selected_option_ids,
    q.auto_grade,
    q.model_answer,
    sa.text_answer
FROM session_questions sq
JOIN questions q ON q.id = sq.question_id
LEFT JOIN session_answers sa
    ON sa.session_id = sq.session_id AND sa.question_id = sq.question_id
WHERE sq.session_id = $1`

	qRows, err := tx.Queryx(questionsQ, sessionID)
	if err != nil {
		return 0, false, fmt.Errorf("aiGradeSession: fetch questions: %w", err)
	}
	var questions []aiGradingQuestionRow
	for qRows.Next() {
		var q aiGradingQuestionRow
		if err := qRows.StructScan(&q); err != nil {
			qRows.Close()
			return 0, false, fmt.Errorf("aiGradeSession: scan question: %w", err)
		}
		questions = append(questions, q)
	}
	qRows.Close()
	if err := qRows.Err(); err != nil {
		return 0, false, fmt.Errorf("aiGradeSession: iterate questions: %w", err)
	}

	// 3. Grade each question.
	var totalScore, totalMax float64

	for _, q := range questions {
		var score, maxScore float64
		var status GradingStatus
		var aiReasoning *string

		if q.QuestionType == "shorttext" {
			score, maxScore, status, aiReasoning = e.gradeShortText(sessionID, q.QuestionID, q.AutoGrade, q.ModelAnswer, q.TextAnswer)
		} else {
			// Non-shorttext: use the standard grading helpers.
			const optionsQ = `
SELECT id, is_correct, likert_weight, likert_polarity
FROM answer_options
WHERE question_id = $1`

			optRows, oErr := tx.Queryx(optionsQ, q.QuestionID)
			if oErr != nil {
				return 0, false, fmt.Errorf("aiGradeSession: fetch options for %s: %w", q.QuestionID, oErr)
			}
			var options []gradingOptionRow
			for optRows.Next() {
				var opt gradingOptionRow
				if err := optRows.StructScan(&opt); err != nil {
					optRows.Close()
					return 0, false, fmt.Errorf("aiGradeSession: scan option for %s: %w", q.QuestionID, err)
				}
				options = append(options, opt)
			}
			optRows.Close()
			if err := optRows.Err(); err != nil {
				return 0, false, fmt.Errorf("aiGradeSession: iterate options for %s: %w", q.QuestionID, err)
			}

			var selectedIDs []string
			if len(q.SelectedOptionIDs) > 0 {
				if err := json.Unmarshal(q.SelectedOptionIDs, &selectedIDs); err != nil {
					return 0, false, fmt.Errorf("aiGradeSession: parse selected_option_ids for %s: %w", q.QuestionID, err)
				}
			}

			score, maxScore, status, err = gradeQuestion(q.QuestionType, options, selectedIDs)
			if err != nil {
				return 0, false, fmt.Errorf("aiGradeSession: grade question %s: %w", q.QuestionID, err)
			}
		}

		const insertQ = `
INSERT INTO session_question_scores (session_id, question_id, score, max_score, grading_status, ai_reasoning)
VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := tx.Exec(insertQ, sessionID, q.QuestionID, score, maxScore, string(status), aiReasoning); err != nil {
			return 0, false, fmt.Errorf("aiGradeSession: insert score for %s: %w", q.QuestionID, err)
		}

		totalScore += score
		totalMax += maxScore
	}

	// 4. Compute aggregate score_pct.
	var scorePct float64
	if totalMax > 0 {
		scorePct = math.Round(totalScore/totalMax*100*100) / 100
	}

	passed := scorePct >= exam.PassingScorePct

	// 5. Update exam_sessions aggregate.
	const updateQ = `UPDATE exam_sessions SET score_pct = $2, passed = $3 WHERE id = $1`
	if _, err := tx.Exec(updateQ, sessionID, scorePct, passed); err != nil {
		return 0, false, fmt.Errorf("aiGradeSession: update session: %w", err)
	}

	return scorePct, passed, nil
}

// gradeShortText calls the LLM to score a short-text answer.
// Falls back to pending_manual on any error or missing data.
func (e *AIGradingEngine) gradeShortText(
	sessionID, questionID string,
	autoGrade bool,
	modelAnswer, textAnswer *string,
) (score, maxScore float64, status GradingStatus, aiReasoning *string) {
	// Cannot auto-grade without a model answer or employee answer.
	if !autoGrade || modelAnswer == nil || strings.TrimSpace(*modelAnswer) == "" || textAnswer == nil {
		return 0.0, 1.0, GradingStatusPendingManual, nil
	}

	prompt := fmt.Sprintf(`You are an exam grader. Compare the employee's answer to the model answer and provide a score.

Model Answer: %s

Employee's Answer: %s

Respond with ONLY valid JSON in this exact format, no other text:
{"score_pct": <integer 0-100>, "reasoning": "<brief explanation>"}

Where score_pct is 100 if the answer is fully correct, 0 if completely wrong, and a proportional value in between for partial credit.`,
		*modelAnswer, *textAnswer)

	text, tokensUsed, err := e.aiClient.GenerateText(context.Background(), prompt, e.model)
	if err != nil {
		e.logger.Warn("aiGradeShortText: GenerateText failed",
			"session_id", sessionID,
			"question_id", questionID,
			"error", err,
		)
		return 0.0, 1.0, GradingStatusPendingManual, nil
	}

	// Extract the first JSON object from the response.
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end < start {
		e.logger.Error("aiGradeShortText: no JSON object in response",
			"session_id", sessionID,
			"question_id", questionID,
			"response", text,
		)
		return 0.0, 1.0, GradingStatusPendingManual, nil
	}

	var result aiGradingResult
	if err := json.Unmarshal([]byte(text[start:end+1]), &result); err != nil {
		e.logger.Error("aiGradeShortText: JSON parse failed",
			"session_id", sessionID,
			"question_id", questionID,
			"error", err,
			"response", text,
		)
		return 0.0, 1.0, GradingStatusPendingManual, nil
	}

	if result.ScorePct < 0 || result.ScorePct > 100 {
		e.logger.Error("aiGradeShortText: score_pct out of range",
			"session_id", sessionID,
			"question_id", questionID,
			"score_pct", result.ScorePct,
		)
		return 0.0, 1.0, GradingStatusPendingManual, nil
	}

	e.logAIUsage(context.Background(), "short_text_grading", tokensUsed)
	reasoning := result.Reasoning
	return float64(result.ScorePct) / 100.0, 1.0, GradingStatusAIGraded, &reasoning
}

// logAIUsage records an ai_usage_log entry. Errors are logged but not propagated.
func (e *AIGradingEngine) logAIUsage(ctx context.Context, feature string, tokensUsed int) {
	if e.db == nil {
		return
	}
	const q = `INSERT INTO ai_usage_log (user_id, feature, tokens_used, model) VALUES (NULL, $1, $2, $3)`
	if _, err := e.db.ExecContext(ctx, q, feature, tokensUsed, e.model); err != nil {
		e.logger.Error("aiGradeShortText: ai_usage_log insert failed", "error", err)
	}
}
