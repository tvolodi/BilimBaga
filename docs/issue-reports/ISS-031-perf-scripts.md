---
id: ISS-031
title: FR-BB65 D1/D3 perf tooling missing (single-endpoint k6, no lighthouse/bundle scripts)
status: resolved
severity: low
layer: config
module: reports
tags: [k6, lighthouse, bundle-check, FR-BB65]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
`docs/test-reports/k6-load-test.js` hit only `GET /api/v1/exams` against a hardcoded localhost; no Lighthouse or bundle-budget tooling.

## Root Cause
FR-BB65 AC-2/AC-3/AC-7 evidence tooling (D1, D3) was never written.

## Fix Applied
k6 script extended to six tagged groups with per-group p95<200, login in setup(), host guard (BASE_URL required, local-only, ALLOW_REMOTE=1 escape, demo host always refused). Added `scripts/perf/lighthouse.sh` and `scripts/perf/bundle-check.sh` (paths per FR-BB65 D3), usage in `docs/test-reports/perf/README.md`.

## Files Changed
| File | Change |
|------|--------|
| docs/test-reports/k6-load-test.js | six groups, thresholds, guard |
| scripts/perf/lighthouse.sh | new |
| scripts/perf/bundle-check.sh | new |
| docs/test-reports/perf/README.md | usage |

## Regression Test
None added (scripts). Verified: `bash -n`, host-guard matrix for lighthouse.sh with a stub npx, host-guard logic of the k6 script exercised under node, bundle-check run against a fresh `npm run build`. k6 and Lighthouse were NOT executed (k6 not installed; nothing run against any host).

## Resolution Results
- Tests: guard matrix passed; bundle-check ran: initial JS 174.2 kB gzip vs 170 budget -> OVER (real finding for UAT/D2), CSS 8.9 kB, 47 lazy chunks.
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
