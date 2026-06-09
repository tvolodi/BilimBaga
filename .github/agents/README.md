---
description: BilimBaga Agent Workflow System — 13 agents with central Orchestrator
---

# BilimBaga Agent Workflow System

> **13-agent system** with a central Orchestrator driving focused subagents.
> Orchestrator manages pipeline sequencing, retry limits, and model selection.
> Subagents each do ONE job — no handoff logic inside subagents.
> Compatible with GitHub Copilot agents and Claude Code.

---

## Architecture

```
Tier 1: Orchestrator (1 agent)
         Routes requests, sequences steps, manages validation revision loop, escalates on failure

Tier 2: Subagents (12 agents)
         Each does exactly one job; returns structured output to the Orchestrator
```

---

## Structure

```
.github/agents/
├── README.md                               ← This file
├── 00-orchestrator.agent.md                ← Central workflow controller
├── requirement-development.agent.md        ← Write FR-BBxxx requirement docs
├── requirement-validation.agent.md         ← Validate FR-BBxxx docs for completeness
├── requirement-implementation.agent.md     ← Implement feature end-to-end
├── 04-code-reviewer.agent.md               ← Review code: quality, security, AC coverage
├── 05-code-fixer.agent.md                  ← Fix Code Reviewer findings
├── test-run-error-resolution.agent.md      ← Run test suites, fix failures
├── issue-resolution.agent.md               ← Root cause analysis + bug fix
├── infrastructure-configuration.agent.md   ← Env, Docker, migrations, CORS
├── 06-release-finalizer.agent.md           ← Git commit + inner report
├── e2e-repair.agent.md                     ← Visual E2E walkthrough repair loop
├── explore.agent.md                        ← Read-only codebase exploration (subagent only)
├── business-analyst.agent.md               ← Business process definition, UAT scenarios, UAT decisions
├── uat-runner.agent.md                     ← Execute UAT scenario scripts against live GUI
└── functions/                              ← Reusable workflow atoms
    ├── ANALYZE_CONTEXT.md
    ├── DESIGN_SOLUTION.md
    ├── IMPLEMENT_FIX.md
    ├── GIT_COMMIT.md
    └── GENERATE_REPORT.md
```

---

## Pipelines

### Pipeline A: Feature Development

```
Step 1   Requirement Development    → docs/requirements/{slug}.md
Step 2   Requirement Validation     → PASS or FAIL+findings
         ↳ on FAIL (revision < 3):
Step 2a    Requirement Development  → revision mode; re-saves doc
           go back to Step 2 (increment revision counter)
         ↳ on FAIL (revision = 3):
           ESCALATE to user
Step 3   Requirement Implementation → code, code review, test, docs, release (internal loop)
```

**Trigger**: "Implement...", "Add feature...", "Create...", "Define feature...", "FR-BBxxx"

### Pipeline B: Bug Fix

```
Step 1  Issue Resolution → root cause analysis, fix, code review, test execution, release (internal)
```

**Trigger**: "Fix bug...", "Error when...", stack trace, broken behavior, ISS-xxx

### Pipeline C: Documentation

```
Step 1  Requirement Development → creates or edits documentation
Step 2  Release Finalizer       → git commit
```

**Trigger**: "Update docs...", "Add requirement doc...", "Documentation..."

### Pipeline Infra: Infrastructure Configuration

```
Step 1  Infrastructure Configuration → env, Docker, migrations, CORS, source code
Step 2  Release Finalizer            → git commit
```

**Trigger**: "Configure...", "Set up env...", "Change Docker...", "Run migration..."

### Pipeline E2E: Visual Walkthrough Repair

```
Step 1  E2E Repair Loop
        ├── npm run test:e2e:live  → parses e2e-results.json
        ├── Failing test → ISS-{NNN}-e2e-failure.md → Issue Resolution
        ├── After fixes: re-run suite
        └── Loop until all 22 tests pass (retry cap: 3 per test → escalate)
```

**Trigger**: "Run E2E tests", "test everything visually", "@e2e-repair"
**Prerequisite**: `make dev` must be running (DB + backend :8080 + Vite :5173)

---

### Pipeline BA: Business Process Definition

```
Step 1  Business Analyst (Mode A — Process Definition)
        → docs/requirements/{slug}-process.md
Step 2  Requirement Development
        → converts process description to FR-BBxxx requirement doc
Step 3  Continue as Pipeline A from Step 2 (Validation → Implementation)
```

**Trigger**: "Define business process for...", "Document workflow for...", "BA spec for..."

---

### Pipeline UAT: Business Process Verification

```
Step 1  Business Analyst (Mode B — UAT Scenario Authoring)
        → docs/uat-scenarios/{slug}-{date}.md
Step 2  UAT Runner
        → executes scenario against live GUI (hybrid Playwright + browser tool)
        → docs/uat-reports/{run-id}.md
Step 3  Business Analyst (Mode C — UAT Decision)
        ├── PASS       → Release Finalizer (marks requirement uat-verified)
        ├── DEFECT     → Issue Resolution → re-run UAT (back to Step 2)
        ├── REQ GAP    → Requirement Development → Pipeline A
        └── ENV ISSUE  → Infrastructure Configuration → re-run UAT (back to Step 2)
        Retry cap: 3 iterations per defect; escalate to user if unresolved
```

**Trigger**: "Verify business process", "UAT for...", "Test <feature> as a user", "BA check"
**Prerequisite**: `make dev` must be running

---

## Handoff File Convention

Every subagent writes its output to:

```
docs/handoffs/{run-id}/step-{NN}-{agent-slug}.json
```

- `run-id` = requirement slug (e.g. `auth-login`) or issue ID (e.g. `ISS-001`)
- Orchestrator passes file paths to downstream agents; agents read from disk, not prompt paste
- Prevents context overflow and ensures lossless information passing

---

## Requirements Numbering

Format: `FR-BB{phase}{section}` (two-digit phase + two-digit section)

Examples:
- `FR-BB11` = Phase 1.1 — Project scaffold
- `FR-BB14` = Phase 1.4 — Authentication
- `FR-BB35` = Phase 3.5 — Session creation
- `FR-BB41` = Phase 4.1 — Result retrieval

Custom / cross-cutting: `FR-BB00{N}` (e.g. `FR-BB001` = API response contract)

---

## Functions Library

Reusable workflow atoms in `functions/`:

| File | Purpose |
|------|---------|
| `ANALYZE_CONTEXT.md` | Determine scope and affected components |
| `DESIGN_SOLUTION.md` | Plan implementation before writing code |
| `IMPLEMENT_FIX.md` | Apply a targeted bug fix |
| `GIT_COMMIT.md` | Auto-commit with Conventional Commits format |
| `GENERATE_REPORT.md` | Produce final pipeline summary for user |

---

## Routing Quick Reference

| User says | Start with |
|-----------|------------|
| "Create requirement for..." | Pipeline A → Requirement Development |
| "Implement FR-BBxx..." | Pipeline A → Requirement Validation → Requirement Implementation |
| "Fix bug..." / stack trace | Pipeline B → Issue Resolution |
| "Update docs..." | Pipeline C → Requirement Development |
| "Configure..." / "Set up env..." | Pipeline Infra → Infrastructure Configuration |
| "Run E2E tests" / "test everything" / "@e2e-repair" | Pipeline E2E → E2E Repair Loop |
| "Define business process for..." / "BA spec for..." | Pipeline BA → Business Analyst (Mode A) |
| "Run UAT for..." / "Verify <feature> as a user" | Pipeline UAT → Business Analyst (Mode B) → UAT Runner → Business Analyst (Mode C) |
