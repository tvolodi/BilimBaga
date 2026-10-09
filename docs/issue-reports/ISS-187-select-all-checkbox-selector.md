# ISS-187 - select-all e2e selector hit the "Show previous versions" checkbox

Root cause: `question-management.spec.ts` used `input[type="checkbox"].first()` for select-all. Since PR #143 the
first checkbox on /admin/questions is the include_versions filter, so the test never exercised select-all.
The three `.nth(1)` tests only worked by accident (nth(1) was the header select-all, not a row).

Fix (test only): select-all is `thead` + role checkbox, name /select all|выбрать все|барлығын таңдау/i
(the component's aria-label is hardcoded "Select all"; regexp covers locale variants). Row tests now use
`getByRole('checkbox', { name: /^select question /i }).first()` (aria-label "Select question <id>").

Other specs: ai-grading.spec.ts `.first()` is on the question editor page (no filter checkbox); exam-taking
already scoped to `main`. No change needed.

Verification: `playwright test --list` parses (216 tests). E2E not executed (no stack). Separate Code Reviewer
skipped: one-file selector change.
