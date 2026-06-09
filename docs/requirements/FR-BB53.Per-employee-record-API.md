# FR-BB53 — Per-Employee Record API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB53 |
| Phase | 5 — Analytics & Reporting |
| Priority | 1 |
| Status | uat-verified |
| Depends On | FR-BB311 (Grading Engine), FR-BB43 (Certificate Generation) |

## Description
Enables examiners and department admins to retrieve a complete learning record for any individual employee. Provides a paginated session history (all exams attempted, scores, certificates) and a track-level progress summary showing how the employee is performing across the three compliance tracks: security, safety, and loyalty. This backs the employee record frontend page (FR-BB58).

## Acceptance Criteria
- [ ] AC-1: Both endpoints require `role IN (examiner, department_admin, super_admin)`; non-admin users receive HTTP 403.
- [ ] AC-2: Both endpoints return HTTP 404 if the target user ID does not exist.
- [ ] AC-3: `GET /admin/users/:id/record` returns sessions in descending `started_at` order; includes sessions with any status except `in_progress` (abandoned/expired sessions are included).
- [ ] AC-4: `certificate_id` in the record response is the `certificates.id` UUID if a certificate exists for that session; null otherwise.
- [ ] AC-5: The record endpoint supports pagination via `page` (default 1) and `per_page` (default 20, max 100); pagination metadata is returned in the top-level `meta` object (`{ page, per_page, total }`), not inside `data`.
- [ ] AC-6: `GET /admin/users/:id/progress` returns exactly three objects in the `tracks` array — one each for `security`, `safety`, and `loyalty` — even if the employee has no activity in a track.
- [ ] AC-7: `questions_answered` is the count of distinct questions the employee has answered across all submitted sessions for that track.
- [ ] AC-8: `last_activity` is the `submitted_at` of the most recent submitted session that involved questions from that track; null if no such session exists.
- [ ] AC-9: `required_exams` in the progress response lists all active exams assigned to the target employee (directly, via their department, or to all users) whose primary track (inferred via `exam_question_rules → categories.track`) matches the track; each item includes `passed: true/false/null` (null = never attempted).
- [ ] AC-10: Cross-user data access is prevented by verifying the target user ID exists before returning any data (HTTP 404 guard). The application is single-tenant per deployment; no SQL-level `tenant_id` filter is required. Authentication and role checks (JWT middleware) are the sole access control mechanism.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/users/:id/record` | examiner+ | Paginated session history for one employee |
| GET | `/api/v1/admin/users/:id/progress` | examiner+ | Track-level progress summary |

#### Request / Response Shapes

**GET /api/v1/admin/users/:id/record?page=1&per_page=20 — Response (200)**
```json
{
  "data": {
    "user_id": "uuid-user",
    "full_name": "Aibek Seitkali",
    "department": "Operations",
    "sessions": [
      {
        "session_id": "uuid-session",
        "exam_id": "uuid-exam",
        "exam_title": "Fire Safety Fundamentals",
        "started_at": "2026-05-14T10:00:00Z",
        "submitted_at": "2026-05-14T10:30:00Z",
        "score_pct": 84.50,
        "passed": true,
        "time_taken_seconds": 1800,
        "status": "submitted",
        "certificate_id": "uuid-cert"
      }
    ]
  },
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 7
  },
  "error": null
}
```

**GET /api/v1/admin/users/:id/progress — Response (200)**
```json
{
  "data": {
    "user_id": "uuid-user",
    "full_name": "Aibek Seitkali",
    "tracks": [
      {
        "track": "security",
        "questions_answered": 120,
        "last_activity": "2026-05-14T10:30:00Z",
        "required_exams": [
          {
            "exam_id": "uuid-exam-1",
            "title": "Security Awareness",
            "passed": true,
            "attempts": 2
          },
          {
            "exam_id": "uuid-exam-2",
            "title": "Data Protection",
            "passed": false,
            "attempts": 1
          }
        ]
      },
      {
        "track": "safety",
        "questions_answered": 45,
        "last_activity": "2026-04-10T14:20:00Z",
        "required_exams": [
          {
            "exam_id": "uuid-exam-3",
            "title": "Fire Safety Fundamentals",
            "passed": true,
            "attempts": 1
          }
        ]
      },
      {
        "track": "loyalty",
        "questions_answered": 0,
        "last_activity": null,
        "required_exams": []
      }
    ]
  },
  "error": null
}
```

### Repository Queries

```sql
-- Session history for one employee
-- $1 = target user_id, $2 = LIMIT (per_page), $3 = OFFSET ((page-1)*per_page)
SELECT
  es.id AS session_id,
  es.exam_id,
  e.title AS exam_title,
  es.started_at,
  es.submitted_at,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT AS time_taken_seconds,
  es.status,
  c.id AS certificate_id
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
LEFT JOIN certificates c ON c.session_id = es.id
WHERE es.user_id = $1
  AND es.status != 'in_progress'
ORDER BY es.started_at DESC
LIMIT $2 OFFSET $3;

-- Total count for pagination
-- $1 = target user_id
SELECT COUNT(*)
FROM exam_sessions
WHERE user_id = $1
  AND status != 'in_progress';

-- Track progress: questions answered + last activity per track
-- $1 = target user_id
-- Returns only tracks that have activity; Go service fills in zero-value entries for
-- missing tracks to guarantee the fixed three-element response array.
SELECT
  cat.track,
  COUNT(DISTINCT sqs.question_id) AS questions_answered,
  MAX(es.submitted_at) AS last_activity
FROM exam_sessions es
JOIN session_question_scores sqs ON sqs.session_id = es.id
JOIN questions q ON q.id = sqs.question_id
JOIN categories cat ON cat.id = q.category_id
WHERE es.user_id = $1
  AND es.status IN ('submitted', 'grading_pending')
  AND cat.track IN ('security', 'safety', 'loyalty')
GROUP BY cat.track;

-- Required exams per track with user's pass status.
-- "Required" = active exams assigned to this user directly, via their department, or to all users.
-- Exam track is inferred from the category of the first exam_question_rule (by sort_order).
-- $1 = target user_id
SELECT
  e.id AS exam_id,
  e.title,
  exam_track.track,
  BOOL_OR(es.passed) AS passed,
  COUNT(es.id) AS attempts
FROM exams e
-- Resolve exam's primary track from its first question rule's category
JOIN LATERAL (
  SELECT cat.track
  FROM exam_question_rules eqr
  JOIN categories cat ON cat.id = eqr.category_id
  WHERE eqr.exam_id = e.id
    AND cat.track IN ('security', 'safety', 'loyalty')
  ORDER BY eqr.sort_order
  LIMIT 1
) AS exam_track ON true
-- Scope to exams assigned to this user (direct, department-based, or blanket)
JOIN exam_assignments ea ON ea.exam_id = e.id
  AND (
    (ea.assignee_type = 'user'       AND ea.assignee_id = $1)
    OR (ea.assignee_type = 'department'
        AND ea.assignee_id = (SELECT department_id FROM users WHERE id = $1))
    OR (ea.assignee_type = 'all')
  )
-- Left-join sessions to get pass status; NULL attempts = never attempted
LEFT JOIN exam_sessions es ON es.exam_id = e.id
  AND es.user_id = $1
  AND es.status IN ('submitted', 'grading_pending')
WHERE e.status = 'active'
GROUP BY e.id, e.title, exam_track.track;
```

### Track Assembly in Go

The progress endpoint assembles results in the Go service layer: it fetches the track activity rows and required exam rows, then merges them into a fixed three-element `tracks` array, filling in zero-value entries for tracks with no activity.

```go
var allTracks = []string{"security", "safety", "loyalty"}

func buildTrackProgress(activity []TrackActivity, exams []ExamProgress) []TrackSummary {
    // index by track
    actMap := map[string]TrackActivity{}
    for _, a := range activity { actMap[a.Track] = a }

    examsByTrack := map[string][]ExamProgress{}
    for _, ex := range exams { examsByTrack[ex.Track] = append(examsByTrack[ex.Track], ex) }

    result := make([]TrackSummary, 3)
    for i, t := range allTracks {
        a := actMap[t]
        result[i] = TrackSummary{
            Track:             t,
            QuestionsAnswered: a.QuestionsAnswered,
            LastActivity:      a.LastActivity,
            RequiredExams:     examsByTrack[t],
        }
    }
    return result
}
```

## Notes
- An exam's track is not stored on the `exams` table. It is inferred at query time from `exam_question_rules → categories.track`. The LATERAL subquery selects the `track` of the category referenced by the lowest-`sort_order` rule for each exam. Exams whose rules reference no category with a valid track are excluded from `required_exams`.
- `passed` in `required_exams` uses `BOOL_OR`: returns `true` if the employee has at least one passed session, `false` if attempted but never passed, and `null` if never attempted (LEFT JOIN produces no rows).
- `time_taken_seconds` is null for sessions where `submitted_at` is null (expired/abandoned without a submission timestamp).
- The application is single-tenant per deployment. `exam_sessions`, `exams`, and `users` have no `tenant_id` column. No SQL-level tenant filter is required; access control is enforced solely by the JWT middleware and role checks.
