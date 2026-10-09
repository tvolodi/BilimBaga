---
id: ISS-102
title: No dedicated e2e specs for employee record, certificates, tab-switch, AI assist, password reset, locale switcher
status: resolved
severity: low
layer: frontend
module: e2e
tags: [playwright, e2e, coverage, issue-16]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-059]
regression_test: frontend/e2e/*.spec.ts (see below)
---

## Symptom
docs/test-reports/e2e-static-audit-2026-10-09.md listed these flows with no dedicated Playwright spec (GitHub issue #16).

## Root Cause
Coverage gap, no spec authored.

## Fix Applied
Added six specs, registered in `playwright.live.config.ts` (admin project: employee-record, certificates, ai-assist, password-reset; employee project: tab-switch, locale-switcher). Selectors derived from components and locale files (ru/en regexes, roles, aria labels). Backend-dependent behavior that is non-deterministic or billable (AI generation, tab-switch policy, certificate payloads) is stubbed with `page.route`.

## Files Changed
| File | Change |
|------|--------|
| frontend/e2e/employee-record.spec.ts | 6 tests (FR-BB58, cert download AC-4) |
| frontend/e2e/certificates.spec.ts | 6 tests + 1 fixme (FR-BB43/44/48) |
| frontend/e2e/tab-switch.spec.ts | 5 tests (FR-BB38) |
| frontend/e2e/ai-assist.spec.ts | 4 tests (FR-BB71), AI + question-create stubbed |
| frontend/e2e/locale-switcher.spec.ts | 6 tests (FR-BB316) |
| frontend/e2e/password-reset.spec.ts | 4 tests, all fixme (FR-BB115 / #33 not merged) |
| frontend/playwright.live.config.ts | testMatch entries |

## Regression Test
The specs themselves. NOT executed live (stack owned by UAT); UAT must run them.

## Resolution Results
- Static only: tsc --noEmit clean, eslint clean, `playwright test --list` discovers all specs in both configs, e2e files type-check (only pre-existing seed.ts:393 error), vitest 66 files / 422 tests pass.
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
