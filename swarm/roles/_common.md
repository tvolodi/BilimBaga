# Common swarm role rules: workflow

Read at every startup and after every compaction, together with `PROTOCOL.md`.

1. The user is absent: decide, act, report. Needs the user (production, external comms, installs, credentials) -> mark the issue `status:blocked` with the reason and continue.
2. Never idle: right after startup start a dynamic `/loop` (5-10 min) running the role's tick.
3. Incoming `<cross-session-message from="...">`: reply with `SendMessage` to that `from`, PROTOCOL message format.
4. Dispatch your own subagent pipelines via `Agent` using `.claude/commands/*.md`.
5. State lives in GitHub and files. After restart: `gh issue list --label swarm --state open --json number,title,labels`, find own `in-progress` issues, resume.
6. Every state change on an issue gets a short `gh issue comment` (what, evidence path, branch/PR).
7. Max 3 attempts per issue, then report `failed`.
8. Working files go in `docs/` subdirectories.
9. Conventional commits ending with `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.
10. Nothing to do -> role default work, then schedule the next tick.
11. Startup handshake: send `bb-supervisor` `{"type":"pong","status":"idle","role":"<role>"}` with first line "<role> online and ready".
12. Run `/compact` when context grows large.
13. Checkpoint each step change and each tick (PROTOCOL, checkpoints).
