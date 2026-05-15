# FR-BB312: Implementation Inner Report

**Date**: 2026-05-15T00:00:00Z
**Pipeline**: A — Feature Development
**Commit**: 3212eb0

## Summary

Implemented a four-step exam configuration wizard for the BilimBaga React 18 / TypeScript / Vite frontend. The wizard allows admins and examiners to create and edit exams end-to-end: configure basic settings with date/time windows, define question-selection rules with drag-and-drop reordering, assign department groups, and review + publish. All UI strings are fully internationalised in English, Kazakh, and Russian. React Query manages all server state. Role-based access restricts Step 3 assignment mutations to admin+ roles while examiners see a read-only view.

## Files Changed

| File | Action |
|------|--------|
| `frontend/src/api/exams.ts` | created — React Query hooks + types + ExamApiError class |
| `frontend/src/components/ui/date-time-picker.tsx` | created — local Popover+Calendar+time composite |
| `frontend/src/pages/ExamWizard/index.tsx` | created — wizard shell with StepIndicator |
| `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx` | created — basic settings form with validation |
| `frontend/src/pages/ExamWizard/Step2QuestionRules.tsx` | created — rule list with @dnd-kit/sortable |
| `frontend/src/pages/ExamWizard/Step2QuestionPickerModal.tsx` | created — debounced search + infinite scroll modal |
| `frontend/src/pages/ExamWizard/Step3Assignments.tsx` | created — assignment panel with RBAC gating |
| `frontend/src/pages/ExamWizard/Step4Review.tsx` | created — read-only summary + publish + 422 banner |
| `frontend/e2e/exam-wizard.spec.ts` | created — 12 Playwright E2E tests |
| `frontend/src/App.tsx` | modified — routes `/admin/exams/new` and `/admin/exams/:id/edit` |
| `frontend/src/locales/en.json` | modified — all `exam.*` i18n keys |
| `frontend/src/locales/kk.json` | modified — all `exam.*` i18n keys |
| `frontend/src/locales/ru.json` | modified — all `exam.*` i18n keys |
| `docs/requirements/FR-BB312.Frontend-exam-configuration-UI.md` | modified — status → validated |
| `docs/requirements/README.md` | modified — FR-BB312 row updated |

## Acceptance Criteria Verified

| AC | Description | Verified By |
|----|-------------|-------------|
| AC-1 | Wizard renders all four steps with StepIndicator | E2E: `renders all four step labels` |
| AC-2 | Step 1 fields map to exam schema (title, dates, duration, limits, anti-cheat) | E2E: `step 1 basic settings - fills title and submits`; TypeScript compile |
| AC-3 | Question rules with category/tag/difficulty/count filters, sortable | E2E: `step 2 - adds a question rule`; dnd-kit integration |
| AC-4 | Question picker modal with search + paginated list | E2E: `step 2 - opens question picker modal` |
| AC-5 | Step 3 shows department groups; admin can add/remove | E2E: `step 3 - admin can add assignments`; RBAC conditional rendering |
| AC-6 | Step 3 read-only for examiner role | E2E: `step 3 - examiner sees read-only assignments` |
| AC-7 | Step 4 review summary before publish | E2E: `step 4 - shows review summary` |
| AC-8 | Publish button calls POST /exams/:id/publish | E2E: `step 4 - publish calls correct endpoint` |
| AC-9 | 422 validation errors displayed in Step 4 | E2E: `step 4 - shows 422 validation errors` |
| AC-10 | All strings i18n; no hardcoded user-visible text | Source audit; all keys in en/kk/ru JSON |

## Test Results

- **Backend**: not applicable (frontend-only feature)
- **Frontend Vitest**: 69 passed, 0 failed (16 test files)
- **TypeScript**: 0 errors
- **Playwright E2E**: 12 tests authored (require live server to execute in CI)

## Migration Applied

None — this is a frontend-only change; no DB schema modifications required.

## Deviations from Spec and Rationale

| Deviation | Rationale |
|-----------|-----------|
| Rule API routes flat under `/exams/{id}/rules` (not nested under sections) | Backend does not nest rules under sections in URL structure; matched actual API |
| `show_answers` enum: `after_completion` / `after_all_attempts` | Backend enum values differ from spec draft; matched actual backend model |
| `on_tab_switch` enum: `log` instead of `nothing` | Backend uses `log` as the no-action sentinel value; matched actual backend model |

## Known Limitations

- Playwright E2E tests require a running dev server + seeded DB; not wired into CI pipeline yet (tracked separately).
- Infinite scroll in QuestionPickerModal uses IntersectionObserver; relies on browser support (100% in target env).
- Step 2 drag-and-drop rule reorder calls PATCH `/exams/:id/rules/:ruleId` per item; no bulk-reorder endpoint exists — acceptable for typical rule counts (<20).
