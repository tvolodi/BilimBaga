# ISS-032: FR-BB65 performance evidence run (UAT)

Target: local stack, main `5a59bd1` (E3 re-run) and `935c746` (E1, E2, E4, 2026-10-11). Run by bb-uat, owner approval for #32 on the local stack recorded in the #32 comment of 2026-10-11.

| Item | Result | Evidence |
|------|--------|----------|
| E1 EXPLAIN ANALYZE, 7 queries | PASS: each plan uses its named index (tiny data, plan shape only) | `docs/test-reports/explain-analyze-FR-BB65.md` |
| E2 k6, 100 VUs, 2 min | PASS: 124,526 requests, 0.00% failed, worst group p95 68.6 ms (portal_exams) | `docs/test-reports/k6-summary-20261011.{json,txt}` |
| E4 bundle split | PASS: initial JS 148.2 kB gzip vs 170 kB budget, 69 lazy chunks, CSS 8.3 kB | #32 comment 2026-10-11 |
| E3 `/portal` mobile (gating), 3 runs | **FAIL**: 88 / 88 / 88, median 88 vs 90 (was 66 before #513 and #520). FCP 2.5 s, LCP 3.5 s | `docs/test-reports/lighthouse/portal-main5a59bd1-r{1,2,3}-mobile-run1.report.html`; baseline `portal-r1-mobile-run1.report.html` |
| E3 `/portal` desktop (recorded) | 98 | `portal-main5a59bd1-desktop-desktop-run1.report.html` |
| E3 exam-taking page mobile (gating), 3 runs | **FAIL**: 88 / 81 / 81, median 81. FCP 3.3 s, LCP 3.9 s | `exampage-main5a59bd1-mobile-r{1,2,3}-mobile-run1.report.html` |
| E3 exam-taking page desktop (recorded) | 99 | `exampage-main5a59bd1-desktop-r1-desktop-run1.report.html` |

## Remaining defects
- #511 `/portal` mobile 88 vs 90: unused JS 32 KiB, `/api/v1/portal/exams` not gzipped (6.7 KiB), render-blocking index CSS and theme-init.js. Set back to `role:dev status:ready` (reopen: 1).
- #600 exam-taking page mobile median 81: Google Fonts stylesheet reported render-blocking (1,040 ms) on this route, plus the same unused JS.

## Verdict
FR-BB65 AC-3 (Lighthouse mobile >= 90 on `/portal` and the exam page) is NOT met. AC for E1, E2, E4 met. #32 stays open until #511 and #600 pass UAT.

Method: Lighthouse 11 via `scripts/perf/lighthouse.sh`, Playwright Chromium 1228 (already installed), signed-in employee (fresh login per run) by refresh-token cookie header, one session started via `POST /api/v1/portal/exams/{id}/sessions` for the exam page.
