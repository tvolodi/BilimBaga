# FR-BB53 — Per-Employee Record API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB53 |
| Phase | 5 — Analytics & Reporting |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB311 |

## Description
Enables HR admins and examiners to retrieve a complete learning record for any individual employee. Provides a paginated session history (all exams attempted, scores, certificates) and a track-level progress summary showing how the employee is performing across the three compliance tracks: security, safety, and loyalty. This backs the employee record frontend page (FR-BB58).

## Acceptance Criteria
- [ ] AC-1: Both endpoints require `role IN (examiner, hr_admin, super_admin)`; non-admin users receive HTTP 403.
- [ ] AC-2: Both endpoints return HTTP 404 if the target user ID does not exist within the caller's tenant.
- [ ] AC-3: `GET /admin/users/:id/record` returns sessions in descending `started_at` order; includes sessions with any status except `in_progress` (abandoned/expired sessions are included).
- [ ] AC-4: `certificate_id` in the record response is the `certificates.id` UUID if a certificate exists for that session; null otherwise.
- [ ] AC-5: The record endpoint supports pagination via `page` (default 1) and `per_page` (default 20, max 100); response includes `total_count`.
- [ ] AC-6: `GET /admin/users/:id/progress` returns exactly three objects in the `tracks` array — one each for `security`, `safety`, and `loyalty` — even if the employee has no activity in a track.
- [ ] AC-7: `questions_answered` is the count of distinct questions the employee has answered across all submitted sessions for that track.
- [ ] AC-8: `last_activity` is the `submitted_at` of the most recent submitted session that involved questions from that track; null if no such session exists.
- [ ] AC-9: `required_exams` in the progress response lists all active exams whose `track = <track>` (from exam configuration); each item includes `passed: true/false/null` (null = never attempted).
- [ ] AC-10: All queries are scoped by `tenant_id` to prevent cross-tenant data leakage.

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
    ],
    "total_count": 7,
    "page": 1,
    "per_page": 20
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
  AND es.tenant_id = $2
  AND es.status != 'in_progress'
ORDER BY es.started_at DESC
LIMIT $3 OFFSET $4;

-- Track progress: questions answered + last activity
SELECT
  c.track,
  COUNT(DISTINCT sqs.question_id) AS questions_answered,
  MAX(es.submitted_at) AS last_activity
FROM exam_sessions es
JOIN session_question_scores sqs ON sqs.session_id = es.id
JOIN questions q ON q.id = sqs.question_id
JOIN categories cat ON cat.id = q.category_id
  JOIN (VALUES ('security'), ('safety'), ('loyalty')) c(track) ON cat.track = c.track
WHERE es.user_id = $1
  AND es.tenant_id = $2
  AND es.status IN ('submitted','grading_pending')
GROUP BY c.track;

-- Required exams per track with user's pass status
SELECT
  e.id AS exam_id,
  e.title,
  e.track,
  BOOL_OR(es.passed) AS passed,
  COUNT(es.id) AS attempts
FROM exams e
LEFT JOIN exam_sessions es ON es.exam_id = e.id
  AND es.user_id = $1
  AND es.status IN ('submitted','grading_pending')
WHERE e.tenant_id = $2
  AND e.status = 'active'
  AND e.track IN ('security','safety','loyalty')
GROUP BY e.id, e.title, e.track;
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
- The `exams.track` column needs to be added in a migration if not already present; see the exam configuration model (FR-BB31).
- `passed` in `required_exams` uses `BOOL_OR` to return `true` if the employee has at least one passed session; `false` if attempted but never passed; `null` if never attempted (no rows → LEFT JOIN returns null).
- `time_taken_seconds` is null for sessions where `submitted_at` is null (expired/abandoned without submission).
