# ISS-107: Freeze bilimbaga-test (production-class); env-configurable URLs

Pipeline B (docs/config). Aligned with DEC-001.

## Changes
- `swarm/PROTOCOL.md` s8, `swarm/roles/infra.md`, `supervisor.md`, `uat.md`, `swarm/README.md`: bilimbaga-test frozen (user-only, Infra read-only health check by default, deploy only on per-action user approval + backup + rollback plan); swarm target = local stack, QA `bilimbaga-qa` (proposed, #106); scenarios declare `Target: local | qa`. Default "redeploy latest main to bilimbaga-test" removed.
- `scripts/seed-test-env.ts`: `E2E_API_URL` required (exit 1); refuses `bilimbaga-test.ai-dala.com` unless `ALLOW_PROTECTED_HOST=1`.
- Playwright: `E2E_BASE_URL` (playwright.config.ts, playwright.live.config.ts, e2e/global-setup.ts, e2e/fixtures/seed.ts); defaults stay local.
- Header comments (protected target) in `deploy/redeploy-test.sh`, `deploy/nginx/bilimbaga-test.conf`.
- `.env.example`, `README.md`: documented E2E_API_URL, E2E_BASE_URL, BB_API_PORT, ALLOW_PROTECTED_HOST.

## CORS finding
The backend sets no CORS headers (grep for cors/Origin in backend: none; browser traffic is same-origin via Vite proxy / nginx). There is no hardcoded origin, so `CORS_ALLOWED_ORIGINS` was NOT added (an unused setting would mislead); this is noted in `.env.example` and README. `PUBLIC_APP_URL` is already a typed Config field.

## Verification
- `go vet ./...`, `go test -p 2 ./...` in backend: pass (no Go changes).
- Seed guard: unset E2E_API_URL -> exit 1; bilimbaga-test host -> exit 1 (run with node strip-types; no network contact).
- Not run: frontend `tsc --noEmit` (sandbox refused creating the node_modules junction); edits are two constants and string replacements.
- Nothing deployed; no SSH or calls to the protected host.
