# FR-BB72 — Adaptive Difficulty

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB72 |
| Phase | 7 — AI Layer |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB311, FR-BB35, FR-BB36, FR-BB37 |

## Description
Adds an optional adaptive question-serving mode to the exam engine. When a published exam is configured with `adaptive = true`, questions are not pre-selected at session start; instead they are served one at a time with difficulty adjusted based on the employee's running performance using a simplified 3PL IRT heuristic. Non-adaptive sessions are entirely unaffected. Adaptive exams show a progress bar indicating questions answered rather than remaining.

## Acceptance Criteria
- [ ] AC-1: Exam configuration gains a nullable boolean field `adaptive` (default `false`); it can be set only for exams in `draft` status and is locked after publishing; existing exams default to `false` and behave identically to current behavior.
- [ ] AC-2: When publishing an adaptive exam, the system validates that the question bank for each rule contains at least 5 questions at each of the three difficulty levels (`easy`, `medium`, `hard`) and rejects publication with error code `INSUFFICIENT_ADAPTIVE_QUESTIONS` if not.
- [ ] AC-3: For adaptive sessions, `GET /api/v1/portal/sessions/:id/next-question` returns the next question chosen by the difficulty adjustment algorithm; the algorithm selects `medium` difficulty for the first question and adjusts based on the rolling 3-answer window.
- [ ] AC-4: The difficulty adjustment rule is: if the last 3 answered questions have >2 correct → increase difficulty one level; if <1 correct → decrease difficulty one level; otherwise maintain current difficulty. At the floor (`easy`) or ceiling (`hard`) no further adjustment is made.
- [ ] AC-5: The selected next question is recorded in session state (e.g. a `current_question_id` or `served_questions` JSONB column) so that repeated calls to `next-question` return the same unanswered question rather than selecting a new one.
- [ ] AC-6: Answering a question in an adaptive session (via the existing answer-save endpoint) marks it as answered in session state; the next call to `next-question` then picks the subsequent adaptive question.
- [ ] AC-7: The exam-taking frontend hides the total question count for adaptive sessions; the progress bar shows "Questions answered: N" rather than "N of M".
- [ ] AC-8: Non-adaptive session behavior is unchanged; all existing exam-taking tests continue to pass.

## Technical Specification

### Configuration / Infrastructure

**New migration** — `migrations/NNN_adaptive_exam.sql`:
```sql
ALTER TABLE exams
  ADD COLUMN adaptive BOOL NOT NULL DEFAULT false;

-- Session state: track served questions and current difficulty
ALTER TABLE exam_sessions
  ADD COLUMN adaptive_state JSONB;

-- adaptive_state schema:
-- {
--   "current_difficulty": "easy|medium|hard",
--   "served_question_ids": ["uuid", ...],
--   "recent_results": [true, false, true],   -- last 3 answers, most recent last
--   "current_question_id": "uuid | null"      -- pending unanswered question
-- }
```

### API Endpoints

#### GET /api/v1/portal/sessions/:id/next-question
- **Auth**: JWT; session must belong to the authenticated user
- **Behaviour for non-adaptive sessions**: returns `404` with code `NOT_ADAPTIVE` (non-adaptive clients should not call this endpoint; they use the pre-loaded question list)
- **Behaviour for adaptive sessions**:
  - If `adaptive_state.current_question_id` is non-null and unanswered: return that question (idempotent)
  - If all exam questions have been served (session complete): return `{ "data": { "done": true }, "error": null }` — triggers auto-submit
  - Otherwise: run selection algorithm, store selected question ID in `adaptive_state.current_question_id`, return question
- **Success response** `200`:
  ```json
  {
    "data": {
      "question": { /* same shape as FR-BB34 question object */ },
      "questions_answered": 7,
      "done": false
    },
    "error": null
  }
  ```

#### PUT /api/v1/admin/exams/:id (modified)
- Add `adaptive: bool` to request body.
- Validation: can only be set when exam `status = "draft"`.

#### POST /api/v1/admin/exams/:id/publish (modified)
- If `adaptive = true`: validate ≥5 questions per difficulty per rule set before publishing.
- Return `422` with `INSUFFICIENT_ADAPTIVE_QUESTIONS` on failure.

### Implementation Details

**Difficulty selection algorithm** (`internal/exams/adaptive.go`):
```go
type Difficulty string
const (
    Easy   Difficulty = "easy"
    Medium Difficulty = "medium"
    Hard   Difficulty = "hard"
)

var difficultyOrder = []Difficulty{Easy, Medium, Hard}

func NextDifficulty(current Difficulty, recentResults []bool) Difficulty {
    if len(recentResults) < 3 {
        return current // not enough data yet, maintain
    }
    last3 := recentResults[len(recentResults)-3:]
    correct := 0
    for _, r := range last3 {
        if r { correct++ }
    }
    idx := difficultyIndex(current)
    if correct > 2 && idx < len(difficultyOrder)-1 { return difficultyOrder[idx+1] }
    if correct < 1 && idx > 0                       { return difficultyOrder[idx-1] }
    return current
}

func difficultyIndex(d Difficulty) int {
    for i, v := range difficultyOrder { if v == d { return i } }
    return 1 // default medium
}
```

**Question selection**:
```go
func (s *SessionService) SelectNextAdaptiveQuestion(
    ctx context.Context,
    session *ExamSession,
) (*Question, error) {
    state := session.AdaptiveState
    nextDiff := NextDifficulty(state.CurrentDifficulty, state.RecentResults)

    // Fetch questions at nextDiff not yet served
    q, err := s.questionRepo.GetRandomByDifficultyExcluding(
        ctx, session.ExamID, nextDiff, state.ServedQuestionIDs,
    )
    if errors.Is(err, ErrNoQuestionsAvailable) {
        // Try adjacent difficulties before giving up
        q, err = s.questionRepo.GetRandomByDifficultyExcluding(
            ctx, session.ExamID, AnyDifficulty, state.ServedQuestionIDs,
        )
    }
    if err != nil { return nil, err }

    state.CurrentDifficulty = q.Difficulty
    state.CurrentQuestionID = &q.ID
    state.ServedQuestionIDs = append(state.ServedQuestionIDs, q.ID)
    if err := s.sessionRepo.UpdateAdaptiveState(ctx, session.ID, state); err != nil {
        return nil, err
    }
    return q, nil
}
```

**Answer save hook for adaptive sessions** — after saving an answer, append correctness to `adaptive_state.recent_results` and clear `current_question_id`:
```go
func (s *SessionService) RecordAdaptiveAnswer(ctx context.Context, sessionID, questionID uuid.UUID, correct bool) error {
    state, err := s.sessionRepo.GetAdaptiveState(ctx, sessionID)
    if err != nil { return err }
    state.RecentResults = append(state.RecentResults, correct)
    state.CurrentQuestionID = nil // cleared; next call to next-question picks new q
    return s.sessionRepo.UpdateAdaptiveState(ctx, sessionID, state)
}
```

This is called from the grading step within the answer-save handler, only when `session.Adaptive = true`.

### Frontend Components

**ExamTaking page** (`src/pages/portal/ExamTaking.tsx`) — adaptive mode changes:
- On mount for adaptive session: do NOT pre-fetch all questions. Instead fetch `GET /portal/sessions/:id/next-question`.
- After each answer submission, trigger `refetch` on the `next-question` query.
- Hide total question count from display; show `{t('session.questionsAnswered', { count: N })}`.
- Progress bar: `value={questionsAnswered}` with no `max` prop (indeterminate style), or `max={estimatedTotal}` if exam config exposes a max.
- When `done: true` is returned: auto-trigger session submit flow.

**Detection of adaptive vs non-adaptive**: read from session object returned by `GET /portal/sessions/:id` — if `session.adaptive === true`, use the next-question flow; otherwise use the pre-loaded question list flow.

## Notes
- The 3PL IRT approximation used here is intentionally simplified (window-based heuristic) rather than a full Bayesian IRT model. A proper IRT implementation could be added in a future phase if the data supports it.
- Exam time limits still apply to adaptive sessions; the timer runs from session start regardless of how many questions have been served.
- The `served_question_ids` array in `adaptive_state` grows throughout the session and is used to prevent re-serving questions. For very large question banks this is efficient enough; indexing is not required.
- Auto-submit (FR-BB310) must handle adaptive sessions: the submission hook should check `adaptive_state.served_question_ids.length` for progress rather than comparing `answered / total_questions`.
