# Common swarm role rules: workflow

Read at every startup and after every compaction, together with `PROTOCOL.md`.

1. The user is absent: decide, act, report. Needs the user (production, external comms, credentials; not downloads or installs, see CLAUDE.md Owner authority 6) -> mark the issue `status:blocked` with the reason and continue.
2. No `/loop`. After the startup handshake and after each task, end your turn and wait for a message (task, ping or reply).
3. Incoming `<cross-session-message from="...">`: reply with `SendMessage` to that `from`, PROTOCOL message format.
4. Dispatch your own subagent pipelines via `Agent` using `.claude/commands/*.md`.
5. State lives in GitHub and files. After restart: `gh issue list --label swarm --state open --json number,title,labels`, find own `in-progress` issues, resume.
6. Every state change on an issue gets a short `gh issue comment` (what, evidence path, branch/PR).
7. Max 3 attempts per issue, then report `failed`.
8. Working files go in `docs/` subdirectories.
9. Conventional commits ending with the `Co-Authored-By:` line your own session's harness gives you. Do not copy a model name from this file or from another role.
10. Nothing to do -> end your turn and wait. Default work arrives from the Supervisor as a `task`. You file an issue yourself only for a defect or gap you met while doing a task, and its body says `found-in: #<n>`. An issue for coverage, a sweep or a clean-up with no `found-in` is closed by the Supervisor unread.
11. Startup handshake: send `bb-supervisor` `{"type":"pong","status":"idle","role":"<role>"}` with first line "<role> online and ready".
12. Run `/compact` when context grows large.
13. Checkpoint each step change (PROTOCOL, checkpoints). No per-tick heartbeat.
14. Never aim a load, performance or probe tool (lighthouse, k6, curl against a site) at a host other than the local stack or a target the issue names. No tool uploads project code, data or logs to a host outside the project. Downloading and installing what the project needs (packages, images, tools) needs no approval (CLAUDE.md Owner authority 6); `npx` only for a named package at a named version. Guard tests stub the tool on PATH and never call the real one.
15. Secrets. An agent never reads, prints, copies or moves the value of a key, token or password: no `cat .env`, no `docker compose config`, no `env` dump of a container, no second file that holds the value, under any name. A stack that needs the owner's `.env` takes it by path: a throwaway compose override, passed with `-f`, that gives the service `env_file: <absolute path to the main checkout's .env>`. `--env-file` alone only fills `${VAR}` in the compose file and puts nothing into the container. The override holds the path, never a value, and is not committed. That a secret is loaded is shown by behaviour (a 200, a startup line), never by its value.
16. Labels. Change `status:*` and `role:*` only with `swarm/bin/set-status.sh <issue> <status> [--role <dev|ba|uat|infra>] [--close] [--pr <n>]`. One edit replaces the other status and, with `--role`, the other role; `done` clears `role:*`; `--close` closes after a MERGED PR. A raw `gh issue edit` for a label leaves two statuses or a `role:*` on a closed issue (retro-011, #518).
17. Issue links. A PR body and a commit message name their issue as `Refs #<n>`. Never `Closes`, `Fixes` or `Resolves`: GitHub then closes the issue at the merge, before its close rule is checked and without `set-status.sh`.
18. Real-Postgres tests. A PR that adds or changes one runs it locally against a throwaway `postgres:16` container (never the UAT stack) and quotes its PASS lines in the PR body. A local run where the test skipped is not a run.
