# Code Review - PR #36 (FR-BB48 public certificate verification page)

Verdict: PASS / APPROVE (zero Critical, zero High)

## AC coverage
- AC-1 covered: Config.PublicAppURL (default http://localhost:5173), .env.example, main.go wiring, BuildVerifyURL trims trailing slashes, used for QR and printed URL; tests for default/override/trimming. Weak spot: no test that a generated PDF actually embeds the URL (AC-8 wording); BuildVerifyURL unit test covers the logic.
- AC-2 covered: AppRoutes renders /verify/* before AuthedRoutes (no useRefreshToken, no RequireAuth).
- AC-3, AC-4 covered (role=status / role=alert, one-decimal %, Intl.DateTimeFormat in UTC with active locale).
- AC-5 covered for network failure and non-2xx; see Medium 1.
- AC-6 covered: tenant name/logo/locale switcher, verify.* in en/ru/kk (check:i18n passes per handoff).
- AC-7 covered: plain fetch with no Authorization, no apiFetch (no refresh on 401), noindex meta added/removed.
- AC-8 covered: valid, invalid, skeleton, 500+retry, network failure, noindex/no-auth, locale switch.

## Security / data exposure
- Public endpoint returns only employee_name, exam_title, score_pct, issued_at (valid=false returns nothing else). No ids, emails, session ids. Matches spec; employee_name exposure is by design.
- No auth leakage: route outside auth bootstrap; no token sent. TenantProvider (public tenant config) still wraps the route, which is public-only.
- encodeURIComponent used on the code. No secrets.

## Conventions
- typed Config, no os.Getenv in handlers; handler thin; React Query used; i18n complete; shadcn components; no console.log.
- Raw fetch in api/verify.ts deviates from the apiFetch rule but is intentional and justified by AC-7 (same pattern as useTenantConfig).

## nginx
deploy/nginx.conf `location /` uses `try_files $uri $uri/ /index.html`, so /verify/<code> is served by the SPA fallback. /api/ proxied separately. CSP (connect-src 'self', img-src 'self') permits the page's calls and /api/v1/tenant/logo. No change needed. PUBLIC_APP_URL is delivered via env_file in compose; operators must set it in prod .env (default is localhost).

## Findings
- [Medium] backend handler.go HandleVerifyCertificate: internal/DB errors are returned as 200 valid=false, so the "temporarily unavailable" state (AC-5) can never trigger for backend faults; a DB outage would show "invalid certificate". Pre-existing FR-BB43 behavior; consider 5xx for internal errors in a follow-up.
- [Medium] Default PUBLIC_APP_URL=http://localhost:5173 silently produces localhost QR codes in prod if unset; consider a warning log or documenting it in the prod deploy docs.
- [Low] No PDF-level test asserting the QR/printed URL uses PublicAppURL.
- [Low] Handoff notes the self-review only; nginx and live QR scan unverified (nginx verified here by reading).
- [Low] AppRoutes uses pathname.startsWith('/verify/'), so a bare /verify (no code) falls to the authed catch-all; acceptable.
