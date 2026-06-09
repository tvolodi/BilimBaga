# uat-exam-assignment-20260609: Implementation Inner Report

**Date**: 2026-06-09T17:12:00Z
**Pipeline**: UAT (Pipeline B — bug fixes applied during run)
**Commit**: b0fa22d

## Summary

The exam-assignment business process (FR-BB33, FR-BB34, FR-BB61) was UAT-verified against
a live `make dev` stack. Initial UAT run failed 8 of 18 steps due to ISS-035: the ISS-034
read-only fix had disabled the Next button on Step 1 of the Exam Wizard for active exams,
inadvertently blocking all navigation to the Assignments tab (Step 3). A second defect
(ISS-036) was discovered during the re-run: `examsFetch` always called `res.json()` even
on `204 No Content` responses, crashing when the DELETE assignment endpoint returned 204
with no body. Both defects were resolved; re-run confirms all 18 UAT steps pass.

## Files Changed

| File | Action |
|------|--------|
| `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx` | modified — short-circuit handleSubmit for active (read-only) exams; remove isReadOnly from button disabled prop |
| `frontend/src/pages/ExamWizard/index.tsx` | modified — read `?step` search param; initialise wizard step from URL |
| `frontend/src/pages/admin/ExamsListPage.tsx` | modified — add "Assign" button for active exams linking to `?step=3` |
| `frontend/src/api/exams.ts` | modified — skip JSON parsing for 204 No Content responses |
| `frontend/src/locales/en.json` | modified — add `exam_list.manage_assignments` key |
| `frontend/src/locales/kk.json` | modified — add `exam_list.manage_assignments` key |
| `frontend/src/locales/ru.json` | modified — add `exam_list.manage_assignments` key |
| `frontend/src/pages/ExamWizard/Step1BasicSettings.test.tsx` | modified — update disabled-button test; add navigation-without-save test |
| `frontend/src/pages/admin/ExamsListPage.test.tsx` | created — Assign button present/absent; href includes `?step=3` |
| `docs/requirements/exam-assignment-process.md` | created — status: uat-verified |
| `docs/uat-scenarios/exam-assignment-20260609.md` | created — UAT scenario script |
| `docs/uat-reports/uat-exam-assignment-20260609.md` | created — initial (partial) UAT report |
| `docs/uat-reports/uat-exam-assignment-20260609-rerun.md` | created — re-run report (all pass) |
| `docs/issue-reports/ISS-035-uat-defect.md` | created — defect report, resolved |
| `.github/agents/00-orchestrator.agent.md` | modified — inline stack-check before UAT pipeline |
| `.github/agents/uat-runner.agent.md` | modified — pipeline configuration update |
| `CLAUDE.md` | modified — orchestrator pipeline notes update |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| FR-BB33 AC-2: Assignment with deadline shows countdown | UAT S2 Step 3 |
| FR-BB34 AC-4: Completion status visible per assignee | UAT S4 Steps 1–3 |
| FR-BB61 AC-5: Removing assignment removes card from portal | UAT S3 Steps 1–4 |
| ISS-035: Assign button on ExamsListPage for active exams | UAT S1 Step 3; ExamsListPage.test.tsx |
| ISS-036: 204 No Content handled correctly in examsFetch | UAT S3 Step 2; api/exams.ts |

## Test Results

- Backend: 25 packages, 0 failed (all cached, go test ./...)
- Frontend: 248 tests passed, 0 failed (44 test files, vitest run + check:i18n)

## Migration Applied

none

## Known Limitations

- The Assignments tab (Step 3) is accessible via the `?step=3` deep-link for active exams,
  but the StepIndicator dots are still non-clickable `<div>` elements; direct step-jumping
  from the indicator UI is not implemented (deferred, not a blocker for this process).
- Bulk assignment (assign exam to entire department in one operation) is not covered by
  FR-BB33 and was not tested in this UAT run.
