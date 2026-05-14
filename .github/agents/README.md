---
description: BilimBaga Agent Workflow System — 11 agents with central Orchestrator
---

# BilimBaga Agent Workflow System

> **11-agent system** with a central Orchestrator driving focused subagents.
> Orchestrator manages pipeline sequencing, retry limits, and model selection.
> Subagents each do ONE job — no handoff logic inside subagents.
> Compatible with GitHub Copilot agents and Claude Code.

---

## Architecture

```
Tier 1: Orchestrator (1 agent)
         Routes requests, sequences steps, manages validation revision loop, escalates on failure

Tier 2: Subagents (10 agents)
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
├── issue-resolution.agent.md              ← Root cause analysis + bug fix
├── infrastructure-configuration.agent.md  ← Env, Docker, migrations, CORS
├── 06-release-finalizer.agent.md          ← Git commit + inner report
├── explore.agent.md                        ← Read-only codebase exploration (subagent only)
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
