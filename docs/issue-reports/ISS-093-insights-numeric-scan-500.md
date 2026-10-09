---
id: ISS-093
title: GET /admin/ai/insights/{examId} returns 500 (numeric passing_score_pct scanned into int)
status: resolved
severity: high
layer: backend
module: ai
tags: [passing_score_pct, numeric, StructScan, GetExamInsightData, examInsightRow]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-082]
regression_test: backend/internal/ai/repository_test.go
---

## Symptom
GET /api/v1/admin/ai/insights/{examId} returns 500 INTERNAL_ERROR with nothing in the log.

## Root Cause
exams.passing_score_pct is DECIMAL(5,2); pgx returns "70.00", which cannot scan into `examInsightRow.PassingScorePct int`. HandleGetInsights default branch also did not log the error.

## Fix Applied
Scan as float64 and `math.Round` to int for ExamInsightData (API/prompt contract unchanged; no frontend consumer). Handler now `slog.Error`s the wrapped error. Audited other ai queries: loyalty `ROUND(...)::int`, COUNT/MIN(sort_order) ints, aggregates already *float64 - no other mismatch. No SQL changed.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/ai/repository.go | float64 scan + round |
| backend/internal/ai/handler.go | log in default branch |
| backend/internal/ai/repository_test.go, handler_test.go | regression tests |

## Regression Test
TestGetExamInsightData_NumericPassingScore (fake driver returns "70.00"); TestHandler_GetInsights_UnexpectedErrorLoggedAnd500. schemaguard checks column names only; type checking would need a live DB (not done).

## Resolution Results
- Tests: all backend packages pass; build and vet clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
