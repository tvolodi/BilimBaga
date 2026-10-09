# Code Review ISS-127 (frontend Docker build / target-guard)

Branch: swarm/127-docker-build. Scope: `git diff origin/main...HEAD` (merge-base 0312c96).

Result: PASS

Findings:
- [Medium] .github/workflows/docker-build.yml: third-party action pinned by tag (`actions/checkout@v4`), consistent with other workflows; permissions are least-privilege (contents: read). No action needed.
- [Medium] frontend/vite.config.ts: `server.fs.allow: ['..']` widens the dev server's file-serving root to the repo root (dev only, not in production build). Acceptable; scoping to `../scripts` would be tighter but vitest needs the root too.
- [Low] workflow `paths` filter omits `.dockerignore`/root-level files; low risk.

Verification:
- Fix addresses root cause: `tsconfig` `include` is `["src"]`, test no longer lives in `src`, so Docker context (frontend/) no longer needs ../scripts.
- scripts/lib/target-guard.test.ts imports `./target-guard` only; scripts/package.json is ESM.
- Ran `npx vitest run ../scripts/lib` from frontend: 32/32 pass.
- No secrets, no push/login in CI; build only. deploy/Dockerfile exists.
- No backend or API/i18n changes; frontend API checklist items not applicable.

Summary: Minimal, correct fix with a regression-guard CI workflow; no Critical or High findings.
