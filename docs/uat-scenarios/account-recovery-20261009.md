---
slug: account-recovery
title: "Account Recovery (Forgot/Reset Password, Admin Unlock) — UAT Scenario"
feature: account-recovery (FR-BB115; GitHub issue #33, parent #28)
version: 1
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **Issue #33** (FR-BB115 implementation: migration for `password_reset_tokens`, `POST /auth/forgot-password`, `POST /auth/reset-password`, `POST /users/{id}/unlock`, `is_locked` on `GET /users`, pages `/forgot-password` and `/reset-password`, login-page link, unlock action in the admin user list). Merged.
- **PR #144** (ISS-105, hardening: constant-time response floor, atomic throttle, Referrer-Policy, access-token revocation; migration 032 `users.password_changed_at`). Merged.
- **FR-BB48 / `PUBLIC_APP_URL`** (PR #36, already in git history as `ac9c9ba`): reset link base.
- Migrations apply automatically at API startup (`make migrate` is unreliable per README), so rebuilding/restarting the API is sufficient.
- `PUBLIC_APP_URL` must be set to the URL the tester can open; reset links are built from it. SMTP precondition Gap 1 below is unchanged by these PRs.

## Email observability (how to read the reset link)

- `docker-compose.yml` defines a `mailhog` service (`mailhog/mailhog:v1.0.1`): SMTP on host port `${HOST_MAILHOG_SMTP_PORT:-1025}`, Web UI and API on `${HOST_MAILHOG_UI_PORT:-8025}`.
- Read mail via `GET http://localhost:8025/api/v2/messages` (JSON; `items[].Content.Body`, `items[].To`). Delete all via `DELETE http://localhost:8025/api/v1/messages`.
- **PRECONDITION GAP 1:** root `.env.example` sets `SMTP_HOST=localhost`, `SMTP_PORT=1025` (host-run API); `backend/.env.example` has empty `SMTP_HOST`; the `api` service in `docker-compose.yml` has no `SMTP_*` override (grep found none). The Runner must verify before S1 that the running API delivers to Mailhog (`SMTP_HOST=mailhog` in-container, or `localhost` host-run; port 1025; `SMTP_TLS=false`). Check with `POST /api/v1/admin/notifications/test` (super_admin) and look for a new Mailhog message. If none arrives: ENV ISSUE (route to Infrastructure Configuration), rerun.
- Email sending may be asynchronous (`backend/internal/email` has a scheduler). Poll Mailhog up to 30 s per mail.
- Token extraction: regex `reset-password\?token=([A-Za-z0-9_-]+)` on the plain-text body (decode quoted-printable soft breaks `=\r\n` and `=3D`).

## Preconditions and accounts

| Account | Role | Password | Source |
|---------|------|----------|--------|
| `admin@bilimbaga.local` | super_admin | `Admin1234!` (or current) | seeded, migration 029 |
| `uat.employee@test.com` | employee | `NewPass123!` | earlier UAT scenarios (`user-onboarding-20260609.md`) |
| `uat.deptadmin@test.com` | department_admin | `NewPass123!` | create per `route-guards-20261009.md` S0 (S7 step 7) |
| `uat.recovery@test.com` | employee | `NewPass123!` | create in S0 below (dedicated, so lockouts do not break other scenarios) |
| `uat.inactive@test.com` | employee, deactivated | any | create in S0 below |

- Platform at `http://localhost`; API at `http://localhost/api/v1`; `{public_app_url}` = configured `PUBLIC_APP_URL`.
- DB read access (`psql`) for hash/expiry checks (S3, S6). If unavailable mark those steps SKIPPED, not failed.
- AuthLimiter is 10 req/min per IP on `/auth/*`. Pace steps (wait 60 s between bursts). A 429 caused by test pacing is not a defect.
- Fresh browser context for anonymous steps.

## Scenario S0: Setup

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Create `uat.recovery@test.com` (employee, password `NewPass123!`); complete forced password change once if prompted | User active, can log in | |
| 2 | Admin | Create `uat.inactive@test.com`, then deactivate | Status inactive | |
| 3 | Tester | `DELETE http://localhost:8025/api/v1/messages` | Mailhog empty | |
| 4 | Tester | `POST /api/v1/admin/notifications/test` as admin | 200 and message in Mailhog; else ENV ISSUE (Gap 1) | |

## Scenario S1: Login-page entry point and forgot form (UI)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Anonymous | Open `/login` | Localized "Forgot password?" link visible (AC-7) | |
| 2 | Anonymous | Switch locale en, ru, kk | Link text translated in each (`auth.recovery.*`), no raw keys | |
| 3 | Anonymous | Click the link | URL `/forgot-password`; email form only; no admin/portal navigation; no `Authorization` header in network log | |
| 4 | Anonymous | Submit malformed email (`abc`) | Validation error; no neutral success message | |
| 5 | Anonymous | Submit `uat.recovery@test.com` | Neutral confirmation message | |
| 6 | Anonymous | Submit `nobody-unknown@test.com` | Identical message and visual state as step 5 | |
| 7 | Tester | Check Mailhog | Exactly 1 message, to `uat.recovery@test.com`; none for the unknown address | |

## Scenario S2: No account enumeration (API)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | `POST /api/v1/auth/forgot-password {"email":"uat.recovery@test.com"}` | 200, `{data:{message:M},error:null}` | |
| 2 | Tester | Same with unknown email | 200, body byte-identical to step 1 | |
| 3 | Tester | Same with `uat.inactive@test.com` | 200, same body | |
| 4 | Tester | Time 5 calls each for known and unknown email | Every response (known, unknown, inactive) takes >= ~400 ms (response floor); known vs unknown medians differ by < ~100 ms or ratio < 1.5 | |
| 5 | Tester | Body `{}` and `{"email":"not-an-email"}` | 422 `VALIDATION_ERROR` (was 400 before PR #197) | |
| 6 | Tester | Check Mailhog | Mail only for the active known user | |

## Scenario S3: Token properties and email content

Pre-step: clear Mailhog; request a reset for `uat.recovery@test.com`; wait for the mail.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Open the mail (plain and HTML parts) | Both parts exist; link `{public_app_url}/reset-password?token=<T>` | |
| 2 | Tester | Inspect `T` | URL-safe chars only; >= 32 bytes (>= 43 base64url chars) | |
| 3 | Tester | DB: `SELECT token_hash, expires_at, used_at FROM password_reset_tokens WHERE user_id=<id>` | One row; `token_hash` != `T` and equals SHA-256 of `T`; `expires_at` = created + 60 min (+-1 min); `used_at` NULL | |
| 4 | Tester | Request a second reset for the same user | New mail; old row invalidated (`used_at` set or deleted) | |
| 5 | Tester | `POST /auth/reset-password` with the OLD token and a valid password | 400 `INVALID_TOKEN` | |
| 6 | Tester | Grep API logs and `audit_log` for `T` | Not present; no API response contains it | |
| 7 | Tester | Set user `preferred_locale` to `ru` (DB update, or via FR-BB116 once merged), request reset; repeat with `kk` and NULL | Mail in Russian; Kazakh; tenant default locale respectively | |
| 8 | Tester | Request reset for `uat.inactive@test.com` | No mail, no `password_reset_tokens` row | |

## Scenario S4: Reset flow (UI)

Use the newest valid token `T2` from S3.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Anonymous | Open `{public_app_url}/reset-password?token=T2` | New password + confirm form; no nav; no Authorization header | |
| 1a | Tester | `GET {public_app_url}/reset-password?token=x` (response headers) and inspect DOM while the page is open and after navigating away | `Referrer-Policy: no-referrer` and `Cache-Control: no-store` headers; `<meta name="referrer" content="no-referrer">` present while the page is mounted and removed after leaving; no outbound `Referer` carries the token | |
| 2 | Anonymous | Enter `abc` / mismatched confirmation | Client-side rules block submit (same as change-password) | |
| 3 | Tester | API: `POST /auth/reset-password {token:T2,new_password:"alllowercase1"}` | 422 `VALIDATION_ERROR` (was 400 before PR #197); token NOT consumed (DB `used_at` NULL; step 4 still works) | |
| 4 | Anonymous | Enter `ResetPass123!` twice, submit | Redirect to `/login` with success notice | |
| 5 | Anonymous | Log in as `uat.recovery@test.com` / `ResetPass123!` | Success; no forced password change | |
| 6 | Anonymous | Log in with old `NewPass123!` | 401 | |
| 7 | Anonymous | Reopen the same reset link | Error "invalid or expired" with link to `/forgot-password` | |
| 8 | Tester | Replay `POST /auth/reset-password` with `T2` | 400 `INVALID_TOKEN`, same body as for garbage and expired tokens | |
| 9a | Anonymous | Open `/reset-password` with no token (or `token=`) | Error state on open, with link to `/forgot-password`; no form; no crash | |
| 9b | Anonymous | Open `/reset-password?token=garbage`, enter a valid new password twice, submit | The form is shown on open (token validity is judged only by the server, there is no pre-check endpoint); after submit the "invalid or expired" error with link to `/forgot-password` (API 400 `INVALID_TOKEN`); no crash | |

## Scenario S5: Reset clears lock and revokes sessions

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as recovery user; save `refresh_token_A` and `access_token_A` | Success | |
| 2 | Tester | 5 failed logins (wrong password; wait 60 s between bursts) | Account locked (30 min) | |
| 3 | Tester | Login with correct password | Rejected while locked | |
| 4 | Tester | Request reset, take token from Mailhog, reset to `ResetPass456!` | 200 | |
| 5 | Tester | Login with `ResetPass456!` immediately | Success (`failed_attempts=0`, `locked_until` NULL) | |
| 6 | Tester | `POST /auth/refresh` with `refresh_token_A` | 401 (all prior refresh tokens revoked) | |
| 7 | Tester | Wait 11 s (epoch cache window ~10 s), then call any protected endpoint with `access_token_A`; then log in fresh | 401 `TOKEN_REVOKED` for `access_token_A`; fresh login works. Optionally repeat after an admin reset-password (FR-BB61), which also stamps `password_changed_at` | |

## Scenario S6: Expiry and throttle

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Request reset; DB: `UPDATE password_reset_tokens SET expires_at = now() - interval '1 minute' WHERE user_id=<id> AND used_at IS NULL` | Row aged | |
| 2 | Tester | Reset with that token | 400 `INVALID_TOKEN`, generic message identical to used/unknown | |
| 3 | Tester | Call forgot-password 4 times for one email within an hour (paced under 10/min) | All return identical 200; only 3 mails in Mailhog (silent per-user throttle, 3 per hour) | |
| 3a | Tester | Concurrency variant: send 5 parallel forgot-password requests for one known user | Exactly 3 `password_reset_tokens` rows created in the hour and 3 mails; an unknown email still returns 200 with no row | |
| 4 | Tester | 12 requests in under a minute from one IP | 429 `RATE_LIMITED` with `Retry-After` by the 11th | |
| 5 | Tester | After a later request or job run, check expired rows | Tokens expired more than 24 h ago are purged (a just-expired row is NOT purged immediately; age a row with `expires_at = now() - interval '25 hours'` to test); flag if the table only grows | |

## Scenario S7: Admin unlock (UI and API)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Lock `uat.recovery@test.com` with 5 bad logins | Locked | |
| 2 | Admin | Open `/admin/users` | Recovery user row shows "Unlock account"; unlocked users show none (AC-8) | |
| 3 | Tester | `GET /api/v1/users` as admin | Recovery user `is_locked:true`; others `false` | |
| 4 | Admin | Click "Unlock account" | Localized success toast; row refreshes, action disappears | |
| 5 | Employee | Log in with correct password | Success without waiting for expiry | |
| 6 | Tester | `POST /api/v1/users/{random-uuid}/unlock` as admin | 404 | |
| 7a | Tester | Unlock with the employee token | 403 `FORBIDDEN` (no `users:manage`) | |
| 7b | Tester | Unlock with the department_admin token for a locked user of strictly lower rank in its OWN department | 200 | |
| 7c | Tester | Unlock with the department_admin token for a user in ANOTHER department | 404 `NOT_FOUND` (out-of-scope and unknown ids are indistinguishable, FR-BB117 D-4c / D-5); the target stays locked | |
| 7d | Tester | Unlock with the department_admin token for a peer `department_admin` or a `super_admin` in its own department | 403 `FORBIDDEN` (rank rule, FR-BB117 D-1); target unchanged | |
| 8 | Tester | Unlock without token | 401 | |

## Scenario S8: Audit trail

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | View audit log filtered for the recovery user | `auth.password_reset_requested` (actor = the target user), `auth.password_reset_completed`, `auth.password_reset_failed` (from S4 step 8 / S6 step 2; no entity id), `users.unlock` (actor admin, target user id) | |
| 2 | Tester | Inspect metadata | No token value anywhere | |
| 3 | Tester | Look for entries about `nobody-unknown@test.com` | None revealing existence | |

## Pass / Fail criteria

- PASS: all steps match; mail arrives in Mailhog with correct link; enumeration checks identical; lock cleared by reset and by admin unlock; audit present.
- FAIL (defect): differing response for known/unknown/inactive; plaintext token stored or logged; reusable or non-expiring token; weak password consumes token; reset leaves lock or refresh tokens; employee can unlock; missing login link; hardcoded strings.
- ENV ISSUE: Mailhog unreachable or SMTP not wired (Gap 1); no DB access (skip DB-only steps); #33 not merged; migration not applied.
- REQ GAP candidates: (resolved: unlock uses users:manage, department_admin limited to own department; decision #346: another department answers 404, a peer or higher rank in its own department answers 403); purge mechanism.

## Acceptance Criteria Coverage

| FR | AC# | Covered by |
|----|-----|------------|
| FR-BB115 | AC-1 | S1 5-6, S2 |
| FR-BB115 | AC-2 | S3 2-5, 8 |
| FR-BB115 | AC-3 | S3 1, 6, 7 |
| FR-BB115 | AC-4 | S4, S5 |
| FR-BB115 | AC-5 | S7 |
| FR-BB115 | AC-6 | S8 |
| FR-BB115 | AC-7 | S1, S4 |
| FR-BB115 | AC-8 | S7 2-4 |
| FR-BB115 | AC-9 | S2 4, S6 |
| FR-BB115 | AC-10 | Not UAT (tests; file-existence check only) |

## Out of Scope

MFA/SSO; password history; SMS recovery; visual regression of email HTML; FR-BB61 admin temporary-password email.
