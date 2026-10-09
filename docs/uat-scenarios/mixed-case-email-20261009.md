---
slug: mixed-case-email
title: "Mixed-Case Email (Login, Forgot Password, Create, Duplicate Detection) — UAT Scenario"
feature: mixed-case-email (FR-BB14, FR-BB18, FR-BB115; GitHub issue #164)
version: 1
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **PR #177** (ISS-164, cycle 2 head `fd59e6b`): `api.NormalizeEmail` (trim + lower) in `auth.Login`, `auth.ForgotPassword`, `users.CreateUser`, `users.ImportUsers` per row and login audit entries; lookups `WHERE lower(u.email)=$1`; duplicate probe `EXISTS(lower(email)=lower($1))` before INSERT (also for CSV import). **OPEN at authoring time (not in `origin/main` `a45b92b`).** Until merged run in baseline mode (see below).
- Merged prerequisites: FR-BB115 account recovery (issue #33) and PR #144 (needed by S2 and S3); PR #162 (admin force-change).
- No migration, no unique index (follow-up: `UNIQUE INDEX` on `lower(email)`). Race on concurrent create is therefore NOT covered by DB constraint; see gap 2.

## Email observability

Reset mail is read from Mailhog (`GET http://localhost:8025/api/v2/messages`, clear with `DELETE http://localhost:8025/api/v1/messages`); see `account-recovery-20261009.md` section "Email observability" for the SMTP precondition (Gap 1 there) and token regex. Send a probe `POST /admin/notifications/test` as admin first; if no mail arrives S2 is ENV ISSUE. On QA, read mail from the QA mail sink the operator provides; if none exists, S2 steps check only the API response and log/audit evidence and are marked SKIPPED (not failed).

## Accounts and data (QA starts with an EMPTY database)

Only `admin@bilimbaga.local` exists (password `{ADMIN_PW}` from the environment; never hard-coded; change it first if `force_password_change=true`). Created by S0:

| Alias | Email as entered by admin | Stored (expected) | Role | Password after S0 |
|-------|--------------------------|-------------------|------|-------------------|
| MC1 | `UAT.MixedCase@Test.COM` | `uat.mixedcase@test.com` | employee | `UatCase123!` |
| LC1 | `uat.lower@test.com` | `uat.lower@test.com` | employee | `UatCase123!` |
| WS1 | `  UAT.Spaces@Test.com  ` (leading/trailing spaces) | `uat.spaces@test.com` | employee | `UatCase123!` |

AuthLimiter is 10 req/min per IP on `/auth/*`: pace calls (wait 60 s between bursts); a 429 caused by pacing is not a defect.

## Scenario S0: Setup

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Login (change password if forced); `GET /users/roles` for the employee role id | Token; role id | |
| 2 | Admin | `POST /users {email:"UAT.MixedCase@Test.COM", full_name:"UAT Mixed", role_id}` | 201; `data.email` = `uat.mixedcase@test.com`; `temporary_password` returned | |
| 3 | Admin | `POST /users` for `uat.lower@test.com` and `"  UAT.Spaces@Test.com  "` | 201; stored emails lowercase and trimmed (`uat.spaces@test.com`) | |
| 4 | Each | Log in with the exact stored lowercase email and the temporary password; `POST /auth/change-password` to `UatCase123!` | 200; later login `force_password_change=false` | |
| 5 | Tester | `DELETE` Mailhog messages | Empty | |
| 6 | DB (optional) | `SELECT count(*) FROM users WHERE email <> lower(btrim(email))` | 0 (legacy-row check from the developer note); if > 0 record the count and the emails: those users are unreachable after the fix (REQ GAP 1) | |

## Scenario S1: Login with different casing (API and UI)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | `POST /auth/login {email:"uat.mixedcase@test.com", password:"UatCase123!"}` | 200 (control) | |
| 2 | Tester | Same with `"UAT.MixedCase@Test.COM"` (as the admin typed it) | 200; `data.user.email` = `uat.mixedcase@test.com` | |
| 3 | Tester | Same with `"Uat.MIXEDcase@test.com"` and `"  uat.mixedcase@test.com  "` | 200 | |
| 4 | Tester | Mixed-case email, wrong password | 401 `INVALID_CREDENTIALS`, body identical to unknown-user login failure | |
| 5 | Tester | `"UAT.LOWER@TEST.COM"` for LC1 | 200 | |
| 6 | Tester | `"UAT.SPACES@test.com"` for WS1 | 200 | |
| 7 | Anonymous | UI `/login`: enter `UAT.MixedCase@Test.COM` + password | Login succeeds, lands on `/portal`; header shows lowercase email on profile | |
| 8 | Tester | Audit: `GET /audit?...` (super_admin) or DB for the login events of steps 2-3 | Login audit entries carry the normalised (lowercase) email; same user id as step 1 | |
| 9 | Tester | Lockout: 5+ wrong-password attempts using different casings of the same address (mind the AuthLimiter pacing), then correct login | Failed attempts count against ONE account (casing does not evade lockout): account locked (423 `ACCOUNT_LOCKED`) per FR-BB14 threshold; admin `POST /users/{id}/unlock` restores | |

## Scenario S2: Forgot password with different casing

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | `POST /auth/forgot-password {email:"UAT.MixedCase@Test.COM"}` | 200 neutral body | |
| 2 | Tester | Poll Mailhog up to 30 s | Exactly 1 mail, `To` = `uat.mixedcase@test.com`; link `reset-password?token=...` (this mail was missing pre-fix) | |
| 3 | Tester | Same body with lowercase address; compare responses | Byte-identical 200 body to step 1 | |
| 4 | Tester | Unknown address `Nobody.Unknown@Test.com` | Same 200 body; no mail | |
| 5 | Anonymous | UI `/forgot-password`: submit `UAT.MixedCase@Test.COM` | Neutral confirmation, identical to the lowercase case; mail arrives | |
| 6 | Anonymous | Open the link from step 2, set `ResetCase123!`; log in with `uat.mixedcase@test.com` and with `UAT.MIXEDCASE@test.com` | Reset succeeds; both logins 200; old password 401 | |
| 7 | Tester | Response-time floor for known mixed-case vs unknown | Both >= ~400 ms, no timing oracle introduced by normalisation (median ratio < 1.5) | |

## Scenario S3: User creation and duplicate detection across case

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | `POST /users` with `"uat.mixedcase@test.com"` (lowercase of existing MC1) | 409 `DUPLICATE_EMAIL` | |
| 2 | Admin | `POST /users` with `"UAT.MIXEDCASE@TEST.COM"` | 409 `DUPLICATE_EMAIL` | |
| 3 | Admin | `POST /users` with `" uat.mixedcase@test.com "` (spaces) | 409 `DUPLICATE_EMAIL` | |
| 4 | Admin | `POST /users {email:"Uat.New.Person@Test.com"}` | 201; stored `uat.new.person@test.com`; login with exact typed form works after password change | |
| 5 | Admin | `POST /users/import` CSV with rows: `UAT.Import.One@Test.com`, `uat.import.one@test.com` (case-duplicate inside the file), `UAT.MixedCase@Test.com` (duplicate of existing), `uat.import.two@test.com` | Import result: rows 1 and 4 created (stored lowercase); row 2 and row 3 reported as duplicates/errors per row (no abort of the whole import); `GET /users?search=import` lists exactly 2 | |
| 6 | Admin | Edit user (`PUT /users/{id}`) changing email of LC1 to `UAT.MixedCase@test.com` (if the API accepts email updates; else mark N/A) | 409 `DUPLICATE_EMAIL` or the field is ignored; never a second account with a case-variant | |
| 7 | Admin | UI `/admin/users`: search `MIXEDCASE` and `mixedcase` | Same single row both times (search is case-insensitive) | |
| 8 | DB (optional) | `SELECT lower(email), count(*) FROM users GROUP BY 1 HAVING count(*)>1` | 0 rows | |
| 9 | Admin | Deactivate MC1, then login with mixed-case address | 401 `ACCOUNT_INACTIVE` (same as lowercase) | |

## Pass / fail / env-issue criteria

- **PASS:** S1-S3 all as expected, including mail delivered for the mixed-case forgot request and case-insensitive duplicate detection (single create and CSV import).
- **FAIL (defect):** any 401 `INVALID_CREDENTIALS` for a casing/whitespace variant of a valid account (issue #164 symptom); no reset mail for mixed-case request; duplicate created across case; different 200 body or markedly different timing between known and unknown addresses; casing evades lockout.
- **ENV ISSUE:** SMTP/Mailhog not delivering the control mail (S2 only; route to Infrastructure Configuration); AuthLimiter 429 from pacing.

## Pre-fix baseline (on `main` `a45b92b`, before PR #177)

S0 steps 2-3 PASS (create already lowercases). S1 step 1 PASS, steps 2, 3, 5, 6, 7 FAIL (401 `INVALID_CREDENTIALS`). S2 step 1 returns 200 but step 2 FAILS (0 mails); step 6 cannot proceed. S3 steps 1 PASS (lowercase duplicate -> 409 even pre-fix), step 2 and 3 expected FAIL if the duplicate probe is case-sensitive pre-fix (issue says uniqueness is case-insensitive for lowercase input; behaviour for mixed-case input is unobserved, record actual). Record as "baseline reproduced"; mark BLOCKED on PR #177.

## Acceptance criteria coverage

| Source | Criterion | Covered by |
|--------|-----------|-----------|
| Issue #164 | Login with email typed as admin entered it succeeds | S1 steps 2, 3, 5, 6, 7 |
| Issue #164 | Forgot-password with mixed case sends mail | S2 steps 1, 2, 5, 6 |
| Issue #164 | Duplicate create detected case-insensitively | S3 steps 1-3 |
| Issue #164 | Import normalised | S3 step 5 |
| PR #177 | Trim + lowercase; audit entries normalised | S0 step 3, S1 step 8 |
| FR-BB115 | No account enumeration, timing floor preserved | S2 steps 3, 4, 7 |
| FR-BB14 | Lockout counted per account | S1 step 9 |
| api-conventions 2.2/2.3 | `INVALID_CREDENTIALS` 401, `ACCOUNT_LOCKED` 423, `ACCOUNT_INACTIVE` 401, `DUPLICATE_EMAIL` 409 | S1, S3 |

## Gaps and REQ GAP candidates

1. **Legacy mixed-case rows:** after the fix such users (if any exist from imports before normalisation) become unreachable; needs a data-migration requirement (blocked on `migration.lock`) and a pre-deploy check query (S0 step 6).
2. **No DB-level uniqueness** (`UNIQUE INDEX ON lower(email)` is a follow-up): concurrent creates can still produce case-variant duplicates; requirement missing in FR-BB18.
3. **FR-BB14 / FR-BB18 do not state the email normalisation rule** (trim + lowercase) or whether email update is permitted; the scenario step S3.6 depends on it.
4. Unicode / IDN case folding (e.g. `İ`, Cyrillic domains) is unspecified; `lower()` in Postgres vs Go `strings.ToLower` may differ; out of scope here but should be decided for ru/kk tenants.
5. Whether `+tag` addresses are distinct is unspecified (assumed distinct).
