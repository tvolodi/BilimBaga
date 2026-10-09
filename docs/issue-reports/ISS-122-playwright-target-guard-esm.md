# ISS-122: Playwright cannot import scripts/lib/target-guard (0 tests since #112)

- Severity: high (blocks every Playwright run on main). Layer: config. Module: e2e.
- Symptom: `SyntaxError: Named export 'requireTarget' not found. The requested module '../../scripts/lib/target-guard' is a CommonJS module`; `playwright test --list` reports 0 tests.
- Root cause: `scripts/` has no package.json with `"type": "module"` and the repo root has none, so Playwright's loader treats scripts/lib/target-guard.ts as CommonJS while frontend/e2e imports it with ESM named imports.
- Fix: add `scripts/package.json` with `{"type": "module"}`.
- Verified: `E2E_API_URL=http://localhost:8080 npx playwright test --list --config=playwright.live.config.ts` lists 204 tests; `vitest src/test/target-guard.test.ts` 32 pass; `npx tsx scripts/seed-test-env.ts` still refuses without E2E_API_URL. No spec was run.
