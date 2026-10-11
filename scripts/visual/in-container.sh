#!/usr/bin/env bash
# FR-BB320 AC-11: runs inside the pinned Playwright container. scripts/visual/run.sh starts it with VISUAL_MODE set.
set -euo pipefail
cd /work/frontend

npm ci --no-audit --no-fund
# The gallery route exists only in builds that set VITE_ENABLE_DESIGN_GALLERY (FR-BB320 AC-9); the page tests use the same
# build. VITE_E2E_TOKEN_SEED reads the access token that global setup saved, so each page context does not rotate the
# shared refresh cookie (lib/e2eTokenSeed.ts). Both flags are set for this test build only.
VITE_ENABLE_DESIGN_GALLERY=true VITE_E2E_TOKEN_SEED=true npm run build

# The API from the host build. It migrates the throwaway Postgres on start and serves on localhost:8080.
(cd /work/backend && exec ./bin/api) >/tmp/api.log 2>&1 &
# The API log goes to the run's output when the run ends, so a crash during the tests shows up in the evidence.
trap 'echo "--- api.log (last 80 lines) ---"; tail -n 80 /tmp/api.log || true' EXIT
healthy() {
  node -e "fetch('http://localhost:8080/api/v1/health').then((r) => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))"
}
for _ in $(seq 1 90); do
  healthy && break
  sleep 1
done
if ! healthy; then
  cat /tmp/api.log
  exit 1
fi

if [ "$VISUAL_MODE" = update ]; then
  npx playwright test -c playwright.visual.config.ts --update-snapshots
else
  npx playwright test -c playwright.visual.config.ts
fi
