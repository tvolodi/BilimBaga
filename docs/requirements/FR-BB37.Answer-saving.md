# FR-BB37 — Answer Saving

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB37 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB36 |

## Description
Implements the answer-saving (upsert) endpoint and the session-resume endpoint. The save endpoint validates session liveness, verifies option membership, and updates the `session_answers` table atomically. The resume endpoint reconstructs full session state (questions + all saved answers + remaining time) so a user can recover from a page reload or accidental navigation without losing progress.

## Acceptance Criteria
- [ ] AC-1: `PUT /api/v1/portal/sessions/:id/answers/:questionId` upserts the answer row; a second call with the same `questionId` overwrites the previous answer rather than creating a duplicate.
- [ ] AC-2: Returns HTTP 422 with code `SESSION_EXPIRED` if `expires_at <= NOW()` at the time of the request, even if `status` is still `'in_progress'`.
- [ ] AC-3: Returns HTTP 422 with code `SESSION_NOT_ACTIVE` if session `status` is not `'in_progress'`.
- [ ] AC-4: Returns HTTP 404 if `questionId` does not belong to the session's `session_questions` list.
- [ ] AC-5: Returns HTTP 400 if any UUID in `selected_option_ids` does not belong to the specified question's options.
- [ ] AC-6: The response to every successful save includes `remaining_seconds` computed server-side as `GREATEST(0, EXTRACT(EPOCH FROM expires_at - NOW()))`, making the client clock server-authoritative.
- [ ] AC-7: `GET /api/v1/portal/sessions/:id` returns HTTP 403 if the session does not belong to the calling user.
- [ ] AC-8: The resume response includes all questions (in their original shuffled order) and a map of all currently saved answers keyed by `question_id`.
- [ ] AC-9: `time_spent_seconds` in the request body is stored as-is; negative values are rejected with HTTP 400.
- [ ] AC-10: Both endpoints are scoped to the authenticated user; one user cannot save answers or read session state of another user's session.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| PUT | `/api/v1/portal/sessions/:id/answers/:questionId` | any authenticated | Upsert answer for one question |
| GET | `/api/v1/portal/sessions/:id` | any authenticated | Resume: full session state with saved answers |

#### Request / Response Shapes

**PUT /api/v1/portal/sessions/:id/answers/:questionId — Request**
```json
{
  "selected_option_ids": ["opt-uuid-b"],
  "text_answer": null,
  "time_spent_seconds": 45
}
```

**PUT /api/v1/portal/sessions/:id/answers/:questionId — Response (200)**
```json
{
  "data": {
    "question_id": "q-uuid-1",
    "saved_at": "2026-05-14T10:12:34Z",
    "remaining_seconds": 5166
  },
  "error": null
}
```

**GET /api/v1/portal/sessions/:id — Response (200)**
```json
{
  "data": {
    "session_id": "sess-uuid-42",
    "exam_id": "exam-uuid-1",
    "status": "in_progress",
    "started_at": "2026-05-14T10:00:00Z",
    "expires_at": "2026-05-14T11:30:00Z",
    "remaining_seconds": 5166,
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
        "stem": "Describe how Go's garbage collector works.",
        "type": "short_text",
        "options": []
      }
    ],
    "answers": {
      "q-uuid-1": {
        "selected_option_ids": ["opt-uuid-b"],
        "text_answer": null,
        "time_spent_seconds": 45,
        "saved_at": "2026-05-14T10:12:34Z"
      }
    }
  },
  "error": null
}
```

**Error — Session Expired (422)**
```json
{
  "data": null,
  "error": {
    "code": "SESSION_EXPIRED",
    "message": "Your exam session has expired."
  }
}
```

**Error — Invalid Option (400)**
```json
{
  "data": null,
  "error": {
    "code": "INVALID_OPTION",
    "message": "One or more selected option IDs do not belong to this question."
  }
}
```

### Upsert SQL Pattern

```sql
INSERT INTO session_answers
    (id, session_id, question_id, selected_option_ids, text_answer, saved_at, time_spent_seconds)
VALUES
    (gen_random_uuid(), $1, $2, $3::jsonb, $4, NOW(), $5)
ON CONFLICT (session_id, question_id)
DO UPDATE SET
    selected_option_ids = EXCLUDED.selected_option_ids,
    text_answer         = EXCLUDED.text_answer,
    saved_at            = EXCLUDED.saved_at,
    time_spent_seconds  = EXCLUDED.time_spent_seconds;
```

## Notes
- For `short_text` questions, `selected_option_ids` must be `[]` (empty array); a non-empty array for a short-text question is rejected with HTTP 400 code `INVALID_ANSWER_FORMAT`.
- For `single_choice` and `true_false` questions, `selected_option_ids` must contain exactly 0 or 1 elements; more than 1 is rejected with HTTP 400.
- The option-validation step fetches valid option IDs from the `question_options` table; this is a lightweight indexed lookup and is acceptable in the hot answer-save path.
- `remaining_seconds` is clamped to `GREATEST(0, ...)` to avoid negative values when a session is at exactly the expiry boundary.
- The resume endpoint (`GET /sessions/:id`) should also work for sessions with status `submitted` or `auto_submitted` (for the result review screen), but `remaining_seconds` is returned as `0` in those cases.
