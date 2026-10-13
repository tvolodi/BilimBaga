# FR-BB68 — Email Deep Links: Assignment and Pass Mails Must Open the SPA

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB68 |
| Phase | 6 — Polish & Hardening (gap closure; completes roadmap 6.1 "new assignment" and "exam passed with certificate link" emails) |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB61, FR-BB33, FR-BB43, FR-BB313, FR-BB45, FR-BB510 |

## Description
Two FR-BB61 templates build their links from `Config.APIBaseURL` instead of the SPA base URL:
- `exam_passed`: `APIBaseURL + "/portal/sessions/<id>/certificate"` (`backend/internal/email/service.go`, pass-mail branch).
- `new_exam_assigned` (and the deadline reminder that shares its link builder): `APIBaseURL + "/portal/exams/<id>"`.

The API serves no HTML at those paths, and the certificate route sits behind the Bearer-authenticated API group, so a click returns 404 or 401. The overdue reminder (FR-BB510) already does this correctly with `PublicAppURL`. All employee-facing mail links must open the SPA, where the employee signs in and performs the authenticated action.

## Acceptance Criteria
- [ ] AC-1: Every link in an outgoing email is built from `Config.PublicAppURL` (trailing slash trimmed) via one helper in `internal/email`; `APIBaseURL` is no longer used for any link. When `PublicAppURL` is empty the send is refused with a logged `email_log` error rather than falling back to the API host.
- [ ] AC-2: `new_exam_assigned` and `deadline_reminder` link to the SPA route `/portal/exams/{examId}` (the employee portal exam card/detail, FR-BB313).
- [ ] AC-3: `exam_passed` links to the SPA result route `/portal/sessions/{sessionId}/result` (FR-BB45), where the certificate download is offered under the employee's own session. No email link points at an API path (`/api/...`) or at an endpoint requiring a Bearer header.
- [ ] AC-4: `exam_failed` links to the same result route when the template currently carries a link; if it carries none, it stays link-free.
- [ ] AC-5: A user who is not signed in and opens a mail link is sent to the login page and, after successful login, returned to the original target path (existing redirect-after-login behaviour is verified for `/portal/exams/:id` and `/portal/sessions/:id/result`; gaps fixed in this FR).
- [ ] AC-6: A link-target test per template (`new_exam_assigned`, `deadline_reminder`, `exam_passed`, `exam_failed`, `overdue_reminder`, `password_reset_link`) asserts, for en/ru/kk, that every URL in the rendered text and HTML parts starts with the configured `PublicAppURL`, contains no `/api/`, and matches a route registered in the SPA router (route table in a shared fixture).
- [ ] AC-7: Tests: `service_test.go` for the link helper (trailing slash, empty base); template tests per AC-6; frontend Vitest for redirect-after-login on both routes; live E2E: assignment mail in Mailhog, extract the link, open it signed out, log in, land on the exam.

## Technical Specification
- Backend: `internal/email/links.go` with `func (s *Service) appLink(path string) (string, error)`; call sites in `service.go` for pass, assignment and reminder mails replaced. No new config keys (`PUBLIC_APP_URL` exists). No migration.
- Frontend: only touched if AC-5 reveals a gap (login `redirect` state in `RequireAuth`).
- Docs: add AC reference and Status note to FR-BB61 and its README row.
