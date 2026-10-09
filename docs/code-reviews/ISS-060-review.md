# Code Review — ISS-060 (frontend tests: grading, results, employee record, Step4Review)

Result: PASS

Scope: 7 new vitest files (+ ISS-060 report, README row, handoff). No product code changed, so the Go and
React production checklists (hardcoded strings, fetch, apiFetch, i18n) are not applicable.

Verification: `npx vitest run --maxWorkers=2` on the 7 files plus neighbouring results tests: 10 files, 89 tests, all passed (about 13s).

## Quality checks
- Behaviour-level assertions (rendered text, button state, call args and order, navigation). No snapshots, no .skip/.only.
- No fake timers or setTimeout. Only three waitFor/findBy uses, all awaiting real async state (navigation, mutation, download call). No flake risk seen.
- Real i18n is loaded and API hooks are mocked at module level, which matches the project rule of no raw fetch.
- GradingDetailPage tests assert sequential submission order via `toHaveBeenNthCalledWith`. They also assert that the second POST is not fired after a failure and that Submit re-enables. This matches the FR-BB47 integration strategy.

## Findings
- [Medium] GradingDetailPage.test.tsx — AC-9: the success toast and the `['grading-queue']` invalidation are not asserted. Only the navigation is. Add a toast assertion, or note that the invalidation lives in the hook.
- [Medium] FR-BB47 AC-6 (out-of-range input shows an inline error and disables Submit): not covered by the new tests. The ISS report and GradingComponents.test.tsx header claim "AC-6", but `SubmitAllGradesButton` only receives a score map. `src/test/QuestionGrader.test.tsx` covers only AI display (FR-BB73). Add a QuestionGrader test for typing 101 or -1.
- [Medium] FR-BB47 AC-1 / Role-Guard tests (employee gets a 403 redirect; examiner, department_admin and super_admin render): not covered. They would need a route-level `RequireRole` test, and the ISS report does not claim them. Track as follow-up or confirm they exist elsewhere.
- [Low] GradingDetailPage.test.tsx: loading and error assertions use loose regexes (`/failed to load|error/i`). Prefer exact i18n strings.
- [Low] The ISS report states 412 tests and 59.28% coverage. These were not re-verified because the full suite was not run. The 7 files themselves pass.

## AC Coverage (test-relative)
FR-BB47: AC-2 ✓, AC-3 ✓, AC-4 ✓, AC-5 ✓, AC-7 ✓, AC-8 ✓, AC-11 ✓, AC-9 partial (navigation only), AC-6 ✗ (not covered), AC-1 ✗ (not covered).
FR-BB58: AC-2, 3, 4, 6, 7, 8, 11, 12 ✓ (per test titles). AC-1, 9 and 10 are not test-targeted here (non-blocking).

Summary: Meaningful, deterministic tests, all green; no Critical or High findings. Three Medium coverage gaps (FR-BB47 AC-1, AC-6, AC-9 toast) are recommended as follow-up and do not block.

## Addendum (author)
- AC-6 gap addressed: added QuestionGrader score-input tests (slider/field sync, out-of-range error, boundaries) to `frontend/src/test/QuestionGrader.test.tsx`.
- AC-9 toast: the toast cannot be asserted because it is lost on navigation (product gap, filed as a GitHub issue; recorded in ISS-060 follow-ups).
- AC-1: covered by existing routeRoles / RequireRole tests.
- Full suite verified by author: 416 passed; line coverage 51.96% -> 59.28%.
