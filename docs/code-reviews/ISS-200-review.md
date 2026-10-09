# Code Review: ISS-200 bundle budget (run swarm-200)

Result: PASS

Files: frontend/src/App.tsx, .github/workflows/tests.yml, docs/issue-reports/ISS-200-bundle-budget.md

## Findings
- [Medium] frontend/src/App.tsx — Suspense fallback is a full-page spinner for employee routes, so navigating between lazy employee pages inside PortalLayout may flash the spinner instead of keeping the layout chrome. Behaviour is functionally identical; consider a nested Suspense inside the layout Outlet later.
- [Medium] bundle margin only ~2 kB (168.0 vs 170); documented in the issue report. Locale JSON is the next split candidate.
- [Low] CI step uses `bash ../scripts/perf/bundle-check.sh`; works with the job's working-directory: frontend (script resolves repo root itself).

## Checks
- 8 pages converted to React.lazy with named-export adapters; LoginPage stays eager; layouts unchanged.
- lazy/Suspense imported; FullPageSpinner imported and used. Public forgot/reset routes and AuthedRoutes Routes are wrapped in Suspense, so every lazy element is under a boundary. PasswordChangeGuard stays outside (eager).
- No other source file imports the converted pages directly (grep), so no chunk-merging regressions.
- No routes, guards, strings or API calls changed; no secrets; no i18n additions needed (spinner is i18n-free).
- CI: build + offline budget script added after unit tests; script exits non-zero when over budget.

## AC Coverage
- FR-BB65 D2 (initial JS <= 170 kB gzip): covered (168.0 kB measured per report; not re-measured here).
- Identical behaviour: covered by code inspection; vitest 652/652 per report.

Summary: Minimal, correct lazy-loading change with CI enforcement; no Critical or High findings.
