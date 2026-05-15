Investigate and fix the reported bug or issue.

---

## Step 1 — Root Cause Analysis

1. Read any error messages, stack traces, or issue descriptions provided.
2. Search the codebase for the relevant code paths.
3. Identify the exact line(s) causing the bug.
4. Write a one-paragraph root cause summary before touching any code.

## Step 2 — Fix

1. Apply a minimal, targeted fix.
2. Do NOT refactor unrelated code as part of the fix.
3. Run `cd backend && go build ./...` — fix any compile errors introduced.

## Step 3 — Test

1. Write or update a test that would have caught this bug.
2. Run the full backend suite: `cd backend && go test ./...`
3. Fix any regressions before continuing.

## Step 4 — Report

Summarize:
- Root cause (one paragraph)
- Fix applied (files changed, what changed and why)
- Test added or updated
- Test results (all pass confirmation)
