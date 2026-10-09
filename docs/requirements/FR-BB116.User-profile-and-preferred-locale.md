# FR-BB116 — User Profile Page and Persisted Preferred Locale

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB116 |
| Phase | 1/6 — Foundation + Polish (gap closure; completes roadmap 6.2 "locale switcher ... in user profile settings" and the locale resolution of 6.1) |
| Priority | 2 |
| Status | Validated |
| Depends On | FR-BB14, FR-BB18, FR-BB61, FR-BB62, FR-BB110, FR-BB111, FR-BB313, FR-BB316 |

## Description
Roadmap 6.2 requires a locale switcher "on the login screen and in user profile settings", and 6.1 requires emails "sent to recipient's locale". Today the column `users.preferred_locale` (migration 023) is read by the email service, but nothing in the API can write it: `User` / `GET /users/me` do not expose it, no endpoint updates it, and the `UserProfilePage` named in FR-BB62 does not exist (no `/profile` route; `LocaleSwitcher` only writes `localStorage`). Consequence: every email falls back to the tenant default locale, and a user's language choice is lost on a new device or after clearing storage. This requirement adds a self-service profile page (view own identity, change language, link to change password) and a self-only endpoint to persist `preferred_locale`, applied automatically at login.

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/users/me` additionally returns `preferred_locale` (`"kk"`, `"ru"`, `"en"` or `null`); `GET /users/{id}` and the list endpoint return the same field for admins. No other existing response field changes.
- [ ] AC-2: `PATCH /api/v1/users/me` (any authenticated role, no extra RBAC permission) accepts `{ "preferred_locale": "<code>|null" }`. A code must be one of the tenant `available_locales`; otherwise 400 `VALIDATION_ERROR`. `null` clears the preference. On success returns 200 with the updated user (`{ data, error: null }`).
- [ ] AC-3: The endpoint can only modify the caller's own `preferred_locale`; any other key in the body (e.g. `role_id`, `department_id`, `email`, `status`) is rejected with 400 `VALIDATION_ERROR` and nothing is persisted (no privilege escalation).
- [ ] AC-4: A successful change writes an `audit_log` entry `user.preferred_locale_updated` (actor = entity = caller, metadata contains old and new value only).
- [ ] AC-5: Frontend: a route `/profile`, reachable by admin and employee roles (inside `RequireAuth`, rendered in the respective layout), shows full name, email, department, role (read-only), a language select (kk/ru/en, native labels as in `LocaleSwitcher`) and a "Change password" link to `/change-password`. Admin `TopBar` and `PortalLayout` each expose a link to it from the user name/menu.
- [ ] AC-6: Selecting a language on the profile page calls `PATCH /users/me` via a React Query mutation, immediately calls `i18n.changeLanguage`, updates `localStorage['i18n-lang']` and `document.documentElement.lang`, shows a localized success toast, and invalidates the `['me']` query. On API failure the previous language is restored and a localized error is shown.
- [ ] AC-7: After login (and on app bootstrap with a valid session), if `preferred_locale` is non-null and differs from the current UI language, the UI switches to it; if null, the existing `localStorage` / tenant default behaviour is unchanged. Using the existing header `LocaleSwitcher` while authenticated also persists via the same endpoint (best-effort; failure does not block the UI switch).
- [ ] AC-8: Emails created after the change (FR-BB61 resolution order `preferred_locale` -> tenant `default_locale` -> `en`) are rendered in the new locale; covered by a test that updates the preference and asserts the locale passed to the email service.
- [ ] AC-9: All new strings exist in `en`, `ru`, `kk` under `profile.*`; zero hardcoded user-visible text; page is usable at >= 375 px width and all controls are keyboard-operable with labels (FR-BB63).
- [ ] AC-10: Tests: backend `service_test.go` + `handler_test.go` (valid locale, locale not in `available_locales`, null clears, extra field rejected, unauthenticated 401, audit written); frontend tests for ProfilePage (renders identity, change language calls PATCH and switches i18n, failure rolls back) and login-time locale application. All run green.

## Technical Specification

### Database Schema
None. Uses existing `users.preferred_locale TEXT NULL` (migration 023). Optional guard in a new migration (next free number; never edit 023): `ALTER TABLE users ADD CONSTRAINT users_preferred_locale_chk CHECK (preferred_locale IS NULL OR preferred_locale ~ '^[a-z]{2,3}$');`

### API Contract
| Method | Path | Auth | Request | Success | Errors |
|--------|------|------|---------|---------|--------|
| GET | `/api/v1/users/me` | any authenticated | - | 200 `{ data: { ...user, preferred_locale }, error: null }` | 401, 404 |
| PATCH | `/api/v1/users/me` | any authenticated | `{ "preferred_locale": "ru" }` or `null` | 200 `{ data: user, error: null }` | 400 `VALIDATION_ERROR` (unknown locale / extra field), 401 |

Register `PATCH /users/me` beside `GET /users/me`, before `/users/{id}` routes. Subject to the global limiter.

### Go Implementation Notes
- Package `users`: add `PreferredLocale *string` (`db:"preferred_locale" json:"preferred_locale"`) to `User` and to the select lists in the repository; `Handler.UpdateMe` -> `Service.UpdateMyLocale(ctx, userID, *string)` -> `Repository.SetPreferredLocale`. Decode with `DisallowUnknownFields` to implement AC-3.
- Allowed locales read from the cached tenant config (inject a small interface; do not import `tenant` into a cycle), never `os.Getenv`.
- Audit via `audit.Write`. Wrap errors with context.

### Frontend Implementation Notes
- Route `/profile`; page `frontend/src/pages/profile/ProfilePage.tsx`; hook `useUpdateMyLocale()` in `src/api/users.ts` (query key `['me']`, existing `useMe()`).
- shadcn/ui `Card`, `Select`, `Button`; reuse the locale list from `LocaleSwitcher` (extract to a shared constant).
- Locale application effect lives next to the auth bootstrap (`TenantProvider` / `RequireAuth`), not in components.
- i18n namespace `profile.*` in `src/locales/{en,ru,kk}.json`.

## Notes
- Out of scope: editing name/email, avatar, notification preferences, timezone, RTL locales (no RTL locale is configured today).
- Closes the dangling `UserProfilePage` reference in FR-BB62 AC-3 / notes.
