# FR-BB311 — Grading Engine

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB311 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Implemented |
| Depends On | FR-BB39 |

## Description
Implements the internal Go grading function that computes a numerical score for a submitted exam session. Handles four question types with distinct scoring rules: single/true-false (binary), multiple-choice (optional partial credit), Likert (weighted polarity), and short-text (deferred to manual grading). Stores per-question scores in `session_question_scores` and updates the session aggregate.

## Acceptance Criteria
- [x] AC-1: Single-choice and true-false questions score 1.0 point when the single selected option matches the correct option exactly, and 0.0 otherwise; no partial credit.
- [x] AC-2: Multiple-choice questions with exact match on the full set of correct options score 1.0; partial credit formula `max(0, correct_selected - incorrect_selected) / total_correct` applies otherwise; the result is clamped to [0, 1]. If `total_correct = 0` for a multiple-choice question (no correct options are marked), the grading engine returns an error and rolls back (treated the same as missing correct answer data under AC-10).
- [x] AC-3: Likert questions are normalised per-question: for the selected option, `contribution = likert_weight * (likert_polarity == 'positive' ? +1 : -1)`; the per-question score is `(contribution - min_q) / (max_q - min_q)` clamped to [0, 1], where `max_q` is the maximum weight among positive-polarity options and `min_q` is the minimum possible contribution; if `max_q - min_q == 0` (e.g. all options have `likert_weight = 0`), the score for that question is `0.0`.
- [x] AC-4: Short-text questions insert a `session_question_scores` row with `grading_status = 'pending_manual'` and `score = 0`; they contribute 0 to the numerator but their `max_score` still appears in the denominator.
- [x] AC-5: `session_question_scores` rows are inserted for every question in the session after grading; one row per question.
- [x] AC-6: Total `score_pct` = `ROUND(SUM(score) / SUM(max_score) * 100, 2)`; if `SUM(max_score) = 0` (edge case: only short-text questions), `score_pct = 0`.
- [x] AC-7: `exam_sessions.score_pct` and `exam_sessions.passed` are updated atomically within the same transaction as `session_question_scores` inserts.
- [x] AC-8: `passed` is set to `TRUE` if `score_pct >= exam.passing_score_pct`, `FALSE` otherwise.
- [x] AC-9: The grading function is a pure Go function (or method on a `GradingEngine` struct) with no HTTP dependencies; it accepts a `*sqlx.Tx` and session ID and returns `(score float64, passed bool, err error)`.
- [x] AC-10: If a question's correct answer data is missing from the database (data integrity error), the grading function returns an error and the calling transaction is rolled back.

## Scope

| Layer | Detail |
|-------|--------|
| Go package | `backend/internal/sessions/` (add `grading.go` and `grading_test.go`) |
| New DB table | `session_question_scores` |
| New migration | `016_session_question_scores.up.sql` / `.down.sql` |
| Frontend | None |

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
For each Likert question q:
    selected_option = session_answers[q].selected_option_ids[0]
    weight = answer_options[selected_option].likert_weight      // e.g. 1..5
    likert_polarity = answer_options[selected_option].likert_polarity  // 'positive' | 'negative'
    contribution = weight * (likert_polarity == 'positive' ? +1 : -1)

    max_q = max(likert_weight of all options for question q with polarity='positive')
    min_q = min(contribution possible) = -max_q  (or 0 if all options are positive)

    if max_q - min_q == 0:
        score_q = 0.0
    else:
        score_q = (contribution - min_q) / (max_q - min_q)
    
    score_q = clamp(score_q, 0.0, 1.0)
    max_score_q = 1.0
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
    Grade(tx *sqlx.Tx, sessionID string) (scorePct float64, passed bool, err error)
}

// DefaultGradingEngine implements GradingEngine.
type DefaultGradingEngine struct{}

func (e *DefaultGradingEngine) Grade(tx *sqlx.Tx, sessionID string) (float64, bool, error) {
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
- `likert_weight` and `likert_polarity` are stored on the `answer_options` table (added in migration 009); no additional migration is required for these columns.
- `ai_graded` status in `grading_status` ENUM is reserved for Phase 7 (AI-assisted grading) and should not be used in Phase 3 implementations.
- When all `pending_manual` scores for a session are manually resolved, a separate endpoint (future requirement) will recompute `score_pct` and update `exam_sessions`; this is not in scope here.
- The partial-credit formula for multiple-choice prevents gaming by selecting all options (selecting all would result in `max(0, N - (total - N)) / N` where extra incorrect choices reduce the score).

## Out of Scope

- Phase 7 `ai_graded` status usage.
- Manual re-score endpoint (future requirement).
- Mixed Likert weight scales within one exam.
- HTTP handler for grading (grading is called internally during session submission).

## Test Strategy

- Unit tests (table-driven) for each question type in `grading_test.go`:
  - Single/true-false: correct, incorrect, no selection.
  - Multiple-choice: exact match (1.0), partial credit formula, all-wrong (0.0), zero `total_correct` (error).
  - Likert: positive/negative polarity, `likert_weight = 0` edge case, clamp to [0, 1].
  - Short-text: always returns `0` + `pending_manual` status.
- Integration test: submit a mixed-type session and confirm `session_question_scores` rows are created, `score_pct` and `passed` are updated atomically.
