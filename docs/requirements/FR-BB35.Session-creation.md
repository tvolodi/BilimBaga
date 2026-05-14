# FR-BB35 — Session Creation

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB35 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB34 |

## Description
Implements the `POST /api/v1/portal/exams/:id/sessions` endpoint that initialises a new exam session for the authenticated user. The endpoint enforces all pre-flight checks (assignment, availability window, attempt limits, no concurrent sessions), resolves questions from rules using deterministic seeded randomness, applies shuffle settings, and returns the full question set (without correct-answer flags) alongside the session token.

## Acceptance Criteria
- [ ] AC-1: Returns HTTP 403 if the exam is not assigned to the calling user.
- [ ] AC-2: Returns HTTP 422 with code `EXAM_NOT_ACTIVE` if the exam `status` is not `'active'`.
- [ ] AC-3: Returns HTTP 422 with code `EXAM_OUTSIDE_WINDOW` if `NOW()` is before `available_from` or after `available_until` (when those fields are set).
- [ ] AC-4: Returns HTTP 422 with code `ATTEMPTS_EXHAUSTED` if the count of finished (non-in_progress) sessions for this user+exam equals or exceeds `exam.max_attempts`.
- [ ] AC-5: Returns HTTP 409 with code `SESSION_ALREADY_OPEN` if an open `in_progress` session already exists for this user+exam with `expires_at > NOW()`.
- [ ] AC-6: For random-mode rules, selects exactly `rule.count` questions from the eligible active question pool using a seeded PRNG; the seed (unix nanoseconds at session creation) is stored in `exam_sessions.seed`.
- [ ] AC-7: `session_questions` rows are inserted with final `sort_order` after applying `shuffle_questions`; `shuffle_options` flag is applied at response serialisation time (option order randomised using same seed per question index).
- [ ] AC-8: The response includes `expires_at` and `remaining_seconds` (computed as `EXTRACT(EPOCH FROM expires_at - NOW())`).
- [ ] AC-9: Correct-answer flags and `is_correct` fields are never included in the response question/option objects.
- [ ] AC-10: The entire session creation (checks + inserts) executes within a single database transaction to prevent race conditions under concurrent requests from the same user.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/portal/exams/:id/sessions` | any authenticated | Start a new exam session |

#### Request / Response Shapes

**POST /api/v1/portal/exams/:id/sessions — Request**
```json
{}
```
*(No request body required; all parameters derived from exam config and calling user.)*

**POST /api/v1/portal/exams/:id/sessions — Response (201)**
```json
{
  "data": {
    "session_id": "sess-uuid-42",
    "exam_id": "exam-uuid-1",
    "started_at": "2026-05-14T10:00:00Z",
    "expires_at": "2026-05-14T11:30:00Z",
    "remaining_seconds": 5400,
    "questions": [
      {
        "id": "q-uuid-1",
        "sort_order": 0,
        "stem": "What does the `defer` keyword do in Go?",
        "type": "single_choice",
        "options": [
          { "id": "opt-uuid-a", "text": "Executes the function immediately." },
          { "id": "opt-uuid-b", "text": "Delays execution until the surrounding function returns." },
          { "id": "opt-uuid-c", "text": "Cancels the function call." },
          { "id": "opt-uuid-d", "text": "Runs the function in a separate goroutine." }
        ]
      },
      {
        "id": "q-uuid-2",
        "sort_order": 1,
        "stem": "Rate your confidence in Go concurrency primitives.",
        "type": "likert",
        "options": [
          { "id": "opt-uuid-e", "text": "Not confident at all" },
          { "id": "opt-uuid-f", "text": "Slightly confident" },
          { "id": "opt-uuid-g", "text": "Moderately confident" },
          { "id": "opt-uuid-h", "text": "Very confident" },
          { "id": "opt-uuid-i", "text": "Extremely confident" }
        ]
      }
    ]
  },
  "error": null
}
```

**Error — Attempts Exhausted (422)**
```json
{
  "data": null,
  "error": {
    "code": "ATTEMPTS_EXHAUSTED",
    "message": "You have used all allowed attempts for this exam."
  }
}
```

**Error — Concurrent Session (409)**
```json
{
  "data": null,
  "error": {
    "code": "SESSION_ALREADY_OPEN",
    "message": "You already have an active session for this exam. Resume it instead."
  }
}
```

### Question Selection Algorithm (Go pseudocode)

```go
// Executed inside the session-creation transaction
func resolveQuestions(db *sqlx.DB, exam Exam, seed int64) ([]SessionQuestion, error) {
    rng := rand.New(rand.NewSource(seed))
    var result []SessionQuestion

    for _, rule := range exam.Rules {
        var questions []Question
        if rule.Mode == "manual" {
            questions = fetchManualQuestions(db, rule.ID) // ordered by sort_order
        } else {
            pool := fetchEligibleQuestions(db, rule) // WHERE status='active', category, tags, difficulty
            shuffleWithRNG(pool, rng)
            questions = pool[:rule.Count]
        }
        result = append(result, questions...)
    }

    if exam.ShuffleQuestions {
        shuffleWithRNG(result, rng)
    }

    for i := range result {
        result[i].SortOrder = i
        if exam.ShuffleOptions {
            shuffleWithRNG(result[i].Options, rng)
        }
    }
    return result, nil
}
```

## Notes
- `fetchEligibleQuestions` filters by: `questions.status = 'active'`, optional `category_id`, optional `difficulty`, and `tags @> rule.tag_ids::uuid[]` (JSONB containment).
- If a random rule's pool has fewer active questions than `rule.count`, session creation fails with HTTP 422 code `INSUFFICIENT_QUESTIONS`; the exam should not have been publishable in this state, but real-time deactivation of questions can cause this edge case.
- The seed stored in `exam_sessions.seed` enables deterministic replay of question selection for audit and dispute resolution purposes.
- Short-text question responses must not include `options` array in the session response (or return an empty array).
