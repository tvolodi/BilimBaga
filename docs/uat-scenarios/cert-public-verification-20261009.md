---
slug: cert-public-verification
title: "Public Certificate Verification Page — UAT Scenario"
feature: cert-public-verification (FR-BB48; depends on FR-BB43, FR-BB44, FR-BB13, FR-BB62, FR-BB64)
version: 1
created: 2026-10-09
author: Business Analyst
---

## Code that must be merged before running

- **PR #36** `feat(certificates): public /verify/:code page and PUBLIC_APP_URL (FR-BB48)` (branch swarm/25-cert-verify) — OPEN at authoring time. Without it `/verify/:code` does not exist and the QR code still encodes `API_BASE_URL`. Do not run until merged and the stack rebuilt.
- The stack must be started with `PUBLIC_APP_URL` set (Precondition 3).
- Issue #37 (Bearer-token downloads) is NOT required: the certificate PDF is obtained via the API with a Bearer token (S1 step 1), not via the UI button.

## Preconditions

1. Platform running at `http://localhost` (nginx port 80, API at `http://localhost/api/v1`). If `BB_API_PORT` is overridden, adjust API URLs only.
2. **Super Admin** `admin@bilimbaga.local` / `Admin1234!` (seeded by migration 029; use whatever password prior UAT runs set after the forced change).
3. API started with `PUBLIC_APP_URL=http://localhost` (SPA origin as seen by a scanner). Variable `{public_app_url}` = this value. If left at the default `http://localhost:5173`, expected URLs become `http://localhost:5173/verify/{cert_code}` and the Vite dev server must be running.
4. **Employee** `uat.employee@test.com` / `NewPass123!` with a submitted passing session on **"UAT Result Exam"** (`certificate_enabled=true`). If absent, run S0a, S0d and S3 of `result-and-certification-20260609.md` first. Record `{passing_session_id}`.
5. `{employee_token}` = JWT for uat.employee (`POST /api/v1/auth/login`).
6. Tenant branding configured: `app_name`, an uploaded logo, and a non-default `primary_color` (e.g. `#0A7D4F`). If not, set via `/admin/settings/branding` as admin. Record `{app_name}`, `{primary_color}`.
7. `{cert_code}` is recorded in S1 step 3.
8. Playwright (or browser tool) with a fresh context (no cookies, empty localStorage) for all "Anonymous" steps; network log capture enabled.

## Test data

| Name | Value |
|------|-------|
| `{cert_code}` | verification UUID of the certificate for "UAT Result Exam" |
| `{unknown_code}` | `00000000-0000-0000-0000-000000000000` |
| `{malformed_code}` | `not-a-uuid` |

---

## Scenario S1: QR code and printed URL use PUBLIC_APP_URL (AC-1)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Call API: `GET /api/v1/portal/sessions/{passing_session_id}/certificate` with `{employee_token}`; save body as `cert.pdf` | HTTP 200; `Content-Type: application/pdf`; body starts with `%PDF-` | |
| 2 | Tester | Extract text from `cert.pdf` (e.g. `pdftotext`) | Text extracted | |
| 3 | Tester | Read `{cert_code}` from Content-Disposition filename `certificate-{cert_code}.pdf` | UUID recorded | |
| 4 | Tester | Assert PDF text contains printed URL `{public_app_url}/verify/{cert_code}` | URL present; not based on port 8080 / `API_BASE_URL`; no doubled slash (`//verify`) | |
| 5 | Tester | Render the PDF page to an image and decode the QR code (e.g. `zbarimg`, jsQR) | QR payload decodes | |
| 6 | Tester | Assert decoded QR payload | Exactly `{public_app_url}/verify/{cert_code}` | |
| 7 | Tester | Request the certificate a second time | Same `{cert_code}` and same QR payload (reuse, FR-BB43 AC-10) | |

## Scenario S2: Page opens without login — valid certificate (AC-2, AC-3)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Anonymous | In a fresh context open the decoded QR URL `{public_app_url}/verify/{cert_code}` | Page renders; URL stays `/verify/{cert_code}`; no redirect to `/login` | |
| 2 | Anonymous | Observe loading | Skeleton may appear briefly, then resolves to a result | |
| 3 | Anonymous | Assert visible element with `role="status"` containing "Certificate is valid" (`verify.valid`) and a check icon | Valid status shown | |
| 4 | Anonymous | Assert visible: employee full name of uat.employee | Name matches | |
| 5 | Anonymous | Assert visible: exam title "UAT Result Exam" | Title matches | |
| 6 | Anonymous | Assert visible: score as percentage with exactly one decimal (e.g. "100.0%") | One-decimal percentage | |
| 7 | Anonymous | Assert visible: issue date formatted for the active locale (via `Intl.DateTimeFormat`), not a raw ISO string | Locale-formatted date | |
| 8 | Anonymous | Assert not visible: any `role="alert"`, admin/portal navigation, sidebar, logout control | None present | |
| 9 | Anonymous | Reload the page (F5) | Same valid result (nginx SPA fallback serves `/verify/*`) | |

## Scenario S3: Invalid and malformed codes (AC-4)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Anonymous | Open `/verify/{unknown_code}` | Page renders, no login redirect | |
| 2 | Anonymous | Assert visible element with `role="alert"` containing "Certificate not found or invalid" (`verify.invalid`) | Invalid state shown | |
| 3 | Anonymous | Assert not visible: employee name, exam title, score, issue date, valid status | No personal fields | |
| 4 | Anonymous | Assert page text has no "Error:", stack trace, JSON or HTTP status | Clean message only | |
| 5 | Anonymous | Open `/verify/{malformed_code}` | Same invalid state as steps 2-4 | |
| 6 | Tester | Call API (no auth): `GET /api/v1/verify/{unknown_code}` | HTTP 200; `data.valid=false` (never 404) | |

## Scenario S4: Verification temporarily unavailable and Retry (AC-5)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Playwright route intercept: abort `GET **/api/v1/verify/*` with a network error | Interception active | |
| 2 | Anonymous | Open `/verify/{cert_code}` | "Verification temporarily unavailable" (`verify.unavailable`) with a Retry button | |
| 3 | Anonymous | Assert NOT shown: "Certificate not found or invalid" | Unavailable is distinct from invalid | |
| 4 | Tester | Change intercept to respond HTTP 503 and reload | Same unavailable state with Retry | |
| 5 | Tester | Remove interception; Anonymous clicks Retry | A new `GET /api/v1/verify/{cert_code}` is issued (network log); page switches to valid state (S2 steps 3-7) without full reload | |
| 6 | Tester | Intercept to respond HTTP 200 `valid:false`; click Retry (re-enter unavailable state first if needed) | Invalid state, not unavailable | |

## Scenario S5: Branding and locale switcher (AC-6)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Anonymous | Open `/verify/{cert_code}`; observe network | `GET /api/v1/tenant/config` and `/api/v1/tenant/logo` return 200 with no `Authorization` header | |
| 2 | Anonymous | Assert visible: `{app_name}` and logo image (loaded, naturalWidth > 0) | Branding shown | |
| 3 | Anonymous | Assert computed accent colour (heading/status/button) matches `{primary_color}` | Primary colour applied | |
| 4 | Anonymous | Assert visible: LocaleSwitcher | Present | |
| 5 | Anonymous | Switch to `ru` | `verify.*` strings switch to Russian with no navigation/reload; date reformatted for ru | |
| 6 | Anonymous | Switch to `kk` | Strings switch to Kazakh; no raw i18n keys (e.g. `verify.valid`) visible | |
| 7 | Anonymous | Repeat locale switching on `/verify/{unknown_code}` | Invalid message localised in en, ru, kk | |
| 8 | Anonymous | Resize viewport to 375 px wide | Centred layout, no horizontal scroll | |

## Scenario S6: No auth, noindex, no refresh attempts (AC-7)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Anonymous | With network capture on, open `/verify/{cert_code}` | Page loads | |
| 2 | Tester | Inspect all fetch/XHR requests | None carries `Authorization`; no request to `/api/v1/auth/refresh` | |
| 3 | Tester | Evaluate `document.querySelector('meta[name="robots"]').content` while on the page | `noindex` | |
| 4 | Tester | Navigate in the SPA to `/login` | The robots noindex tag is removed (present only while verify page is mounted) | |
| 5 | Tester | Log in as uat.employee in the same context, then open `/verify/{cert_code}` | Valid state renders; no redirect to `/portal`; verify request still has no `Authorization` header | |
| 6 | Tester | Intercept `GET **/api/v1/verify/*` to return HTTP 401 | Unavailable state; no token refresh attempt; no redirect to `/login` | |

## Pass / Fail criteria

- PASS: every step matches expected outcome; AC-1 to AC-7 each covered by at least one passing scenario.
- FAIL (defect): redirect to `/login` on `/verify/*`; QR/printed URL not based on `PUBLIC_APP_URL`; unavailable reported as invalid; personal data in invalid state; Authorization header on verify call; raw i18n keys; missing noindex.
- ENV ISSUE (not a defect): PR #36 not merged or stack not rebuilt; `PUBLIC_APP_URL` not set per Precondition 3; no QR decoder available.

## Acceptance Criteria Coverage

| FR | AC# | Covered by |
|----|-----|------------|
| FR-BB48 | AC-1 | S1 |
| FR-BB48 | AC-2 | S2 steps 1, 9; S6 step 5 |
| FR-BB48 | AC-3 | S2 |
| FR-BB48 | AC-4 | S3 |
| FR-BB48 | AC-5 | S4 |
| FR-BB48 | AC-6 | S5 |
| FR-BB48 | AC-7 | S6 |
| FR-BB48 | AC-8 | Not UAT-applicable (unit tests; Test Runner) |

## Out of Scope

Certificate revocation/expiry, manual code-entry form, rate limiting, PDF layout (covered by `result-and-certification-20260609.md`).
