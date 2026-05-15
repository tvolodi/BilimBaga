Finalize the current pipeline: commit all changes with Conventional Commits format, generate the inner report, and print the final summary.

Do NOT commit if any tests are failing. Do NOT leave any command for the user to run.

---

## Step 1 — Verify Tests Passed (Pipeline A and B only)

Check that the full backend and frontend test suites pass before proceeding:
```
cd backend && go test ./...
```
If any failures exist, stop and fix them before committing.

## Step 2 — Apply Pending Migrations

If any `backend/migrations/` files are in the change set and have not yet been applied:
- Try `make migrate` first.
- Fallback: `docker exec bilimbaga-db-1 psql -U bilimbaga -d bilimbaga < backend/migrations/{NNN}_{slug}.up.sql`
Verify the migration applied before proceeding.

## Step 3 — Git Commit

```bash
# Review what will be staged
git status
git diff --stat

# Stage relevant files explicitly — NEVER git add -A blindly
git add backend/ frontend/src/ docs/requirements/ docs/architecture-guide.md
# Include migrations if changed: backend/migrations/
# Exclude: .env, node_modules/, dist/, vendor/, docs/handoffs/

# Verify staged set
git diff --cached --stat

# Commit
git commit -m "{type}({scope}): {subject}"
```

**Conventional Commits types**: `feat`, `fix`, `docs`, `chore`, `test`, `refactor`

**Scopes**: `auth`, `users`, `departments`, `questions`, `exams`, `sessions`, `grading`, `certs`, `reports`, `audit`, `tenant`, `frontend`, `infra`, `ai`

Rules:
- Subject max 72 characters, imperative mood
- Never commit `.env`, `node_modules/`, `dist/`, `vendor/`

## Step 4 — Generate Inner Report

Create `docs/issue-reports/{run-id}-INNER-REPORT.md`:

```markdown
# {Run ID}: Implementation Inner Report

**Date**: {UTC ISO 8601 date}
**Pipeline**: A | B | C | Infra
**Commit**: {commit hash}

## Summary
{One paragraph: what was built or fixed and why.}

## Files Changed
| File | Action |
|------|--------|
| `...` | created / modified |

## Acceptance Criteria Verified
| AC | Verified By |
|----|-------------|
| AC-1 | test: TestName |

## Test Results
- Backend: {N passed, 0 failed}
- Frontend: {N passed, 0 failed}

## Migration Applied
{Migration file name, or "none"}

## Known Limitations
{Out-of-scope items, deferred work, known edge cases not covered.}
```

## Step 5 — Update Requirement Status

If this is Pipeline A (feature implementation):
1. Update the `Status` field in the requirement document to `implemented`.
2. Update `docs/requirements/README.md` — change status column to `implemented`.

## Step 6 — Print Final Summary

```
✅ Pipeline {A/B/C/Infra} complete.

Commit: {hash} — {commit message}
Files changed: {N}
Tests: {N backend} + {N frontend} passing
Migration applied: {file name, or "none"}

{one sentence summary of what was delivered}
```

**⛔ Zero Manual Work**: the summary must NEVER contain a command for the user to run.
