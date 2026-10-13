# FR-BB68 — Email Deep Links: Assignment and Pass Mails Must Open the SPA

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB68 |
| Phase | 6 — Polish & Hardening (gap closure; completes roadmap 6.1 "new assignment" and "exam passed with certificate link" emails) |
| Priority | 1 |
| Status | Validated |
| Depends On | FR-BB61, FR-BB33, FR-BB313, FR-BB45, FR-BB110, FR-BB115, FR-BB322, FR-BB510 |

## Description
Verified against the code on 2026-10-13. Result mails (`exam_passed`, `exam_failed`, `exam_submitted`, `exam_result_corrected`) already link to the SPA result route through `sessionResultLink` (`PublicAppURL`, FR-BB322 AC-10) and `password_reset_link` uses `PublicAppURL` (FR-BB115). Three gaps remain in `backend/internal/email`:
- `new_exam_assigned`: `service.go` (assignment branch) builds `PortalLink` as `APIBaseURL + "/portal/exams/<id>"`. The API serves no HTML at that path, so a click returns 404.
- `overdue_reminder` (FR-BB510, `SendOverdueReminder`): uses `PublicAppURL` but falls back to `APIBaseURL` when it is empty, and links to `/portal/exams/<id>`.
- `deadline_reminder` (`scheduler.go`): the template data has no `PortalLink` and the template text carries no link, so the reminder cannot be acted on from the mail.

The SPA has no route `/portal/exams/:examId` (only `/portal`, `/portal/results`, `/portal/profile`, `/portal/exams/:examId/result`, `/portal/sessions/:sessionId`, `/portal/sessions/:sessionId/result`), so even with the right host that path matches nothing. Exam cards are listed on `/portal`; that is the link target for assignment and reminder mails. All employee-facing mail links must open the SPA, where the employee signs in and performs the authenticated action.

## Acceptance Criteria
- [ ] AC-1: Every link in an outgoing email is built from `Config.PublicAppURL` (trailing slash trimmed) via one helper in `internal/email` (the existing `sessionResultLink` and the reset link are moved onto it); `APIBaseURL` is no longer used for any link, including the fallback in `SendOverdueReminder`. When `PublicAppURL` is empty the send is refused: no SMTP call, an `email_log` row with status failed, and an error returned to the caller.
- [ ] AC-2: `new_exam_assigned`, `deadline_reminder` and `overdue_reminder` link to the SPA route `/portal` (the employee portal listing the exam cards, FR-BB313), not to `/portal/exams/{examId}`, which is not an SPA route. `deadline_reminder` gains a `PortalLink` in its template data and a link line in the text and HTML parts of `en`, `ru`, `kk` (strings stay in the locale files).
- [ ] AC-3: `exam_passed` keeps linking to the SPA result route `/portal/sessions/{sessionId}/result` (FR-BB45, FR-BB322 AC-10), where the certificate download is offered under the employee's own session; a regression test pins it. No email link points at an API path (`/api/...`) or at an endpoint requiring a Bearer header.
- [ ] AC-4: `exam_failed`, `exam_submitted` and `exam_result_corrected` keep linking to the same result route (`ResultLink`); a regression test pins it.
- [ ] AC-5: A user who is not signed in and opens a mail link is sent to the login page and, after successful login, returned to the original target path. Verified in code: `RequireAuth`/`RequireRole` redirect to `/login` with `state.from = pathname + search` and `LoginPage` navigates to it after sign-in (FR-BB110 AC-11; only in-app paths starting with a single `/` are accepted). This FR adds tests for `/portal` and `/portal/sessions/:id/result`; any gap they reveal is fixed here.
- [ ] AC-6: A link-target test per template (`new_exam_assigned`, `deadline_reminder`, `exam_passed`, `exam_failed`, `exam_submitted`, `exam_result_corrected`, `overdue_reminder`, `password_reset_link`; extends `result_links_test.go`) asserts, for en/ru/kk, that every URL in the rendered text and HTML parts starts with the configured `PublicAppURL`, contains no `/api/`, and matches a route registered in the SPA router (route table in a shared fixture).
- [ ] AC-7: Tests: `service_test.go` for the link helper (trailing slash, empty base); template tests per AC-6; frontend Vitest for redirect-after-login on both routes (`RequireRole` + `LoginPage`); live E2E: assignment mail in Mailhog, extract the link, open it signed out, log in, land on the exam.

## Technical Specification
- Backend: `internal/email/links.go` with `func (s *EmailService) appLink(path string) (string, error)`; call sites in `service.go` (assignment, `sessionResultLink`, reset link, `SendOverdueReminder`) and `scheduler.go` (`deadline_reminder`) use it. No new config keys (`PUBLIC_APP_URL` exists). No migration.
- Frontend: only touched if AC-5 reveals a gap (`from` state in `RequireAuth`/`RequireRole`/`LoginPage`).
- Docs: add AC reference and Status note to FR-BB61 and its README row.
