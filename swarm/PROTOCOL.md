# Swarm Protocol

Normative rules for all swarm roles. Roles: `bb-supervisor`, `bb-dev1`, `bb-dev2`, `bb-ba`, `bb-uat`, `bb-infra`.

## 1. Session registry (addresses)

| Role | Session name (`claude -n`) | cwd | Settings |
|------|----------------------------|-----|----------|
| Supervisor | `bb-supervisor` | repo root | `swarm/roles/supervisor.settings.json` |
| Dev1 | `bb-dev1` | `.claude/worktrees/dev1` | `dev1.settings.json` |
| Dev2 | `bb-dev2` | `.claude/worktrees/dev2` | `dev2.settings.json` |
| BA | `bb-ba` | `.claude/worktrees/ba` | `ba.settings.json` |
| UAT | `bb-uat` | `.claude/worktrees/uat` (owns the live stack) | `uat.settings.json` |
| Infra | `ai-dala-infra-fc` (reused) or `bb-infra` | `..\ai-dala-infra` (reused session's cwd is a per-run dir such as `ai-dala-infra\runs\<run-id>`; project root is still `..\ai-dala-infra`) | `infra.settings.json` |

Addressing is by name via `SendMessage`. Names given with `-n` are kept as is, but sessions without one get a derived suffix (observed: `ai-dala-infra-fc`, `bilimbaga-e0`), so always resolve the exact name from `ListAgents` by prefix (`bb-dev1`...) before sending; `claude agents --json` gives the same list to scripts. Verify liveness with `ListAgents`. Reply to an incoming message by copying its `from` attribute into `to`.

## 2. Permission mode (single common mode)

All roles run with **`bypassPermissions`** (`--permission-mode bypassPermissions`), because the already-running Infra session is in bypass mode and a mismatch makes the receiver hold cross-session messages for approval (a stalled swarm). Safety is provided instead by per-role **deny rules** in `*.settings.json` (deny wins over everything, also in bypass mode) and by the rules below. If any session is found in another mode, the Supervisor reports it (`mode-mismatch`) and the launcher restarts it with the right flag.

**No permission laundering:** never ask a peer to do something your own settings deny or that you would not be allowed to do. If blocked, mark the issue `status:blocked`, comment why, and tell the Supervisor.

## 3. Message format

Transport: `SendMessage`. Line 1 is a self-contained sentence (the human preview). Then one JSON object.

```
Task for Dev1: fix ISS-123 export button (issue #57)
{"v":1,"type":"task","issue":57,"branch":"swarm/57-export-button","action":"implement","role":"dev","prio":"p1","ref":"gh issue view 57","deadline_min":90}
```

Types and fields:
- `task` (Supervisor -> worker): `issue, action (implement|fix|write-requirement|uat-run|e2e-sweep|deploy|retro-apply), branch, prio, deadline_min`
- `result` (worker -> Supervisor): `issue, branch, action, result (done|failed|blocked|needs-uat|pr-open), pr, detail, attempts`
- `question`/`escalate`: `issue, detail`
- `ping`/`pong`: `{"type":"ping"}` / `{"type":"pong","status":"idle|busy","issue":57}`
- `heartbeat` is NOT sent as messages; liveness comes from `status` fields in `swarm/state/workers.json` written by Supervisor from replies and from GitHub activity.

Rules: messages carry pointers (issue numbers, file paths), not content. Everything durable goes into the GitHub issue as a comment and/or `docs/handoffs/<run-id>/`. A message may be lost; the issue must still be enough to resume.

## 4. GitHub label state machine

Labels: `role:dev|ba|uat|infra`, `status:ready|in-progress|review|uat|blocked|done`, `type:bug|feature|infra|tech-debt|test`, `prio:p0|p1|p2`, plus `swarm`, and the optional marker `needs-live-db` (SQL-changing PR without a real-Postgres test; UAT verifies it first).

```
(new) -> status:ready + role:X     Supervisor/BA/UAT files; Supervisor assigns role
ready -> in-progress               worker starts (worker sets it, only if role label matches it)
in-progress -> review              dev opened PR, code review passed
review -> uat                      Supervisor checked conflicts, Release Finalizer merged; role flips to role:uat
uat -> done (issue closed)         UAT verified PASS
uat -> ready + role:dev            UAT FAIL (reopen count +1, comment with report path)
any -> blocked                     comment with reason; Supervisor resolves
```

Claiming: **the Supervisor assigns** (changes `role:*` and sends a `task`). A worker only takes issues that carry its role label, status `ready` (or its own `in-progress`), ordered by prio then number. Dev1/Dev2 are distinguished by an assignee-free comment `claimed-by: bb-devN`; Supervisor never gives one issue to two devs. Exactly one dev per issue.

## 5. Merge and lock rules

- **Worktree-only**: every role except the Supervisor works in its own git worktree (`.claude/worktrees/<role>`), on branches `swarm/<issue>-<slug>`, never on `main` directly. The repo-root checkout is read-only for everyone but the Supervisor: no `git switch`/`checkout`, commit, pull or file edits there (reading is fine).
- **Migration number lock**: before creating a migration, take `swarm/locks/migration.lock` (create with `set -o noclobber`; content = role, issue, UTC time). Pick `max(existing on origin/main)+1`, commit the migration on a pushed branch, and release the lock only after the branch is merged or abandoned. Stale lock (> 60 min) can be broken by Supervisor.
- **Stack lock**: only one session at a time may run `make dev`/docker on the host. `swarm/locks/stack.lock` is owned by `bb-uat` by default (it needs the stack for UAT/E2E). Devs needing a running stack for integration tests ask UAT via Supervisor, or use unit tests and `go test` with their own ports only if non-conflicting. Stack restarts to deploy a merged change are done by UAT before a test run (`git pull --ff-only` in the UAT worktree `.claude/worktrees/uat`, `make dev` rebuild there; the repo root stays read-only).
- Lock dir `swarm/locks/` is git-ignored and lives in the **main checkout** (`C:\Users\tvolo\dev\ai-dala\BilimBaga\swarm\locks`); worktrees use the absolute path.
- **Merge**: devs finish through the existing Release Finalizer pipeline but open a PR instead of pushing to main; Supervisor checks `gh pr view --json mergeable,mergeStateStatus` and conflicts with other open PRs, then tells the dev (or the dev itself, once told `merge-ok`) to merge with `gh pr merge --squash`. Before merge, bring the branch up to date with latest `origin/main`: for an already-pushed branch, force-push is denied, so the sanctioned path is `git merge origin/main` into the branch (not rebase) and a plain push; the squash-merge then hides the merge commit. Docs-only and swarm/state changes by Supervisor go via PR too.
- Never force-push shared branches, never rewrite history, never edit existing migrations.

## 6. Escalation

- A worker makes at most 3 attempts per issue (record `attempts` in its result and as an issue comment). After 3 failures: `result: failed`.
- Supervisor then: (1) reassigns to the other dev; (2) if that also fails, swaps role (e.g. BA clarifies requirement, or Infra checks environment); (3) if still failing, labels `status:blocked`, files a `type:tech-debt` investigation issue, and records it in the next retrospective.
- **The user is informed only through reports** (`docs/retrospectives/`, the final lines of Supervisor ticks, `swarm/state/escalations.json`); the swarm never waits for the user. Actions reserved for the user (see global CLAUDE.md: production, external communication, system installs) are skipped, logged as `blocked` with reason, and the swarm moves on to other work.

## 7. Anti-idle contract

Every worker must always have a next action: (a) assigned `task`; else (b) oldest `status:ready` issue with its role label; else (c) tell the Supervisor `idle` and, while waiting, run its **self-generated default work** from its role file. Every worker runs a `/loop` (dynamic, 5-10 min) that re-checks (b) so it never sits idle if a message was missed. The Supervisor's anti-idle engine guarantees the queue is never empty (see `roles/supervisor.md`).

## 8. Infra scope

**`bilimbaga-test.ai-dala.com` (hetzner-prod `bilimbaga-test`) is FROZEN: it is a customer demo and production-class (user decision, #107).** Per `docs/requirements/DEC-001.Environments-production-class-demo-and-qa.md`: no swarm role runs automated tests, UAT, seeds, load tests, `DISABLE_RATE_LIMIT=true`, destructive DB ops, account creation or ad-hoc config edits against it. Only the user (or Infra on explicit per-action user approval, with a verified off-volume DB backup and a stated rollback plan) may deploy to it; Infra's only default activity is a read-only health check (containers, certificate, disk, backup status, health endpoint). Swarm test traffic goes to local stacks (`make dev`) and to the QA instance `bilimbaga-qa.ai-dala.com` (proposed, NOT built yet; blocked on #106). Until it exists, Infra has no remote deploy target and builds QA only after the user confirms scope. UAT scenarios declare `Target: local | qa`. Scripts that write data (`scripts/seed-test-env.ts`) require an explicit `E2E_API_URL` and refuse the frozen host. Infra handles the QA Keycloak realm `bilimbaga` and the future QA instance through the ai-dala-infra repo workflows. Production always needs the user: Infra marks such issues `status:blocked` ("needs user") and moves on. Note: the ai-dala-infra `shared/agent-team.md` lists only the Letflow team; BilimBaga tasks follow its normal approval protocol unless the user extends that file (only the user edits it). The local `make dev` stack and migrations are handled by dev/UAT using `infra-configuration` pipeline inside this repo.

## 9. Durable state

- GitHub issues + labels = source of truth for work.
- `docs/handoffs/<run-id>/` = pipeline payloads.
- **Report numbering (retro-002)**: issue/review report files use `ISS-<github issue number>` (e.g. issue #88 -> `docs/issue-reports/ISS-088-<slug>.md`, `docs/code-reviews/ISS-088-review.md`). Never take "highest existing + 1": parallel sessions collide. This overrides the numbering step of `.claude/commands/issue-resolution.md` in the swarm.
- `swarm/state/*.json` (git-ignored runtime; examples committed): `workers.json`, `retro.json`, `escalations.json`.
- `docs/retrospectives/retro-NNN.md` = audits.

## 10. Resources

- **SQL-changing PRs**: a dev PR that adds or changes SQL must either include a test run against a real migrated Postgres, or be labelled `needs-live-db` (see `roles/uat.md`). The no-DB `internal/schemaguard` test (SQL column refs vs migrations, all backend packages) must stay green.
- Devs run tests with capped parallelism: `npx vitest run --maxWorkers=2` (frontend), `go test -p 2 ./...` (backend).
- UAT live runs (full stack + browsers) need >= 8 GB free memory; UAT checks before starting and defers the run if below.
- The Supervisor checks free memory each tick while UAT is running and reports low memory in the tick report.

## 11. Checkpoints and heartbeats

Purpose: a restarted session or another worker can continue exactly where a worker stopped, and the Supervisor can detect a stalled worker without waiting for issue activity.

- **File**: one per role, named after the role: `swarm/state/<role>.json` (`dev1`, `dev2`, `ba`, `uat`, `infra`, `supervisor`). `swarm/state/*.json` is git-ignored runtime and lives in the **main checkout** like `swarm/locks`; workers in worktrees MUST use the absolute path `C:\Users\tvolo\dev\ai-dala\BilimBaga\swarm\state\<role>.json` (Git Bash: `/c/Users/tvolo/dev/ai-dala/BilimBaga/swarm/state/<role>.json`). A worker writes only its own file.
- **When**: at every step change (issue picked up, branch created, each pipeline step, PR opened, blocked, done) and at least once per loop tick while busy. Idle workers write `status: idle` with `issue: null` each tick, so the heartbeat keeps running.
- **Schema** (all timestamps UTC ISO 8601), example in `swarm/state/checkpoint.example.json`:

```json
{"role":"dev2","status":"busy","issue":148,"branch":"swarm/148-checkpoints-heartbeats","step":"code-review","next_action":"fix review findings, then open PR","last_tick_utc":"2026-10-09T07:55:00Z","blockers":""}
```

  `status` is `busy`, `idle` or `blocked`; `issue` is the GitHub issue number or `null`; `blockers` is empty when none.
- **Atomic write**: write a temp file in the same directory, then rename it over the target, so readers never see half-written JSON. `swarm/bin/checkpoint.sh <role> <issue> <branch> <step> <next_action> [blockers]` does this (bash, `date`, `printf` only, no `jq`; `SWARM_STATUS=idle|blocked` overrides the default `busy`; `SWARM_STATE_DIR` overrides the target directory for tests only).
- **Issue mirror**: at each step change also post a short `gh issue comment` (step, next action, blockers, branch/PR). **GitHub issue comments stay the source of truth**; the file is a convenience. On restart, trust the issue comments if they disagree with the file.
- **Heartbeat**: `last_tick_utc` is the heartbeat. The Supervisor treats a `busy` worker whose heartbeat is older than `N = 45` minutes (configurable as `stale_heartbeat_min` in `swarm/state/retro.json`; 45 when absent) as stalled and applies the stall handling in `roles/supervisor.md` and section 6. A missing file for a live worker counts as stale after the same interval. `idle` and `blocked` workers are checked only for a very old heartbeat (> 2 x N), which indicates a dead session.
