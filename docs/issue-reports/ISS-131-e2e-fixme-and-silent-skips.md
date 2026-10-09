---
id: ISS-131
title: e2e specs still test.fixme / silently skip (password-reset, tab-switch, certificates)
status: resolved
severity: low
layer: frontend
module: e2e
tags: [test.fixme, test.skip, password-reset.spec.ts, tab-switch.spec.ts, certificates.spec.ts, createTestExam]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-102]
regression_test: frontend/e2e/tab-switch.spec.ts
---

## Symptom
GitHub #131: password-reset.spec.ts wholly test.fixme although FR-BB115 (#33/PR #104) is merged; tab-switch.spec.ts (5 tests) test.skip()ed when POST /portal/exams/{id}/sessions failed (shared "E2E Mixed Exam" attempts exhausted); certificates.spec.ts PDF test fixme (no certificate-enabled exam with passed session).

## Root Cause
Stale fixme markers; test depended on a shared seeded exam whose attempt counter is consumed by repeated runs, and converted the failure into a skip; seed `createExam` hard-coded `certificate_enabled: false`.

## Fix Applied
- seed.ts: createTestExam takes options (maxAttempts, certificateEnabled, passingScorePct, onTabSwitch, assignToUserId); new strict helpers getEmployeeApiToken, startEmployeeSession (throws), createPassedEmployeeSession (answers the Option A question and submits).
- tab-switch.spec.ts: per-run dedicated exam (max_attempts 500) assigned to the seed employee; session creation failure fails the test.
- certificates.spec.ts: fixme replaced by a real test: certified exam + passed session via API, employee context opens result page, downloads PDF, verification code from Content-Disposition is verified on /api/v1/verify/{code}.
- password-reset.spec.ts: un-fixmed and rewritten as stub-based UI-state tests (en forced, confirmed selectors); the happy-path reset is dropped as a duplicate of account-recovery.spec.ts, which already covers forgot -> Mailhog -> reset -> login.

## Files Changed
| File | Change |
|------|--------|
| frontend/e2e/fixtures/seed.ts | options on createTestExam; strict employee-session helpers |
| frontend/e2e/tab-switch.spec.ts | dedicated exam, fail instead of skip |
| frontend/e2e/certificates.spec.ts | un-fixme, real round trip |
| frontend/e2e/password-reset.spec.ts | un-fixme, trimmed to non-duplicate coverage |

## Regression Test
The specs themselves. NOT EXECUTED: no live stack available to the author; UAT must run them locally with E2E_API_URL set (never the frozen bilimbaga-test host).

## Resolution Results
- Tests: vitest 488 passed; tsc and eslint clean; playwright --list discovers the 4 password-reset, 5 tab-switch tests and the certificates PDF test
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
