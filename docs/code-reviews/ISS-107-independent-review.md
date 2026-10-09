# ISS-107 Independent Code Review (PR #112)

Reviewer: Code Reviewer subagent (read-only). Diff: `gh pr diff 112`, branch `swarm/107-freeze-test-env`. Requirement: DEC-001.

## Verdict: CHANGES REQUESTED (one blocking issue, the seed guard is bypassable)

## Blocking

B1. `scripts/seed-test-env.ts` guard is a raw regex over the unparsed string:
`/bilimbaga-test\.ai-dala\.com/i.test(BASE)`. The WHATWG URL parser used by `fetch` normalises
several encodings that the regex never sees. Verified with Node:

| E2E_API_URL | regex blocks? | hostname fetch will use |
|---|---|---|
| `https://BILIMBAGA-TEST.ai-dala.com`, trailing dot, `:443`, `user@`, `http://` | yes | protected host |
| `https://bilimbaga%2Dtest.ai-dala.com` | NO | bilimbaga-test.ai-dala.com |
| `https://bilimbaga-test.ai-dala%2Ecom` | NO | bilimbaga-test.ai-dala.com |
| `https://bilimbaga-test。ai-dala。com` (ideographic dots) | NO | bilimbaga-test.ai-dala.com |
| fullwidth letters (`ＢＩＬＩＭＢＡＧＡ-test...`) | NO | bilimbaga-test.ai-dala.com |

Also unguarded: any IP or DNS alias of the host (inherent to a name check). Fix: `new URL(BASE)`,
take `.hostname`, lowercase, strip trailing dots, and compare (exact or `endsWith('.ai-dala.com')`
suffix check on the suffix `bilimbaga-test.ai-dala.com`). Better, invert to an allowlist: permit only
`localhost`, `127.0.0.1`, `[::1]` and (once built) `bilimbaga-qa.ai-dala.com` unless `ALLOW_PROTECTED_HOST=1`
(also covers the IP/alias case for any remote host). Invalid URL should exit 1.

## Non-blocking

N1. The guard exists only in the seed script. Playwright `global-setup.ts` / `fixtures/seed.ts`
(which create users and log in) and the e2e specs accept any `E2E_API_URL` / `E2E_BASE_URL`, so a mis-set
env still writes to the demo. Consider the same hostname check in `global-setup.ts`.

N2. `swarm/roles/infra.settings.json` still allows `Bash(ssh *)` with no deny for the bilimbaga-test host
or `redeploy-test.sh`; the freeze is prose-only. The deny list has `*bilimbaga.ai-dala.com*` (not the -test host).
Settings files are user-edited, so flag to the user rather than change here.

N3. `swarm/roles/uat.md` requires scenarios to declare `Target: local | qa`, but no existing
`docs/uat-scenarios/*` has a `Target:` line yet (DEC-001 s7 follow-up). Not part of AC-1..AC-8; track it.

N4. `.env.example` / `deploy/` contain no `DISABLE_RATE_LIMIT`, no real SMTP creds; `JWT_SECRET` is the placeholder
(AC-8 holds for the committed files; no `docker-compose.qa.yml` exists yet).

## Check results

1. Swarm docs vs DEC-001: PROTOCOL s8, infra.md (scope, what you do, tick), supervisor.md, uat.md and README are
   consistent with DEC-001 s2/s4 and AC-1 (bilimbaga-test prod-class/user-only, bilimbaga-qa the swarm target,
   read-only health check as the only default, backup and rollback plan precondition on approved changes).
   Grep of `swarm/` and the repo on the PR head (excluding historical docs): remaining `bilimbaga-test` hits are
   prod-class statements, the `deploy/` ops artifacts (now headed PROTECTED), DEC-001 itself, and the unrelated
   `@bilimbaga-test.local` seed e-mail domain. No remaining "redeploy to bilimbaga-test" default in swarm. The
   infra tick still names `bilimbaga-test` read-only health check, which DEC-001 permits. PASS.
2. Seed guard: see B1. `E2E_API_URL` required: PASS. `ALLOW_PROTECTED_HOST=1` override: as specified.
3. Playwright/config: `E2E_BASE_URL` falls back to the previous literals (5173 live, 4173 default), trailing slash
   trimmed in setup/seed; `E2E_API_URL`/`BB_API_PORT` untouched and still resolved identically in specs, seed fixtures
   and `vite.config.ts` (#12 convention intact). Note `playwright.config.ts` uses `E2E_BASE_URL` also as the
   `webServer.url`, so setting it to a remote host makes Playwright wait for it but `npm run preview` still starts
   locally; harmless. PASS. Frontend `tsc` was not run (PR discloses).
4. PR body AC checklist: truthful. AC-1, AC-7, AC-8 ticked correctly (AC-7 grep re-verified: scenarios have no
   matches; roles only contain prod-class documentation). AC-2..AC-6 correctly left unticked as out of scope.
   Claim "seed guard bilimbaga-test URL exits 1" is true only for the plain form (see B1). The "no CORS" claim
   matches the repo (no CORS handling in the PR's context); not re-verified in Go source.

## Required to approve

Fix B1 (parse URL, normalise hostname, prefer allowlist) and re-run the guard with the cases in the table.
