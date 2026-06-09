# FR-BB52 — Per-Exam Analytics API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB52 |
| Phase | 5 — Analytics & Reporting |
| Priority | 1 |
| Status | uat-verified |
| Depends On | FR-BB311 |

## Description
Delivers deep statistical analysis for a single exam: score distribution across 10 percentage buckets, pass rate, mean and median scores, attempt/participant counts, and per-question item analysis including correct rate, average answer time, and option selection distribution. This data powers the per-exam analytics frontend (FR-BB57) and the CSV export (FR-BB54).

## Acceptance Criteria
- [ ] AC-1: The endpoint requires the caller to have `reports.read` permission (roles: `examiner`, `department_admin`, `super_admin`); returns 401 if unauthenticated, 403 if the role lacks the permission, and 404 if no exam with the given ID exists.
- [ ] AC-2: `score_distribution` always contains exactly 10 buckets labelled `"0-10"`, `"10-20"`, …, `"90-100"`, even if count is 0; the last bucket is inclusive of 100.
- [ ] AC-3: `pass_rate` is `passed_count / total_attempts`; if `total_attempts = 0`, returns 0.0.
- [ ] AC-4: `median_score` is computed using the `PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY score_pct)` PostgreSQL aggregate; returns null if no sessions exist.
- [ ] AC-5: `total_attempts` counts all sessions with `status IN ('submitted', 'auto_submitted', 'grading_pending')`; `unique_participants` counts distinct `user_id` values among those sessions. Sessions with status `in_progress` are excluded.
- [ ] AC-6: `per_question_stats` covers every distinct question that appeared in at least one completed session for the exam (sourced from `session_questions`); `correct_rate` is the fraction of `session_question_scores` rows for that question where `score = max_score`.
- [ ] AC-7: `avg_time_seconds` per question is computed as the average of `session_question_scores.time_taken_seconds` where that column is not null; returns null if no timing data exists. The `time_taken_seconds` column is added by migration 021.
- [ ] AC-8: `answer_distribution` in `per_question_stats` lists every `answer_options` row for that question with its selection count across all completed sessions; options with zero selections are included. Option text is sourced from `answer_translations` using the question's `default_locale`.
- [ ] AC-9: `stem_preview` is produced with `LEFT(qt.stem, 100)` where `qt` is the `question_translations` row for the question's `default_locale`.
- [ ] AC-10: The endpoint responds in under 1 second for exams with up to 5,000 sessions and 50 questions.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/exams/:id/analytics` | `reports.read` permission | Full analytics for one exam |

#### Response Shape (200)

```json
{
  "data": {
    "exam_id": "uuid-exam",
    "exam_title": "Fire Safety Fundamentals",
    "score_distribution": [
      { "bucket": "0-10",   "count": 2  },
      { "bucket": "10-20",  "count": 3  },
      { "bucket": "20-30",  "count": 5  },
      { "bucket": "30-40",  "count": 8  },
      { "bucket": "40-50",  "count": 12 },
      { "bucket": "50-60",  "count": 18 },
      { "bucket": "60-70",  "count": 25 },
      { "bucket": "70-80",  "count": 28 },
      { "bucket": "80-90",  "count": 14 },
      { "bucket": "90-100", "count": 5  }
    ],
    "pass_rate": 0.72,
    "avg_score": 74.3,
    "median_score": 76.0,
    "total_attempts": 120,
    "unique_participants": 98,
    "per_question_stats": [
      {
        "question_id": "uuid-q1",
        "stem_preview": "What is the correct evacuation procedure when...",
        "correct_rate": 0.83,
        "avg_time_seconds": 42.1,
        "answer_distribution": [
          { "option_id": "uuid-opt-a", "option_text": "Use the nearest exit", "select_count": 99 },
          { "option_id": "uuid-opt-b", "option_text": "Wait for announcement", "select_count": 18 },
          { "option_id": "uuid-opt-c", "option_text": "Call the fire department", "select_count": 3 }
        ]
      }
    ]
  },
  "error": null
}
```

#### Error Responses

| Status | `error.code` | Condition |
|--------|-------------|-----------|
| 401 | `UNAUTHORIZED` | Missing or invalid JWT |
| 403 | `FORBIDDEN` | Authenticated but role lacks `reports.read` permission |
| 404 | `EXAM_NOT_FOUND` | No exam exists with the given ID |
| 500 | `INTERNAL_ERROR` | Unexpected database or server error |

### Repository Queries

The completed-session subquery used in all queries below:

```sql
-- Reusable subquery: IDs of completed sessions for an exam
-- Included statuses: submitted, auto_submitted, grading_pending
-- auto_submitted sessions (time-expired) are included because they represent
-- genuine attempt data; excluding them would undercount attempts and distort analytics.
-- in_progress sessions are excluded as grading is not yet complete.
SELECT id FROM exam_sessions
WHERE exam_id = $1
  AND status IN ('submitted', 'auto_submitted', 'grading_pending')
```

```sql
-- Exam title lookup (404 if no row returned)
SELECT title FROM exams WHERE id = $1;

-- Score distribution (10 fixed buckets)
SELECT
  bucket,
  COUNT(*) AS count
FROM (
  SELECT
    CASE
      WHEN score_pct >= 90 THEN '90-100'
      WHEN score_pct >= 80 THEN '80-90'
      WHEN score_pct >= 70 THEN '70-80'
      WHEN score_pct >= 60 THEN '60-70'
      WHEN score_pct >= 50 THEN '50-60'
      WHEN score_pct >= 40 THEN '40-50'
      WHEN score_pct >= 30 THEN '30-40'
      WHEN score_pct >= 20 THEN '20-30'
      WHEN score_pct >= 10 THEN '10-20'
      ELSE '0-10'
    END AS bucket
  FROM exam_sessions
  WHERE exam_id = $1
    AND status IN ('submitted', 'auto_submitted', 'grading_pending')
) sub
GROUP BY bucket;

-- Summary stats
SELECT
  ROUND(AVG(score_pct), 1)                                              AS avg_score,
  PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY score_pct)               AS median_score,
  COUNT(*)                                                              AS total_attempts,
  COUNT(DISTINCT user_id)                                               AS unique_participants,
  ROUND(
    COUNT(*) FILTER (WHERE passed = TRUE)::DECIMAL / NULLIF(COUNT(*), 0), 4
  )                                                                     AS pass_rate
FROM exam_sessions
WHERE exam_id = $1
  AND status IN ('submitted', 'auto_submitted', 'grading_pending');

-- Per-question stats
-- Questions are sourced from session_questions (the resolved question set per session).
-- Distinct question_ids across all completed sessions give the full question pool for this exam.
-- Stem text comes from question_translations using the question's default_locale.
-- Questions are ordered by q.id for stable, deterministic output (no exam-level position column exists).
SELECT
  q.id                                                                  AS question_id,
  LEFT(qt.stem, 100)                                                    AS stem_preview,
  ROUND(
    COUNT(sqs.question_id) FILTER (WHERE sqs.score = sqs.max_score)::DECIMAL
    / NULLIF(COUNT(sqs.question_id), 0), 4
  )                                                                     AS correct_rate,
  ROUND(AVG(sqs.time_taken_seconds), 1)                                 AS avg_time_seconds
FROM (
  SELECT DISTINCT question_id
  FROM session_questions
  WHERE session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1
      AND status IN ('submitted', 'auto_submitted', 'grading_pending')
  )
) sq_dist
JOIN questions q ON q.id = sq_dist.question_id
JOIN question_translations qt ON qt.question_id = q.id AND qt.locale = q.default_locale
LEFT JOIN session_question_scores sqs ON sqs.question_id = q.id
  AND sqs.session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1
      AND status IN ('submitted', 'auto_submitted', 'grading_pending')
  )
GROUP BY q.id, qt.stem
ORDER BY q.id;

-- Answer distribution per option (fetched in bulk for all questions, grouped in Go)
-- answer_options replaces the non-existent question_options table.
-- Option text comes from answer_translations using the question's default_locale.
-- selected_option_ids is a JSONB array; containment operator @> is used for matching.
SELECT
  q.id                                                                  AS question_id,
  ao.id                                                                 AS option_id,
  LEFT(at2.text, 100)                                                   AS option_text,
  COUNT(sa.id) FILTER (
    WHERE sa.selected_option_ids @> jsonb_build_array(ao.id::text)
  )                                                                     AS select_count
FROM (
  SELECT DISTINCT question_id
  FROM session_questions
  WHERE session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1
      AND status IN ('submitted', 'auto_submitted', 'grading_pending')
  )
) sq_dist
JOIN questions q ON q.id = sq_dist.question_id
JOIN answer_options ao ON ao.question_id = q.id
JOIN answer_translations at2 ON at2.option_id = ao.id AND at2.locale = q.default_locale
LEFT JOIN session_answers sa ON sa.question_id = ao.question_id
  AND sa.session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1
      AND status IN ('submitted', 'auto_submitted', 'grading_pending')
  )
GROUP BY q.id, ao.id, at2.text
ORDER BY q.id, ao.sort_order;
```

### Score Distribution: Zero-Filling in Go

The SQL query returns only non-empty buckets. The Go handler must merge results against a fixed ordered bucket list to ensure all 10 buckets are always present:

```go
var allBuckets = []string{"0-10","10-20","20-30","30-40","40-50","50-60","60-70","70-80","80-90","90-100"}

func fillBuckets(raw []BucketCount) []BucketCount {
    m := make(map[string]int, len(raw))
    for _, b := range raw { m[b.Bucket] = b.Count }
    out := make([]BucketCount, len(allBuckets))
    for i, b := range allBuckets {
        out[i] = BucketCount{Bucket: b, Count: m[b]}
    }
    return out
}
```

## Notes
- **Tenant isolation**: The system is single-tenant per deployment. `exam_sessions` and `exams` have no `tenant_id` column. Tenant-level access control is enforced at the JWT middleware layer (claims-based). No `tenant_id` filter is applied in SQL queries.
- **`time_taken_seconds` column**: Not present in the current schema. Migration **021** must add `ALTER TABLE session_question_scores ADD COLUMN IF NOT EXISTS time_taken_seconds DECIMAL(8,2);` before this feature is deployed. Until then, `avg_time_seconds` will always be null.
- **`auto_submitted` sessions included**: Sessions with `status = 'auto_submitted'` (timer-expired auto-submit) are included in all analytics aggregates. These represent genuine completed attempts and excluding them would undercount participation and skew score distributions.
- **Question sourcing via `session_questions`**: Questions have no `exam_id` column and are not directly linked to exams. The `session_questions` table records the resolved question set for each session. Distinct `question_id` values across all completed sessions for an exam form the analytics question pool.
- **Stem text locale**: `question_translations.stem` is joined using `q.default_locale` as the fallback. If a question has no translation for its `default_locale`, the question is excluded from results (INNER JOIN semantics). Implementors may relax this to a LEFT JOIN with COALESCE if needed.
- **Option text locale**: `answer_translations` is joined on `at2.locale = q.default_locale` (same locale as the stem). Missing translations cause the option to be excluded (same caveat as above).
- **For short-text questions**: `answer_distribution` is omitted (empty array) since `answer_options` rows do not exist for `type = 'shorttext'` questions.
- **Question ordering**: Questions are ordered by `q.id` (UUID, deterministic) because no exam-level question position column exists. The `session_questions.sort_order` varies per session due to shuffling and cannot be used as a canonical position.
- **Roles with `reports.read` permission** (per migration 005): `examiner`, `department_admin`, and `super_admin`. The `employee` role does not have this permission.
