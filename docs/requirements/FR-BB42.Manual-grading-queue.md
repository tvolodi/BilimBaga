# FR-BB42 — Manual Grading Queue

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB42 |
| Phase | 4 — Results & Certificates |
| Priority | 1 |
| Status | uat-verified |
| Depends On | FR-BB39, FR-BB311 |

## Scope

| Layer | Scope |
|-------|-------|
| Backend — sessions package | New handler/service/repository methods |
| Database | Migration 018: ALTER TABLE session_question_scores; seed grading permissions |
| Tests | Unit tests for score validation, transaction logic, re-grade path |
| Frontend | Out of scope (FR-4.7) |

## Description
Provides the backend API for reviewing and scoring short-text answers that require human judgment. Examiners can list sessions awaiting manual grading, open a session to see individual answers, and submit a numeric score (0–100%) with optional textual feedback per answer. Once every pending answer in a session is graded, the system recalculates the final score and transitions the session to `submitted`.

## Acceptance Criteria
- [ ] AC-1: `GET /admin/grading` returns only sessions with `status = 'grading_pending'`; sessions in any other state are excluded.
- [ ] AC-2: The grading queue supports filtering by `exam_id`, `date_from`, and `date_to` (all optional); date filters apply to `submitted_at`.
- [ ] AC-3: `GET /admin/grading` is paginated with `page` and `per_page` (default 20, max 100) and returns pagination metadata including `meta.total`.
- [ ] AC-4: `GET /admin/grading/:sessionId` returns only questions of type `short_text` that have `grading_status = 'pending_manual'`; already-graded short-text questions are included with their current `score_pct`.
- [ ] AC-5: `POST /admin/grading/:sessionId/answers/:questionId` returns HTTP 422 with `code: INVALID_SCORE` if `score_pct` is not in the range [0, 100].
- [ ] AC-6: After a grade is submitted, if all `session_question_scores` rows for the session have `grading_status != 'pending_manual'`, the system recalculates `exam_sessions.score_pct` using the grading engine logic and transitions `status` to `submitted`.
- [ ] AC-7: Final score recalculation and status transition after last-question grading execute inside a single database transaction.
- [ ] AC-8: Each grade submission writes an audit log entry: `action = 'answer.grade'`, `entity_type = 'session_answer'`, `entity_id = question_id`, metadata includes `grader_id`, `session_id`, `score_pct`.
- [ ] AC-9: `GET /admin/grading`, `GET /admin/grading/:sessionId`, and `POST /admin/grading/:sessionId/answers/:questionId` require the `grading:read` or `grading:write` permission respectively. Roles `examiner`, `department_admin`, and `super_admin` hold these permissions. Employees receive HTTP 403.
- [ ] AC-10: `graded_by`, `graded_at`, and `manual_feedback` are persisted on `session_question_scores` for every scored short-text answer.

## Technical Specification

### Database Schema

```sql
-- Add columns to existing session_question_scores table
ALTER TABLE session_question_scores
  ADD COLUMN manual_feedback    TEXT        NULL,
  ADD COLUMN graded_by          UUID        NULL REFERENCES users(id),
  ADD COLUMN graded_at          TIMESTAMPTZ NULL;
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/grading` | examiner+ | List sessions pending manual grading |
| GET | `/api/v1/admin/grading/:sessionId` | examiner+ | Session detail with answers to grade |
| POST | `/api/v1/admin/grading/:sessionId/answers/:questionId` | examiner+ | Submit a grade for one short-text answer |

#### Request / Response Shapes

**GET /api/v1/admin/grading?exam_id=&date_from=&date_to=&page=1&per_page=20 — Response (200)**
```json
{
  "data": {
    "items": [
      {
        "session_id": "uuid-session",
        "employee_name": "Aibek Seitkali",
        "exam_id": "uuid-exam",
        "exam_title": "Leadership Assessment",
        "submitted_at": "2026-05-13T14:00:00Z",
        "pending_question_count": 3
      }
    ],
    "meta": {
      "page": 1,
      "per_page": 20,
      "total": 12
    }
  },
  "error": null
}
```

**GET /api/v1/admin/grading/:sessionId — Response (200)**
```json
{
  "data": {
    "session_id": "uuid-session",
    "employee_name": "Aibek Seitkali",
    "exam_title": "Leadership Assessment",
    "submitted_at": "2026-05-13T14:00:00Z",
    "questions": [
      {
        "question_id": "uuid-q1",
        "stem": "Describe your approach to conflict resolution.",
        "text_answer": "I listen to both sides and...",
        "grading_status": "pending_manual",
        "current_score_pct": null,
        "manual_feedback": null
      },
      {
        "question_id": "uuid-q2",
        "stem": "Give an example of leadership under pressure.",
        "text_answer": "During a production outage...",
        "grading_status": "graded",
        "current_score_pct": 75.0,
        "manual_feedback": "Good example, but lacked measurable outcomes."
      }
    ]
  },
  "error": null
}
```

**POST /api/v1/admin/grading/:sessionId/answers/:questionId — Request**
```json
{
  "score_pct": 80.0,
  "feedback": "Well-structured answer with clear reasoning."
}
```

**POST /api/v1/admin/grading/:sessionId/answers/:questionId — Response (200, more answers pending)**
```json
{
  "data": {
    "question_id": "uuid-q1",
    "grading_status": "graded",
    "score_pct": 80.0,
    "session_status": "grading_pending",
    "all_graded": false
  },
  "error": null
}
```

**POST — Response (200, last answer graded — session auto-transitions)**
```json
{
  "data": {
    "question_id": "uuid-q3",
    "grading_status": "graded",
    "score_pct": 60.0,
    "session_status": "submitted",
    "final_score_pct": 72.33,
    "passed": true,
    "all_graded": true
  },
  "error": null
}
```

**Error — Score out of range (422)**
```json
{
  "data": null,
  "error": { "code": "INVALID_SCORE", "message": "score_pct must be between 0 and 100" }
}
```

### Repository Queries

```sql
-- Count pending questions per grading session
SELECT
  es.id AS session_id,
  u.full_name AS employee_name,
  e.id AS exam_id,
  e.title AS exam_title,
  es.submitted_at,
  COUNT(sqs.question_id) FILTER (WHERE sqs.grading_status = 'pending_manual') AS pending_question_count
FROM exam_sessions es
JOIN users u ON u.id = es.user_id
JOIN exams e ON e.id = es.exam_id
JOIN session_question_scores sqs ON sqs.session_id = es.id
WHERE es.status = 'grading_pending'
  AND ($1::UUID IS NULL OR e.id = $1)
  AND ($2::DATE IS NULL OR es.submitted_at::DATE >= $2)
  AND ($3::DATE IS NULL OR es.submitted_at::DATE <= $3)
GROUP BY es.id, u.full_name, e.id, e.title, es.submitted_at
ORDER BY es.submitted_at ASC
LIMIT $4 OFFSET $5;

-- Check if all answers graded after scoring one
SELECT COUNT(*) = 0 AS all_graded
FROM session_question_scores
WHERE session_id = $1 AND grading_status = 'pending_manual';
```

### RBAC Permissions

New permissions to seed in migration 018:
- `grading:read` — granted to: `examiner`, `department_admin`, `super_admin`
- `grading:write` — granted to: `examiner`, `department_admin`, `super_admin`

Middleware usage:
- `GET /admin/grading` — guarded by `rbac.RequirePermission(cache, "grading", "read")`
- `GET /admin/grading/:sessionId` — guarded by `rbac.RequirePermission(cache, "grading", "read")`
- `POST /admin/grading/:sessionId/answers/:questionId` — guarded by `rbac.RequirePermission(cache, "grading", "write")`

Migration 018 seeds these into the `permissions` and `role_permissions` tables.

### Final Score Recalculation

When `all_graded = true`, the service calls the same aggregate formula used by the grading engine:

```
score_pct = ROUND(SUM(score) / SUM(max_score) * 100, 2)
passed    = score_pct >= exam.passing_score_pct
```

Both `exam_sessions.score_pct`, `exam_sessions.passed`, and `exam_sessions.status = 'submitted'` are updated in a single transaction.

### Stem Locale Source

The `stem` field in `GET /admin/grading/:sessionId` responses is sourced from `question_translations` using `q.default_locale` from the `questions` table, consistent with existing question detail endpoints.

## Notes
- The `score` stored in `session_question_scores` for a short-text answer is `score_pct / 100 * max_score` to keep the aggregate formula consistent with auto-graded questions.
- Examiners may re-grade an already-graded short-text answer; subsequent submissions overwrite `score`, `manual_feedback`, `graded_by`, and `graded_at`, and retrigger the aggregate recalculation.

## Out of Scope

- **Frontend grading UI**: The examiner-facing grading screen (roadmap section 4.7) is out of scope for this requirement.
- **Certificate auto-generation**: Automatic certificate issuance upon session completion (FR-BB43) is out of scope for this requirement.

## Test Strategy

- **Unit tests**:
  - Score range validation: assert HTTP 422 / `INVALID_SCORE` when `score_pct < 0` or `score_pct > 100`.
  - Transaction atomicity: mock DB failure mid-transaction; assert session status does not transition and score is not persisted.
  - Re-grade path: submit a second grade for the same `questionId`; assert `score`, `manual_feedback`, `graded_by`, and `graded_at` are overwritten, not duplicated.
  - Audit log write: assert `action = 'answer.grade'` entry is written with correct `grader_id`, `session_id`, and `score_pct`.
- **Integration tests**:
  - Full `POST` flow that triggers session auto-transition: grade all pending short-text answers for a session and assert `exam_sessions.status` transitions to `submitted` and `score_pct` is recalculated correctly.
