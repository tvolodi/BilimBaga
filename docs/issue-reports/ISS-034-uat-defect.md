---
id: ISS-034
title: "UAT Defect: Exam Configuration — Active exam form fields not visually disabled in wizard"
status: open
severity: low
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
UAT Scenario: `Exam Configuration`, Step AC#5
Actor: Admin
Action: Opens an active (published) exam in the wizard and views Step 1 Basic Settings fields
Expected: Form fields are visually disabled (no `disabled` attribute or equivalent read-only styling) and a clear message indicates the exam is active and cannot be edited in this state
Actual: All Step 1 wizard form fields in `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx` remain fully editable for active exams — no `disabled` attribute is applied. The backend correctly enforces the constraint by returning a 409 error when a save is attempted, but the UI provides no upfront visual indication that editing is disallowed.
Screenshot: none

## Root Cause
`Step1BasicSettings.tsx` never checked `exam?.status` to derive a read-only state. Every form control (inputs, selects, switch toggles, textarea) was rendered without a `disabled` attribute regardless of whether the exam was active or draft. The local `Switch` sub-component also lacked a `disabled` prop entirely.

## Fix Applied
1. Added `const isReadOnly = exam?.status === 'active'` derived flag inside `Step1BasicSettings`.
2. Added `disabled={isReadOnly}` to: `Input` (title, timeLimitMinutes, passingScorePct, maxAttempts), `textarea` (description), `Select` (showAnswers, onTabSwitch), `DateTimePicker` (availableFrom, availableUntil), all three `Switch` toggles (shuffleQuestions, shuffleOptions, certificateEnabled), and the submit `Button`.
3. Extended the local `Switch` component to accept and propagate a `disabled` prop (HTML attribute + click guard + disabled CSS styles).
4. Added an amber informational banner at the top of Step 1 that renders only when `isReadOnly` is true, using i18n key `exam.wizard.activeReadOnlyNotice`.
5. Added `activeReadOnlyNotice` translation key to `en.json`, `kk.json`, and `ru.json`.
6. Fixed a pre-existing TypeScript error in `Step4Review.tsx` where the archive confirm dialog was missing from the JSX (causing `archiveDialogOpen` and `handleArchiveConfirm` to appear unused).

## Files Changed
- `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx`
- `frontend/src/pages/ExamWizard/Step4Review.tsx` (pre-existing TS blocker fix)
- `frontend/src/locales/en.json`
- `frontend/src/locales/kk.json`
- `frontend/src/locales/ru.json`

## Regression Test
New test file: `frontend/src/pages/ExamWizard/Step1BasicSettings.test.tsx`
- 7 test cases covering: inputs enabled for draft, no banner for draft, all inputs disabled for active, all switches disabled for active, submit button disabled for active, read-only banner visible for active, inputs enabled in create mode (null exam).

## Resolution Results
- `npm run build` — zero TypeScript errors, clean production build.
- `npm test` — 244/244 tests pass (7 new).
- `check:i18n` — all 720 locale keys present in all 3 locales.
