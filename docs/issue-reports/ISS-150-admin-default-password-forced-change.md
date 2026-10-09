---
id: ISS-150
title: Seeded super_admin keeps default password with force_password_change=false
status: resolved
severity: critical
layer: database
module: auth
tags: [force_password_change, 030_reset_admin_password, Admin1234!, BOOTSTRAP_ADMIN_PASSWORD]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-152]
regression_test: backend/internal/auth/bootstrap_test.go
---

## Symptom
admin@bilimbaga.local / Admin1234! logs in on public URLs without being forced to change the
password (found by infra on bilimbaga-qa). #152 is the P0 tracker (same root cause).

## Root Cause
Migration 030 reset the admin to the default bcrypt hash with `force_password_change = false`
(029's comment says true). Separately the backend never enforces the flag: login returns it
and only the SPA LoginPage redirects (follow-up #160).

## Fix Applied
- Migration 033 (new; 029/030 untouched): sets force_password_change=true only while
  password_hash is exactly the 029/030 default hash; idempotent; down restores 030 state while
  the hash is still the 030 default. password_changed_at not stamped.
- `auth.BootstrapAdmin` (backend/internal/auth/bootstrap.go) runs at API startup (main.go):
  admin missing -> no-op; admin not on default password (bcrypt verify) -> untouched;
  default + BOOTSTRAP_ADMIN_PASSWORD -> validated (ValidateComplexity, != default), bcrypt
  hashed with BCRYPT_COST, stored atomically (compare-and-set on old hash), force cleared,
  password_changed_at stamped; default + BOOTSTRAP_ADMIN_GENERATE=true -> random one-time
  password stored, force set, logged once; otherwise force_password_change set and a SECURITY
  warning logged each start. The env value is never logged or put in errors.
- Config fields BootstrapAdminPassword / BootstrapAdminGenerate; documented in both
  .env.example files and README.
- scripts/seed-test-env.ts tries E2E_ADMIN_PASS first.

## Files Changed
| File | Change |
|------|--------|
| backend/migrations/033_force_admin_password_change.{up,down}.sql | new |
| backend/internal/auth/bootstrap.go, bootstrap_test.go | new |
| backend/internal/config/config.go, config_test.go | new fields + test |
| backend/cmd/api/main.go | call BootstrapAdmin, warnings |
| .env.example, backend/.env.example, README.md | docs |
| scripts/seed-test-env.ts | E2E_ADMIN_PASS first |

## Regression Test
bootstrap_test.go (fake store: all branches, secret not leaked, generated password policy,
race, migration 033 targets the 029/030 hashes). Config test. schemaguard green.

## Resolution Results
- Go: vet, staticcheck clean; `go test -p 1 ./...` all pass. Migration NOT applied (no live
  Postgres here): label needs-live-db.
- Backend enforcement not added (needs token claim + frontend + e2e changes): follow-up #160.
- UAT/Infra: apply 033; set BOOTSTRAP_ADMIN_PASSWORD (or GENERATE) on public stacks; e2e
  global-setup logs in with E2E_ADMIN_PASS (default Admin1234!): on a stack with a bootstrap
  password export E2E_ADMIN_PASS to match. With the default and no enforcement the login still
  succeeds (flag only), so the live suite is unchanged. Existing deployments: admin keeps the
  old password until rotated; startup check forces change if still default.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
