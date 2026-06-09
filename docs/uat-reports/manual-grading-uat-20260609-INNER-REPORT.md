# UAT Inner Report — Manual Grading

| Field | Value |
|-------|-------|
| Run ID | manual-grading-uat-20260609 |
| Process | Manual Grading |
| Requirements | FR-BB42, FR-BB47 |
| Iterations | 3 |
| Final Decision | PASS |
| Date | 2026-06-09 |

## Defects Resolved

| ISS | Title | Severity | Layer | Fixed In |
|-----|-------|----------|-------|----------|
| ISS-040 | Grading API 500 — audit query parameter type | High | Backend | Iteration 1 |
| ISS-041 | Questions in wrong sort order on detail page | Low | Backend | Iteration 1 |
| ISS-042 | Employee redirected to blank /admin page | Low | Frontend | Iteration 1 |
| ISS-043 | Result page shows 0%/Failed for grading_pending | High | Both | Iteration 2 |
| ISS-044 | Retake button visible during grading_pending | Medium | Frontend | Iteration 2 |
| ISS-045 | Examiner feedback not shown on employee result | Medium | Both | Iteration 2 |

## Requirement Gaps (non-blocking)

| Gap | Description |
|-----|-------------|
| exam-filter-ui | No exam filter UI on grading queue page; backend supports ?exam_id= |
| grading-queue-pagination-text | meta.total not displayed in queue UI |

## Acceptance Criteria Coverage

- FR-BB42: 10/10 ACs verified
- FR-BB47: 10/11 ACs verified (AC-11 mutation-fail path: partial — network-fail simulation out of scope)
