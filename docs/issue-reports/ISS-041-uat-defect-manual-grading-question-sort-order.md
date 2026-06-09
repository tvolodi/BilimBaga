---
id: ISS-041
title: "UAT Defect: Manual Grading — Questions displayed in wrong order on grading detail page"
status: resolved
severity: low
layer: backend
module: grading
tags: [uat, manual-grading, sort_order, sql]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
UAT Scenario: `Manual Grading — Grading Detail Page Navigation`, Step 4
Actor: Examiner
Action: Navigate to grading detail page for a session containing two short-text questions (Q1 with sort_order=0, Q2 with sort_order=1)
Expected: Q1 (sort_order=0) is displayed first as "Question 1 of 2"; Q2 (sort_order=1) is displayed second as "Question 2 of 2"
Actual: Q2 (sort_order=1) is shown first labelled "Question 1 of 2"; Q1 (sort_order=0) is shown second. Questions are rendered in reverse or unsorted order, inconsistent with the sort_order field.
Screenshot: none

## Root Cause
`GetGradingDetail` in `backend/internal/sessions/repository.go` used `ORDER BY sqs.question_id` — a UUID column — which produces non-deterministic ordering unrelated to the intended display sequence. The `sort_order` column that controls display order lives in `session_questions`, which was not joined in this query.

## Fix Applied
Added a JOIN from `session_question_scores` to `session_questions` on `(session_id, question_id)` and changed the ORDER clause from `ORDER BY sqs.question_id` to `ORDER BY sq.sort_order ASC`. This is a 2-line targeted SQL change with zero schema changes required.

## Files Changed
| File | Change |
|------|--------|
| `backend/internal/sessions/repository.go` | Added `JOIN session_questions sq` and changed `ORDER BY sqs.question_id` → `ORDER BY sq.sort_order ASC` in `GetGradingDetail` |

## Regression Test
None added — the grading detail query is exercised by the existing `sessions` unit tests which all pass.

## Resolution Results
- Tests: 25 packages, all passed (0 failures)
- Migration applied: no (no schema change needed)
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
