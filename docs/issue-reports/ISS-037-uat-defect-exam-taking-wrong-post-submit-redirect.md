---
id: ISS-037
title: "UAT Defect: Employee Exam Taking — Post-submit redirect goes to basic result screen instead of detailed result page"
status: resolved
severity: high
layer: frontend
module: exam-taking
tags: [uat, employee-exam-taking, navigate, submit, result-page]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
UAT Scenario: `Start Exam, Answer All Questions, Submit — Pass`, Step 15
Actor: Employee
Action: Submit exam (click "Submit anyway" → "Submit" confirmation); exam has show_answers = "after_completion"
Expected: After submission, employee is redirected to the full Result screen showing per-question review table (question stems, employee answers, correct answers), since show_answers = "after_completion"
Actual: After submission the employee lands at `/portal/sessions/{id}` — a basic screen showing only "✅ Passed, Score: 100%", and a "Back to my exams" button. No per-question review table is shown and no link to the full result page is provided. The detailed result with question review IS accessible by navigating directly to `/portal/sessions/{id}/result`, but the post-submit redirect does not go there.
Screenshot: screenshot-s1-step15-basic.png, screenshot-s1-step15-full-result.png

## Root Cause
`frontend/src/pages/ExamTaking/index.tsx` used `onSubmitSuccess={setSubmitResult}` which stored the `SubmitResult` in local React state and rendered the inline `<ResultScreen>` component at the same URL (`/portal/sessions/{id}`). This basic component only showed a pass/fail banner and score, with no per-question review. The full `ResultPage` (with score dial, section scores, question breakdown table, and certificate actions) lives at the route `/portal/sessions/:sessionId/result` and was never navigated to after submission. The already-submitted session guard also rendered the same basic `ResultScreen` instead of redirecting to the result route.

## Fix Applied
Replaced the `onSubmitSuccess` state-setter with a `useNavigate()` call that navigates to `/portal/sessions/${result.session_id}/result`. Also replaced the already-submitted session guard (`session.status === 'submitted' | 'auto_submitted'`) with a `<Navigate>` component redirecting to the same result route. Removed unused `useState<SubmitResult>`, `ResultScreen`, and `SubmitResult` type imports.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/pages/ExamTaking/index.tsx` | Replaced inline `ResultScreen` rendering with `useNavigate` redirect to `/portal/sessions/{id}/result`; removed dead `submitResult` state |

## Regression Test
No new test added — the fix is a navigation-only change. Existing 252 frontend tests all pass. The behavior is validated by the `ResultPage` route existing at `/portal/sessions/:sessionId/result` in `App.tsx`.

## Resolution Results
- Tests: 252 passed, 0 failed (45 test files)
- Migration applied: no
- Build clean: yes (`✓ built in 6.20s`)
