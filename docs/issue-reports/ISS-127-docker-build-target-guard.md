# ISS-127 frontend Docker build fails (TS2307 target-guard)

Root cause: frontend/src/test/target-guard.test.ts (PR #112) imported ../../../scripts/lib/target-guard, outside the frontend/ Docker context, so `tsc` in `npm run build` failed.

Fix: test moved to scripts/lib/target-guard.test.ts (relative import `./target-guard`); frontend/vite.config.ts `test.include` adds `../scripts/lib/**/*.test.ts` and `server.fs.allow: ['..']` so vitest can load it. `tsc` include is `src` only, so neither the test nor frontend/e2e (imports scripts/lib) is in the production build.

CI: .github/workflows/docker-build.yml builds frontend, api and deploy/Dockerfile images (no push, no secrets).

Verified: 32 guard tests pass via vitest (maxWorkers=2); tsc, lint, build pass; docker build of frontend (./frontend), api (./backend), deploy/Dockerfile (repo root) from `git archive HEAD` all succeeded; temp images removed.
