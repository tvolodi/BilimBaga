# infra-seed-20260901: Implementation Inner Report

**Date**: 2026-09-01
**Pipeline**: Infra → B
**Commit**: (see next commit hash)

## Summary

This run resumed an interrupted infra task: migration 030 (resetting the locked-out
`admin@bilimbaga.local` account on the live test environment) had been written and
committed in a prior session but never actually deployed. This session deployed it to
`bilimbaga-test.ai-dala.com` via `redeploy-test.sh`, confirmed the schema version
advanced from 29 to 30, and re-ran `scripts/seed-test-env.ts`. That surfaced three
further, previously-hidden bugs in the seed script itself (masked until now by the
admin lockout): missing HTTP 429 retry handling, wrong question `type`/translation
field names sent to the questions API, and an invalid `likert_polarity` value that
violated a DB CHECK constraint. All three were fixed and verified live (ISS-050).
Also cleaned up abandoned, uncommitted work-in-progress found in the working tree: an
incorrect guessed-password workaround in README.md/seed-test-env.ts, and an
uncompilable scratch debug tool at `backend/cmd/genhash/`.

## Files Changed
| File | Action |
|------|--------|
| `scripts/seed-test-env.ts` | modified — added 429 retry/backoff helper; fixed question `type` wire values and answer-option translation JSON key; fixed Likert `likert_polarity` value |
| `docs/issue-reports/ISS-050-seed-script-blocked-live-test-env.md` | created |
| `docs/issue-reports/README.md` | modified — added ISS-050 row |
| `docs/handoffs/infra-seed-20260901/step-01a-pre-review.json` | created |
| `docs/handoffs/infra-seed-20260901/step-02-infrastructure-configuration.json` | created |
| `README.md` | reverted to committed state (guessed-password edit discarded) |
| `backend/cmd/genhash/` | deleted (uncompilable scratch debug tool) |

## Acceptance Criteria Verified
| AC | Verified By |
|----|-------------|
| Admin account unlocked on live test env | `schema_migrations` version 29→30, seed script logged in with initial password and normalised it |
| Seed script completes end-to-end | Two full live runs against bilimbaga-test.ai-dala.com, both exit code 0 |
| Seed script is idempotent | Second run reported all entities already-exist |

## Test Results
- Backend: all packages passing (`go build ./...` clean, `go test ./...` — all `ok`)
- Frontend: not touched, not re-run
- Live verification: `npx tsx scripts/seed-test-env.ts` run twice against the live test environment — both succeeded

## Migration Applied
`backend/migrations/030_reset_admin_password.up.sql` — already committed in a prior
session (b6fafe6); this run deployed it to the live test server for the first time.

## Known Limitations
- `scripts/` has no automated test harness in this repo; ISS-050's fix was verified via
  live end-to-end runs rather than an added unit/integration test.
- The questions API returns a generic `ERR_INTERNAL` (500) for any repository-layer
  failure (including DB CHECK constraint violations) without logging the underlying
  error server-side, which made root-causing the `likert_polarity` bug slower than
  necessary. Improving that error visibility is a candidate for a future backend issue
  but was out of scope here (fixing the caller was sufficient).
