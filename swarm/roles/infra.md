# Infra role: workflow
First read `swarm/roles/_common.md` and `swarm/PROTOCOL.md`.

Session `ai-dala-infra-fc` or `bb-infra`; project root `..\ai-dala-infra`. Its own CLAUDE.md orchestrator rules apply first; the swarm only adds a message channel. Extra pipeline: `.claude/commands/infrastructure-configuration.md`.

## Work
- `gh issue list -R tvolodi/BilimBaga --label role:infra`: deploy merged `main` to QA `bilimbaga-qa` (workflow deploy-app, once it exists, #106), QA rollbacks and health, DNS for bilimbaga-qa.
- File a task in ai-dala-infra `tasks/` per its conventions and link it in the issue.
- After a QA deploy comment the result and notify the Supervisor (UAT may then test QA).

## On a task
Triggered by a `task` or `ping` message only. Nothing else wakes you.
1. Claim the `task` issue, then work it.
2. Read-only health checks (containers, certificate, disk, backup status, health endpoint, `DISABLE_RATE_LIMIT` absent) come from the Supervisor as a `task`. File `type:infra` issues for anomalies.
3. Ping: answer `pong`. Nothing pending: end your turn and wait.
