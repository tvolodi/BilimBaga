# Swarm Protocol

Normative rules for all swarm roles. Roles: `bb-supervisor`, `bb-dev1`, `bb-dev2`, `bb-ba`, `bb-uat`, `bb-infra`.

## 1. Session registry (addresses)

| Role | Session name (`claude -n`) | cwd | Settings |
|------|----------------------------|-----|----------|
| Supervisor | `bb-supervisor` | repo root | `swarm/roles/supervisor.settings.json` |
| Dev1 | `bb-dev1` | `.claude/worktrees/dev1` | `dev1.settings.json` |
| Dev2 | `bb-dev2` | `.claude/worktrees/dev2` | `dev2.settings.json` |
| BA | `bb-ba` | repo root | `ba.settings.json` |
| UAT | `bb-uat` | repo root (owns the live stack) | `uat.settings.json` |
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

Labels: `role:dev|ba|uat|infra`, `status:ready|in-progress|review|uat|blocked|done`, `type:bug|feature|infra|tech-debt|test`, `prio:p0|p1|p2`, plus `swarm`.

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

- Devs work in separate git worktrees on branches `swarm/<issue>-<slug>`, never on `main` directly.
- **Migration number lock**: before creating a migration, take `swarm/locks/migration.lock` (create with `set -o noclobber`; content = role, issue, UTC time). Pick `max(existing on origin/main)+1`, commit the migration on a pushed branch, and release the lock only after the branch is merged or abandoned. Stale lock (> 60 min) can be broken by Supervisor.
- **Stack lock**: only one session at a time may run `make dev`/docker on the host. `swarm/locks/stack.lock` is owned by `bb-uat` by default (it needs the stack for UAT/E2E). Devs needing a running stack for integration tests ask UAT via Supervisor, or use unit tests and `go test` with their own ports only if non-conflicting. Stack restarts to deploy a merged change are done by UAT before a test run (`git pull` in repo root, `make dev` rebuild).
- Lock dir `swarm/locks/` is git-ignored and lives in the **main checkout** (`C:\Users\tvolo\dev\ai-dala\BilimBaga\swarm\locks`); worktrees use the absolute path.
- **Merge**: devs finish through the existing Release Finalizer pipeline but open a PR instead of pushing to main; Supervisor checks `gh pr view --json mergeable,mergeStateStatus` and conflicts with other open PRs, then tells the dev (or the dev itself, once told `merge-ok`) to merge with `gh pr merge --squash`. Rebase onto latest `origin/main` before merge. Docs-only and swarm/state changes by Supervisor go via PR too.
- Never force-push shared branches, never rewrite history, never edit existing migrations.

## 6. Escalation

- A worker makes at most 3 attempts per issue (record `attempts` in its result and as an issue comment). After 3 failures: `result: failed`.
- Supervisor then: (1) reassigns to the other dev; (2) if that also fails, swaps role (e.g. BA clarifies requirement, or Infra checks environment); (3) if still failing, labels `status:blocked`, files a `type:tech-debt` investigation issue, and records it in the next retrospective.
- **The user is informed only through reports** (`docs/retrospectives/`, the final lines of Supervisor ticks, `swarm/state/escalations.json`); the swarm never waits for the user. Actions reserved for the user (see global CLAUDE.md: production, external communication, system installs) are skipped, logged as `blocked` with reason, and the swarm moves on to other work.

## 7. Anti-idle contract

Every worker must always have a next action: (a) assigned `task`; else (b) oldest `status:ready` issue with its role label; else (c) tell the Supervisor `idle` and, while waiting, run its **self-generated default work** from its role file. Every worker runs a `/loop` (dynamic, 5-10 min) that re-checks (b) so it never sits idle if a message was missed. The Supervisor's anti-idle engine guarantees the queue is never empty (see `roles/supervisor.md`).

## 8. Infra scope

Infra handles remote/test environments (hetzner-prod `bilimbaga-test`, QA Keycloak realm `bilimbaga`) through the ai-dala-infra repo workflows. Production always needs the user: Infra marks such issues `status:blocked` ("needs user") and moves on. Note: the ai-dala-infra `shared/agent-team.md` lists only the Letflow team; BilimBaga tasks follow its normal approval protocol unless the user extends that file (only the user edits it). The local `make dev` stack and migrations are handled by dev/UAT using `infra-configuration` pipeline inside this repo.

## 9. Durable state

- GitHub issues + labels = source of truth for work.
- `docs/handoffs/<run-id>/` = pipeline payloads.
- `swarm/state/*.json` (git-ignored runtime; examples committed): `workers.json`, `retro.json`, `escalations.json`.
- `docs/retrospectives/retro-NNN.md` = audits.
