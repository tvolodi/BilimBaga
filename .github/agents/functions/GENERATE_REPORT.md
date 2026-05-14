# Function: GENERATE_REPORT

> Produce the final pipeline summary report for the user.

## Metadata

| Property | Value |
|----------|-------|
| **Category** | Reporting |
| **Used By** | Release Finalizer |
| **Depends On** | All previous pipeline steps complete |

---

## Purpose

Consolidate all pipeline step outputs into a single, readable summary that tells the user exactly what was done.

---

## Steps

1. Read handoff files from all pipeline steps in `docs/handoffs/{run-id}/`.
2. Read the inner report from `docs/issue-reports/{run-id}-INNER-REPORT.md`.
3. Compose the final summary.

---

## Summary Format

```
✅ Pipeline {A/B/C/Infra} complete — {Run ID}

Commit: {hash} on {branch}
  {type}({scope}): {subject}

─────────────────────────────────────────
WHAT WAS DELIVERED
─────────────────────────────────────────
{2-4 sentences describing what was built or fixed.}

─────────────────────────────────────────
FILES CHANGED ({N} files)
─────────────────────────────────────────
  backend/internal/{domain}/handler.go          modified
  backend/internal/{domain}/service.go          modified
  backend/internal/{domain}/repository.go       modified
  backend/migrations/{NNN}_{slug}.sql           created
  frontend/src/pages/{Page}.tsx                 created
  frontend/src/api/{domain}.ts                  modified
  frontend/src/locales/en.json                  modified
  docs/requirements/{slug}.md                   modified (status → implemented)

─────────────────────────────────────────
TESTS
─────────────────────────────────────────
  Backend:  {N} passed, 0 failed
  Frontend: {N} passed, 0 failed

─────────────────────────────────────────
ACCEPTANCE CRITERIA
─────────────────────────────────────────
  ✅ AC1: {text}
  ✅ AC2: {text}
  ✅ AC3: {text}

─────────────────────────────────────────
MIGRATION
─────────────────────────────────────────
  {NNN}_{slug}.sql — applied ✅
  (or "none")

─────────────────────────────────────────
KNOWN LIMITATIONS
─────────────────────────────────────────
  {Any deferred items or edge cases not covered. "None." if clean.}
```

---

## Failure Summary Format (when pipeline aborted)

```
❌ Pipeline {A/B/C} aborted — {Run ID}

Reason: {concise explanation}
Step failed: Step {N} — {Agent Name}

Unresolved findings:
  - [{Severity}] {file}: {issue}

Required user action:
  {What the user must decide or provide to unblock the pipeline}
```
