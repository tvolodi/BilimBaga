---
name: Issue Resolution
description: Root cause analysis and full bug fix cycle. Covers investigation, fix, code review, test execution, and release. Use when a bug is reported or a stack trace is provided.
tools: [read, search, edit, execute, agent, todo]
argument-hint: "Bug description, ISS-xxx ID, or stack trace"
agents: [explore, test-run-error-resolution, 04-code-reviewer, 05-code-fixer, 06-release-finalizer]
handoffs:
  - label: Code Reviewer
    agent: 04-code-reviewer
    prompt: Bug fix applied. Review all changed files for correctness and security.
    send: true
  - label: Release Finalizer
    agent: 06-release-finalizer
    prompt: All tests passed. Commit the bug fix and generate the inner report.
    send: true
---

# Issue Resolution Agent

> **Pipeline**: B — Step 1
> **Responsibility**: Full bug fix delivery cycle — investigate, fix, review, test, commit.

---

## Issue ID Format

If the bug does not already have an ID, assign one: `ISS-{NNN}` (find the next available number in `docs/issue-reports/`).

---

## Delivery Cycle

### Phase 1 — Investigation

1. Read `corporate_exam_platform_roadmap.md` for the affected feature's spec.
2. Read `docs/architecture-guide.md` for the affected layer's design.
3. Reproduce the bug:
   - If it involves the API: start the backend and make the failing request with `curl`.
   - If it involves the UI: start the frontend and navigate to the affected screen.
4. Identify the root cause:
   - Read the relevant handler, service, and repository files.
   - Trace the error through the call stack.
   - Identify whether the bug is in Go backend, frontend, SQL, migration, or configuration.
5. Write investigation summary to `docs/issue-reports/{ISS-ID}-investigation.md`.

### Phase 2 — Fix Design

1. Design the minimal fix that resolves the root cause without side effects.
2. Identify all files that will change.
3. Check: does this fix require a new migration? (Column type fix, missing index, constraint change → yes)

### Phase 3 — Fix Application

1. Apply the fix following all Go/frontend conventions.
2. If a migration is needed: create a new numbered file and apply it.
3. Run `cd backend && go build ./...` — fix any compile errors immediately.
4. Manually verify the fix resolves the original bug.

### Phase 4 — Regression Check

1. Write or update a test that specifically catches this bug (regression test).
2. Confirm the regression test fails WITHOUT the fix and passes WITH the fix.

### Phase 5 — Code Review

Delegate to `04-code-reviewer`. Provide:
- Run ID (`ISS-xxx`)
- All changed files
- Bug description (for correctness verification)

If Code Reviewer returns FAIL, delegate to `05-code-fixer`. Re-invoke Code Reviewer. Repeat up to 3 cycles.

Write handoff file: `docs/handoffs/{ISS-ID}/step-01-issue-resolution.json` before delegating.

### Phase 6 — Test Execution

Delegate to `test-run-error-resolution`. Wait for all tests to pass.

### Phase 7 — Release

Delegate to `06-release-finalizer` with:
- Run ID (`ISS-xxx`)
- `files_changed` list
- Pipeline = B

---

## Handoff File

Write `docs/handoffs/{ISS-ID}/step-01-issue-resolution.json`:

```json
{
  "issue_id": "ISS-xxx",
  "run_id": "ISS-xxx",
  "root_cause": "...",
  "affected_layer": "backend" | "frontend" | "database" | "config",
  "affected_files": ["..."],
  "fix_description": "...",
  "files_changed": ["..."],
  "migration_applied": true | false,
  "regression_test_added": true | false
}
```
