---
id: ISS-049
title: "UAT Defect: Analytics & Reporting — Exam Analytics average/median score inflated (×100 double multiply)"
status: resolved
severity: high
layer: frontend
module: analytics / per-exam
tags: [uat, analytics, frontend]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/components/analytics/__tests__/StatsSummaryRow.test.tsx
---

## Symptom
UAT Scenario: `Per-Exam Analytics Page`, Step 4
Actor: Super Admin
Action: Assert summary statistics on Exam Analytics page — Average Score and Median Score values
Expected: avg_score and median_score are sent by the backend already in percentage form (e.g., 33.3 means 33.3%); they must be displayed as-is (e.g., "33.3%")
Actual: Average Score displays as 3330.0% (should be 33.3%); Median Score displays as 1666.5% (should be 16.7%). StatsSummaryRow.tsx multiplies these values by 100 a second time even though the backend has already sent them as percentage values.
Screenshot: frontend/screenshots/uat-analytics/s2-04-summary-stats.png

## Root Cause

`frontend/src/components/analytics/StatsSummaryRow.tsx` multiplied `avgScore` and `medianScore` by 100 when rendering (`(avgScore * 100).toFixed(1)`). However, the backend API (FR-BB52) already returns `avg_score` and `median_score` as percentage values (e.g., `33.3` = 33.3%, computed via `AVG(score_pct)` and `PERCENTILE_CONT(0.5)` directly on `score_pct` which is stored as a 0–100 value). This caused a double multiplication: 33.3 → 3330.0%.

`pass_rate` is correctly sent as a 0–1 decimal (e.g., `0.72`) so its `* 100` multiplication is correct.

## Fix Applied

In `StatsSummaryRow.tsx`, removed `* 100` from the `avgScore` and `medianScore` display expressions:
- Before: `` `${(avgScore * 100).toFixed(1)}%` ``
- After: `` `${avgScore.toFixed(1)}%` ``

Updated the regression test `StatsSummaryRow.test.tsx` to pass API-realistic values (`avgScore={75.0}` instead of `avgScore={0.75}`) matching the actual 0–100 percentage scale sent by the backend.

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/components/analytics/StatsSummaryRow.tsx` | Removed `* 100` from `avgScore` and `medianScore` display expressions |
| `frontend/src/components/analytics/__tests__/StatsSummaryRow.test.tsx` | Updated test fixtures from 0–1 decimal scale to 0–100 percentage scale to match API contract |

## Regression Test

`frontend/src/components/analytics/__tests__/StatsSummaryRow.test.tsx` — test "renders numeric values correctly" passes `avgScore={75.0}` and asserts `75.0%` is displayed (no multiplication).

## Resolution Results

- Tests: 259 passed, 0 failed (46 test files)
- Migration applied: no
- Build clean: yes
