# FR-BB52 — Per-Exam Analytics API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB52 |
| Phase | 5 — Analytics & Reporting |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB311 |

## Description
Delivers deep statistical analysis for a single exam: score distribution across 10 percentage buckets, pass rate, mean and median scores, attempt/participant counts, and per-question item analysis including correct rate, average answer time, and option selection distribution. This data powers the per-exam analytics frontend (FR-BB57) and the CSV export (FR-BB54).

## Acceptance Criteria
- [ ] AC-1: The endpoint requires `role IN (examiner, hr_admin, super_admin)` and scopes data to the caller's tenant; returns 404 if the exam does not exist in the tenant.
- [ ] AC-2: `score_distribution` always contains exactly 10 buckets labelled `"0-10"`, `"10-20"`, …, `"90-100"`, even if count is 0; the last bucket is inclusive of 100.
- [ ] AC-3: `pass_rate` is `passed_count / total_attempts`; if `total_attempts = 0`, returns 0.0.
- [ ] AC-4: `median_score` is computed using the `PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY score_pct)` PostgreSQL aggregate; returns null if no sessions exist.
- [ ] AC-5: `total_attempts` counts all sessions with `status IN ('submitted','grading_pending')`; `unique_participants` counts distinct `user_id` values among those sessions.
- [ ] AC-6: `per_question_stats` includes every question currently assigned to the exam (not deleted); `correct_rate` is the fraction of sessions where `score = max_score` for that question.
- [ ] AC-7: `avg_time_seconds` per question is computed as the average of `session_question_scores.time_taken_seconds` where that column is not null; returns null if no timing data exists.
- [ ] AC-8: `answer_distribution` in `per_question_stats` lists every option for that question with its selection count across all sessions; options with zero selections are included.
- [ ] AC-9: `stem_preview` is the first 100 characters of `questions.stem`; truncation does not cut mid-word (uses `LEFT(stem, 100)`).
- [ ] AC-10: The endpoint responds in under 1 second for exams with up to 5,000 sessions and 50 questions.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/exams/:id/analytics` | examiner+ | Full analytics for one exam |

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

### Repository Queries

```sql
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
    AND tenant_id = $2
    AND status IN ('submitted','grading_pending')
) sub
GROUP BY bucket;

-- Summary stats
SELECT
  ROUND(AVG(score_pct), 1) AS avg_score,
  PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY score_pct) AS median_score,
  COUNT(*) AS total_attempts,
  COUNT(DISTINCT user_id) AS unique_participants,
  ROUND(
    COUNT(*) FILTER (WHERE passed = TRUE)::DECIMAL / NULLIF(COUNT(*), 0), 4
  ) AS pass_rate
FROM exam_sessions
WHERE exam_id = $1
  AND tenant_id = $2
  AND status IN ('submitted','grading_pending');

-- Per-question stats
SELECT
  q.id AS question_id,
  LEFT(q.stem, 100) AS stem_preview,
  ROUND(
    COUNT(sqs.id) FILTER (WHERE sqs.score = sqs.max_score)::DECIMAL
    / NULLIF(COUNT(sqs.id), 0), 4
  ) AS correct_rate,
  ROUND(AVG(sqs.time_taken_seconds), 1) AS avg_time_seconds
FROM questions q
LEFT JOIN session_question_scores sqs ON sqs.question_id = q.id
  AND sqs.session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND tenant_id = $2 AND status IN ('submitted','grading_pending')
  )
WHERE q.exam_id = $1
GROUP BY q.id, q.stem
ORDER BY q.position;

-- Answer distribution per option (fetched in bulk, grouped in Go)
SELECT
  q.id AS question_id,
  qo.id AS option_id,
  LEFT(qo.text, 100) AS option_text,
  COUNT(sa.id) AS select_count
FROM questions q
JOIN question_options qo ON qo.question_id = q.id
LEFT JOIN session_answers sa ON sa.selected_option_id = qo.id
  AND sa.session_id IN (
    SELECT id FROM exam_sessions
    WHERE exam_id = $1 AND tenant_id = $2 AND status IN ('submitted','grading_pending')
  )
WHERE q.exam_id = $1
GROUP BY q.id, qo.id, qo.text
ORDER BY q.position, qo.position;
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
- `time_taken_seconds` on `session_question_scores` is not currently in the schema. The migration for this column should be added alongside the FR-BB52 implementation migration.
- For short-text questions, `answer_distribution` is omitted (empty array) since there are no discrete options.
- Per-question stats are sorted by question position to match the exam's question order in the UI.
