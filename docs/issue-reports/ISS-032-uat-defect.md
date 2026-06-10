---
id: ISS-032
title: "UAT Defect: Exam Configuration — Missing Archive button in exam management UI"
status: open
severity: high
layer: frontend
module: exams
tags: [uat, exam-configuration]
created: 2026-06-09
resolved: null
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
UAT Scenario: `Exam Configuration`, Step AC#7
Actor: Admin
Action: Attempts to archive a published or unpublished exam from ExamsListPage or Step4Review
Expected: An Archive button is present in the exam management UI; clicking it archives the exam and updates its status accordingly
Actual: No Archive button exists anywhere in the frontend. The backend endpoint `POST /api/v1/exams/:id/archive` exists and functions correctly, but there is no `useArchiveExam` hook in `frontend/src/api/exams.ts` and no Archive button on `ExamsListPage` or `Step4Review`.
Screenshot: none

## Root Cause
The `useArchiveExam` mutation hook was never implemented in `frontend/src/api/exams.ts`, despite the backend exposing a `POST /api/v1/exams/:id/archive` endpoint. Without the hook, no frontend component could call the archive endpoint. Consequently, no Archive button was added to either `ExamsListPage.tsx` or `Step4Review.tsx`. The omission also meant no `exam.archive.*` i18n keys existed in any locale file.

## Fix Applied
1. Added `useArchiveExam()` mutation hook to `frontend/src/api/exams.ts` following the same pattern as `useUnpublishExam`.
2. Added Archive button to `ExamsListPage.tsx` for exams with `draft` or `active` status. The button opens a confirmation dialog; on success it shows a success message and invalidates the exam list query; on error it shows an error message.
3. Added Archive button to `Step4Review.tsx` alongside the Unpublish button (visible when exam is `active`). On success it navigates to `/admin/exams`.
4. Added `exam.archive.*` translation keys (`button`, `confirm`, `confirmDetail`, `success`, `error`) to all three locale files (`en.json`, `kk.json`, `ru.json`).
5. Build (`npm run build`) passes with zero TypeScript errors.

## Files Changed
- `frontend/src/api/exams.ts` — added `useArchiveExam` hook
- `frontend/src/pages/admin/ExamsListPage.tsx` — added Archive button with confirmation dialog
- `frontend/src/pages/ExamWizard/Step4Review.tsx` — added Archive button with confirmation dialog
- `frontend/src/locales/en.json` — added `exam.archive.*` keys
- `frontend/src/locales/kk.json` — added `exam.archive.*` keys
- `frontend/src/locales/ru.json` — added `exam.archive.*` keys

## Regression Test
`frontend/src/pages/admin/ExamsListPage.test.tsx` — 4 new tests:
- Renders Archive button for active exams
- Renders Archive button for draft exams
- Opens confirmation dialog when Archive button is clicked
- Calls archive endpoint and closes dialog on success

## Resolution Results
- All 237 frontend tests pass
- i18n check: 720 keys present in all 3 locales
- TypeScript build: clean
