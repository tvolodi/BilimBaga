# Rules twins

Twin files hold the **workflow, pipeline, convention and skill rules** of the original files, with all permission, block and restriction rules removed. The originals stay unchanged and remain authoritative for permissions (see the `*.settings.json` deny lists).

| Original | Twin |
|----------|------|
| `CLAUDE.md` | `CLAUDE2.md` |
| `swarm/PROTOCOL.md` | `PROTOCOL2.md` |
| `swarm/roles/_common.md` | `_common2.md` |
| `swarm/roles/ba.md` | `ba2.md` |
| `swarm/roles/dev1.md` | `dev1_2.md` |
| `swarm/roles/dev2.md` | `dev2_2.md` |
| `swarm/roles/infra.md` | `infra2.md` |
| `swarm/roles/supervisor.md` | `supervisor2.md` |
| `swarm/roles/uat.md` | `uat2.md` |

Left out on purpose (kept only in the originals): deny/allow rules, write scopes, frozen-environment rules, "never edit X", reserved-for-user actions, force-push and install bans, lock ownership limits.
