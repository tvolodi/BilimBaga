# ISS-107: Freeze bilimbaga-test (production-class); env-configurable URLs

Pipeline B (docs/config). Aligned with DEC-001.

## Changes
- `swarm/PROTOCOL.md` s8, `swarm/roles/infra.md`, `supervisor.md`, `uat.md`, `swarm/README.md`: bilimbaga-test frozen (user-only, Infra read-only health check by default, deploy only on per-action user approval + backup + rollback plan); swarm target = local stack, QA `bilimbaga-qa` (proposed, #106); scenarios declare `Target: local | qa`. Default "redeploy latest main to bilimbaga-test" removed.
- `scripts/lib/target-guard.ts` (shared, pure): parses with `new URL`, decides on the normalised hostname by allowlist (localhost, *.localhost, 127.0.0.1, ::1, bilimbaga-qa.ai-dala.com; http/https only). Anything else, including bilimbaga-test, is refused unless `ALLOW_PROTECTED_HOST=1`; the protected host prints a loud warning even then. Closes the review bypasses (percent-escaped host, ideographic dots, fullwidth letters, userinfo, trailing dot).
- `scripts/seed-test-env.ts`: `E2E_API_URL` required (exit 1) and checked with the guard. `frontend/e2e/global-setup.ts` and `fixtures/seed.ts` check `E2E_BASE_URL` / `E2E_API_URL` with the same guard (local defaults unchanged).
- Test: `frontend/src/test/target-guard.test.ts` (vitest, 32 cases).
- Playwright: `E2E_BASE_URL` (playwright.config.ts, playwright.live.config.ts, e2e/global-setup.ts, e2e/fixtures/seed.ts); defaults stay local.
- Header comments (protected target) in `deploy/redeploy-test.sh`, `deploy/nginx/bilimbaga-test.conf`.
- `.env.example`, `README.md`: documented E2E_API_URL, E2E_BASE_URL, BB_API_PORT, ALLOW_PROTECTED_HOST.

## CORS finding
The backend sets no CORS headers (grep for cors/Origin in backend: none; browser traffic is same-origin via Vite proxy / nginx). There is no hardcoded origin, so `CORS_ALLOWED_ORIGINS` was NOT added (an unused setting would mislead); this is noted in `.env.example` and README. `PUBLIC_APP_URL` is already a typed Config field.

## Verification
- `go vet ./...`, `go test -p 2 ./...` in backend: pass (no Go changes).
- Seed guard via tsx: unset URL, percent-escaped bilimbaga-test host and example.com all exit 1 (no network contact).
- `npx vitest run src/test/target-guard.test.ts --maxWorkers=2`: 32 passed. `npx tsc --noEmit` in frontend: clean. (node_modules was a temporary copy, removed afterwards.)
- Nothing deployed; no SSH or calls to the protected host.
