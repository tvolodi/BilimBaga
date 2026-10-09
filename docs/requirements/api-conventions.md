# API Conventions (as implemented)

> Status: Accepted. Source: GitHub issue #172. Derived from the backend code on `main`, not from aspiration. Refreshed 2026-10-09 against `main` d4c03ec (PRs #162, #174, #180, #196, #197 and the frontend-only #186 merged since the first version). Where code is inconsistent the inconsistency is recorded in section 13; fixed items are marked `Resolved by PR #n`. Paths are relative to `backend/` unless noted.

## 1. Envelope

Every JSON response is `{ "data": <payload|null>, "error": <null|{code,message,...}> }`.

- Shared helpers: `api.WriteJSON(w, status, payload)` (`internal/api/response.go:9`) and `api.WriteError(w, status, code, message)` (`internal/api/response.go:18`). Most domains use these.
- Two packages keep a private copy of the same envelope: `auth` (`internal/auth/handler.go:48-62`) and `tenant` (`internal/tenant/handler.go:32`). The rate limiter has a third (`internal/ratelimit/middleware.go:38-50`). The wire format is identical.
- Success: handlers pass `map[string]any{"data": x, "error": nil}` to `api.WriteJSON` (e.g. `internal/users/handler.go:77`). The error body is always `{code, message}`; some errors add keys: `details` (`exams/handler.go:330-346`; for `INSUFFICIENT_ADAPTIVE_QUESTIONS` it is an **object** `{rule_id, difficulty, required, available}`, not an array; `EXAM_RULES_UNSATISFIED`) or `fields` (`ERR_VALIDATION`, `exams/handler.go:96-104`, `questions/handler.go:84-93`).
- Content-Type is `application/json`, except file downloads (section 9).
- Frontend contract: `apiFetch` (`frontend/src/api/apiFetch.ts:21-44`) throws an `ApiError{code, message, details}` whenever `body.error` is non-null; the code is the machine key.

## 2. Error code catalogue

Status is what the code returns today. "Where" gives representative source locations; many codes are reused in several handlers.

### 2.1 Cross-cutting

| Code | HTTP | Where |
|------|------|-------|
| MISSING_TOKEN | 401 | `internal/auth/middleware.go:61`, `internal/rbac/middleware.go:19`, `internal/ai/handler.go:35` |
| INVALID_TOKEN | 401 | `auth/middleware.go:65,74,82,87,95,101` (malformed header, bad signature, bad claims, missing `iat`, user gone) |
| TOKEN_EXPIRED | 401 | `auth/middleware.go:72` |
| TOKEN_REVOKED | 401 | `auth/middleware.go:110` (access token issued before the user's last password change: self change-password since ISS-171, admin reset, token reset) |
| UNAUTHORIZED | 401 | `audit/handler.go:30,67`, `auth/handler.go:160,170,185`, `email/handler.go:27` (missing user/tenant in context) |
| FORBIDDEN | 403 | `rbac/middleware.go:23`; in-handler scope checks `users/handler.go:102`, `sessions/handler.go:178`, `ai/handler.go:142` |
| NOT_FOUND | 404 | `api/uuid.go:89` (malformed path id), `users/handler.go:100`, `departments/handler.go:56` |
| VALIDATION_ERROR | 422 | `api/uuid.go:53,65`, `users/handler.go:137,173`, `auth/recovery_handler.go:24,70`, `auth/recovery_service.go:56,105`, `ai/handler.go:49`. Aligned to 422 everywhere by PR #197 (I-1); forgot-password, reset-password and AI generate used to answer 400 |
| INVALID_BODY | 400 | undecodable JSON/multipart: `ai/handler.go:41`, `auth/handler.go:96,178`, `departments/handler.go:41`, `users/handler.go:260` |
| RATE_LIMITED | 429 | `ratelimit/middleware.go:45` (sets `Retry-After: 60`, line 40) |
| INTERNAL_ERROR | 500 | catch-all: `auth/handler.go:72`, `auth/middleware.go:105` (fails closed), `audit/handler.go:44` |

There is no `ROLE_*` family of codes in the code; role-related endpoints use the cross-cutting codes.

### 2.2 Auth and account

| Code | HTTP | Where |
|------|------|-------|
| INVALID_CREDENTIALS | 401 on login, 400 on change-password (wrong current password) | `auth/service.go:71,113` (401); `auth/service.go:252` (400) |
| PASSWORD_CHANGE_REQUIRED | 403 | `auth/middleware.go:114` (PR #180, ISS-160). Returned by `Authenticate` on every protected route except `POST /auth/change-password` and `GET /users/me` while `users.force_password_change` is true; see section 4 |
| ACCOUNT_LOCKED | 423 | `auth/service.go:82,106` |
| ACCOUNT_INACTIVE | 401 | `auth/service.go:91` |
| INVALID_REFRESH_TOKEN | 401 | `auth/handler.go:117`, `auth/service.go:155,167,175` |
| WEAK_PASSWORD | 400 | `auth/service.go:239`; policy `auth/password.go:20-34` (min 8, upper, lower, digit) |
| INVALID_TOKEN (reset link) | 400 | `auth/recovery_service.go:98` (same code as the 401 JWT case; I-2) |

### 2.3 Users, departments, tenant, audit, email

| Code | HTTP | Where |
|------|------|-------|
| DUPLICATE_EMAIL | 409 | `users/handler.go:161` (email is trimmed and lowercased by `api.NormalizeEmail` at create, CSV import, login and forgot-password; uniqueness is case-insensitive: pre-check `lower(email)` in `users/repository.go:115`, plus the unique index `idx_users_email_lower_unique` from migration 035 (ISS-181), which is created only when no legacy case-variant duplicates exist, otherwise skipped with a NOTICE and reported at startup as a WARN plus audit action `users.duplicate_emails_detected`) |
| MISSING_FILE | 400 | `users/handler.go:266` |
| FILE_TOO_LARGE | 413 | `users/handler.go:257` (whole body over the cap, rejected before parsing, PR #197), `users/handler.go:280` (file content over 10 MiB) |
| INVALID_FILE_TYPE | 415 | `users/handler.go:282` |
| INVALID_CSV | 400 | `users/handler.go:290,295` |
| TOO_MANY_ROWS | 400 | `users/handler.go:301` (more than 500 data rows) |
| DUPLICATE_NAME | 409 | `departments/handler.go:58,93` |
| DEPARTMENT_HAS_CHILDREN, DEPARTMENT_NOT_EMPTY | 409 | `departments/handler.go:118,120` |
| LOGO_TOO_LARGE | 413 | `tenant/handler.go:58-64` (status chosen in the handler) |
| INVALID_LOGO_TYPE, INVALID_LOCALE | 400 | `tenant/service.go:177,251` (default status 400, `tenant/handler.go:59`) |
| EMAIL_UNAVAILABLE | 503 | `email/handler.go:38` |
| USER_NOT_FOUND | 404 (reports), 500 (email test) | `reports/handler.go:98,128`, `email/handler.go:33` (I-3) |

### 2.4 Categories, tags, questions

| Code | HTTP | Where |
|------|------|-------|
| ERR_VALIDATION | 422 with `fields[]` | `questions/handler.go:84-93`, `questions/translation_handler.go:67`. Also raised for a blank default-locale answer option (PR #196): `questions/option_validation.go:31-45`, wired at `questions/handler.go:146,273,346`, `questions/service.go:194,240`, bulk import `questions/import_export_service.go:91`; see section 2.7 |
| ERR_NOT_FOUND | 404 | `categories/handler.go:102`, `exams/handler.go:264` |
| ERR_INVALID_BODY | 400 | `categories/handler.go:36`, `exams/handler.go:206`, `questions/import_export_handler.go:28,34,47,68` |
| ERR_INVALID_PARAM | 400 | `questions/import_export_handler.go:129`, `reports/handler.go:53,75,121,172,193` |
| ERR_INTERNAL | 500 | `categories/handler.go:26`, `exams/handler.go:183`, `certificates/handler.go:97` |
| ERR_CATEGORY_CYCLE, ERR_PARENT_NOT_FOUND, ERR_INVALID_NAME | 400 | `categories/handler.go:104-110` |
| ERR_CATEGORY_IN_USE | 409 | `categories/handler.go:108` |
| ERR_TAG_DUPLICATE, ERR_TAG_IN_USE | 409 | `tags/handler.go:84,86` |
| ERR_INVALID_NAME (tags) | 400 | `tags/handler.go:88` |
| ERR_NOT_DRAFT | 409 | `exams/handler.go:481`, `questions/handler.go:446` |
| ERR_INVALID_TRANSITION | 409 | `exams/handler.go:386`, `questions/handler.go:414` |
| ERR_INVALID_STATUS | 422 | `questions/handler.go:356` |
| ERR_STEM_REQUIRED | 400 | `questions/handler.go:419` |
| INVALID_FIELD_FOR_TYPE, MISSING_MODEL_ANSWER | 400 | `questions/handler.go:152,157` (field-level), `questions/handler.go:343,352` (no `ERR_` prefix; I-4) |
| ERR_QUESTION_NOT_FOUND | 404 | `questions/translation_handler.go:43` |
| ERR_UNSUPPORTED_LOCALE, ERR_MISSING_OPTION_TRANSLATIONS, ERR_CANNOT_DELETE_DEFAULT_LOCALE | 4xx (see source) | `questions/translation_handler.go:94,102,137` |
| ERR_FILE_TOO_LARGE | 413 | `questions/import_export_handler.go:25` (whole body over the cap, rejected before parsing, PR #197), `:52` (file content over 10 MiB) |
| ERR_INVALID_FILE_TYPE | 415 | `questions/import_export_handler.go:54` |
| ERR_IMPORT_VALIDATION, ERR_BATCH_TOO_LARGE | 4xx (see source) | `questions/import_export_handler.go:76,90` |

### 2.5 Exams, assignments

| Code | HTTP | Where |
|------|------|-------|
| INSUFFICIENT_QUESTIONS | 422 | publish with no rules `exams/handler.go:349-351`; session start with too-small pool `sessions/handler.go:58-60` |
| INSUFFICIENT_ADAPTIVE_QUESTIONS | 422 with `details` | `exams/handler.go:332-338` (adaptive needs at least 5 questions per difficulty per rule) |
| EXAM_RULES_UNSATISFIED | 422 with `details` | `exams/handler.go:340-346` |
| EXAM_NOT_DRAFT | 409 | `exams/handler.go:303-309,358` |
| EXAM_NOT_ACTIVE | 409 (assign), 422 (start session) | `exams/handler.go:714-716`, `sessions/handler.go:46-48` (I-5) |
| ASSIGNMENT_ALREADY_EXISTS | 409 | `exams/handler.go:720-726` |
| ERR_FORBIDDEN | 403 | `exams/handler.go:729,761` (assign target outside caller scope) |
| ERR_DEADLINE_IN_PAST | 400 | `exams/handler.go:731` |
| ERR_ACTIVE_SESSIONS | 409 | `exams/handler.go:422` |
| RULE_NOT_MANUAL | 400 | `exams/handler.go:806` |
| EXAM_NOT_FOUND | 404 | `exams/handler.go:823`, `reports/handler.go:60`, `ai/handler.go:95` (exam CRUD itself uses ERR_NOT_FOUND; I-4) |

### 2.6 Sessions, results, certificates, AI

| Code | HTTP | Where |
|------|------|-------|
| EXAM_NOT_ASSIGNED | 403 | `sessions/handler.go:40-42`, `portal/handler.go:50` |
| EXAM_ARCHIVED | 403 | `sessions/handler.go:43-45` |
| EXAM_OUTSIDE_WINDOW, ATTEMPTS_EXHAUSTED | 422 | `sessions/handler.go:49-54` |
| SESSION_ALREADY_OPEN | 409 | `sessions/handler.go:55-57` |
| SESSION_NOT_FOUND | 404 | `sessions/handler.go:146,176,198,220`, `certificates/handler.go:94`, `ai/handler.go:138` |
| SESSION_FORBIDDEN | 403 | `sessions/handler.go:97,148,200,222,401`, `certificates/handler.go:82` |
| SESSION_NOT_ACTIVE | 422 | `sessions/handler.go:100,151` |
| SESSION_EXPIRED | 422 | `sessions/handler.go:103,154` |
| SESSION_IN_PROGRESS | 422 | `sessions/handler.go:225,246` |
| INVALID_ANSWER_OPTION, INVALID_ANSWER_FORMAT, INVALID_TIME_SPENT, INVALID_EVENT_TYPE, INVALID_DATE | 400 | `sessions/handler.go:107-117,91,143,316,324` |
| INVALID_SCORE | 422 | `sessions/handler.go:377` |
| NOT_ADAPTIVE | 404 | `sessions/handler.go:404` |
| SESSION_NOT_SUBMITTED, SESSION_NOT_PASSED, EXAM_NOT_CERTIFIABLE | 422 | `certificates/handler.go:84-92` |
| NOT_A_LOYALTY_SESSION | 400 | `ai/handler.go:140` |
| AI_RATE_LIMITED | 429 | `ai/handler.go:51` (hourly generation cap) |
| AI_UNAVAILABLE | 503 | `ai/handler.go:54,144` |
| INVALID_PARAM | 400 | `ai/handler.go:84,128` (cf. ERR_INVALID_PARAM) |

### 2.7 Blank answer option text (PR #196, issue #173)

Create (`POST /questions`), update (`PUT /questions/{id}`) and bulk import of a choice question (single, multiple, truefalse, likert) reject any answer option whose text for the question's **default locale** is empty or whitespace-only. Other locales may stay empty. `shorttext` questions have no options and are not checked. The failure is 422 `ERR_VALIDATION` with `fields[]` entries `{field: "answer_options[i].translations.<locale>.text", message}` (create: `questions/handler.go:146`; service guard `questions/service.go:194` for create and `:240` for update, mapped at `questions/handler.go:273,346`; import: per-row message in `ERR_IMPORT_VALIDATION`, `questions/import_export_service.go:91`). Unknown JSON keys are still ignored (no `DisallowUnknownFields`), so a misnamed field such as `body` is caught only because the resulting text is blank. Known limit: an update that sends zero options is still accepted. Editing a legacy question that already has blank options now returns 422 until the options are filled.

## 3. Malformed UUID rule (PR #147, ISS-141)

- A malformed UUID in a **path** parameter returns **404 NOT_FOUND** (`"resource not found"`), because such a resource cannot exist. Implemented by `api.RequireUUIDPathParams(names...)` (`internal/api/uuid.go:83-95`), mounted once on the authenticated route group (`internal/router/router.go:95`). Names covered: `id, userId, sessionId, examId, questionId, tagId, sectionId, ruleId, assignmentId` (`api/uuid.go:76`, `UUIDPathParamNames`). A router test enumerates registered routes against this list (comment at `api/uuid.go:72-75`), so a new UUID path parameter name must be added to the list.
- A malformed UUID in a **query** parameter returns **422 VALIDATION_ERROR** (`"<name> must be a valid UUID"`). Helper `api.UUIDQuery(w, r, name)` (`api/uuid.go:47-57`): an absent parameter is fine; the caller must return when `ok == false`. Comma lists use `api.ValidateUUIDList(w, param, ids)` (`api/uuid.go:62-70`). Current users: `audit/handler.go:135` (actor_id), `questions/handler.go:172,185` (category_id, tag_ids), `questions/import_export_handler.go:132-145` (ids, category_id, tag_ids), `sessions/handler.go:304` (exam_id), `users/handler.go:48,55` (department_id, role_id).
- Validity means canonical 36-character hyphenated only (`api.IsUUID`, `api/uuid.go:36-42`).
- **Exceptions**: the middleware is mounted only in the authenticated group, so public routes are not covered. `GET /verify/{code}` (`router.go:86`) takes a non-UUID code; an unknown or malformed code returns **200** with `data.valid == false` and no error, to avoid information leakage (`certificates/handler.go:59-67`). A DB failure there is 500 `INTERNAL_ERROR`.
- UUIDs inside JSON bodies have no shared helper; each handler validates its own (I-6).

## 4. Authentication and authorization

- Access token: `Authorization: Bearer <JWT>` on every route in the protected group (`router.go:90-96`). Claims used: `sub`, `role`, optional `department_id`, `iat` (`auth/middleware.go:80-98`). Missing header: 401 `MISSING_TOKEN`; wrong scheme, bad signature or bad claims: 401 `INVALID_TOKEN`; expired: 401 `TOKEN_EXPIRED`; issued before the last password change: 401 `TOKEN_REVOKED` (`auth/middleware.go:109-112`). If the per-user account-state lookup fails the middleware fails closed with 500 (`auth/middleware.go:104-106`).
- **Forced password change** (PR #162, #180; ISS-150/152/160): the same lookup returns `users.force_password_change` (cached per user; the cache entry is invalidated right after a password change or reset, `auth/handler.go:22-29`, `router.go:41-51`). While it is true, `Authenticate` answers **403 `PASSWORD_CHANGE_REQUIRED`** (`auth/middleware.go:113-115`) for every protected route except the allow-list `forcePasswordChangeAllowedPaths` (`auth/middleware.go:36-47`): `POST|any /api/v1/auth/change-password` and `GET /api/v1/users/me` only. `/auth/login`, `/auth/refresh`, `/auth/logout`, forgot/reset password and `/verify/{code}` are public routes and are unaffected. The login response also carries `force_password_change` (`auth/model.go:64`, `auth/service.go:141`). The flag is set by migration 033 for the seeded admin while it still has the default password and is cleared by a successful change-password (`auth/service.go:234`). ISS-171: a successful self change-password also stamps `users.password_changed_at`, revokes all of the user's refresh tokens and returns `data.access_token` (+ `token_type`, `expires_in`) with a new `refresh_token` cookie; all earlier access tokens then get 401 `TOKEN_REVOKED`. The SPA redirects to the change-password page on this code and never retries it.
- **Bootstrap admin** (PR #162, ISS-150): at API startup `auth.BootstrapAdmin` (`cmd/api/main.go:154-181`, `internal/auth/bootstrap.go`) applies `BOOTSTRAP_ADMIN_PASSWORD` to the seeded `admin@bilimbaga.local` only while it still has the default password (8-72 bytes, upper, lower, digit, not the default; an invalid value is ignored with a warning), or, with `BOOTSTRAP_ADMIN_GENERATE=true`, generates a random one-time password and logs it once. Both come from the typed Config (`config/config.go:58-66,158-161`), are documented in `backend/.env.example:26-33`, and the env password is never logged (only the generated one-time password is, once). With neither set the admin keeps the default password, a startup warning is logged and the first login forces a change.
- Refresh token: opaque value in cookie `refresh_token`, `HttpOnly`, `SameSite=Strict`, `Path=/api/v1/auth`, `Secure` from config (`auth/service.go:329-337`). `POST /auth/refresh` reads the cookie (`auth/handler.go:101`) and returns a new access token in the body; `POST /auth/logout` clears it. Both sit in the public, tightly rate-limited group (`router.go:68-77`). Failures: 401 `INVALID_REFRESH_TOKEN`.
- Frontend flow: `apiFetch` attaches Bearer from the React Query cache; `downloadFile` refreshes once on a 401 and retries (`frontend/src/api/download.ts:52-62`).
- **401 vs 403**: 401 means no usable credentials (missing, invalid, expired or revoked token, or no role in context, `rbac/middleware.go:18-20`). 403 means authenticated but not allowed (`rbac/middleware.go:22-25` `FORBIDDEN`; ownership checks `SESSION_FORBIDDEN`, `ERR_FORBIDDEN`; `EXAM_NOT_ASSIGNED`).
- Permission middleware: `rbac.RequirePermission(cache, resource, action)` (`internal/rbac/middleware.go:14`), applied per route with `r.With(...)`. Permissions are the string `resource:action`, loaded from the DB into an in-memory `Cache` at startup (`rbac/cache.go:28-67`). In use today (from `router.go`): `tenant:manage`, `departments:read|manage`, `users:read|manage`, `audit:read`, `categories:manage`, `tags:read|manage`, `questions:read|write`, `exams:read|write|assign`, `reports:read`, `grading:read|write`. The verb `manage` versus `write` is not unified (I-7).
- Routes without `RequirePermission` ("any authenticated user") are commented as such in the router: `/users/me`, `/portal/*`, `/auth/change-password`, `GET /categories`, and `GET /users/{id}` with an in-handler self/scope check (`router.go:125-126`). Static sub-paths (`/users/roles`, `/users/import`, `/questions/import`, `/questions/export`) are registered before the `/{id}` routes.
- Tenancy: `auth.TenantContext()` runs on every request (`router.go:64`) and puts `tenant_id` in the context; handlers and repositories scope by it.

## 5. Pagination, filtering, sorting

- Query names: `page` (1-based) and `per_page`. Non-numeric or non-positive values silently fall back to defaults and do not produce an error: `parseIntParam` in `users/handler.go`, `exams/handler.go:150-160`, `audit/handler.go`; `parsePagination` in `sessions/handler.go:278-296`; inline in `reports/handler.go:79-93`.
- Defaults and caps: users 20, cap 100 (`users/service.go:54-62`); questions 20 and exams 20 (`parseIntParam`, no cap in the handler); audit 50 (`audit/handler.go:35`); grading queue, history and My Results 20, cap 100 (`sessions/handler.go:278-296`); user record 20, cap 100 (`reports/handler.go:85-92`).
- List response shape: `data: { items: [...], meta: { page, per_page, total } }` for users (`users/types.go:46-56`), audit (`audit/handler.go:48-58`), questions (`questions/handler.go:222-232`), exams (`exams/handler.go:189-199`), grading queue (`sessions/model.go:349`). `items` is normalised to `[]`, not null (`questions/handler.go:219`). Some lists (departments, categories) return a tree or array without meta.
- Filters: snake_case query names, allow-listed per handler: `status`, `search`, `locale`, `category_id`, `tag_ids` (comma list), `department_id`, `role_id`, `exam_id`, `actor_id`, `actor` (ILIKE on name), `action`, `entity_type`. Unknown parameters are ignored.
- Sorting: `sort` + `order` for exams and questions (`exams/handler.go:174-175`, `questions/handler.go:211-212`); `sort` + `dir` for My Results, allow-listed to `date|score` and `asc|desc`, defaulting to `date`/`desc` (`sessions/handler.go:420-427`). Naming differs (I-8). Invalid values fall back silently.
- Dates: `from`/`to` are RFC 3339 on audit (`audit/handler.go:151-160`; an unparseable value is silently dropped) and are `YYYY-MM-DD` on the dashboard report (`reports/handler.go:219-224`, unparseable value silently ignored); `date_from`/`date_to` are `YYYY-MM-DD` on the grading queue and give 400 `INVALID_DATE` (`sessions/handler.go:313-326`) (I-9).

## 6. Identifiers, timestamps, bodies

- IDs are UUID v4 strings, validated as in section 3. Timestamps are UTC ISO 8601 / RFC 3339 (project rule in `CLAUDE.md`). Date-only filters use `YYYY-MM-DD`.
- JSON field names are snake_case.
- JSON request bodies are decoded with `json.NewDecoder`; failure maps to 400 `INVALID_BODY` (or `ERR_INVALID_BODY`). Only the public recovery endpoints (`auth/recovery_handler.go:23,69`) and the two multipart import endpoints (`upload.ParseImportMultipart`, section 8) cap the body with `http.MaxBytesReader`; other JSON bodies are unbounded (I-10).

## 7. Rate limiting

- Limiters (`internal/ratelimit/middleware.go`): `AuthLimiter` 10 req/min per IP on `/health` and `/auth/*` (`ratelimit/middleware.go:53`; `router.go:68-77`); `GlobalLimiter` 300 req/min per IP on the public and protected groups (`:67`); `AnswerSaveLimiter` 60 req/min keyed by session id on `PUT /portal/sessions/{id}/answers/{questionId}` (`:84`; `router.go:252-254`). AI generation has its own hourly cap in the service (`AI_RATE_LIMITED`, 429).
- 429 response: the envelope with `RATE_LIMITED` / "Too many requests" and header `Retry-After: 60`, a constant (`ratelimit/middleware.go:40`). The repo sets no `X-RateLimit-*` headers itself; the `httprate` library (v0.15.0, `go.mod:12`) may add its own, which this document does not guarantee.
- `DISABLE_RATE_LIMIT=true|1` disables the Auth, Global and AnswerSave limiters (test-only: E2E and k6 load runs; never set on shared or production-class instances; default is rate limiting ON). Each limiter checks `disabled()` when it is constructed (`ratelimit/middleware.go:18-21,54,68,85`), so the variable is read once at router build, not per request. I-11 resolved by PR #197. Documented in `backend/.env.example:67-70`.
- Client IP comes from `chimw.RealIP` (`router.go:60`); `auth` additionally parses `X-Forwarded-For` itself (`auth/handler.go:77-90`).

## 8. Uploads

| Endpoint | Field | Parse limit | Content limit | Validation | Errors |
|----------|-------|-------------|---------------|------------|--------|
| `POST /users/import[?commit=true]` | `file` (CSV) | `upload.ParseImportMultipart` (same cap as questions import; over-cap body gives 413 `FILE_TOO_LARGE`) | 10 MiB `upload.MaxCSVBytes` (`upload/validate.go:16`); 500 rows | magic-byte check `upload.ValidateCSVFile` | `MISSING_FILE`, `FILE_TOO_LARGE` 413, `INVALID_FILE_TYPE` 415, `INVALID_CSV`, `TOO_MANY_ROWS` |
| `POST /questions/import[?dry_run=true]` | `file` (CSV, or JSON by `.json` extension) | `upload.ParseImportMultipart`: body capped at `upload.MaxImportBodyBytes` (10 MiB + 1 MiB multipart overhead) via `http.MaxBytesReader`, memory limited to 10 MiB; over-cap body gives 413 `ERR_FILE_TOO_LARGE` | 10 MiB, bounded read via `io.LimitReader` | same magic-byte check | `ERR_INVALID_BODY`, `ERR_FILE_TOO_LARGE` 413, `ERR_INVALID_FILE_TYPE` 415 |
| `PUT /tenant/config` (logo) | key inside the JSON update map (`tenant/handler.go:50`), not multipart | n/a | 2 MiB `upload.MaxLogoBytes` (`upload/validate.go:13`) | PNG/JPEG magic bytes, never the client Content-Type | `LOGO_TOO_LARGE` 413, `INVALID_LOGO_TYPE` 400 |

File type is always detected from magic bytes (`upload/validate.go`, `DetectMIME`).

## 9. File downloads

All download endpoints are ordinary GETs inside the protected group, so **Bearer is required**; a plain `<a href>` navigation cannot send it. The frontend must use `downloadFile` (fetch with Authorization, blob, object URL; `frontend/src/api/download.ts:46-75`). Responses set `Content-Disposition: attachment; filename="..."`:

- CSV (`text/csv; charset=utf-8`): `GET /audit/export` (`audit/handler.go:83`), `GET /questions/export` (`questions/import_export_handler.go:170`), `GET /admin/exams/{id}/results/export` and `GET /admin/users/{id}/record/export` (`reports/handler.go:169,190`).
- The two reports CSVs are built in memory and sent only after the export succeeded (`writeBufferedCSV`, `reports/handler.go:155-166`, PR #174, ISS-163). A failure returns the JSON envelope with 500 `INTERNAL_ERROR` (`reports/handler.go:188,213`) instead of an empty 200 file. The buffer has **no row/size cap** (FR-BB54 AC-8 asks for one; open gap G1 in `conformance/PR194-PR198-conformance-20261009.md`).
- CSV rules (PR #194, ISS-191, merged at 6f3f417; FR-BB54 AC-8):
  - Formula-injection guard: `api.CSVSafe` (`internal/api/csv.go:26`) prefixes `'` to a text cell that starts with `=`, `+`, `-`, `@`, TAB or CR; a cell that already starts with `'` plus one of those is also prefixed so `api.CSVUnsafe` (`csv.go:53`) is an exact inverse. Leading-space values and server-formatted numbers/timestamps/booleans are not touched (a negative score stays `-1`). No BOM.
  - Guarded exports and cells: results (`employee_name`, `department`; `reports/service.go:434-435`), user record (`exam_title`, `status`; `service.go:511,517`), audit (every column except `timestamp`; `audit/handler.go:119-125`), questions (every cell via `rowToCSV`; `questions/import_export_handler.go:466`, round-trip inverse at `:263`). There is no users CSV export (only the `POST /users/import` reader, `users/handler.go:307`) and the dashboard export is PDF, so no export is left unguarded. The questions JSON export is not CSV and is not guarded.
  - Results export rows: sessions with status `submitted`, `auto_submitted` or `grading_pending` (`reports/repository.go:783,813`; the same set as per-exam analytics). `graded` is not a `session_status` value, so FR-BB54 AC-8 wording is satisfied by `grading_pending`.
  - Errors: unknown exam id gives 404 `NOT_FOUND` (`reports/handler.go:183-186`, lookup `reports/service.go:342`, `repository.go:448`); the lookup is not tenant-scoped (`WHERE id = $1`). A non-UUID id is not special-cased in the handler (Postgres rejects it, so expect 500 `INTERNAL_ERROR`, not 404; open gap G2). Unknown user on the record export gives 404 `USER_NOT_FOUND` (`reports/handler.go:207-209`). Empty id gives 400 `ERR_INVALID_PARAM` (`handler.go:172,197`).
- PDF (`application/pdf`): `GET /portal/sessions/{id}/certificate` and `GET /admin/sessions/{id}/certificate` (`certificates/handler.go:111`), `GET /admin/dashboard/export` (`reports/handler.go:269`).
- Errors on a download endpoint are still the JSON envelope with the normal status, so the client must check `res.ok` before treating the body as a blob (`download.ts:56-62`).
- Public, unauthenticated reads: `GET /tenant/config`, `GET /tenant/logo` (stored image content type, `tenant/handler.go:88`), `GET /verify/{code}`.

## 10. Audit logging

- `audit.Writer.Write(ctx, r, action, entityType, entityID *string, metadata any)` (`internal/audit/writer.go`) inserts into `audit_log`. It never returns an error, no-ops on a nil writer, and drops the event with a warning when the tenant is missing. Actor comes from the context, IP from `r.RemoteAddr`.
- Convention: every state-changing handler writes one event after success, from the handler (not the service). Action names are dotted lower-case `<entity>.<verb>`; sub-entities add a segment: `category.create`, `exam.publish`, `exam.section.create`, `exam.rule.delete`, `exam.assign`, `question.status_transition`, `question.tag_add`, `question_translation.upsert`, `auth.login.success`, `auth.login.failure`, `auth.password_reset_requested` (e.g. `exams/handler.go:250,513,600,738`; `auth/handler.go:102-107`). `entity_type` is a snake_case noun; `entity_id` is the UUID or nil; `metadata` is a small JSON object (names, diffs; never secrets).
- Reads are generally not audited; exports are (`question.export`, `questions/import_export_handler.go:211`).
- Read endpoints: `GET /audit` and `GET /audit/export`, permission `audit:read`, filters as in section 5.

## 11. Messages and i18n

- Server `message` strings are English, human-readable and not localized (all `WriteError` calls above). They are for logs and as a fallback, not primary UI text.
- The `code` is the stable contract. The frontend branches on `error.code` and shows localized text through `react-i18next` keys (for example `frontend/src/components/auth/LoginForm.tsx:49-52` maps `ACCOUNT_LOCKED` and `INVALID_CREDENTIALS`; category and tag errors live under `categories.errors.*` and `tags.errors.*` in `frontend/src/locales/en.json:298,335`). Locale files: `frontend/src/locales/{en,kk,ru}.json`.
- Some validation messages embed dynamic text from `err.Error()` (`users/handler.go:137`, `ai/handler.go:49`); those are English and unlocalized.
- Download failures map to `download.failed` / `download.session_expired` (`download.ts:14-18`).

## 12. Checklist: when adding an endpoint

1. Register the route in `internal/router/router.go` in the right group: public (`GlobalLimiter`), auth (`AuthLimiter`), or protected. Register static sub-paths before `/{id}`.
2. Protected route: add `r.With(rbac.RequirePermission(rbacCache, "<resource>", "<read|write|manage>"))`, or comment explicitly that any authenticated user may call it. A new permission needs a new migration (never edit old ones) seeding it for the intended roles.
3. Path params that carry UUIDs: use a name from `api.UUIDPathParamNames`; if you introduce a new name, add it to that list (the router test fails otherwise).
4. Query UUIDs: `api.UUIDQuery` / `api.ValidateUUIDList` (422). Body UUIDs: validate and return 422 `VALIDATION_ERROR`.
5. Respond only through `api.WriteJSON` / `api.WriteError` with `{data,error}`. Lists: `{items, meta:{page,per_page,total}}`, default `per_page` 20, cap 100, never `items: null`.
6. Reuse a code from section 2 where one fits. A new code is SCREAMING_SNAKE, stable, and gets a locale key in all three locale files.
7. Scope every query by `tenant_id`; apply department scoping for `department_admin` where relevant.
8. Keep the handler thin: logic in the service, SQL in the repository, errors wrapped with context; a sentinel error maps to a status in the handler.
9. State change: call `writer.Write` with `<entity>.<verb>`.
10. Upload: magic-byte validation, bounded read, documented size cap. Download: set `Content-Type` and `Content-Disposition`, state that Bearer is required, use `downloadFile` on the frontend.
11. Tests: `service_test.go` and `handler_test.go` (plus a router test if you add a route group or path-param name); run `go test ./...`.
12. Add the endpoint to the owning FR-BB requirement doc.

## 13. Inconsistencies in the code (not fixed here)

- I-1 Resolved by PR #197: `VALIDATION_ERROR` is 422 everywhere (was 400 in `auth/recovery_*` and `ai/handler.go:49`). Questions and exams still use the separate code `ERR_VALIDATION` (422) for the same meaning, which falls under I-4.
- I-2 `INVALID_TOKEN` is 401 for a bad JWT (`auth/middleware.go`) and 400 for an invalid reset link (`auth/recovery_service.go:98`).
- I-3 `USER_NOT_FOUND` is 404 in `reports/handler.go:98,128` but 500 in `email/handler.go:33`; the users domain itself uses `NOT_FOUND`.
- I-4 Two code families coexist: `ERR_*` (categories, tags, questions, exam CRUD, certificates 500, reports params) and unprefixed (auth, users, departments, sessions, ai, tenant). Pairs for the same meaning: `ERR_NOT_FOUND` / `EXAM_NOT_FOUND` for the same exam resource (`exams/handler.go:264` vs `:823`), `ERR_INVALID_PARAM` / `INVALID_PARAM`, `ERR_INVALID_BODY` / `INVALID_BODY`, `ERR_INTERNAL` / `INTERNAL_ERROR`, `ERR_FORBIDDEN` / `FORBIDDEN`. `certificates/handler.go` uses both `INTERNAL_ERROR` (line 69) and `ERR_INTERNAL` (line 97).
- I-5 `EXAM_NOT_ACTIVE` is 409 on assign and 422 on session start.
- I-6 Bad ids/params outside the UUID helpers get 400 (`reports/handler.go:52` `ERR_INVALID_PARAM`, `ai/handler.go:84` `INVALID_PARAM`), while the query-UUID helper gives 422.
- I-7 Permission verbs: `write` (questions, exams, grading) versus `manage` (users, departments, tags, categories, tenant) with no documented rule; `exams:read` also gates the admin session result and certificate routes (`router.go:267,307`).
- I-8 Sort params: `sort`+`order` (exams, questions) versus `sort`+`dir` (My Results). Invalid sort values silently default instead of 422.
- I-9 Date filters: RFC 3339 `from`/`to` (invalid value silently ignored on audit) versus `date_from`/`date_to` `YYYY-MM-DD` (400 `INVALID_DATE` on grading).
- I-10 Import part Resolved by PR #197: both imports cap the body at 10 MiB + 1 MiB before parsing and answer 413 (`upload/validate.go:90-104`). **Still open:** invalid pagination input silently falls back; JSON request bodies other than the recovery endpoints have no size limit.
- I-11 (answer-save part Resolved by PR #197; `AnswerSaveLimiter` now honours `DISABLE_RATE_LIMIT`, `ratelimit/middleware.go:84-87`). **Still open:** the variable is read with `os.Getenv` in middleware (`ratelimit/middleware.go:18-21`), contrary to the typed-Config rule in `CLAUDE.md`.
- I-12 Envelope construction is duplicated (`auth`, `tenant`, `ratelimit`), and `exams`/`questions` hand-build envelopes with `map[string]any` instead of `api.WriteError`. The error object gains extra keys (`details`, `fields`) with no declared schema.
- I-13 Duplicate-name conflict codes differ: `DUPLICATE_NAME` (departments), `ERR_TAG_DUPLICATE` (tags), `DUPLICATE_EMAIL` (users).
- I-14 `Retry-After` is a fixed 60; `AI_RATE_LIMITED` returns 429 without `Retry-After`.
- I-15 Wrong-password status differs: 401 on login, 400 on change-password (`auth/service.go:252`).
- I-17 (new, from the PR #196 review) The blank-default-locale-option rule (section 2.7) is not applied by `PUT /questions/{id}/translations/{locale}` for the default locale (`questions/translation_service.go:122-153`), so blank option text can still be stored through that route.
- I-16 Exam/question deletion of a non-draft returns 409 `ERR_NOT_DRAFT`, while publish on a non-draft returns 409 `EXAM_NOT_DRAFT`: the same condition under two names.

## 14. Broken references to fix elsewhere (outside BA scope)

Checked with `ls docs/` and grep over `CLAUDE.md`, `README.md` and `.github/`.

| Reference | Where referenced | State | Proposed replacement |
|-----------|------------------|-------|----------------------|
| `docs/architecture-guide.md` | `CLAUDE.md:146`; `.github/agents/06-release-finalizer.agent.md:45`; `.github/agents/business-analyst.agent.md:59`; `.github/agents/functions/GIT_COMMIT.md:32`; `.github/agents/infrastructure-configuration.agent.md:94` | **Missing** (`docs/` holds only business-process-map.md, product-summary-en/ru.md, ui-forms-inventory.md, ui-workflows-inventory.md and subdirectories) | Point API questions to `docs/requirements/api-conventions.md`, topology to `docs/business-process-map.md` and `docs/requirements/DEC-001.Environments-production-class-demo-and-qa.md`; or create the guide. Remove `docs/architecture-guide.md` from the `git add` lines in the release agents (they stage a nonexistent path). |
| `docs/backend-development-guide.md` | `CLAUDE.md:147` | **Missing** | `.github/instructions/backend-conventions.instructions.md` (exists) plus `docs/requirements/api-conventions.md` |
| `docs/frontend-development-guide.md` | `CLAUDE.md:148` | **Missing** | `.github/instructions/frontend-conventions.instructions.md` (exists) |
| `docs/requirements/requirements-backlog.md`, `docs/requirements/README.md`, `corporate_exam_platform_roadmap.md` | `CLAUDE.md` Key Docs | Exist | none |
| `make migrate` | `CLAUDE.md:10,22,132`; `README.md:60`; `.github/agents/06-release-finalizer.agent.md:29`; `.github/agents/infrastructure-configuration.agent.md:65` | **Resolved by PR #198 (ISS-146, merged at 96bf413).** `backend/cmd/api/run.go:31-43` dispatches `api migrate` (applies pending migrations through `dbpkg.RunMigrations`, prints `migrations applied`, exit 0; exit 1 on failure; unknown args exit 2 with usage). `Makefile:14-15` runs `docker compose run --rm api migrate` (image `ENTRYPOINT ["/app/api"]`, `backend/Dockerfile:26`). The API still also applies migrations at startup (`main.go:92`), so the target is optional. Remaining nits: it prints `migrations applied` even when nothing changed (FR-BB12 AC-7 wants a no-change message, gap G3). | None required; the references above are now correct. The agent prompts may keep mandating `make migrate`. |
| CLAUDE.md Repository Layout (`internal/` domain list) | `CLAUDE.md` Repository Layout | Stale: packages also include `ai, api, categories, config, ctxkeys, db, departments, email, health, middleware, portal, ratelimit, rbac, router, schemaguard, tags, tenant, upload`; the certificates package is `certificates`, not `certs` | Update the layout block and note that shared HTTP helpers live in `internal/api` |
| "Never `os.Getenv()` outside the startup Config" | `CLAUDE.md` Non-Negotiable Conventions | Violated by `internal/ratelimit/middleware.go:18-21` (still open after PR #197) | Code fix (move to `config.Config`), outside BA scope |
| Frontend port 5173 | `CLAUDE.md` Repository Layout | Only the Vite dev server; the Docker stack serves the SPA on port 80 (`README.md` table) | Clarify in `CLAUDE.md`/`README.md` |
| Root `README.md` | n/a | No link to API documentation | Add a link to `docs/requirements/api-conventions.md` |

Recommendation: in the `CLAUDE.md` Key Docs table replace the three missing guide rows with `docs/requirements/api-conventions.md` (API), `.github/instructions/backend-conventions.instructions.md` and `.github/instructions/frontend-conventions.instructions.md`.
