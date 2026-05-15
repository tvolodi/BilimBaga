# FR-BB41 — Result Retrieval API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB41 |
| Phase | 4 — Results & Certificates |
| Priority | 1 |
| Status | implemented |
| Depends On | FR-BB311 |

## Description
Provides REST endpoints for retrieving graded exam results. Employees can fetch their own session result (score, pass/fail, question breakdown) and full session history for any exam. Admins and examiners can access any session result with the full answer breakdown regardless of `show_answers` configuration. Visibility of per-question detail for employees is controlled by the exam's `show_answers` setting.

## Scope

| Layer | Items |
|-------|-------|
| Database | Requires migration 017 (`backend/migrations/017_session_questions_rule_id.up.sql`). Reads from: `exam_sessions`, `exams`, `exam_sections`, `session_question_scores`, `session_questions`, `session_answers`, `questions`, `users` |
| API endpoints | `GET /api/v1/portal/sessions/:id/result`, `GET /api/v1/admin/sessions/:id/result`, `GET /api/v1/portal/exams/:id/history` |
| Go package | `backend/internal/sessions/` — new handler, service interface, and repository methods; routes registered in `backend/internal/router/router.go` |
| Frontend pages/components | Out of scope for this requirement |
| i18n keys | None — backend-only requirement |

## Acceptance Criteria
- [ ] AC-1: `GET /portal/sessions/:id/result` returns HTTP 403 if the session does not belong to the authenticated user.
- [ ] AC-2: `GET /portal/sessions/:id/result` returns HTTP 422 with `code: SESSION_IN_PROGRESS` if the session status is `in_progress`; employees may only view results after submission.
- [ ] AC-3: When `exam.show_answers = 'never'`, `per_question_breakdown` is absent from the JSON response (Go struct uses `omitempty`); no `null` or empty array is serialised for that field.
- [ ] AC-4: When `exam.show_answers = 'after_completion'` or `'after_all_attempts'`, `per_question_breakdown` is present and each item includes `employee_answer`, `correct_answer`, `points_earned`, `max_points`, and `explanation` (nullable).
- [ ] AC-5: `GET /admin/sessions/:id/result` is gated on the `exams:read` permission enforced via `rbac.RequirePermission(cache, "exams", "read")` (held by `examiner`, `department_admin`, and `super_admin` roles); always includes full `per_question_breakdown` regardless of `show_answers`.
- [ ] AC-6: `GET /portal/exams/:id/history` returns sessions in ascending `started_at` order; sessions with status `in_progress` are excluded.
- [ ] AC-7: `per_section_scores` is present and non-empty only when the exam has sections (`exam_sections` rows exist); otherwise the field is an empty array.
- [ ] AC-8: `time_taken_seconds` is computed as `EXTRACT(EPOCH FROM (submitted_at - started_at))`; returns null if `submitted_at` is null.
- [ ] AC-9: The session endpoints (`GET /portal/sessions/:id/result` and `GET /admin/sessions/:id/result`) return HTTP 404 with `code: SESSION_NOT_FOUND` if the session does not exist. The history endpoint (`GET /portal/exams/:id/history`) returns HTTP 404 with `code: EXAM_NOT_FOUND` if the exam does not exist.
- [ ] AC-10: The history endpoint supports pagination via `page` (default 1) and `per_page` (default 20, max 100) query parameters and includes `meta.total` in the response envelope alongside `meta.page` and `meta.per_page`.
- [ ] AC-11: `GET /admin/sessions/:id/result` returns HTTP 422 with `code: SESSION_IN_PROGRESS` when the referenced session has status `in_progress`.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/portal/sessions/:id/result` | any authenticated | Employee retrieves own session result |
| GET | `/api/v1/admin/sessions/:id/result` | `exams:read` permission | Admin retrieves any session result with full breakdown |
| GET | `/api/v1/portal/exams/:id/history` | any authenticated | Employee's chronological attempt history for one exam |

#### Request / Response Shapes

**GET /api/v1/portal/sessions/:id/result — Response (200, show_answers = 'after_completion')**
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
    "show_answers_mode": "after_completion",
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
    "per_section_scores": []  // empty because this exam fixture has no sections, not because show_answers='never' hides them; per AC-7, section scores are present or absent solely based on whether exam_sections rows exist
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
    "meta": {
      "page": 1,
      "per_page": 20,
      "total": 2
    }
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
-- Employee result: scoped to the authenticated user (tenant isolation via user ownership)
-- attempt_number is derived via window function; exam_sessions has no stored attempt_number column
SELECT
  es.id AS session_id,
  es.exam_id,
  e.title AS exam_title,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT AS time_taken_seconds,
  ROW_NUMBER() OVER (PARTITION BY es.user_id, es.exam_id ORDER BY es.started_at)::INT AS attempt_number,
  es.submitted_at,
  e.show_answers AS show_answers_mode
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
WHERE es.id = $1 AND es.user_id = $2;

-- Admin result: scoped by session ID only; tenant isolation is enforced via JWT authentication
SELECT
  es.id AS session_id,
  es.exam_id,
  e.title AS exam_title,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT AS time_taken_seconds,
  ROW_NUMBER() OVER (PARTITION BY es.user_id, es.exam_id ORDER BY es.started_at)::INT AS attempt_number,
  es.submitted_at,
  e.show_answers AS show_answers_mode
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
WHERE es.id = $1;

-- Per-section scores (requires migration 017: session_questions.rule_id)
SELECT
  eqr.section_id,
  esec.title,
  ROUND(
    SUM(sqs.score) / NULLIF(SUM(sqs.max_score), 0) * 100, 2
  ) AS score_pct
FROM session_question_scores sqs
JOIN session_questions sq ON sq.session_id = sqs.session_id AND sq.question_id = sqs.question_id
JOIN exam_question_rules eqr ON eqr.id = sq.rule_id
JOIN exam_sections esec ON esec.id = eqr.section_id
WHERE sqs.session_id = $1
  AND sq.rule_id IS NOT NULL
GROUP BY eqr.section_id, esec.title
ORDER BY esec.sort_order;

-- Per-question breakdown: stem and explanation come from question_translations, not questions
-- $1 = session_id, $2 = locale (default 'ru' when the user has no preferred locale)
SELECT
  q.id AS question_id,
  qt.stem,
  sqs.score AS points_earned,
  sqs.max_score AS max_points,
  qt.explanation
FROM session_question_scores sqs
JOIN questions q ON q.id = sqs.question_id
JOIN question_translations qt ON qt.question_id = q.id AND qt.locale = $2
JOIN session_questions sq ON sq.session_id = sqs.session_id AND sq.question_id = sqs.question_id
WHERE sqs.session_id = $1
ORDER BY sq.sort_order;
```

> **Note**: The query above is simplified. It must be extended with joins to `session_answers` (for `employee_answer`) and `question_options` (for `correct_answer`) when `show_answers != 'never'` or on the admin endpoint. See service implementation for the full query.

### Required Migration

```sql
-- 017_session_questions_rule_id.up.sql
ALTER TABLE session_questions
  ADD COLUMN rule_id UUID REFERENCES exam_question_rules(id);
```

Nullable — rows where `rule_id IS NULL` represent questions added outside of any rule (edge case); section scores are only computed for rows with a non-null `rule_id`.

### Go Implementation Notes

- All new methods belong to the `backend/internal/sessions/` package: handler methods `GetSessionResult`, `GetAdminSessionResult`, `GetExamHistory`; corresponding service interface methods; repository interface additions with `sqlx` implementations.
- Routes registered in `backend/internal/router/router.go` under the authenticated group:
  - `r.Get("/portal/sessions/{id}/result", sessionsHandler.GetSessionResult)`
  - `r.With(rbac.RequirePermission(rbacCache, "exams", "read")).Get("/admin/sessions/{id}/result", sessionsHandler.GetAdminSessionResult)`
  - `r.Get("/portal/exams/{id}/history", sessionsHandler.GetExamHistory)`
- Migration 017 (`017_session_questions_rule_id.up.sql`) must be applied before deploying this feature.
- The `PerQuestionBreakdown` field on the result response struct must be tagged `json:"per_question_breakdown,omitempty"` so it is entirely absent (not `null`) when `show_answers = 'never'`.
- `employee_answer` and `correct_answer` are populated from `session_answers` and `question_options` respectively; the per-question breakdown query must be extended with these joins when `show_answers != 'never'` or on the admin endpoint.
- The admin route uses the `exams:read` permission (no new permission or migration needed); `rbac.RequirePermission(rbacCache, "exams", "read")` is the middleware pattern used throughout `router.go`.

## Notes
- `employee_answer` and `correct_answer` are arrays of option texts (not IDs) to keep the response self-contained and survive future option deletions.
- For `short_text` questions, `employee_answer` is the raw text string wrapped in a single-element array; `correct_answer` is null since there is no canonical answer.
- The admin endpoint at `/admin/sessions/:id/result` shares the same response shape but always populates `per_question_breakdown`.
- Valid `show_answers_policy` enum values (from migration 012): `'never'`, `'after_completion'`, `'after_all_attempts'`. The values `'after_submission'` and `'always'` do not exist in the database schema.
- The application is single-tenant at the DB level; `users` has no `tenant_id` column. Tenant isolation relies on JWT authentication alone.
- `stem` and `explanation` live in `question_translations(question_id, locale, stem, explanation)`, not in `questions`. Always join `question_translations` with the caller's preferred locale; default to `'ru'` when none is specified.

## Out of Scope

- Certificate PDF generation (FR-BB42)
- Frontend result page
- Push/email notifications
- Manual grading UI

## Test Strategy

- Handler unit tests for AC-1 (ownership check returns 403), AC-2 (in-progress returns 422), and AC-5 (RBAC permission middleware gates admin endpoint).
- Repository integration test asserting that `attempt_number` is correctly derived via `ROW_NUMBER() OVER (PARTITION BY user_id, exam_id ORDER BY started_at)`.
- Handler test asserting that `per_question_breakdown` is entirely absent (not `null`, not `[]`) from the serialised JSON when `show_answers = 'never'` (AC-3, `omitempty` tag).
- AC-4: Integration test seeding a submitted session with `show_answers = 'after_completion'`; assert each item in `per_question_breakdown` contains `employee_answer`, `correct_answer`, `points_earned`, `max_points`, and `explanation` (nullable).
- AC-6: Repository integration test seeding two submitted sessions and one in-progress session; assert history query returns only the two submitted sessions in ascending `started_at` order.
- AC-7: Repository integration test with two fixture exams — one with `exam_sections` rows and one without; assert `per_section_scores` is non-empty for the first and an empty array for the second.
- AC-8: Repository/SQL test asserting `time_taken_seconds` equals `EXTRACT(EPOCH FROM (submitted_at - started_at))::INT` for a known session; and returns `null` when `submitted_at IS NULL`.
- AC-9: Handler unit tests for all three endpoints with a non-existent session UUID or exam UUID; assert HTTP 404 with `code: SESSION_NOT_FOUND` for the two session endpoints (`GET /portal/sessions/:id/result` and `GET /admin/sessions/:id/result`), and HTTP 404 with `code: EXAM_NOT_FOUND` for the history endpoint (`GET /portal/exams/:id/history`).
- AC-10: Handler integration test issuing requests with `?page=2&per_page=5`; assert correct page of results is returned and response `data.meta` contains `page: 2`, `per_page: 5`, and accurate `total`; also assert `per_page=200` is clamped to 100.
