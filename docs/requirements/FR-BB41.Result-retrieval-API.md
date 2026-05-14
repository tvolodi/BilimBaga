# FR-BB41 — Result Retrieval API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB41 |
| Phase | 4 — Results & Certificates |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB311 |

## Description
Provides REST endpoints for retrieving graded exam results. Employees can fetch their own session result (score, pass/fail, question breakdown) and full session history for any exam. Admins and examiners can access any session result with the full answer breakdown regardless of `show_answers` configuration. Visibility of per-question detail for employees is controlled by the exam's `show_answers` setting.

## Acceptance Criteria
- [ ] AC-1: `GET /portal/sessions/:id/result` returns HTTP 403 if the session does not belong to the authenticated user.
- [ ] AC-2: `GET /portal/sessions/:id/result` returns HTTP 422 with `code: SESSION_IN_PROGRESS` if the session status is `in_progress`; employees may only view results after submission.
- [ ] AC-3: When `exam.show_answers = 'never'`, the response omits `per_question_breakdown`; the field is absent from the JSON, not null.
- [ ] AC-4: When `exam.show_answers = 'after_submission'` or `'always'`, `per_question_breakdown` is present and each item includes `employee_answer`, `correct_answer`, `points_earned`, `max_points`, and `explanation` (nullable).
- [ ] AC-5: `GET /admin/sessions/:id/result` requires `role IN (examiner, hr_admin, super_admin)`; always includes full `per_question_breakdown` regardless of `show_answers`.
- [ ] AC-6: `GET /portal/exams/:id/history` returns sessions in ascending `started_at` order; sessions with status `in_progress` are excluded.
- [ ] AC-7: `per_section_scores` is present and non-empty only when the exam has sections (`exam_sections` rows exist); otherwise the field is an empty array.
- [ ] AC-8: `time_taken_seconds` is computed as `EXTRACT(EPOCH FROM (submitted_at - started_at))`; returns null if `submitted_at` is null.
- [ ] AC-9: All three endpoints return HTTP 404 with `code: NOT_FOUND` if the referenced session or exam does not exist within the caller's tenant.
- [ ] AC-10: The history endpoint supports pagination via `page` (default 1) and `per_page` (default 20, max 100) query parameters and includes `total_count` in the response envelope.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/portal/sessions/:id/result` | any authenticated | Employee retrieves own session result |
| GET | `/api/v1/admin/sessions/:id/result` | examiner+ | Admin retrieves any session result with full breakdown |
| GET | `/api/v1/portal/exams/:id/history` | any authenticated | Employee's chronological attempt history for one exam |

#### Request / Response Shapes

**GET /api/v1/portal/sessions/:id/result — Response (200, show_answers enabled)**
```json
{
  "data": {
    "session_id": "uuid-session",
    "exam_id": "uuid-exam",
    "exam_title": "Fire Safety Fundamentals",
    "score_pct": 84.50,
    "passed": true,
    "time_taken_seconds": 1423,
    "attempt_number": 1,
    "submitted_at": "2026-05-14T10:30:00Z",
    "show_answers_mode": "after_submission",
    "per_section_scores": [
      { "section_id": "uuid-sect-1", "title": "Theory", "score_pct": 90.0 },
      { "section_id": "uuid-sect-2", "title": "Practical", "score_pct": 78.0 }
    ],
    "per_question_breakdown": [
      {
        "question_id": "uuid-q1",
        "stem": "What is the correct evacuation procedure?",
        "employee_answer": ["Option B text"],
        "correct_answer": ["Option B text"],
        "points_earned": 1.0,
        "max_points": 1.0,
        "explanation": "Evacuation must start from the nearest exit."
      }
    ]
  },
  "error": null
}
```

**GET /api/v1/portal/sessions/:id/result — Response (200, show_answers = 'never')**
```json
{
  "data": {
    "session_id": "uuid-session",
    "exam_id": "uuid-exam",
    "exam_title": "Fire Safety Fundamentals",
    "score_pct": 84.50,
    "passed": true,
    "time_taken_seconds": 1423,
    "attempt_number": 1,
    "submitted_at": "2026-05-14T10:30:00Z",
    "show_answers_mode": "never",
    "per_section_scores": [],
    "per_question_breakdown": null
  },
  "error": null
}
```

**GET /api/v1/portal/exams/:id/history — Response (200)**
```json
{
  "data": {
    "exam_id": "uuid-exam",
    "exam_title": "Fire Safety Fundamentals",
    "sessions": [
      {
        "session_id": "uuid-session-1",
        "started_at": "2026-04-01T09:00:00Z",
        "submitted_at": "2026-04-01T09:24:00Z",
        "score_pct": 60.00,
        "passed": false,
        "status": "submitted"
      },
      {
        "session_id": "uuid-session-2",
        "started_at": "2026-05-14T10:00:00Z",
        "submitted_at": "2026-05-14T10:30:00Z",
        "score_pct": 84.50,
        "passed": true,
        "status": "submitted"
      }
    ],
    "total_count": 2,
    "page": 1,
    "per_page": 20
  },
  "error": null
}
```

**Error — Session in progress (422)**
```json
{
  "data": null,
  "error": { "code": "SESSION_IN_PROGRESS", "message": "Result is not available while session is in progress" }
}
```

### Repository Queries

```sql
-- Employee result with section scores
SELECT
  es.id AS session_id,
  es.exam_id,
  e.title AS exam_title,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT AS time_taken_seconds,
  es.attempt_number,
  es.submitted_at,
  e.show_answers AS show_answers_mode
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
WHERE es.id = $1 AND es.tenant_id = $2;

-- Per-section scores (computed from session_question_scores)
SELECT
  esec.id AS section_id,
  esec.title,
  ROUND(
    SUM(sqs.score) / NULLIF(SUM(sqs.max_score), 0) * 100, 2
  ) AS score_pct
FROM exam_sections esec
JOIN questions q ON q.section_id = esec.id
JOIN session_question_scores sqs ON sqs.question_id = q.id AND sqs.session_id = $1
WHERE esec.exam_id = $2
GROUP BY esec.id, esec.title;

-- Per-question breakdown
SELECT
  q.id AS question_id,
  q.stem,
  sqs.score AS points_earned,
  sqs.max_score AS max_points,
  q.explanation
FROM session_question_scores sqs
JOIN questions q ON q.id = sqs.question_id
WHERE sqs.session_id = $1
ORDER BY sqs.position;
```

## Notes
- `employee_answer` and `correct_answer` are arrays of option texts (not IDs) to keep the response self-contained and survives future option deletions.
- For `short_text` questions, `employee_answer` is the raw text string wrapped in a single-element array; `correct_answer` is null since there is no canonical answer.
- The admin endpoint at `/admin/sessions/:id/result` shares the same response shape but always populates `per_question_breakdown`.
