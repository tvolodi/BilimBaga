# FR-BB67 — E2E Coverage: Employee Portal, Exam Taking, Result & Grading Flows

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB67 |
| Phase | 6 — Polish & Hardening |
| Priority | 1 |
| Status | implemented |
| Depends On | FR-BB313, FR-BB314, FR-BB41, FR-BB42, FR-BB45, FR-BB46 |

## Description

The BilimBaga E2E test suite currently covers 22 admin-side walkthrough tests but has zero coverage of the employee portal, exam-taking session (all 5 question input types), exam result screens, and the admin manual-grading workflow. These are the highest-risk user-facing flows: a broken exam-taking screen or grading page would be invisible to the current test suite. This requirement adds 36 new Playwright tests across 4 new spec files, a shared test-data seeding fixture, a second auth storage state for the employee role, and 5 targeted additions to the existing `full-walkthrough.spec.ts`. After implementation, element-interaction E2E coverage rises from ~27 % to ~85 %.

## Acceptance Criteria

- [ ] AC-1: A `frontend/e2e/fixtures/seed.ts` module exports `seedEmployeeFixtures()` which, when called from `global-setup.ts`, creates (idempotently): one employee user (`employee@bilimbaga.local / Employee1234!`), one exam containing exactly one question of each of the 5 types (single_choice, multiple_choice, true_false, likert, short_text) assigned to that employee, one short-text-only exam for the pending-grading result test, and saves the employee JWT to `.auth/employee.json` in the same format as `.auth/admin.json`.
- [ ] AC-2: `playwright.live.config.ts` is updated so every spec in `e2e/employee-portal.spec.ts`, `e2e/exam-taking.spec.ts`, and `e2e/exam-result.spec.ts` runs with the employee storage state; `e2e/admin-grading.spec.ts` continues to use the admin storage state.
- [ ] AC-3: `frontend/e2e/employee-portal.spec.ts` contains exactly 8 tests covering: portal load with exam cards, empty-portal state, Start Exam modal open/cancel/confirm, Continue CTA, View Result CTA, and My Results page list + sort + pagination — all assertions are hard `expect(...).toBeVisible()` with no silent `if (isVisible)` guards on required elements.
- [ ] AC-4: `frontend/e2e/exam-taking.spec.ts` contains exactly 11 tests covering: top-bar and navigator render, each of the 5 question input types (single-choice radio, multiple-choice checkbox, true/false radio, Likert scale buttons, short-text textarea), flag/unflag with navigator colour change, question-navigator jump, finish-exam → review screen (unanswered + flagged lists + Go Back + Submit Anyway), submit-confirmation modal full happy path, and tab-switch warning modal.
- [ ] AC-5: `frontend/e2e/exam-result.spec.ts` contains exactly 5 tests covering: Passed result screen (✅ icon, green heading, score %), Failed result screen (❌ icon, red heading, score %), Pending-grading state (⏳ icon, no score), result detail page route (`/portal/sessions/:id/result`) with ScoreDial and breakdown table, and Back to Portal navigation.
- [ ] AC-6: `frontend/e2e/admin-grading.spec.ts` contains exactly 7 tests covering: empty grading queue message, session row render and navigation to detail, question display and Previous/Next navigation, score input validation (invalid value → error message, valid value → clears), feedback textarea, Submit All Grades happy path (loading state → success toast → redirect), and pagination boundary states.
- [ ] AC-7: `frontend/e2e/full-walkthrough.spec.ts` receives 5 targeted additions: (a) test 09 asserts Save Draft button visibility; (b) test 13 hard-asserts From/To date filter inputs and Export CSV button exist; (c) test 16 removes the `/reports` skip and asserts the "coming soon" page renders without crashing; (d) test 18 asserts Values Profile section heading or pagination controls exist; (e) test 20 asserts at least one `<h1>` or heading renders on the analytics page.
- [ ] AC-8: All 58 tests (22 existing + 36 new) pass when run via `npm run test:e2e:live` against a live `make dev` stack. Zero tests are skipped or conditionally guarded with `test.skip()`.
- [ ] AC-9: TypeScript compiles cleanly (`npx tsc --noEmit`) with no errors introduced by the new spec files or the seed fixture.
- [ ] AC-10: Screenshots are taken at every key interaction point in all new spec files, following the `shot(page, 'NN-description')` convention from the existing suite.

## Technical Specification

### Database Schema

No schema changes required. All data is created via the existing API at test-setup time using the admin JWT.

### API Contract

No new API endpoints required. The seed fixture calls existing endpoints:

```
POST /api/v1/users                        — create employee user (admin auth)
POST /api/v1/exams                        — create mixed-type exam (admin auth)
POST /api/v1/questions                    — create one question per type (admin auth)
POST /api/v1/exams/{id}/rules             — add question rules (admin auth)
POST /api/v1/exams/{id}/assignments       — assign exam to employee user (admin auth)
POST /api/v1/exams/{id}/publish           — publish exam (admin auth)
POST /api/v1/auth/login                   — login as employee, capture JWT
```

All responses follow the standard `{ data, error }` envelope. The seed fixture must be idempotent: if `employee@bilimbaga.local` already exists it skips creation and proceeds to login.

### Go Implementation Notes

None — this is a frontend-only requirement. No Go files are changed.

### Frontend Implementation Notes

#### File: `frontend/e2e/fixtures/seed.ts`

```typescript
// Exported function — called once from global-setup.ts after admin login
export async function seedEmployeeFixtures(adminToken: string): Promise<void>
```

- Uses `node-fetch` or the `chromium` browser context's `fetch` (via `page.evaluate`) to call the backend API with the admin token.
- Idempotency: attempt `POST /api/v1/auth/login` for `employee@bilimbaga.local` first; if it succeeds, skip creation. If it fails with `INVALID_CREDENTIALS`, create the user, then login.
- After login, saves employee JWT to `.auth/employee.json` using the same `context.storageState()` pattern as the admin setup.
- Saves the raw token to `.auth/employee-token.txt`.
- The mixed-type exam needs 5 questions created and added as rules, then published.
- The short-text-only exam needs 1 short_text question, assigned to the employee, published.

#### File: `frontend/e2e/global-setup.ts` (patch)

Add one line at the end of the existing `globalSetup()` function:

```typescript
import { seedEmployeeFixtures } from './fixtures/seed'
// ... existing admin setup ...
await seedEmployeeFixtures(accessToken)
```

#### File: `frontend/playwright.live.config.ts` (patch)

Add a second Playwright project entry so employee-facing specs use the employee storage state:

```typescript
projects: [
  {
    name: 'chromium-live-admin',
    testMatch: ['**/full-walkthrough.spec.ts', '**/admin-grading.spec.ts'],
    use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 }, storageState: ADMIN_STORAGE_STATE },
  },
  {
    name: 'chromium-live-employee',
    testMatch: ['**/employee-portal.spec.ts', '**/exam-taking.spec.ts', '**/exam-result.spec.ts'],
    use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 }, storageState: EMPLOYEE_STORAGE_STATE },
  },
]
```

Where `EMPLOYEE_STORAGE_STATE = path.join(__dirname, '.auth', 'employee.json')`.

#### File: `frontend/e2e/employee-portal.spec.ts`

8 tests. Uses a `loginAsEmployee(page)` helper mirroring `loginAsAdmin` but navigating to `/portal` and asserting `toHaveURL(/\/portal/)`.

| Test | Key assertions |
|------|---------------|
| 01 Portal loads — exam cards render | `getByRole('article')` or exam card container present; title, status badge, time limit, passing score, attempts, deadline all visible |
| 02 Empty portal state | `getByText(t('portal.empty.title'))` visible |
| 03 Start Exam modal opens | Click Start button → modal title visible, time limit line, attempts remaining, warning text |
| 04 Start Exam modal cancel | Click Cancel → modal gone, URL still `/portal` |
| 05 Start Exam modal confirm | Click Confirm → loading state on button → navigate to `/portal/sessions/` |
| 06 Continue CTA | Card with `in_progress` status → Continue button → navigates directly to session |
| 07 View Result CTA | Card with `passed`/`failed` status → View Result button → navigates to result URL |
| 08 My Results page | Navigate to `/portal/results` → table visible → sort buttons → pagination |

#### File: `frontend/e2e/exam-taking.spec.ts`

11 tests. Uses a `startExamSession(page): string` helper that creates a fresh session via the Start modal and returns the session URL.

| Test | Key assertions |
|------|---------------|
| 01 Top bar and navigator render | ExamTopBar title, progress `0 / N answered`, CountdownTimer (HH:MM:SS format), QuestionNavigator grid, legend items |
| 02 Single-choice answer | `role="radiogroup"` present, click option → radio `checked`, SaveIndicator shows "Saving…" then "Saved ✓" |
| 03 Multiple-choice answer | Checkboxes present, click two → both checked; click one again → unchecked |
| 04 True/False answer | 2-option radio group, click True → selected state |
| 05 Likert answer | `role="radiogroup"` with `role="radio"` buttons, click one → `aria-checked="true"` |
| 06 Short-text answer | `textarea` with placeholder "Type your answer here…" → type text → SaveIndicator debounce |
| 07 Flag/unflag | FlagButton `aria-pressed="false"` → click → `aria-pressed="true"` → navigator button turns yellow → click again → `aria-pressed="false"` |
| 08 Question navigator jump | Click question 3 button → question 3 stem displayed |
| 09 Finish exam → review screen | Click "Finish exam" → FinishReviewScreen title, unanswered count, flagged count, Go Back button, Submit Anyway button |
| 10 Submit confirmation happy path | Click Submit Anyway → SubmitConfirmModal title and message → Click Submit → loading → ResultScreen visible |
| 11 Tab switch warning modal | `document.dispatchEvent(new Event('visibilitychange'))` + `Object.defineProperty(document, 'visibilityState', {value:'hidden'})` → TabSwitchWarningModal title and message → click Close → modal gone |

#### File: `frontend/e2e/exam-result.spec.ts`

5 tests. Admin seeds sessions to specific result states before each test.

| Test | Key assertions |
|------|---------------|
| 01 Passed result | ✅ icon (or green colour), `t('exam.taking.result.passed')` heading, score `%` text, Back to Portal button |
| 02 Failed result | ❌ icon (or red colour), `t('exam.taking.result.failed')` heading, score `%` text, Back to Portal button |
| 03 Pending grading | ⏳ icon or pending message, no score visible, Back to Portal button |
| 04 Result detail page | `/portal/sessions/:id/result` → heading, score indicator, breakdown table rows |
| 05 Back to Portal navigation | Click "Back to my exams" → `toHaveURL(/\/portal/)` |

#### File: `frontend/e2e/admin-grading.spec.ts`

7 tests. Uses admin storage state. Requires the short-text session seeded by `seedEmployeeFixtures` to be in a submitted-but-ungraded state.

| Test | Key assertions |
|------|---------------|
| 01 Empty grading queue | Navigate to `/admin/grading` → `t('grading.queue_empty')` visible |
| 02 Session row and navigation | Row with employee name, exam title, submitted date, pending count → click row → URL matches `/admin/grading/` |
| 03 Question display and nav | Detail title, employee + exam subtitle, question counter "Question 1 of N", question stem, employee answer, Previous/Next buttons |
| 04 Score input validation | Enter `150` → `t('grading.score_error')` visible; change to `85` → error gone |
| 05 Feedback textarea | Textarea with placeholder "Add comments…" → type feedback → value persists |
| 06 Submit All Grades | Click Submit → loading state → success toast `t('grading.success_toast')` → redirected to `/admin/grading` |
| 07 Pagination | Previous button disabled at page 1; if >1 page, Next enabled; click Next → page 2 |

#### Shared helper additions in `frontend/e2e/support/helpers.ts` (new file)

```typescript
export async function loginAsEmployee(page: Page): Promise<void>
export async function shot(page: Page, name: string): Promise<void>  // re-exported from walkthrough
export async function waitForContent(page: Page): Promise<void>      // re-exported from walkthrough
```

Moving shared helpers to a support file avoids duplication across 4 spec files.

#### i18n keys referenced (English values for selector guidance)

| Key | English value |
|-----|--------------|
| `portal.empty.title` | "No exams assigned" (or similar) |
| `portal.modal.title` | Start exam modal heading |
| `portal.modal.confirm` | "Start" |
| `portal.modal.cancel` | "Cancel" |
| `exam.taking.finishButton` | "Finish exam" |
| `exam.taking.flagButton` | "Flag for review" |
| `exam.taking.unflagButton` | "Unflagged" |
| `exam.taking.shortTextPlaceholder` | "Type your answer here…" |
| `exam.taking.submit.confirm.title` | "Submit exam" |
| `exam.taking.tabswitch.title` | "Warning" |
| `exam.taking.result.passed` | "Passed" |
| `exam.taking.result.failed` | "Failed" |
| `exam.taking.result.pending` | "Your exam is being reviewed by an instructor." |
| `exam.taking.result.backToPortal` | "Back to my exams" |
| `grading.queue_empty` | "No sessions are awaiting manual grading." |
| `grading.score_error` | "Score must be between 0 and 100" |
| `grading.success_toast` | "All grades submitted. Session has been finalized." |

## Notes

- **Out of scope**: adaptive exam mode (FR-BB72 flow), certificate download button, clipboard copy on password reset modal, mobile-viewport responsive tests, export CSV download verification. These are deferred to a follow-up FR-BB82.
- **Constraint**: The seed fixture must be fully idempotent — running `npm run test:e2e:live` twice against the same database must not fail on duplicate-key errors.
- **Constraint**: Tab-switch detection in test 11 of `exam-taking.spec.ts` requires injecting a fake `visibilitychange` event because Playwright does not natively hide the document. The test should use `page.evaluate()` to dispatch the event and manipulate `document.visibilityState` before the assertion.
- **Constraint**: The short-text exam for pending-grading tests (AC-5 tests 03 and 04) requires that the employee submits the session during seeding (via `POST /api/v1/sessions/:id/submit`) so it lands in `submitted` status awaiting manual grading.
- **Deferred**: `frontend/e2e/full-walkthrough.spec.ts` patch for test 16 (`/reports` route) depends on FR-BB113 (Departments) and a reports stub being present. If neither exists, the test should assert a "coming soon" or 404-style message rather than a real page.
- The `e2e/support/helpers.ts` extraction is optional — implementations may keep helpers inline in each spec file if that is simpler to review.
