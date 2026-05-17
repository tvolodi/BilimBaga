---
name: E2E Repair Loop
description: Runs the full visual E2E walkthrough test suite against the live stack, registers every failing test as an ISS-xxx issue, drives Issue Resolution for each, and loops until all 22 tests pass. Trigger with "run E2E tests", "test everything visually", or "e2e-repair".
tools: [read, search, edit, execute, agent, todo]
argument-hint: "Optional: specific test numbers to focus on, e.g. '04, 08, 11'"
agents: [issue-resolution, 06-release-finalizer]
handoffs:
  - label: Issue Resolution
    agent: issue-resolution
    prompt: E2E test failure registered. Investigate and fix the failing screen/component. Run ID and failure details are in the ISS-xxx doc.
    send: true
  - label: Return to Orchestrator
    agent: 00-orchestrator
    prompt: E2E Repair complete. All results and ISS IDs are in the inner report.
    send: true
---

# E2E Repair Loop Agent

> **Pipeline**: E2E — Step 1 (the only step; internally loops)
> **Responsibility**: Run the full 22-test visual walkthrough, diagnose failures, delegate fixes to Issue Resolution, repeat until all pass.

---

## Prerequisites

Before running, confirm the full dev stack is live:
- Frontend Vite dev server: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- PostgreSQL: running via Docker

If not running, spawn **Infrastructure Configuration** to execute `make dev` first.

---

## Step 1 — Run the E2E Suite

```bash
cd frontend && npm run test:e2e:live
```

The JSON reporter writes `e2e-results.json` to the repo root automatically.

Read `e2e-results.json`. For each test where `status !== "passed"`, extract:
- `title` — test name (e.g. "04 — Users list — displays table and filters")
- `error.message` — assertion or timeout message
- `error.stack` — full stack trace
- `attachments` — screenshot paths if present

If all tests pass: jump to Step 5.

---

## Step 2 — Register Issues

Find the highest existing `ISS-{NNN}` number in `docs/issue-reports/`. Assign sequential IDs starting from the next available number.

**Before creating a new ISS file**, search `docs/issue-reports/ISS-*.md` for the failing test title and error message keywords. If a matching resolved issue is found (same test, same error), reuse that file: increment its `recurrence_count`, set `status: recurring`, append to its `## Recurrence Log`, and pass that existing ISS ID to Issue Resolution instead of creating a new file.

For each failing test with **no matching prior issue**, write `docs/issue-reports/ISS-{NNN}-e2e-failure.md` using the [Issue File Format](issue-resolution.agent.md):

```markdown
---
id: ISS-NNN
title: "E2E: {test title}"
status: open
severity: high
layer: frontend
module: (infer from test title)
tags: [e2e, (keywords from error message)]
created: YYYY-MM-DD
resolved: null
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
Test: `{test title}` in `frontend/e2e/full-walkthrough.spec.ts`
Screenshot: {attachment path, or "none"}

```
{error.message}
```

```
{error.stack}
```

## Root Cause
(to be filled by Issue Resolution)

## Fix Applied
(to be filled by Issue Resolution)

## Files Changed
(to be filled by Issue Resolution)

## Regression Test
(to be filled by Issue Resolution)

## Resolution Results
(to be filled by Issue Resolution)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
```

Group failures with the same root cause under one ISS ID (note both test titles in the Symptom section).

---

## Step 3 — Spawn Issue Resolution Per Failure

For each registered ISS ID, spawn the **Issue Resolution** agent. Pass:
- Full content of `docs/issue-reports/ISS-{NNN}-e2e-failure.md`
- The exact failing test block from `frontend/e2e/full-walkthrough.spec.ts`
- Screenshot path (if present)
- ISS ID to use as the run ID

Wait for each Issue Resolution to complete and record its commit hash.

---

## Step 4 — Re-run Suite

After all Issue Resolution agents from this iteration complete:

```bash
cd frontend && npm run test:e2e:live
```

Re-read `e2e-results.json`:
- All tests pass → go to Step 5.
- Failures remain → go back to Step 2 with remaining failures (new ISS IDs for new failures; same ISS IDs if the same test still fails).

**Retry cap**: if the same test title fails in 3 consecutive iterations without improvement → escalate that test to the Orchestrator; continue repairing others.

Track: `iteration_count`, `failures_per_test_title` (map of title → attempt count).

---

## Step 5 — Final Report

Write `docs/issue-reports/E2E-REPAIR-{YYYYMMDD}-INNER-REPORT.md`:

```markdown
# E2E Repair Run — {UTC date}

## Result: ALL PASS | PARTIAL ({N} tests still failing)

## Iterations
| Iteration | Tests Run | Passed | Failed |
|-----------|-----------|--------|--------|
| 1 | 22 | N | N |

## Issues Resolved
| ISS ID | Test | Fix Summary | Commit |
|--------|------|-------------|--------|

## Escalated (if any)
| ISS ID | Test | Reason |
|--------|------|--------|

## Final Count
- Total: 22 | Passed: N | Failed: N (escalated)
```

---

## Return to Orchestrator

```
E2E Repair complete.

Iterations: N
Tests: 22 total, N passed, N failed (N escalated)
Issues resolved: [ISS-001, ISS-002, ...]
Issues escalated: [ISS-XXX, ...] (or "none")
Report: docs/issue-reports/E2E-REPAIR-{date}-INNER-REPORT.md
```

Include full failure messages for any escalated tests.
