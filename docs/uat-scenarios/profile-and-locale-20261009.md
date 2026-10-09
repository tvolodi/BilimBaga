---
slug: profile-and-locale
title: "User Profile Page and Preferred Locale — UAT Scenario"
feature: profile-and-locale (FR-BB116; GitHub issue #42, parent #34)
version: 1
created: 2026-10-09
author: Business Analyst
---

## Code that must be merged before running

- **Issue #42** (FR-BB116): `PATCH /users/me`, `preferred_locale` in `GET /users/me`, `/profile` route and `ProfilePage`, TopBar/PortalLayout links, login-time locale application, `profile.*` i18n keys. Status ready, no PR at authoring time.
- Rebuild frontend and API. No migration required (column from migration 023; an optional CHECK-constraint migration may exist).
- Expected pre-fix baseline: `GET /users/me` lacks `preferred_locale`; `PATCH /users/me` returns 404/405; `/profile` falls to not-found/redirect. All scenarios FAIL.

## Preconditions and accounts

| Account | Role | Password | Source |
|---------|------|----------|--------|
| `admin@bilimbaga.local` | super_admin | `Admin1234!` (or current) | seeded |
| `uat.employee@test.com` | employee | `NewPass123!` | earlier UAT scenarios |
| `uat.deptadmin@test.com` | department_admin | `NewPass123!` | create per `route-guards-20261009.md` S0 |
| `uat.examiner@test.com` | examiner | `NewPass123!` | same (route-guards S0) |

- Tenant `available_locales` includes `kk, ru, en` (check tenant config API or `/admin/settings`). `xx` is never available (used for the rejection test).
- Mailhog is needed only for S6. **PRECONDITION GAP:** SMTP wiring in `docker-compose.yml` is unverified (see `account-recovery-20261009.md`, "Email observability"). If mail does not arrive, S6 is ENV ISSUE.
- Fresh browser context per role, empty `localStorage` (`i18n-lang` unset).
- Reset between scenarios: `PATCH /users/me {"preferred_locale": null}` for each test user.

## Scenario S1: Profile page content and navigation

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in; open the portal user menu | Profile link in `PortalLayout`; click goes to `/profile` | |
| 2 | Employee | Review page | Full name, email, department, role shown read-only (no inputs); language select with native labels (Қазақша, Русский, English); "Change password" link to `/change-password` | |
| 3 | Admin | Log in; open TopBar user menu | Profile link present; `/profile` renders in the admin layout | |
| 4 | Dept admin, examiner | Repeat step 3 | Same | |
| 5 | Anonymous | Open `/profile` | Redirect to `/login` | |
| 6 | Employee | Click "Change password" | Lands on `/change-password` | |
| 7 | Tester | Compare `GET /users/me` with page | Values match | |

## Scenario S2: API contract of /users/me

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | `GET /api/v1/users/me` | 200; `preferred_locale` key present (`null` initially); all earlier fields unchanged | |
| 2 | Admin | `GET /api/v1/users/{employee_id}` and `GET /api/v1/users` | Both expose `preferred_locale` | |
| 3 | Anonymous | `GET /users/me` without token | 401 | |

## Scenario S3: PATCH /users/me validation and self-only

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | `PATCH /users/me {"preferred_locale":"ru"}` | 200 `{data:user,error:null}` with `preferred_locale:"ru"`; also works with examiner and dept admin tokens (no extra permission) | |
| 2 | Employee | `GET /users/me` | `ru` persisted | |
| 3 | Employee | `PATCH {"preferred_locale":null}` | 200; cleared | |
| 4 | Employee | `PATCH` with `en`, then `kk` | Each 200 | |
| 5 | Employee | `PATCH {"preferred_locale":"xx"}` | 400 `VALIDATION_ERROR`; unchanged | |
| 6 | Employee | `PATCH {"preferred_locale":"ru","role_id":"<admin role id>"}` | 400 `VALIDATION_ERROR`; nothing persisted (locale and role unchanged) | |
| 7 | Employee | Each of `{"email":"x@y.z"}`, `{"department_id":"..."}`, `{"status":"inactive"}`, `{"id":"<other>"}`, `{}` | 400 for unknown fields; record `{}` behaviour | |
| 8 | Employee | `PATCH /users/{admin_id}` with employee token | 403/404/405; admin locale unchanged | |
| 9 | Anonymous | `PATCH /users/me` without token | 401 | |
| 10 | Tester | Malformed JSON; `{"preferred_locale":123}` | 400; no 500 | |

## Scenario S4: Profile UI locale change

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | On `/profile` choose Русский | `PATCH /users/me` 200; UI immediately Russian; localized success toast; `localStorage['i18n-lang']='ru'`; `<html lang="ru">` | |
| 2 | Employee | Reload | Still Russian; select shows Русский | |
| 3 | Admin | View audit log filtered for the employee | `user.preferred_locale_updated`, actor = entity = employee, metadata only old and new value | |
| 4 | Employee | Choose Қазақша then English | Each persists with toast; `lang` attribute follows | |
| 5 | Employee | Intercept `PATCH /users/me` to return 500, choose another language | Previous language restored; localized error shown; select reverts | |
| 6 | Employee | Keyboard only | All controls reachable and operable, labelled | |
| 7 | Employee | Viewport 375 px | No horizontal scroll; usable | |
| 8 | Tester | Cycle en/ru/kk | No raw `profile.*` keys; no hardcoded English in ru/kk | |

## Scenario S5: Locale applied at login and bootstrap

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Set employee `preferred_locale='kk'` via API | Persisted | |
| 2 | Anonymous | New context, empty storage; log in as employee | UI switches to Kazakh after login; `localStorage['i18n-lang']='kk'` | |
| 3 | Employee | Close tab, reopen `/portal` with valid session | Kazakh | |
| 4 | Tester | Preference null; new context with `localStorage['i18n-lang']='ru'`; log in | Stays Russian (unchanged behaviour) | |
| 5 | Employee | Preference `kk`; use header `LocaleSwitcher` to choose `en` | UI switches at once; `GET /users/me` returns `en` | |
| 6 | Employee | Block `PATCH` and use the header switcher | UI still switches; no blocking error | |
| 7 | Anonymous | Use LocaleSwitcher on `/login` | Works as before; no PATCH sent | |
| 8 | Two contexts | Set `ru` in context A; log in as same user in fresh context B | B shows Russian | |

## Scenario S6: Email locale

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Clear Mailhog; set employee `ru`; admin resets the employee's password (FR-BB61 temp-password mail) | Mail body Russian | |
| 2 | Tester | Set `kk`, repeat | Kazakh | |
| 3 | Tester | Set `null`, repeat | Tenant `default_locale` language | |
| 4 | Tester | Set `en`, repeat | English | |

Restore the employee password to `NewPass123!` afterwards (force change flow).

## Pass / Fail criteria

- PASS: contract, validation, self-only, persistence, UI switching, login application, audit and email locale as specified.
- FAIL (defect): any extra key accepted; other user's locale modifiable; locale outside `available_locales` accepted; no rollback on failure; locale not applied at login; missing audit entry; hardcoded strings.
- ENV ISSUE: #42 not merged; Mailhog/SMTP not wired (S6 only).
- REQ GAP candidates: behaviour of `{}` body; whether the optional CHECK migration shipped.

## Acceptance Criteria Coverage

| FR | AC# | Covered by |
|----|-----|------------|
| FR-BB116 | AC-1 | S2 |
| FR-BB116 | AC-2 | S3 1-5, 9-10 |
| FR-BB116 | AC-3 | S3 6-8 |
| FR-BB116 | AC-4 | S4 3 |
| FR-BB116 | AC-5 | S1 |
| FR-BB116 | AC-6 | S4 1-2, 4-5 |
| FR-BB116 | AC-7 | S5 |
| FR-BB116 | AC-8 | S6 |
| FR-BB116 | AC-9 | S4 6-8 |
| FR-BB116 | AC-10 | Not UAT (tests; file-existence check only) |

## Out of Scope

Editing name/email; avatar; timezone; notification preferences; RTL locales.
