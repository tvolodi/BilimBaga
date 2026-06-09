---
id: ISS-036
title: "UAT Defect: Employee Exam Taking — Auto-save 'Saved ✓' indicator not displayed after answer selection"
status: resolved
severity: medium
layer: frontend
module: exam-taking
tags: [uat, employee-exam-taking]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/pages/ExamTaking/__tests__/QuestionDisplay.test.tsx
---

## Symptom
UAT Scenario: `Start Exam, Answer All Questions, Submit — Pass`, Steps 7, 9, 10
Actor: Employee
Action: Select an answer on Q1 (Step 7), Q2 (Step 9), Q3 (Step 10)
Expected: Answer is highlighted; an explicit "Saved ✓" (or "Saving…") indicator appears within 2 seconds of selection
Actual: Radio button is selected and highlighted (blue border); progress counter updates (e.g., "1 / 3 answered"); navigator button turns blue (answered state). No "Saved ✓" text indicator, toast, or transient label is shown at any point.
Screenshot: none

## Root Cause
The `SaveIndicator` component was implemented (`SaveIndicator.tsx`) and already wired to `saveStatus` state in `ExamLayout`, but it was only rendered inside `ExamTopBar` — in the top bar of the screen. When an employee selects an answer, their visual focus is on the answer options area (the question card in the main content area), not on the top bar. The tiny `text-xs` indicator in the crowded header was not noticed, making it effectively invisible from a UX perspective.

## Fix Applied
Added a per-question `SaveIndicator` rendered **inside each `QuestionDisplay` card**, directly below the answer options. This ensures the save status indicator appears right where the user's attention is focused when they interact with answers.

Changes made:
1. **`QuestionDisplay.tsx`**: Added optional `saveStatus?: SaveStatus` prop (defaults to `'idle'`). When `saveStatus !== 'idle'`, renders `<SaveIndicator status={saveStatus} />` right-aligned below the answer input area. Imports `SaveIndicator` and `SaveStatus` from the same directory.

2. **`ExamLayout.tsx`**: Added `savingQuestionId: string | null` state to track which question is actively being saved. Set `savingQuestionId` at the start of `triggerSave`. Updated `scheduleSaveStatusReset` to also clear `savingQuestionId` when the indicator resets to idle. Each `QuestionDisplay` in the non-adaptive render receives `saveStatus={question.id === savingQuestionId ? saveStatus : 'idle'}` so only the question being saved shows the indicator. In adaptive mode, the single active question always receives `saveStatus` directly.

The existing `SaveIndicator` in `ExamTopBar` is retained (secondary indicator in the header bar) but is no longer the only location.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/pages/ExamTaking/QuestionDisplay.tsx` | Added `saveStatus` prop; render `SaveIndicator` below answer options |
| `frontend/src/pages/ExamTaking/ExamLayout.tsx` | Added `savingQuestionId` state; pass per-question `saveStatus` to `QuestionDisplay` |
| `frontend/src/pages/ExamTaking/__tests__/QuestionDisplay.test.tsx` | New regression test file (4 tests) |

## Regression Test
`frontend/src/pages/ExamTaking/__tests__/QuestionDisplay.test.tsx`

Four tests covering: no indicator when idle (default), "Saving…" when status=saving, "Saved ✓" when status=saved, "Connection lost" when status=error.

## Resolution Results
- Tests: 252 passed (248 pre-existing + 4 new), 0 failed
- Migration applied: no
- Build clean: yes (tsc + vite build, no errors)
