# FR-BB51 — Dashboard Metrics API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB51 |
| Phase | 5 — Analytics & Reporting |
| Priority | 1 |
| Status | uat-verified |
| Depends On | FR-BB311, FR-BB33 |

## Description
Provides a single aggregated dashboard endpoint for HR admins and examiners. Returns four metric groups: per-exam completion/pass rates, a list of overdue employees, a recent activity feed, and average scores per compliance track. All data is scoped to the caller's tenant. Heavy queries are designed to be index-friendly and complete within a single SQL round-trip per group.

## Acceptance Criteria
- [ ] AC-1: The endpoint requires `role IN (examiner, hr_admin, super_admin)`; employees receive HTTP 403.
- [ ] AC-2: `completion_rate_by_exam` includes every active exam in the tenant; `assigned_count` is the number of distinct users assigned to the exam; `completed_count` is those with at least one `submitted` or `grading_pending` session; `passed_count` is those with at least one `passed = true` session.
- [ ] AC-3: `overdue_employees` contains at most 20 entries; an assignment is overdue when `deadline < NOW()` AND the user has no session with `passed = true` for that exam.
- [ ] AC-4: `recent_activity` contains the last 20 sessions (by `submitted_at` DESC) with `status IN ('submitted', 'grading_pending')`.
- [ ] AC-5: `avg_score_by_track` is computed over sessions submitted within the last 90 days; a track's average is null (not 0) if no sessions exist for that track in the period.
- [ ] AC-6: Track attribution is resolved by joining `session_question_scores → questions → categories` and reading `categories.track`; tracks are `security`, `safety`, `loyalty`.
- [ ] AC-7: The endpoint returns HTTP 200 with all four keys present even if individual arrays are empty or track averages are null.
- [ ] AC-8: All queries are scoped by `tenant_id` and never cross tenant boundaries.
- [ ] AC-9: The endpoint responds in under 500 ms for tenants with up to 10,000 sessions (enforced via integration test or documented index requirements).
- [ ] AC-10: An audit log entry is NOT written for dashboard reads (read-only reporting does not require audit).

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/dashboard` | examiner+ | Aggregated dashboard metrics |

#### Response Shape (200)

```json
{
  "data": {
    "completion_rate_by_exam": [
      {
        "exam_id": "uuid-exam",
        "title": "Fire Safety Fundamentals",
        "assigned_count": 150,
        "completed_count": 112,
        "passed_count": 98
      }
    ],
    "overdue_employees": [
      {
        "user_id": "uuid-user",
        "name": "Aibek Seitkali",
        "exam_title": "Security Awareness",
        "deadline": "2026-04-30T23:59:59Z"
      }
    ],
    "recent_activity": [
      {
        "session_id": "uuid-session",
        "employee_name": "Aibek Seitkali",
        "exam_title": "Fire Safety Fundamentals",
        "score_pct": 84.50,
        "passed": true,
        "submitted_at": "2026-05-14T10:30:00Z"
      }
    ],
    "avg_score_by_track": {
      "security": 71.4,
      "safety": 83.2,
      "loyalty": null
    }
  },
  "error": null
}
```

### Repository Queries

```sql
-- Completion rate by exam
SELECT
  e.id AS exam_id,
  e.title,
  COUNT(DISTINCT ea.user_id) AS assigned_count,
  COUNT(DISTINCT CASE WHEN es.status IN ('submitted','grading_pending') THEN ea.user_id END) AS completed_count,
  COUNT(DISTINCT CASE WHEN es.passed = TRUE THEN ea.user_id END) AS passed_count
FROM exams e
JOIN exam_assignments ea ON ea.exam_id = e.id
LEFT JOIN exam_sessions es ON es.exam_id = e.id AND es.user_id = ea.user_id
WHERE e.tenant_id = $1 AND e.status = 'active'
GROUP BY e.id, e.title;

-- Overdue employees (limit 20)
SELECT
  u.id AS user_id,
  u.full_name AS name,
  e.title AS exam_title,
  ea.deadline
FROM exam_assignments ea
JOIN users u ON u.id = ea.user_id
JOIN exams e ON e.id = ea.exam_id
WHERE ea.tenant_id = $1
  AND ea.deadline < NOW()
  AND NOT EXISTS (
    SELECT 1 FROM exam_sessions es
    WHERE es.exam_id = ea.exam_id AND es.user_id = ea.user_id AND es.passed = TRUE
  )
ORDER BY ea.deadline ASC
LIMIT 20;

-- Recent activity (last 20 submitted sessions)
SELECT
  es.id AS session_id,
  u.full_name AS employee_name,
  e.title AS exam_title,
  es.score_pct,
  es.passed,
  es.submitted_at
FROM exam_sessions es
JOIN users u ON u.id = es.user_id
JOIN exams e ON e.id = es.exam_id
WHERE es.tenant_id = $1
  AND es.status IN ('submitted', 'grading_pending')
ORDER BY es.submitted_at DESC
LIMIT 20;

-- Average score by track (last 90 days)
SELECT
  c.track,
  ROUND(AVG(es.score_pct), 1) AS avg_score
FROM exam_sessions es
JOIN session_question_scores sqs ON sqs.session_id = es.id
JOIN questions q ON q.id = sqs.question_id
JOIN categories c ON c.id = q.category_id
WHERE es.tenant_id = $1
  AND es.status = 'submitted'
  AND es.submitted_at >= NOW() - INTERVAL '90 days'
  AND c.track IN ('security','safety','loyalty')
GROUP BY c.track;
```

### Required Indexes

```sql
-- Ensure these indexes exist (add in a new migration if missing)
CREATE INDEX IF NOT EXISTS idx_exam_assignments_deadline ON exam_assignments(tenant_id, deadline)
  WHERE deadline IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_exam_sessions_submitted_at ON exam_sessions(tenant_id, submitted_at DESC)
  WHERE status IN ('submitted','grading_pending');
CREATE INDEX IF NOT EXISTS idx_questions_category_id ON questions(category_id);
```

## Notes
- If a track has no data in the last 90 days, its value in `avg_score_by_track` is JSON `null`, not `0`, to distinguish "no data" from "zero score".
- The four query groups can be executed concurrently in Go using goroutines with a `sync.WaitGroup` or `errgroup` to minimise total response latency.
- Dashboard data is not cached server-side; the frontend applies a 5-minute `staleTime` in React Query.
