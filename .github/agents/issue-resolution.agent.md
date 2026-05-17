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
> **Knowledge base**: Every resolved issue is stored in `docs/issue-reports/ISS-{NNN}-{slug}.md`. Search it first — recurring bugs skip the investigation phase entirely.

---

## Issue ID Format

If the bug does not already have an ID, assign one: `ISS-{NNN}` (find the next available number in `docs/issue-reports/` by listing existing `ISS-*.md` files).

---

## Issue File Format

Every issue lives in **one file** for its lifetime. Recurrences are appended to the same file, not filed as new ones.

File path: `docs/issue-reports/ISS-{NNN}-{slug}.md`  
Slug: 3–5 words from the symptom, kebab-case (e.g. `jwt-token-not-refreshed`).

```markdown
---
id: ISS-NNN
title: One-line symptom description
status: open | resolved | recurring
severity: critical | high | medium | low
layer: backend | frontend | database | config
module: auth | users | exams | sessions | certificates | reports | audit | tenant
tags: [keyword1, keyword2]   # exact error tokens, function names, SQL keywords
created: YYYY-MM-DD
resolved: YYYY-MM-DD         # null if open
recurrence_count: 1
related_issues: []           # [ISS-XXX] if same root area
regression_test: null        # path/to/test_file once added
---

## Symptom
<!-- EXACT error message or observable behavior. This text is used for similarity search. -->

## Root Cause
<!-- Technical explanation: which function/query/component fails and why. -->

## Fix Applied
<!-- What was changed and why. Enough detail that a future agent can reapply it. -->

## Files Changed
| File | Change |
|------|--------|

## Regression Test
<!-- Path and description of the test that catches this bug. "None added" if skipped. -->

## Resolution Results
- Tests: N passed, M failed
- Migration applied: yes / no
- Build clean: yes / no

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
```

---

## Delivery Cycle

### Phase 0 — Similarity Search *(always first)*

Before assigning an issue ID or starting investigation:

1. Extract search keywords from the bug report: module name, error message fragments, file/function names.
2. Run `grep` across `docs/issue-reports/ISS-*.md` for those keywords (search both YAML frontmatter `tags` and the `## Symptom` body).
3. Read the top 3 candidate files. Evaluate whether the root cause matches the current bug.

**If a match is found (same root cause):**
- Update the existing file:
  - Increment `recurrence_count` in frontmatter.
  - Set `status: recurring`.
  - Append a row to the `## Recurrence Log`.
- **If `regression_test` is set**: run it first. If it fails, the previous fix regressed → full investigation required (treat as new bug, link as `related_issues`).
- **If `regression_test` passes but the bug is still observed**: the fix was incomplete → full investigation required, same file.
- **If the documented fix is still valid**: apply it directly, skip Phases 1–2, jump to Phase 3.

**If no match is found:**
- Assign the next `ISS-{NNN}` and create the new issue file with frontmatter + `## Symptom` filled in.
- Continue to Phase 1.

---

### Phase 1 — Investigation *(new issues only)*

1. Read `corporate_exam_platform_roadmap.md` for the affected feature's spec.
2. Read `docs/architecture-guide.md` for the affected layer's design.
3. Reproduce the bug:
   - If it involves the API: start the backend and make the failing request with `curl`.
   - If it involves the UI: start the frontend and navigate to the affected screen.
4. Identify the root cause:
   - Read the relevant handler, service, and repository files.
   - Trace the error through the call stack.
   - Identify whether the bug is in Go backend, frontend, SQL, migration, or configuration.
5. Fill in `## Root Cause` in the issue file.

### Phase 2 — Fix Design *(new issues only)*

1. Design the minimal fix that resolves the root cause without side effects.
2. Identify all files that will change.
3. Check: does this fix require a new migration? (Column type fix, missing index, constraint change → yes)

### Phase 3 — Fix Application

1. Apply the fix following all Go/frontend conventions.
2. If a migration is needed: create a new numbered file and apply it.
3. Run `cd backend && go build ./...` — fix any compile errors immediately.
4. Manually verify the fix resolves the original bug.
5. Update `## Fix Applied` and `## Files Changed` in the issue file.

### Phase 4 — Regression Check

1. Write or update a test that specifically catches this bug (regression test).
2. Confirm the regression test fails WITHOUT the fix and passes WITH the fix.
3. Update `regression_test` in the issue file frontmatter with the test path.
4. Fill in `## Regression Test` section.

### Phase 5 — Code Review

Delegate to `04-code-reviewer`. Provide:
- Run ID (`ISS-xxx`)
- All changed files
- Bug description (for correctness verification)

If Code Reviewer returns FAIL, delegate to `05-code-fixer`. Re-invoke Code Reviewer. Repeat up to 3 cycles.

Write handoff file: `docs/handoffs/{ISS-ID}/step-01-issue-resolution.json` before delegating.

### Phase 6 — Test Execution

Delegate to `test-run-error-resolution`. Wait for all tests to pass.  
After tests pass: fill in `## Resolution Results` in the issue file and set `status: resolved` + `resolved: YYYY-MM-DD` in frontmatter.

### Phase 7 — Release

Delegate to `06-release-finalizer` with:
- Run ID (`ISS-xxx`)
- `files_changed` list
- Pipeline = B

After release: update `docs/issue-reports/README.md` — add or update the row for this issue in the index table.

---

## Issue Index Maintenance

`docs/issue-reports/README.md` is the knowledge base index. After every resolved issue, ensure it contains an up-to-date row:

```markdown
| ISS-NNN | title | module | severity | status | recurrence_count | resolved |
```

If the file does not exist, create it with this header:

```markdown
# Issue Reports Index

| ID | Title | Module | Severity | Status | Recurrences | Resolved |
|----|-------|--------|----------|--------|-------------|----------|
```

---

## Handoff File

Write `docs/handoffs/{ISS-ID}/step-01-issue-resolution.json`:

```json
{
  "issue_id": "ISS-xxx",
  "run_id": "ISS-xxx",
  "is_recurring": false,
  "root_cause": "...",
  "affected_layer": "backend | frontend | database | config",
  "affected_files": ["..."],
  "fix_description": "...",
  "files_changed": ["..."],
  "migration_applied": true,
  "regression_test_added": true,
  "regression_test_path": "..."
}
```
