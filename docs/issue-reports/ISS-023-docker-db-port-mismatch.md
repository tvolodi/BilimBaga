---
id: ISS-023
title: make dev fails when DB_PORT is set to host-mapped 5442
status: resolved
severity: high
layer: config
module: auth
tags: [make dev, docker compose, DB_PORT, HOST_DB_PORT, lookup db, no such host]
created: 2026-05-25
resolved: 2026-05-25
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
`make dev` failed to bring up a usable stack. API startup log showed:

`startup error: open database: db.New connect: dial tcp: lookup db on 127.0.0.11:53: no such host`

The failure appeared after changing PostgreSQL port to `5442` to avoid host conflict.

## Root Cause
Root `.env` used by `docker-compose.yml` was configured with `DB_PORT=5442`, which is the host-side published port, not the internal container network port.

Inside Docker Compose network, the API must connect to service `db` on PostgreSQL internal port `5432`. Setting `DB_PORT` to host mapping value conflated internal and external networking concerns and broke API-to-DB startup behavior.

## Fix Applied
Updated root environment configuration to keep container-to-container DB connection on internal port `5432` and map host access separately via `HOST_DB_PORT=5442`.

Also updated `.env.example` comments to clarify the difference between `DB_PORT` (internal) and `HOST_DB_PORT` (host bind) to prevent recurrence.

## Files Changed
| File | Change |
|------|--------|
| `.env` | Set `DB_PORT=5432`; added `HOST_DB_PORT=5442`; clarified internal vs host port comments |
| `.env.example` | Added clarifying comments for `DB_PORT` and `HOST_DB_PORT` semantics |

## Regression Test
None added. This is a Docker configuration/environment wiring issue; verified via runtime stack startup and API health checks.

## Resolution Results
- Tests: manual runtime verification passed (`docker compose ps` healthy; `GET /api/v1/health` returned 200 with `db_ok=true`)
- Migration applied: no
- Build clean: yes (`make dev` stack startup reached healthy API state)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-25 | `make dev` startup failure after changing PostgreSQL port to 5442 | Corrected env semantics: internal `DB_PORT=5432`, host mapping via `HOST_DB_PORT=5442`; added env template clarifications |
