# FR-BB311 — Grading Engine

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB311 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB39 |

## Description
Implements the internal Go grading function that computes a numerical score for a submitted exam session. Handles four question types with distinct scoring rules: single/true-false (binary), multiple-choice (optional partial credit), Likert (weighted polarity), and short-text (deferred to manual grading). Stores per-question scores in `session_question_scores` and updates the session aggregate.

## Acceptance Criteria
- [ ] AC-1: Single-choice and true-false questions score 1.0 point when the single selected option matches the correct option exactly, and 0.0 otherwise; no partial credit.
- [ ] AC-2: Multiple-choice questions with exact match on the full set of correct options score 1.0; partial credit formula `max(0, correct_selected - incorrect_selected) / total_correct` applies otherwise; the result is clamped to [0, 1].
- [ ] AC-3: Likert questions: for each selected option, the contribution is `option.likert_weight`; positive polarity contributes positively, negative polarity contributes negatively; the sum is normalised to [0, 1] relative to the maximum possible Likert score in the session.
- [ ] AC-4: Short-text questions insert a `session_question_scores` row with `grading_status = 'pending_manual'` and `score = 0`; they contribute 0 to the numerator but their `max_score` still appears in the denominator.
- [ ] AC-5: `session_question_scores` rows are inserted for every question in the session after grading; one row per question.
- [ ] AC-6: Total `score_pct` = `ROUND(SUM(score) / SUM(max_score) * 100, 2)`; if `SUM(max_score) = 0` (edge case: only short-text questions), `score_pct = 0`.
- [ ] AC-7: `exam_sessions.score_pct` and `exam_sessions.passed` are updated atomically within the same transaction as `session_question_scores` inserts.
- [ ] AC-8: `passed` is set to `TRUE` if `score_pct >= exam.passing_score_pct`, `FALSE` otherwise.
- [ ] AC-9: The grading function is a pure Go function (or method on a `GradingEngine` struct) with no HTTP dependencies; it accepts a `*sqlx.Tx` and session ID and returns `(score float64, passed bool, err error)`.
- [ ] AC-10: If a question's correct answer data is missing from the database (data integrity error), the grading function returns an error and the calling transaction is rolled back.

## Technical Specification

### Grading Algorithm

#### Single Choice / True-False

```
score = 1.0  if  len(selected) == 1  AND  selected[0] == correct_option_id
score = 0.0  otherwise
max_score = 1.0
```

#### Multiple Choice (Partial Credit)

```
correct_set     = set of option IDs where is_correct = TRUE
selected_set    = set of selected_option_ids
total_correct   = |correct_set|
correct_selected = |selected_set ∩ correct_set|
incorrect_selected = |selected_set \ correct_set|

raw_score = max(0, correct_selected - incorrect_selected)
score     = raw_score / total_correct   (clamped to [0, 1])
max_score = 1.0
```

#### Likert

```
For each Likert question q in session:
    selected_option = session_answers[q].selected_option_ids[0]
    weight = question_options[selected_option].likert_weight    // e.g. 1..5
    polarity = question_options[selected_option].polarity       // 'positive' | 'negative'
    contribution = weight * (polarity == 'positive' ? +1 : -1)

max_likert_contribution = max(likert_weight) * question_count  // theoretical maximum
min_likert_contribution = -(max(likert_weight) * question_count)

// Normalise to [0, 1]
score = (sum_contributions - min_likert_contribution) / (max_likert_contribution - min_likert_contribution)
score = clamp(score, 0.0, 1.0)
max_score = 1.0 per Likert question (already normalised)
```

#### Short Text

```
score     = 0.0
max_score = 1.0
grading_status = 'pending_manual'
```

#### Aggregate

```
total_score   = SUM(score for all questions)
total_max     = SUM(max_score for all questions)
score_pct     = ROUND(total_score / total_max * 100, 2)   [0 if total_max == 0]
passed        = score_pct >= exam.passing_score_pct
```

### Database Schema (new table)

```sql
CREATE TYPE grading_status AS ENUM ('graded', 'pending_manual', 'ai_graded');

CREATE TABLE session_question_scores (
    session_id      UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id     UUID NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
    score           DECIMAL(6, 4) NOT NULL DEFAULT 0,
    max_score       DECIMAL(6, 4) NOT NULL DEFAULT 1,
    grading_status  grading_status NOT NULL DEFAULT 'graded',
    PRIMARY KEY (session_id, question_id)
);

CREATE INDEX idx_session_question_scores_session_id
    ON session_question_scores(session_id);

CREATE INDEX idx_session_question_scores_pending
    ON session_question_scores(session_id)
    WHERE grading_status = 'pending_manual';
```

### Go Interface

```go
// GradingEngine grades a fully-submitted session.
type GradingEngine interface {
    Grade(tx *sqlx.Tx, sessionID uuid.UUID) (scorePct float64, passed bool, err error)
}

// DefaultGradingEngine implements GradingEngine.
type DefaultGradingEngine struct{}

func (e *DefaultGradingEngine) Grade(tx *sqlx.Tx, sessionID uuid.UUID) (float64, bool, error) {
    exam := fetchExamForSession(tx, sessionID)
    questions := fetchSessionQuestionsWithAnswers(tx, sessionID)

    var totalScore, totalMax float64

    for _, q := range questions {
        score, maxScore, status := e.gradeQuestion(q)
        insertSessionQuestionScore(tx, sessionID, q.ID, score, maxScore, status)
        totalScore += score
        totalMax   += maxScore
    }

    var scorePct float64
    if totalMax > 0 {
        scorePct = math.Round(totalScore/totalMax*100*100) / 100
    }
    passed := scorePct >= exam.PassingScorePct

    return scorePct, passed, nil
}
```

## Notes
- `likert_weight` and `polarity` must be stored on `question_options` table; if those columns do not exist yet, a migration must add them (coordinate with FR-BB22 / question model).
- The Likert normalisation formula assumes all Likert options within a session share the same weight scale (e.g. 1–5); mixed scales within one exam are not supported in Phase 3.
- `ai_graded` status in `grading_status` ENUM is reserved for Phase 7 (AI-assisted grading) and should not be used in Phase 3 implementations.
- When all `pending_manual` scores for a session are manually resolved, a separate endpoint (future requirement) will recompute `score_pct` and update `exam_sessions`; this is not in scope here.
- The partial-credit formula for multiple-choice prevents gaming by selecting all options (selecting all would result in `max(0, N - (total - N)) / N` where extra incorrect choices reduce the score).
