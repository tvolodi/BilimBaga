---
id: ISS-039
title: "UAT Defect: Employee Exam Taking — 'Retake Exam' button on result page navigates to non-existent route"
status: resolved
severity: high
layer: frontend
module: exam-taking
tags: [uat, employee-exam-taking, ResultActions, retake, navigate, route]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: [ISS-038]
regression_test: null
---

## Symptom
UAT Scenario: `Auto-Save Survives Browser Reload` / `Attempts Exhausted After Using Both Attempts` (result page)
Actor: Employee
Action: Click "Retake Exam" button visible on the exam result page after completing an attempt
Expected: Employee is navigated to a valid screen where they can start a new attempt (e.g., back to Employee Portal or to a route that allows starting a new session)
Actual: Clicking "Retake Exam" navigates to `/portal/exams/{id}`, which matches no route in the React Router configuration. A blank page is rendered with a console warning: "No routes matched location '/portal/exams/{id}'".
Screenshot: none

## Root Cause
`ResultActions.tsx` used `navigate(\`/portal/exams/${examId}\`)` for the "Retake Exam" button. The route `/portal/exams/:examId` does not exist in the React Router configuration (`App.tsx`). The valid portal routes are `/portal` (employee portal index), `/portal/sessions/:sessionId`, and `/portal/sessions/:sessionId/result`. Navigating to a non-existent route causes React Router to render nothing (blank page) with a console warning "No routes matched location".

## Fix Applied
Changed the "Retake Exam" button navigation in `ResultActions.tsx` from the non-existent `/portal/exams/${examId}` to `/portal`. The employee lands on the portal page where they can start a new attempt via the exam card "Start exam" CTA. The unused `examId` parameter was removed from the function destructuring (kept in the interface for non-breaking compatibility) to satisfy `noUnusedParameters` TypeScript strict mode.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/components/results/ResultActions.tsx` | Changed `navigate(\`/portal/exams/${examId}\`)` to `navigate('/portal')`; removed unused `examId` from function destructuring |

## Regression Test
None added — existing routing is covered by integration tests; the fix is a one-line navigation target change with no branching logic to test in isolation.

## Resolution Results
- Tests: 253 passed, 0 failed (45 test files)
- Migration applied: no
- Build clean: yes
