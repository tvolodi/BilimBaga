Run the full backend and frontend test suites. Fix every failure. Confirm all tests pass before finishing.

---

## Step 1 — Run Backend Tests

```
cd backend && go test ./... -v
```

- List every PASS and FAIL result.
- For each FAIL: capture test name, error message, and stack trace.

## Step 2 — Run Frontend Tests (if frontend code was changed)

```
cd frontend && npm test -- --run
```

## Step 3 — Diagnose and Fix Failures

For each failure:
1. Read the test file and the production code it covers.
2. Determine the cause:
   - **Test code wrong** (stale fixture, wrong mock, wrong assertion) → fix the test
   - **Production code wrong** (bug) → fix the production code
3. Apply the fix. Do NOT delete tests, do NOT use `t.Skip()`, do NOT weaken assertions.

## Step 4 — Re-run to Confirm

After all fixes:
```
cd backend && go test ./...
cd frontend && npm test -- --run   (if applicable)
```

Confirm zero failures. If failures persist after 3 fix cycles, report them clearly and explain why they cannot be resolved.

## Step 5 — Report

Summarize:
- Total tests run
- Tests passed / failed
- Fixes applied (file, type: test|production, description)
- Any unresolved failures with reason
