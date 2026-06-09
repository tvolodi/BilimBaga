---
id: ISS-046
title: "UAT Defect: Result & Certification — per_section_scores non-empty for flat exams"
status: resolved
severity: medium
layer: both
module: results
tags: [uat, result-and-certification]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/components/results/SectionScores.test.tsx
---

## Symptom
UAT Scenario: S1 (Flat Exam — Happy Path Result), Step 7; S7 (API assertion — per_section_scores for flat exam)
Actor: Employee
Action: View result page for a flat exam (no exam_sections rows); inspect API response `GET /api/v1/sessions/{id}/result`
Expected: `per_section_scores` is `[]` (empty array); no "SECTION SCORES" card is rendered in the UI
Actual: API returns `[{section_id: <rule_uuid>, title: "", score_pct: N}]` — a non-empty array using the question-rule UUID as a surrogate section ID. Frontend renders a spurious "SECTION SCORES" card on every flat exam result page.
Screenshot: none

**Violated acceptance criteria:**
- FR-BB41 AC-7: "per_section_scores is present and non-empty only when the exam has sections"
- FR-BB45 AC-4: Frontend "SECTION SCORES" card must not appear for flat exams

## Root Cause
`GetSectionScores` in `backend/internal/sessions/repository.go` grouped by `sq.rule_id` — which is always populated for every session question — with only a LEFT JOIN on `exam_sections`. For flat exams (no `exam_sections` rows), the LEFT JOIN produced `es.id = NULL` and `COALESCE(es.title, '') = ''` but still emitted one row per rule because the WHERE clause only checked `sq.rule_id IS NOT NULL`, not `eqr.section_id IS NOT NULL`. The frontend `SectionScores` component trusted the API response without filtering empty-title entries.

## Fix Applied
1. **Backend** (`repository.go`): Added `AND eqr.section_id IS NOT NULL` to the WHERE clause of the `GetSectionScores` SQL query. This ensures only rules that belong to an actual named `exam_sections` row are included, so flat exams return an empty array.
2. **Frontend** (`SectionScores.tsx`): Added a secondary guard — `sections.filter(s => s.title.trim() !== '')` — before rendering. The component now returns null when all entries have empty titles.

## Files Changed
| File | Change |
|------|--------|
| `backend/internal/sessions/repository.go` | Added `AND eqr.section_id IS NOT NULL` to `GetSectionScores` SQL WHERE clause |
| `frontend/src/components/results/SectionScores.tsx` | Filter entries with empty titles before rendering; use `validSections` instead of `sections` |
| `frontend/src/components/results/SectionScores.test.tsx` | New regression test file (5 tests covering empty array, empty titles, whitespace titles, valid sections, mixed) |

## Regression Test
`frontend/src/components/results/SectionScores.test.tsx` — 5 tests:
- Renders nothing for empty sections array
- Renders nothing when all sections have empty titles (ISS-046 direct regression)
- Renders nothing when all sections have whitespace-only titles
- Renders cards for valid non-empty title sections
- Filters out empty-title entries while rendering valid ones

## Resolution Results
- Tests: 259 frontend tests passed, 25 backend packages passed
- Migration applied: no (SQL fix is in query logic, not schema)
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
