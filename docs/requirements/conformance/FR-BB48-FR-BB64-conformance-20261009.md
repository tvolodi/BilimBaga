# AC Conformance Review: FR-BB48 and FR-BB64 (2026-10-09)

Mode: Business Analyst static code review (no live stack). Baseline: origin/main at d22a1ec. Nothing executed; test results are not re-run, only existence is checked.

## 1. FR-BB48 vs PR #36 (ac9c9ba)

| AC | Verdict | Evidence | Tests |
|----|---------|----------|-------|
| AC-1 PUBLIC_APP_URL, QR/printed URL | PASS | `backend/internal/config/config.go` (field `PublicAppURL`, default `http://localhost:5173`); `cmd/api/main.go:213` passes `cfg.PublicAppURL`; `certificates/pdf.go` `BuildVerifyURL` trims trailing `/`; `.env.example` documents it. `API_BASE_URL` no longer used for certificates. | `config_test.go` default and override; `pdf_test.go` `TestBuildVerifyURL`. See G1 about the "printed URL" wording. |
| AC-2 public route, no RequireAuth | PASS | `frontend/src/App.tsx` `AppRoutes` branches on `pathname.startsWith('/verify/')` and renders only `VerifyCertificatePage`, before `AuthedRoutes` (which owns `useRefreshToken` and the guards). | Test "renders the valid state ... no login redirect". |
| AC-3 valid state | PASS | `VerifyCertificatePage.tsx`: `role="status"` + CheckCircle2, employee, exam, `toFixed(1)%`, `Intl.DateTimeFormat(i18n.language, dateStyle long, UTC)`. | valid-state test. |
| AC-4 invalid state | PASS | `data && !data.valid` renders `role="alert"` with no personal fields. Raw errors never rendered. | invalid-state test. |
| AC-5 unavailable and retry | PASS | `api/verify.ts` throws on `!res.ok` and on network failure; page shows `verify.unavailable` + Retry (`refetch`), distinct from invalid. `retry:false`. | HTTP 500 + retry test and network-failure test. |
| AC-6 branding, LocaleSwitcher, i18n | PASS with minor gap (G2) | `useTenantConfig` (public fetch `/api/v1/tenant/config`), `TenantLogo` (`/api/v1/tenant/logo`), `LocaleSwitcher`, app_name shown. All 11 `verify.*` keys present in en/ru/kk (`locales/*.json` line ~1009). `primary_color` is applied by `TenantProvider` globally (it wraps `AppRoutes` in `App.tsx`), not by the page. | locale-switch test. Branding not asserted. |
| AC-7 no auth, noindex | PASS | `fetchVerify` is a plain `fetch` with no headers; `/verify/` bypasses `useRefreshToken`. `useEffect` adds and removes `<meta name="robots" content="noindex">`. Neither `TenantProvider` nor `useTenantConfig` sends credentials. | noindex + no-Authorization test (inspects all fetch calls). |
| AC-8 tests | PASS | Frontend: 7 tests (skeleton, valid, invalid, 500+retry, network, noindex/no-auth, locale). Backend: config default/override, BuildVerifyURL. | n/a |

Nginx fallback: `deploy/nginx.conf:20` `try_files $uri $uri/ /index.html` serves `/verify/*`; the spec's "verification test" for this is not present (G3). Public route `GET /api/v1/verify/{code}` is registered in `router/router.go:69`.

Verdict FR-BB48: 8/8 AC met in code; 3 minor gaps.

### FR-BB48 gaps
| ID | Gap | Severity |
|----|-----|----------|
| G1 | `GeneratePDF` test asserts the URL builder only, not that the PDF's QR/printed text uses it end to end (acceptable; builder is the single call site at `pdf.go`). Also the FR text and status in the requirement file were set to Implemented while AC checkboxes remain `[ ]`. | p3 |
| G2 | Tenant `primary_color` and logo rendering are not asserted by the page tests (AC-6). | p3 |
| G3 | No automated check that nginx serves `/index.html` for `/verify/<code>` (spec asked for a "verification test"). Covered by UAT scenario `docs/uat-scenarios/cert-public-verification-20261009.md` only. | p3 |
| G4 | NOT-VERIFIABLE-STATICALLY: real scan of a downloaded PDF QR, and CSP `connect-src 'self'` allowing logo/config calls in the built container. Needs the live UAT run. | p2 (verification pending, not a defect) |

No p1 gaps for FR-BB48. The verify endpoint exposes only name, exam, score and date for a possession-of-UUID code, per FR-BB43 design.

## 2. FR-BB64 remaining work vs origin/main

| Item | State on origin/main | Evidence |
|------|----------------------|----------|
| AC-1 rate limiting tests | MERGED (bbbb565, #13) | `backend/internal/ratelimit/middleware_test.go`: block after limit for auth (10) and global (300), Retry-After "60" and `RATE_LIMITED` envelope (lines ~77-90), disabled pass-through, answer-save per session, fallback to IP. |
| AC-2 HSTS | OPEN | `grep -i strict-transport|hsts` over `deploy/` returns nothing. `deploy/nginx.conf:5-8` has CSP, X-Frame-Options, nosniff, Referrer-Policy only. No commit touched it since the audit. |
| AC-3 question import validation | OPEN | `questions/import_export_handler.go:21` still `ParseMultipartForm(32<<20)`; `ValidateCSVFile` is called only in `users/handler.go:234`. No 10 MB cap or 413 for question import. |
| AC-7 CI workflow | OPEN | `.github/` has no `workflows/` directory; `Makefile:21` `security-check` target only. |
| Tests: temp password vs ValidateComplexity | PARTIAL | `users/service_test.go:409` `TestGenerateTempPassword` (10 iterations) checks length and character classes directly but does not call `auth.ValidateComplexity` or loop many times. Acceptable equivalent; weaker than the spec's "many iterations". |
| Tests: question import upload validation | OPEN | blocked by the AC-3 code change. |
| AC-4, AC-5, AC-6, AC-8 | Satisfied per validated delta | no regression found in the touched files. |

Open work is tracked by dev issue #19.

### FR-BB64 gaps and severity
| ID | Gap | Severity | Rationale |
|----|-----|----------|-----------|
| S1 | Question import has no magic-byte check and no 10 MB cap (only 32 MB multipart memory parse; body size otherwise unbounded by the handler) | p1 | Security/data-exposure class: authenticated upload endpoint without the required validation or size limit (resource exhaustion). Mitigation: it needs an authenticated admin or author role, so exploitability is limited; keep p1 only because the requirement classifies upload validation as a security AC and the fix is small. |
| S2 | No HSTS header | p2 | Behind a TLS-terminating proxy that can add HSTS itself; AC-2 is unmet as written. |
| S3 | No CI workflow running `security-check` | p2 | Process control; the checks exist locally. |
| S4 | Temp-password test does not call `ValidateComplexity` | p3 | Rule is satisfied by construction. |

## 3. Recommendations
1. Implement S1 first (small change): `http.MaxBytesReader` at 10 MB returning 413 `FILE_TOO_LARGE`, `upload.ValidateCSVFile` for `.csv`, plus handler tests (413, bad content, valid). Reuse the `users/handler.go:234` pattern.
2. S2: add the `map $http_x_forwarded_proto $hsts` plus `add_header Strict-Transport-Security $hsts always;` as the spec dictates; verify with `curl -H 'X-Forwarded-Proto: https'` in the nginx container.
3. S3: add `.github/workflows/security.yml` running `make security-check` on push and pull_request.
4. Run the pending FR-BB48 UAT scenario live to close G4; add a P3 nginx fallback check to the infra smoke test.
5. Tick the FR-BB48 AC checkboxes (all `[ ]` although Status is Implemented) and keep FR-BB64 at Validated until S1-S3 merge.

## Summary
FR-BB48: 8 PASS, 0 GAP at p1/p2, 3 p3 notes, 1 verification pending. FR-BB64: 1 test item merged (ratelimit), 3 open (AC-2, AC-3, AC-7), 1 partial test. p1 gaps: S1 only.
