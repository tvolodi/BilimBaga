# FR-BB34 — Employee Exam Portal API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB34 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Implemented |
| Depends On | FR-BB33 |

## Description
Provides read-only API endpoints for employees to discover which exams are assigned to them and to view their attempt history. Computes a per-exam status (not_started, in_progress, passed, failed, expired) by evaluating the user's assignment records, session history, and deadline. No session mutation occurs in this layer.

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/portal/exams` returns only exams the calling user is assigned to (directly, via department, or via 'all' assignment) and that are currently `active`.
- [ ] AC-2: Computed `user_status` is `expired` when the deadline has passed and the user has no `passed` session; `expired` takes priority over `failed`.
- [ ] AC-3: Computed `user_status` is `passed` when at least one session for this user and exam has `passed = TRUE`, regardless of deadline.
- [ ] AC-4: Computed `user_status` is `in_progress` when the user has an open session with `status = 'in_progress'` and `expires_at > NOW()`.
- [ ] AC-5: Computed `user_status` is `failed` when all allowed attempts are exhausted and no session has `passed = TRUE` and the deadline has not passed.
- [ ] AC-6: Computed `user_status` is `not_started` in all other cases.
- [ ] AC-7: `GET /api/v1/portal/exams/:id` returns HTTP 403 if the exam exists but is not assigned to the calling user.
- [ ] AC-8: `attempts_used` accurately reflects the count of non-`in_progress` sessions (submitted + auto_submitted) for the calling user on that exam.
- [ ] AC-9: The `deadline` returned is the earliest applicable deadline from all matching assignments for that user (individual > department > all).
- [ ] AC-10: Both endpoints require a valid JWT with any role; the response is scoped strictly to the authenticated user's assignments.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/portal/exams` | any authenticated | List assigned exams with computed status |
| GET | `/api/v1/portal/exams/:id` | any authenticated | Exam detail and attempt history |

#### Request / Response Shapes

**GET /api/v1/portal/exams — Response (200)**
```json
{
  "data": [
    {
      "id": "exam-uuid-1",
      "title": "Go Developer Certification",
      "description": "Tests core Go language proficiency.",
      "time_limit_minutes": 90,
      "passing_score_pct": 70.0,
      "max_attempts": 2,
      "attempts_used": 1,
      "deadline": "2026-07-01T23:59:59Z",
      "user_status": "in_progress",
      "open_session_id": "session-uuid-42"
    },
    {
      "id": "exam-uuid-2",
      "title": "Safety Induction",
      "description": null,
      "time_limit_minutes": 30,
      "passing_score_pct": 80.0,
      "max_attempts": 3,
      "attempts_used": 0,
      "deadline": null,
      "user_status": "not_started",
      "open_session_id": null
    }
  ],
  "error": null
}
```

**GET /api/v1/portal/exams/:id — Response (200)**
```json
{
  "data": {
    "id": "exam-uuid-1",
    "title": "Go Developer Certification",
    "description": "Tests core Go language proficiency.",
    "time_limit_minutes": 90,
    "passing_score_pct": 70.0,
    "max_attempts": 2,
    "shuffle_questions": true,
    "shuffle_options": true,
    "show_answers": "after_completion",
    "certificate_enabled": true,
    "deadline": "2026-07-01T23:59:59Z",
    "user_status": "in_progress",
    "attempts_used": 1,
    "open_session_id": "session-uuid-42",
    "attempt_history": [
      {
        "session_id": "session-uuid-10",
        "started_at": "2026-05-01T09:00:00Z",
        "submitted_at": "2026-05-01T10:20:00Z",
        "status": "submitted",
        "score_pct": 65.0,
        "passed": false
      }
    ]
  },
  "error": null
}
```

**Status Computation Logic (server-side, pseudocode)**
```
function computeUserStatus(exam, userSessions, deadline, now):
  openSession = userSessions.find(s => s.status == 'in_progress' AND s.expires_at > now)
  passedSession = userSessions.find(s => s.passed == true)
  finishedSessions = userSessions.filter(s => s.status IN ['submitted','auto_submitted'])

  if passedSession: return 'passed'
  if openSession:   return 'in_progress'
  if deadline != null AND deadline < now: return 'expired'
  if finishedSessions.count >= exam.max_attempts: return 'failed'
  return 'not_started'
```

**Error — Not Assigned (403)**
```json
{
  "data": null,
  "error": {
    "code": "EXAM_NOT_ASSIGNED",
    "message": "You do not have access to this exam."
  }
}
```

## Notes
- The assignment resolution query uses a UNION of three sub-queries: direct user assignment, department membership (with recursive CTE for sub-departments), and 'all' assignments.
- `open_session_id` is included in the list response so the frontend can deep-link directly to the resume flow without a second request.
- `show_answers` is intentionally exposed in the detail endpoint so the frontend can conditionally show the result review screen after submission.
- Attempt history is only returned in the detail endpoint to avoid bloating the list response.
