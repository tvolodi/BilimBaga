# FR-BB48 — Public Certificate Verification Page

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB48 |
| Phase | 4 — Results & Certificates |
| Priority | 2 |
| Status | Implemented |
| Depends On | FR-BB43, FR-BB44, FR-BB13, FR-BB62, FR-BB64 |

## Description
Roadmap 4.3 requires that a third party (HR, auditor, regulator) can scan the QR code on a certificate and confirm its authenticity. The JSON endpoint `GET /api/v1/verify/:code` (FR-BB43) exists, but the end-to-end flow is broken: the QR code in the PDF encodes `{API_BASE_URL}/verify/{code}` (default `http://localhost:8080/verify/...`), a path no server route serves (the API route lives under `/api/v1`), and the SPA has no unauthenticated `/verify/:code` page, so a scan would land on a blank or login-redirected screen. This requirement closes the gap: a public, branded, localized SPA page at `/verify/:code` that calls the existing endpoint and shows valid or invalid, plus a dedicated `PUBLIC_APP_URL` setting so the QR code points at the SPA page.

## Acceptance Criteria
- [x] AC-1: A new env/config value `PUBLIC_APP_URL` (typed `Config.PublicAppURL`, default `http://localhost:5173`, documented in `backend/.env.example`) is the base used for the verification URL; `GeneratePDF` encodes `{PublicAppURL}/verify/{verification_code}` (no trailing slash duplication) in the QR code and printed URL. `API_BASE_URL` is no longer used for this purpose.
- [x] AC-2: Navigating, without any session, to `/verify/{code}` renders the verification page; the route is not wrapped in `RequireAuth`/`RequireRole`, and no redirect to `/login` occurs.
- [x] AC-3: For a code where `GET /api/v1/verify/:code` returns `valid: true`, the page shows a "Certificate is valid" status (with check icon and `role="status"`), the employee name, exam title, score as a percentage with one decimal, and the issue date formatted with `Intl.DateTimeFormat` in the active locale.
- [x] AC-4: For a code returning `valid: false` (unknown code or malformed UUID), the page shows a "Certificate not found or invalid" status (`role="alert"`) and none of the personal fields; the page never shows a raw error or stack trace.
- [x] AC-5: If the verify request fails at the network level or returns HTTP 5xx, the page shows a distinct "Verification temporarily unavailable" state with a Retry button that re-issues the request; this state is not reported as "invalid".
- [x] AC-6: The page shows the tenant branding (`app_name`, logo from `/api/v1/tenant/logo`, `primary_color`) from the public `GET /api/v1/tenant/config`, and includes the `LocaleSwitcher`; all strings exist in `en`, `ru` and `kk` locale files under `verify.*` and switching locale updates the page without reload.
- [x] AC-7: The page contains no admin/portal navigation, makes no authenticated API call (no `Authorization` header sent, no token refresh attempt on 401/403), and sets `<meta name="robots" content="noindex">` while mounted.
- [x] AC-8: Frontend tests (Vitest + RTL + mocked fetch) cover valid, invalid, loading skeleton, and error/retry states; backend tests cover the `PublicAppURL` config default/override and that the generated PDF verify URL uses it.

## Technical Specification

### Database Schema
No changes. Uses `certificates` (migration 019).

### API Contract
No new endpoints. Consumes existing public endpoints (no auth):

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| GET | `/api/v1/verify/:code` | public | 200 `{ "data": { "valid": true, "employee_name": "...", "exam_title": "...", "score_pct": 84.5, "issued_at": "2026-05-14T10:35:00Z" }, "error": null }`; unknown code 200 `{ "data": { "valid": false }, "error": null }` (never 404, per FR-BB43 AC-7). IDs are UUID v4; timestamps UTC ISO 8601. |
| GET | `/api/v1/tenant/config` | public | branding (FR-BB13) |
| GET | `/api/v1/tenant/logo` | public | logo bytes |

Nginx (`deploy/nginx.conf`, `deploy/nginx/*.conf`) already falls back to `/index.html` for `/verify/*`; CSP (`connect-src 'self'`) permits the calls. No change required beyond a verification test.

### Go Implementation Notes
- Package `config`: add `PublicAppURL string` loaded once in `Load()` from `PUBLIC_APP_URL`; no `os.Getenv` elsewhere.
- `cmd/api/main.go`: pass `cfg.PublicAppURL` to `certificates.NewHandler` in place of `cfg.APIBaseURL`.
- `certificates` handler: rename the `baseURL` field semantics to the public SPA base; strip trailing `/` before building the URL in `pdf.go`.
- `HandleVerifyCertificate` unchanged (thin handler -> service -> repository).

### Frontend Implementation Notes
- Route: `/verify/:code` registered in `App.tsx` outside every auth wrapper, before the `/` redirect. Eagerly or lazily loaded (lazy preferred; small standalone chunk).
- Files: `src/pages/VerifyCertificatePage.tsx`, `src/api/verify.ts` (hook `useVerifyCertificate(code)`, React Query key `['verify', code]`, `retry: false`, `staleTime: 5 * 60_000`, uses plain unauthenticated fetch, not the authenticated `apiFetch`).
- shadcn/ui `Card`, `Skeleton`, `Button`; Tailwind layout, centered, mobile-first (>= 375 px).
- i18n namespace `verify`: `verify.title`, `verify.valid`, `verify.invalid`, `verify.unavailable`, `verify.retry`, `verify.employee`, `verify.exam`, `verify.score`, `verify.issuedOn`, `verify.loading`, `verify.footer`.

## Notes
- Out of scope: certificate revocation, expiry dates, verification by certificate ID entry form, rate-limit changes (FR-BB43 notes already recommend 60 req/min/IP; global limiter applies), PDF layout changes.
- Existing certificates already issued keep their code; since the URL is derived at PDF generation time, re-downloading yields corrected QR codes.
