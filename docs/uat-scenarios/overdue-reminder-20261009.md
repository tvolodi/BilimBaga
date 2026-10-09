---
slug: overdue-reminder
title: "Overdue Employee Reminder (Send Reminder Action) — UAT Scenario"
feature: overdue-reminder (FR-BB510; GitHub issue #61, parent roadmap 5.6)
version: 1
created: 2026-10-09
author: Business Analyst
---

## Code that must be merged before running

- **Issue #61** (FR-BB510 implementation): `exam_id` on `overdue_employees[]`, real `POST /api/v1/admin/users/{userId}/remind`, `overdue_reminder` email template (en/ru/kk), `exam_reminders` migration (next number after 030), audit `users.remind`, frontend `OverdueTable` changes. Status at authoring time: not merged.
- Rebuild api and frontend images and apply migrations (`make migrate`) before the run.
- Expected pre-fix baseline (current main): `RemindEmployee` (`backend/internal/users/handler.go:308-312`) returns `200 {data:null}` unconditionally and sends nothing; S1 step 2 and every later step FAIL. Record that as "feature not merged" (ENV ISSUE), not as a defect.
- Mail sink: the `api` service in `docker-compose.yml` sets `SMTP_HOST: mailhog` and `SMTP_PORT: "1025"` (overrides `.env`, no credentials; lines 30-33). Service `mailhog` (image `mailhog/mailhog:v1.0.1`) publishes SMTP on host `${HOST_MAILHOG_SMTP_PORT:-1025}` and the UI/API on host `${HOST_MAILHOG_UI_PORT:-8025}`. Mailhog UI: `http://localhost:8025`; messages API: `http://localhost:8025/api/v2/messages`; clear all: `DELETE http://localhost:8025/api/v1/messages`. `SMTP_USER`, `SMTP_PASS` must be empty and `SMTP_TLS=false`; `SMTP_FROM` comes from `.env` (`noreply@example.com` in `.env.example`).

## Spec reference

`docs/requirements/FR-BB510.Overdue-reminder-action.md` AC-1..AC-9. Endpoint `POST /api/v1/admin/users/{userId}/remind`, body `{ "exam_id": "<uuid>" }`, permission `reports:read` (super_admin, department_admin, examiner). Dashboard `GET /api/v1/admin/dashboard`.

## Pacing note (rate limits)

- `ratelimit.AuthLimiter()` (10 requests/min per IP) covers only `/health`, `/auth/login`, `/auth/refresh`, `/auth/logout` (`backend/internal/router/router.go:53-59`). Log in each account ONCE and reuse tokens; never log in inside a loop. If a login returns 429 `RATE_LIMITED`, wait 60 s (`Retry-After`). Note `/health` shares this limiter, so do not poll health more than a few times per minute.
- The remind endpoint is under the global limiter (300/min), not the auth limiter. The 24-hour per (user, exam) limit is a business rule (AC-5) and cannot be reset through the UI: use a fresh (user, exam) pair for each success step, or rewind `exam_reminders.sent_at` via DB (S5 step 6).

## Preconditions and accounts

| Account | Role | Locale | Password | Source |
|---------|------|--------|----------|--------|
| `admin@bilimbaga.local` | super_admin | n/a | `Admin1234!` (or current) | seeded |
| `uat.examiner@test.com` | examiner | n/a | `NewPass123!` | create per `route-guards-20261009.md` S0 |
| `uat.deptadmin@test.com` | department_admin | n/a | `NewPass123!` | same |
| `uat.employee@test.com` | employee | n/a | `NewPass123!` | earlier UAT |
| `uat.rem.en@test.com` | employee | `en` | `NewPass123!` | create now, `preferred_locale` = en |
| `uat.rem.ru@test.com` | employee | `ru` | `NewPass123!` | create now, `preferred_locale` = ru |
| `uat.rem.kk@test.com` | employee | `kk` | `NewPass123!` | create now, `preferred_locale` = kk |
| `uat.rem.off@test.com` | employee | `en` | `NewPass123!` | create now, then deactivate |

Create users through `/admin/users` (see `user-onboarding-20260609.md`) and set locale via admin edit or the profile page (`profile-and-locale-20261009.md`). Confirm exact field names from the API first and record them.

Test exams and assignments (deadline in the past so they are overdue; if the UI blocks past deadlines, create via API or `psql`; record the method used):

| Exam | Assigned to | Deadline | Passing session? | Purpose |
|------|-------------|----------|------------------|---------|
| `UAT-Rem-A` (active) | `uat.rem.en`, `uat.rem.ru`, `uat.rem.kk` | yesterday | none | success, locales |
| `UAT-Rem-B` (active) | `uat.rem.en` | yesterday | none | second exam for same user (limit is per exam) |
| `UAT-Rem-Future` (active) | `uat.rem.en` | next week | none | probe whether a deadline is required (S3 step 4) |
| `UAT-Rem-Passed` (active) | `uat.employee` | yesterday | passed | NOT_OVERDUE (already passed) |
| `UAT-Rem-Unassigned` (active) | nobody | n/a | n/a | NOT_OVERDUE (not assigned) |
| `UAT-Rem-Inactive` (active) | `uat.rem.off` | yesterday | none | USER_INACTIVE |

Before starting: `DELETE http://localhost:8025/api/v1/messages` (empty Mailhog); save the baseline count of audit entries with action `users.remind` (audit UI/API or `psql`); save the baseline `GET /admin/dashboard` JSON.

Variables: `{admin_token}`, `{examiner_token}`, `{employee_token}`, `{uid_en}`, `{uid_ru}`, `{uid_kk}`, `{uid_off}`, `{exam_A}`, `{exam_B}`, `{exam_pass}`, `{exam_unassigned}`, `{exam_inactive}`.

## Scenario S1: Dashboard exposes exam_id (AC-1)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | `GET /api/v1/admin/dashboard` | 200; `overdue_employees` contains rows for `uat.rem.en/ru/kk` on `UAT-Rem-A` | |
| 2 | Admin | Inspect a row | Has `exam_id` (UUID equal to `{exam_A}`); existing fields unchanged; never empty | |
| 3 | Admin | Count rows and ordering vs baseline | At most 20 rows; ordering as before | |
| 4 | Admin | Open `/admin/dashboard` overdue table | Each row shows the Send Reminder button | |

## Scenario S2: Validation and not-found (AC-2)

All with `{admin_token}` against `POST /api/v1/admin/users/{uid_en}/remind` unless stated. After each step the Mailhog message count must be unchanged.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Body `{}` | 400 `VALIDATION_ERROR` | |
| 2 | Admin | Body `{"exam_id":""}` | 400 `VALIDATION_ERROR` | |
| 3 | Admin | Body `{"exam_id":"not-a-uuid"}` | 400 `VALIDATION_ERROR` | |
| 4 | Admin | Invalid JSON body | 400, not 500 | |
| 5 | Admin | Valid body, userId = random UUID | 404 `NOT_FOUND` | |
| 6 | Admin | userId = `not-a-uuid` | 4xx (400 or 404), not 500; record | |
| 7 | Admin | exam_id = random UUID | 404 `NOT_FOUND` | |
| 8 | Anonymous | No `Authorization` header | 401 | |

## Scenario S3: Not a reminder target (AC-3)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Remind `uat.employee` for `{exam_pass}` (already passed) | 409 `NOT_OVERDUE` | |
| 2 | Admin | Remind `{uid_en}` for `{exam_unassigned}` | 409 `NOT_OVERDUE` | |
| 3 | Admin | Remind `{uid_off}` for `{exam_inactive}` | 409 `USER_INACTIVE` | |
| 4 | Admin | Remind `{uid_en}` for `UAT-Rem-Future` | Record outcome: 200 if an assignment alone qualifies, 409 `NOT_OVERDUE` if a past deadline is required. AC-3 text says "assignment resolving to that user" with the same resolution as `GetOverdueEmployees` (which requires `deadline < NOW()`); a send for a not-yet-due exam is a finding to raise to the BA | |
| 5 | Tester | Mailhog `GET /api/v2/messages` | `total` = 0 (excluding step 4 if it returned 200) | |
| 6 | Tester | Audit entries `users.remind` | Unchanged from baseline (rejected requests write nothing) | |
| 7 | Tester | `exam_reminders` row count (psql) | Unchanged | |

## Scenario S4: Successful send, localized email in Mailhog (AC-4, AC-6, AC-7)

One send per locale on a fresh pair. Delivery is synchronous, so the message must already be in Mailhog when the 200 arrives (allow 5 s).

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Remind `{uid_en}` / `{exam_A}` | 200 `{ data: { sent_at: "<UTC ISO 8601>" }, error: null }` | |
| 2 | Tester | `GET http://localhost:8025/api/v2/messages` | `total` = 1; recipient is `uat.rem.en@test.com`; sender = configured `SMTP_FROM` | |
| 3 | Tester | Inspect subject and body (UI `http://localhost:8025`, Plain text and HTML tabs) | English text; names `UAT-Rem-A`; deadline in UTC, human formatted and correct; link to the exam on the portal (base from config); no `{{` placeholders | |
| 4 | Tester | Check MIME structure | multipart/alternative with text/plain and text/html parts | |
| 5 | Admin | Remind `{uid_ru}` / `{exam_A}` | 200; new message to `uat.rem.ru@test.com`; Russian subject and body (Cyrillic correct, no mojibake, subject encoded) | |
| 6 | Admin | Remind `{uid_kk}` / `{exam_A}` | 200; message to `uat.rem.kk@test.com` in Kazakh (letters such as ә ғ қ ң ө ұ ү һ і render) | |
| 7 | Admin | Optional: clear `uat.rem.en` locale and remind `{uid_en}` / `{exam_B}` | 200; email in tenant default locale (note which) | |
| 8 | Admin | Response bodies of steps 1, 5, 6 | None contains the recipient's email address (AC-7) | |
| 9 | Tester | `docker compose logs api` for the requests | No recipient email in request logs or errors | |
| 10 | Tester | `email_log` rows (confirm column names first) | One row per attempt, template `overdue_reminder`, success status | |
| 11 | Tester | `exam_reminders` rows | One row per success with correct `user_id`, `exam_id`, `sent_by` = admin id | |
| 12 | Tester | Audit log (`/admin/audit` as super_admin or API filter action=`users.remind`) | New entries equal to the number of successes; `entity_type=user`, `entity_id` = target user id, actor = admin; metadata has `exam_id` and `exam_title`; no email body or address | |

## Scenario S5: 24-hour rate limit (AC-5)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Repeat S4 step 1 request immediately | 429 `REMINDER_RATE_LIMITED`; `error.details.retry_after_seconds` is a positive integer at most 86400, close to 86400 minus elapsed | |
| 2 | Tester | Mailhog total | Unchanged | |
| 3 | Tester | Audit entries and `exam_reminders` rows | Unchanged by the 429 | |
| 4 | Admin | Same user, other exam: remind `{uid_en}` / `{exam_B}` | 200 (limit is per user+exam); email delivered | |
| 5 | Examiner | Remind `{uid_ru}` / `{exam_A}` with `{examiner_token}` | 429 (limit is per pair, not per sender) | |
| 6 | Tester | psql: `UPDATE exam_reminders SET sent_at = now() - interval '25 hours' WHERE user_id = '{uid_en}' AND exam_id = '{exam_A}'` | Done | |
| 7 | Admin | Remind `{uid_en}` / `{exam_A}` again | 200; email delivered; new `exam_reminders` row | |

## Scenario S6: SMTP failure honesty (AC-4, AC-5)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | `docker compose stop mailhog` | Mail sink down | |
| 2 | Admin | Remind a fresh pair (create `UAT-Rem-C` assigned to `uat.rem.kk`, overdue) | 502 `EMAIL_SEND_FAILED` | |
| 3 | Tester | `exam_reminders` | No row for this pair | |
| 4 | Tester | `email_log` | An attempt row with failure status | |
| 5 | Tester | Audit | No `users.remind` entry for the failure | |
| 6 | Admin (UI) | Click Send Reminder on that row | Error toast `dashboard.reminder_error`; no success toast; button re-enabled | |
| 7 | Tester | `docker compose start mailhog`; retry the request | 200 (failed attempt did not consume the window); message appears in Mailhog | |

## Scenario S7: RBAC (AC-7)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | POST remind with `{employee_token}` for a valid pair | 403 | |
| 2 | Examiner | POST remind on a fresh pair | 200 | |
| 3 | Dept admin | Same | 200 (or 403 if department scope excludes the user; record and compare with dashboard visibility) | |
| 4 | Tester | Mailhog after step 1 | No message for the 403 request | |
| 5 | Tester | Audit entry for step 2 | Actor = examiner id | |

## Scenario S8: Frontend button states and messages (AC-8)

As admin on `/admin/dashboard` with fresh overdue rows (create `UAT-Rem-D` for two employees so two rows exist). Use ru and kk as well as en.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Open dashboard, inspect overdue table | Rows list employee, exam, deadline; Send Reminder enabled | |
| 2 | Admin | Click Send Reminder on row 1 (throttle network in dev tools) | Button disabled while pending; only that row affected | |
| 3 | Admin | Result | Toast from `dashboard.reminder_sent`; row button stays disabled with a "Reminder sent" label | |
| 4 | Admin | Reload | Button enabled again (state is session-only per AC-8); record actual behaviour | |
| 5 | Admin | Click row 1 again after reload | Toast `dashboard.reminder_rate_limited`; no email in Mailhog | |
| 6 | Admin | Make a row non-overdue after page load (pass the exam via DB or remove assignment) and click | Toast `dashboard.reminder_not_overdue` | |
| 7 | Admin | Trigger 502 (S6) from the UI | Toast `dashboard.reminder_error` | |
| 8 | Admin | Trigger network error (stop api briefly) | Toast `dashboard.reminder_error` | |
| 9 | Admin | Repeat toasts in ru and kk | Translated text for all messages and the "Reminder sent" label; no raw keys such as `dashboard.reminder_sent`; `reminder_already_sent` shown where used | |
| 10 | Admin | Row 2 while row 1 is pending or sent | Row 2 independent (keyed by user_id + exam_id) | |
| 11 | Admin | Same employee overdue on two exams (rows for `UAT-Rem-A` and `UAT-Rem-B`) | Reminding one row does not disable the other | |
| 12 | Admin | Network tab | POST body carries the row's real `exam_id`, never empty | |

## Scenario S9: Regression

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Load dashboard after all steps | 200; other widgets unaffected | |
| 2 | Tester | Mailhog final count | Equals the number of successes in S4, S5 (4, 7), S6 (7), S7 (2, 3), S8 | |
| 3 | Tester | FR-BB61 daily scheduler | Not triggered by this action; no duplicate emails (record only if observable) | |

## Cleanup

Empty Mailhog; delete `exam_reminders` rows for `UAT-Rem-*`; delete `UAT-Rem-*` exams and `uat.rem.*` users; ensure mailhog is running (`docker compose start mailhog`).

## Pass / Fail criteria

- PASS: S1-S8 meet expected outcomes; each success yields exactly one Mailhog message in the recipient's locale; 400/404/409/429/403/502 paths send nothing and write no audit or reminder row (502 writes only email_log); UI messages match the codes.
- FAIL (defect): 200 without an email; wrong locale or unresolved placeholders; email to a non-target; 429 not enforced or `retry_after_seconds` missing; audit missing or containing the body; recipient address leaked to the caller; success toast on failure; 500 on bad input; empty `exam_id` in the UI request.
- ENV ISSUE: #61 not merged or migration not applied; Mailhog not running; `SMTP_HOST` not `mailhog`; past-deadline assignments cannot be created.

## Acceptance Criteria Coverage

| FR | AC# | Covered by |
|----|-----|------------|
| FR-BB510 | AC-1 exam_id on dashboard | S1, S8 step 12 |
| FR-BB510 | AC-2 validation and 404 | S2 |
| FR-BB510 | AC-3 target rules, 409 | S3 |
| FR-BB510 | AC-4 localized email, 502, email_log | S4, S6 |
| FR-BB510 | AC-5 24h limit, 429 | S5, S6 step 7 |
| FR-BB510 | AC-6 audit users.remind | S3 step 6, S4 step 12, S6 step 5, S7 step 5 |
| FR-BB510 | AC-7 RBAC 403, no address leak | S7, S4 steps 8-9 |
| FR-BB510 | AC-8 frontend states and i18n | S8 |
| FR-BB510 | AC-9 tests | Not covered here (unit, Vitest, live E2E are Test Runner scope) |

## Out of Scope

Bulk reminders, manager escalation, reminder history UI, upcoming-deadline reminders (FR-BB61 scheduler), SMS/push, rendering across mail clients.
