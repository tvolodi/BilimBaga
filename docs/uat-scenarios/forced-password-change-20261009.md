---
slug: forced-password-change
title: "Forced Password Change Enforced by Backend; No Working Default Admin on Fresh Deployments — UAT Scenario"
feature: forced-password-change (FR-BB14, FR-BB18, FR-BB110; GitHub issues #160, #150, #152)
version: 1
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **PR #162** (ISS-150 / ISS-152, commit `bd74606`): **MERGED** in `origin/main`. Migration 033 (forces `force_password_change=true` for `admin@bilimbaga.local` while its hash is still the 029/030 default hash), `auth.BootstrapAdmin` at startup (`BOOTSTRAP_ADMIN_PASSWORD` | `BOOTSTRAP_ADMIN_GENERATE` | warning), `scripts/seed-test-env.ts` reads `E2E_ADMIN_PASS`.
- **PR #180** (ISS-160): server-side enforcement (`403 PASSWORD_CHANGE_REQUIRED` in `auth.Authenticate`, allow-list `POST /auth/change-password` and `GET /users/me`; cache of 10 s per user, invalidated on change/reset; frontend `PasswordChangeGuard`; e2e helpers). **Merged to `origin/main` as d4c03ec (2026-10-09).** S2-S4 and S6 run in enforcement mode; baseline mode is only needed against an older build.
- Migrations 029 (seed admin, force=true), 030 (resets admin to the documented default hash, force=false), 033 apply automatically at API startup. 030 and 029 are NOT to be edited; behaviour is verified, not changed.

## Important rule for the Runner (about the default password)

The old documented default admin password is **not** an expected-to-work credential in this scenario and must not be written into any step as the thing to log in with. Where a step needs "the initial admin credential", the Runner takes it from the deployment under test:

| Deployment mode | Initial admin credential (`{INIT_PW}`) comes from |
|-----------------|---------------------------------------------------|
| A. `BOOTSTRAP_ADMIN_PASSWORD` set | that env value (secret store / compose env) |
| B. `BOOTSTRAP_ADMIN_GENERATE=true` | the single log line printed at first startup (`docker compose logs api`); the value must be redacted in the report |
| C. neither set (migration-seeded DB; local developer default) | the migration-seeded credential documented in `README.md`; this mode is tested ONLY to prove the migration-seeded account is forced to change (S1), and is never used on `qa` |

Report the mode used. On `qa` only modes A or B are allowed; if the QA deployment is mode C, S5 is a FAIL (see criteria). Secrets are never written to the report.

## Accounts and data (QA starts with an EMPTY database)

| Alias | Email | Role | Password |
|-------|-------|------|----------|
| SA | `admin@bilimbaga.local` | super_admin | `{INIT_PW}` initially, then `UatForce123!` after S1 |
| EMP-F | `uat.force.emp@test.com` | employee | temporary (from create response), then `UatForce456!` |
| EXM-F | `uat.force.examiner@test.com` | examiner | temporary, then `UatForce456!` |
| EMP-R | `uat.force.reset@test.com` | employee | used for admin-reset check (S3) |

Platform at `http://localhost`, API `http://localhost/api/v1`. Run first on a FRESH database (drop the volume, restart the API) so migrations 029-033 apply in order. AuthLimiter is 10 req/min per IP on `/auth/*`; pace.

Representative blocked endpoints (the "probe set" P): `GET /users`, `GET /admin/dashboard`, `GET /exams`, `GET /questions`, `GET /audit`, `POST /exams` (body any), `GET /portal/exams`, `GET /portal/results`, `GET /categories`, `GET /departments`.

## Scenario S1: Migrations 029/030/033 on a fresh DB (state of the seeded admin)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Start API on an empty DB; `SELECT version, dirty FROM schema_migrations` | Version >= 33, `dirty=false` (QA: >= 34 if PR #185 merged; not required here) | |
| 2 | Tester | `SELECT email, force_password_change, password_changed_at FROM users WHERE email='admin@bilimbaga.local'` | One row; mode C: `force_password_change=true` (033 corrected 030); mode A/B: see S5; `password_changed_at` NULL in mode C | |
| 3 | Tester | Read the API startup log | Mode C: exactly one `SECURITY` warning about the default hash; mode A: "admin password set from BOOTSTRAP_ADMIN_PASSWORD" and the secret appears 0 times in the log; mode B: one-time generated password logged once | |
| 4 | SA | `POST /auth/login` with `{INIT_PW}` | 200; `data.user.force_password_change=true` (modes B and C; mode A: false by design, see S5) | |
| 5 | Tester | Restart API (second startup) | Mode C: SECURITY warning again while default hash still present; mode B: generated password NOT logged again; mode A: no repeated "set from env" if hash already matches | |
| 6 | Tester | Idempotency of 033: re-run it (down then up in a rolled-back transaction, or restart) on a DB whose admin already changed the password | Admin untouched (flag false, hash unchanged), no warning | |

## Scenario S2: Backend rejects API calls while a change is pending (needs PR #180)

Use SA (mode B/C) with the token `T0` obtained in S1 step 4, before changing the password.

**Session rule (decision of issue #342, FR-BB14 / ISS-171):** a password change revokes every access token issued before it, including the token that made the change, and returns a replacement session (`data.access_token`, new refresh cookie). Steps after a change use the replacement token `TN`, never the old one. Revocation compares `iat` with the change time at second resolution, so a token issued in the SAME second as the change survives (documented caveat, not a defect): wait at least 2 seconds between obtaining `T0` and the change, otherwise step 8b is not meaningful.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | Call every endpoint of the probe set P with `Authorization: Bearer T0` | Every call: HTTP 403, `{data:null, error:{code:"PASSWORD_CHANGE_REQUIRED", message}}` (not 401, not 200, no data leakage) | |
| 2 | SA | `GET /users/me` with `T0` | 200; `force_password_change=true` visible | |
| 3 | SA | `POST /auth/logout` with `T0` | 200 (logout never blocked) | |
| 4 | SA | Re-login for a fresh token; `POST /auth/refresh` (no cookie) | 200 for login; refresh without cookie 401 `INVALID_REFRESH_TOKEN` (unaffected) | |
| 5 | SA | `POST /auth/change-password` with `T0`, wrong `current_password` | 400 `INVALID_CREDENTIALS`; flag stays true; P still 403 | |
| 6 | SA | `POST /auth/change-password` with weak `new_password` `alllower1` | 400 `WEAK_PASSWORD`; flag stays true | |
| 7 | SA | `POST /auth/change-password` valid (`UatForce123!`) with `T0` (obtained at least 2 s earlier) | 200; flag cleared; body carries `data.access_token` (`TN`) and a new `refresh_token` cookie | |
| 8a | SA | Immediately (no wait, no re-login) call the probe set with the replacement token `TN` | All 200 (the 10 s cache is invalidated on change); at most one retry within 1 s tolerated, record any delay | |
| 8b | SA | Call `GET /users/me` and one probe endpoint with the OLD `T0` | 401 `TOKEN_REVOKED` ("password was changed; please sign in again"); the old session is gone by design | |
| 9 | SA | Login with the old initial credential | 401 `INVALID_CREDENTIALS`; login with `UatForce123!` returns `force_password_change=false` | |
| 10 | Anonymous | Public/unauthenticated routes while another user is pending: `GET /health`, `GET /verify/...` (public certificate verification), `POST /auth/forgot-password`, `POST /auth/login` | Unaffected (200 or their normal status) | |
| 11 | SA | Request with a missing token to a probe endpoint | 401 `MISSING_TOKEN` (not 403 `PASSWORD_CHANGE_REQUIRED`) | |

## Scenario S3: Admin-created and admin-reset users

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | `POST /users` for EMP-F (employee) and EXM-F (examiner) | 201; `temporary_password` returned | |
| 2 | EMP-F | Login with the temporary password | 200; `force_password_change=true` | |
| 3 | EMP-F | `GET /portal/exams`, `GET /portal/results`, `GET /users` | `PASSWORD_CHANGE_REQUIRED` 403 on all (portal included) | |
| 4 | EMP-F | `GET /users/me`, then (at least 2 s after the login of step 2) `POST /auth/change-password` to `UatForce456!` | 200 and 200; the change response carries the replacement `access_token` (`TN`) | |
| 5 | EMP-F | `GET /portal/exams` with `TN` | 200 immediately | |
| 5b | EMP-F | `GET /portal/exams` with the OLD token from step 2 | 401 `TOKEN_REVOKED` (200 only if the login and the change fell in the same second: repeat with a longer wait, record the gap) | |
| 6 | EXM-F | Same as steps 2-5 against `GET /admin/dashboard` and `GET /admin/exams/{id}/analytics` | 403 `PASSWORD_CHANGE_REQUIRED` until changed, then 200/403-by-role as normal (examiner role rules unchanged) | |
| 7 | SA | `POST /users/{EMP-R}/reset-password` for a user who has already changed password and holds a valid token `TR` | 200 with new `temporary_password` | |
| 8 | EMP-R | Using the OLD token `TR`, call `GET /portal/exams` | Within 10 s (cache expiry) calls start returning 403 `PASSWORD_CHANGE_REQUIRED` (or 401 `TOKEN_REVOKED` per PR #144); record time to take effect; must not stay 200 beyond ~15 s | |
| 9 | EMP-R | Login with the temporary password, change it | Back to normal access | |
| 10 | SA | CSV-import a user via `POST /users/import`; log in as that user with the issued temporary password | Same forced behaviour as step 3 (the developer did not test import) | |
| 11 | Forgot-flow | EMP-R requests forgot-password and resets via link | After reset the user is NOT forced to change again (existing FR-BB115 behaviour) and normal calls return 200 | |

## Scenario S4: UI behaviour

Fresh browser context.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | EMP-F (forced) | Log in via `/login` | Redirected to `/change-password`; no portal/admin chrome | |
| 2 | EMP-F | Reload while on `/change-password` | Stays; form works | |
| 3 | EMP-F | Type `/portal` and `/portal/results` into the address bar (full page load) | Redirected to `/change-password` promptly (target <= 2 s; UAT on the branch observed ~7.8 s and NO redirect on `/portal/results`: record the actual time; > 5 s or no redirect = FAIL, see developer note about `retry:false`) | |
| 4 | EMP-F | Same with `/admin` after granting nothing | Guard order: change-password first or the normal role redirect, but never a screen with data | |
| 5 | EMP-F | Change password in en, ru, kk (use a new forced user per locale) | Labels and `WEAK_PASSWORD`/`INVALID_CREDENTIALS` messages localized; no raw keys | |
| 6 | EMP-F | After successful change | Lands on `/portal` (employee) / `/admin` (admin, examiner) without a second login | |
| 7 | SA | Same flow for admin on a fresh deployment (mode B/C) | `/change-password` first; after change `/admin` | |

## Scenario S5: No documented default admin credential works on a fresh deployment

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Mode A (`BOOTSTRAP_ADMIN_PASSWORD` valid, complexity ok) on a fresh DB: login with `{INIT_PW}` | 200; `force_password_change=false` by design; log says set from env, secret absent from logs | |
| 2 | Tester | Mode A: log in with the migration-seeded default credential (the documented one in the README; this is a NEGATIVE check) | 401 `INVALID_CREDENTIALS` | |
| 3 | Tester | Mode B: login with generated password | 200; `force_password_change=true`; negative check from step 2 also 401; password logged exactly once | |
| 4 | Tester | Mode A with a WEAK `BOOTSTRAP_ADMIN_PASSWORD` on a default-hash DB | API startup aborts with a complexity message; value not echoed in the log | |
| 5 | Tester | Mode A with weak or 94-char value on an already-rotated admin | API starts; one warning "BOOTSTRAP_ADMIN_PASSWORD is set but invalid ... ignored"; rotated password still works | |
| 6 | Tester | Mode C on a fresh DB intended as public/QA | Treated as FAIL for `qa`: public deployment must use mode A or B. On `local` it is accepted (S1 verifies the forced change) | |
| 7 | Tester | `grep` the repo `docker-compose*.yml`, `.env.example`, `deploy/` for a literal default password in compose env files used for QA | No literal admin password value in deploy configs (examples with empty value only) | |
| 8 | Tester | Existing deployment upgrade (non-empty DB at schema 32 with default hash): start new API | 033 applies; admin forced; if `BOOTSTRAP_ADMIN_*` unset the default credential still logs in (with force=true) until used: record as the known residual risk, not a defect | |
| 9 | Tester | Existing deployment whose admin was already rotated (QA admin) | 033 leaves hash and flag untouched; no SECURITY warning | |
| 10 | Tester | e2e/seed on a fresh DB: `scripts/seed-test-env.ts` with `E2E_ADMIN_PASS` exported, and without it | With the variable: admin password changed then seeds; without: documented behaviour (candidate list or clear failure message), never silent success on a locked admin; re-run is idempotent | |

## Pass / fail / env-issue criteria

- **PASS:** S1-S5 as expected on a fresh DB; every probe call is `403 PASSWORD_CHANGE_REQUIRED` while pending and 200 right after the change; no documented default credential logs in on any deployment configured per mode A/B.
- **FAIL (defect):** any probe endpoint returns 200/data with a pending-change token (issue #160 symptom); `change-password`, `GET /users/me` or `logout` blocked; the replacement `access_token` from the change response needs a re-login or does not work immediately, or the old token still works 2 s or more after the change; a user whose admin reset is pending keeps full access beyond ~15 s; default credential works on a mode A/B deployment; secret appears in logs; migration 033 modifies a rotated admin.
- **ENV ISSUE:** cannot recreate a fresh DB or read startup logs; QA deploy mode unknown (REQ GAP 1); `E2E_ADMIN_PASS` unavailable.

## Pre-fix baseline (what a Runner sees on `main` `a45b92b`: PR #162 merged, PR #180 not)

S1 PASS (033 present: admin `force_password_change=true`; SECURITY warning). S2 steps 1 and 8 FAIL/N-A: probe calls with `T0` return **200** (UI-only enforcement, issue #160); step 2, 3, 5, 6, 7 PASS (change-password works). S3 steps 3, 6, 8 FAIL (portal/admin endpoints 200 for forced users); S3 steps 1, 2, 4 PASS. S4 step 1 PASS (login redirect exists); steps 3, 4 FAIL (no route guard for direct navigation). S5 steps 1-5, 9 PASS (PR #162). Record as "baseline reproduced"; mark S2-S4 BLOCKED on PR #180.

## Acceptance criteria coverage

| Source | Criterion | Covered by |
|--------|-----------|-----------|
| #160 | Backend rejects authenticated routes except change-password / logout / me with `PASSWORD_CHANGE_REQUIRED` | S2 steps 1-3, 11; S3 steps 3, 6 |
| #160, #342 | Fresh access right after change-password without re-login (replacement token); old token revoked | S2 steps 8a, 8b; S3 steps 5, 5b; S4 step 6 |
| #160 | Admin-reset of a user re-imposes the block | S3 steps 7-9 |
| #160 | Frontend guard and global 403 handling | S4 steps 1-4 |
| #160 | E2E/seed use env password or change first | S5 step 10 |
| #150 | Migration 033 forces change only while default hash present; idempotent; rotated untouched | S1 steps 1-2, 6; S5 step 9 |
| #150 | `BOOTSTRAP_ADMIN_PASSWORD` / `BOOTSTRAP_ADMIN_GENERATE` | S5 steps 1-5 |
| #150 | Startup warning while default hash present | S1 steps 3, 5 |
| #152 | Fresh public deployment has no working known super_admin credential | S5 steps 2, 3, 6, 7 |
| api-conventions 2.2 | `INVALID_CREDENTIALS`, `WEAK_PASSWORD`, `TOKEN_REVOKED`, `MISSING_TOKEN` | S2, S3 |

## Gaps and REQ GAP candidates

1. **`PASSWORD_CHANGE_REQUIRED` is not in `docs/requirements/api-conventions.md`** (the 403 catalogue lists only `FORBIDDEN`) and no FR (FR-BB14/FR-BB18) states the allow-list (`change-password`, `users/me`, plus logout and public routes). Add an AC and the code to the catalogue.
2. **Deployment mode contract missing:** no requirement says a public (QA/test/prod) deployment MUST set `BOOTSTRAP_ADMIN_PASSWORD` or `BOOTSTRAP_ADMIN_GENERATE`. Today mode C silently runs with a known credential plus a log warning. Decide whether startup should refuse in a production flag. The QA deployment mode is unknown to the BA.
3. **Cache window:** enforcement uses a 10 s per-user cache per replica (stale up to 10 s on another replica; admin reset takes up to 10 s). Acceptable tolerance (15 s used above) needs to be a stated requirement.
4. **Direct-navigation redirect latency** (~7.8 s, none on `/portal/results`) is a UI gap noted by the developer's UAT; S4 step 3 encodes a 2 s target that is not yet in any requirement.
5. **Interaction with PR #144 token revocation: RESOLVED (issue #342, BA decision 2026-10-09).** The design stands: change-password revokes the session that made the change and issues a replacement (FR-BB14 "Implementation note", ISS-171 / PR #204). The scenario was wrong to expect the same token to keep working; S2 step 8 is now 8a (replacement token works) and 8b (old token 401 `TOKEN_REVOKED`). The UAT observation in S3 step 5 (old token still 200) is the documented second-granularity caveat (login and change in the same second), not a defect; steps now wait 2 s.
6. **bilimbaga-test (production-class)** admin state is escalated to the user (issue #152 comment); this scenario never runs there.
7. CSV-imported users and `hr_admin` were not tested by the developer; hr_admin does not exist (renamed to `examiner`, migration 005).
8. kk/ru change-password UI not verified by the developer (S4 step 5 is the first check).
