# uat-exam-configuration-20260609: Implementation Inner Report

**Date**: 2026-06-09T00:00:00Z
**Pipeline**: B
**Commit**: 824691b

## Summary
Fixed ISS-034: when an examiner opened an active (published) exam in the wizard at `/admin/exams/:id/edit`, all Step 1 Basic Settings form fields were fully editable — no `disabled` attribute was applied and no explanatory message was shown. The fix derives an `isReadOnly` flag from `exam?.status === 'active'` inside `Step1BasicSettings`, propagates it as `disabled` to every form control (all `Input`s, `textarea`, both `Select`s, two `DateTimePicker`s, three `Switch` toggles, and the submit `Button`), and renders an amber informational banner using i18n key `exam.wizard.activeReadOnlyNotice` when the exam is active. The local `Switch` sub-component was extended to accept and honour the `disabled` prop. A pre-existing TypeScript blocker in `Step4Review.tsx` (missing archive confirm dialog causing two variables to be flagged as unused) was also resolved in the same commit. The `useArchiveExam` hook and archive button/dialog in `ExamsListPage` were included as they were co-developed in this UAT run and are required by `Step4Review`.

## Files Changed
| File | Action |
|------|--------|
| `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx` | modified — isReadOnly flag, disabled on all controls, amber banner |
| `frontend/src/pages/ExamWizard/Step4Review.tsx` | modified — restored missing archive confirm dialog (TS blocker fix) |
| `frontend/src/locales/en.json` | modified — added `exam.wizard.activeReadOnlyNotice` |
| `frontend/src/locales/kk.json` | modified — added `exam.wizard.activeReadOnlyNotice` |
| `frontend/src/locales/ru.json` | modified — added `exam.wizard.activeReadOnlyNotice` |
| `frontend/src/api/exams.ts` | modified — added `useArchiveExam` hook |
| `frontend/src/pages/admin/ExamsListPage.tsx` | modified — added archive button and confirm dialog |
| `frontend/src/pages/ExamWizard/Step1BasicSettings.test.tsx` | created — 7 regression tests |
| `docs/issue-reports/ISS-034-uat-defect.md` | modified — root cause, fix, and resolution filled in |

## Acceptance Criteria Verified
| AC | Verified By |
|----|-------------|
| AC#5: Active exam controls disabled + explanatory message | test: `Step1BasicSettings > disables all form inputs when exam status is active`, `Step1BasicSettings > shows the read-only banner when exam status is active` |
| AC#5: Draft exam remains fully editable | test: `Step1BasicSettings > renders all inputs as enabled for a draft exam` |
| AC#5: Create mode (null exam) remains fully editable | test: `Step1BasicSettings > renders inputs as enabled when no exam is passed (create mode)` |
| Switch toggles disabled | test: `Step1BasicSettings > disables all switch toggles when exam status is active` |
| Submit button disabled | test: `Step1BasicSettings > disables the submit button when exam status is active` |
| No banner for draft | test: `Step1BasicSettings > does not show the read-only banner for a draft exam` |

## Test Results
- Backend: N/A (no backend changes)
- Frontend: 244 passed, 0 failed (44 test files)
- i18n check: 720 locale keys present in all 3 locales ✓

## Migration Applied
none

## Known Limitations
- The `isReadOnly` flag only targets Step 1. Steps 2–4 do not restrict editing for active exams at the UI level (the backend enforces this). If full wizard lock-down is desired, Steps 2 and 3 would need similar treatment — deferred to a separate issue.
- The amber banner does not include a direct "Unpublish" action button; the user must navigate to Step 4 to unpublish. This is intentional per AC#5 wording but could be enhanced.
