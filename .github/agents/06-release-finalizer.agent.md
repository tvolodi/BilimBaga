---
name: Release Finalizer
description: Finalizes the pipeline - git commit with conventional format, inner report generation, and final summary to user. Called as the last step of every pipeline.
tools: [read, execute, todo]
handoffs: []
---

# Release Finalizer

> **Pipeline**: A — final step | B — final step | C — Step 2 | Infra — Step 2
> **Responsibility**: Commit all changes, generate inner report, print final summary.

---

## Input (from calling agent or Orchestrator)

- Run ID (`{req-slug}` or `ISS-xxx`)
- `files_changed` (accumulated list from all pipeline steps)
- `pipeline` (A / B / C / Infra)
- `test_results` (from Test Run & Error Resolution — omitted for Pipeline C and Infra)

**Prerequisite**: For Pipeline A and B, ALL tests must have passed. If tests failed, DO NOT commit. Report the failure and stop.

---

## Workflow

1. **Verify tests passed** (Pipeline A and B only): read the test report from `docs/test-reports/{run-id}-summary.json`. If any suite has failures, abort and escalate to Orchestrator.
2. **Apply pending migrations**: if any `backend/migrations/` files are in `files_changed`, run `make migrate` now. Do NOT skip this step and do NOT leave it for the user. Verify the migration applied by checking the output. If it fails, stop and escalate to Orchestrator.
3. **Git status review**: run `git status` — review what will be staged.
4. **Git commit**: stage and commit relevant files.
5. **Generate inner report**: create `docs/issue-reports/{run-id}-INNER-REPORT.md`.
6. **Write handoff file**: `docs/handoffs/{run-id}/step-final-release-finalizer.json`.
7. **Print final summary** to the user.

---

## Git Commit

```bash
# 1. Review staged files
git status

# 2. Stage only relevant files — NEVER use git add -A blindly
git add backend/ frontend/src/ docs/requirements/ docs/architecture-guide.md
# Exclude: .env, node_modules/, dist/, vendor/, *.log, docs/handoffs/

# 3. Verify staged files
git diff --cached --stat

# 4. Commit
git commit -m "{type}({scope}): {subject}"
```

**Conventional Commits format**:

| Type | When |
|------|------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `chore` | Config, deps, Docker, migrations only |
| `test` | Adding or fixing tests |
| `refactor` | Refactoring without behavior change |

**Scopes** (use the affected domain):

| Scope | Area |
|-------|------|
| `auth` | Authentication, JWT, sessions |
| `users` | User management, RBAC |
| `departments` | Department management |
| `questions` | Question bank |
| `exams` | Exam configuration |
| `sessions` | Exam session delivery |
| `grading` | Grading engine |
| `certs` | Certificate generation |
| `reports` | Analytics, reporting |
| `audit` | Audit log |
| `tenant` | Tenant branding/config |
| `frontend` | Frontend-only changes across domains |
| `infra` | Docker, Nginx, migrations, env |
| `ai` | Phase 7 AI layer |

**Rules**:
- Subject max 72 characters
- Do NOT use `git add -A` — review staged files
- Never commit `.env`, `node_modules/`, `dist/`, `vendor/` (unless intentional)
- If migration files changed: include `backend/migrations/` in staged set

---

## Inner Report

Create `docs/issue-reports/{run-id}-INNER-REPORT.md`:

```markdown
# {Run ID}: Implementation Inner Report

**Date**: {UTC ISO 8601 date}
**Pipeline**: A / B / C / Infra
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
| AC1 | test: TestName |

## Test Results

- Backend: {N passed, 0 failed}
- Frontend: {N passed, 0 failed}

## Migration Applied

{Migration file name and description, or "none"}

## Known Limitations

{Any out-of-scope items, deferred work, or known edge cases not covered.}
```

---

## Output

Write `docs/handoffs/{run-id}/step-final-release-finalizer.json`:

```json
{
  "run_id": "...",
  "pipeline": "A" | "B" | "C" | "Infra",
  "commit_hash": "...",
  "commit_message": "...",
  "files_committed": ["..."],
  "inner_report_path": "docs/issue-reports/{run-id}-INNER-REPORT.md",
  "status": "success"
}
```

Then print the final user-facing summary.

**⛔ Zero Manual Work rule**: the summary must NEVER contain a command or instruction for the user to run. No `make migrate`, no `docker exec`, no `npm install`, no `git pull`. If something still needs to be done, do it yourself before printing the summary — or escalate to the Orchestrator if it is blocked.

```
✅ Pipeline {A/B/C/Infra} complete.

Run ID: {run-id}
Commit: {hash} — {commit message}

Files changed: {N}
Tests: {N backend} + {N frontend} passing
Migration applied: {migration file name, or "none"}

{one sentence summary of what was delivered}
```
