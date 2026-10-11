# Owner brief: full redeploy of the demo (bilimbaga-test) from main

Prepared by BA, 2026-10-11, from issues and the repo only. No host was contacted. Reference: `main` at `c1d5e06`; pin the SHA when the window is set.

## Decision needed

Approve a window for one full redeploy of `bilimbaga-test` (`deploy/redeploy-test.sh`, run by you or by Infra on your direct order). Nothing else is pending: the edge vhost (PR 482) and the nginx healthcheck are already live on the demo (2026-10-11).

## Baseline

The last full deploy recorded in the issues is `90fee47` on 2026-10-09 (#74). The demo api runs image `7f908e81` (#396). Since then `main` has about 240 commits and migrations 031 to 038. Infra confirms the running ref read-only before the window; if a later deploy happened, only the baseline moves.

## What changes for users

| Area | Before | After the redeploy |
|------|--------|--------------------|
| Look | Light only | Light, dark or system (default follows the OS), switch in the header, no flash on load, charts and logo follow the theme |
| Brand colours | Primary `#0ea5e9` (2.8:1 on white, fails WCAG AA), accent `#f59e0b` | Migration 038 moves only untouched defaults to `#2E6DB4` and `#C8A84B`. A colour the tenant chose is kept. Saving a primary below 4.5:1 on white is refused (422). Saved colours are stored uppercase |
| Sign-in token | Access token also written to browser storage | Not written in production builds (#415). No visible change; at worst one re-login (JWT secret unchanged, not verified on the demo) |
| Client IP | Everyone behind nginx shared one address: the auth limit (10/min) was for the whole instance, audit IPs were wrong (#475) | Per-client limits, forwarded headers a client sends are ignored, `/health` out of the auth bucket, one helper for audit and log IPs |
| Healthchecks | nginx only | api container reports healthy too (#473); the nginx one is already live |
| AI features | Behaviour depends on the current env | No provider key means AI answers 503 `AI_UNAVAILABLE` and the boot log shows "AI is off". Your call per host: add a key (x.ai or Anthropic; data goes to that provider). Daily cap `AI_INSIGHTS_DAILY_LIMIT` (default applies), CA certificates now in the image |
| Time zone | `TENANT_TIMEZONE=Asia/Almaty` fails in the scratch image and falls back to UTC | Fixed by PR 523 (zone database embedded); email and reminder times use the tenant zone. Closes #522 once rebuilt |
| Also | | Role management, user reactivate, profile page with saved language, password recovery, exam overdue reminders (one per user and exam per 24 h), anti-cheat event hygiene, department-scoped admin views, JSON 413 on oversize uploads, many bug fixes |

## Things that may surprise you

1. If `admin@bilimbaga.local` still has the documented default password, migration 033 forces a password change at its next login.
2. Migration 035 adds a unique index on lower(email) only if no case-only duplicates exist; otherwise it logs a notice and skips. It never fails startup or deletes users.
3. Migrations 031 to 038 apply by themselves at api start. Migration 037 changes the primary key of `ai_insight_cache`, so rolling back the image alone is not verified; the dump is the real rollback.
4. Build time does not take the site down (images build before the recreate); downtime is the recreate plus migrations, expected under a minute.

## Conditions (DEC-001, project rule 5)

- Your direct order for this specific action; Infra states the risk first.
- Window outside an exam or customer session (none is scheduled in the repo; the repo does not hold your customer calendar) and outside the 02:00 UTC backup run.
- Verified DB dump off the container volume before starting, path recorded in #396. Daily backup now exists (#350) but take a manual dump anyway.
- Use `redeploy-test.sh` only; never a bare `docker compose up -d` on the demo checkout. Rollback: the tags the script prints plus the dump.

## After the deploy (read-only checks)

Health 200 with `db_ok`; api and nginx containers healthy; boot log has no "invalid TENANT_TIMEZONE"; tenant primary reads `#2E6DB4` unless customised; theme switch works at sign-in. Then Infra comments on #396, #473 and #522 so the Supervisor can close them.
