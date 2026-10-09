# Code Review - ISS-031 (FR-BB65 perf scripts D1/D3)

Reviewer: Code Reviewer subagent. Scope: staged changes - `docs/test-reports/k6-load-test.js`, `scripts/perf/lighthouse.sh`, `scripts/perf/bundle-check.sh`, `docs/test-reports/perf/README.md` vs `docs/requirements/FR-BB65.Performance.md`.
Method: read in full; `bash -n` on both scripts; guard tested with stubbed `npx` (no Lighthouse run, no network); k6 guard functions extracted and exercised under plain `node` (no k6 run). No source edited.

## Verdict: FAIL (1 High)

## Findings

- [High] `scripts/perf/lighthouse.sh` (npx call, ~L97) - AC-3 correctness: the script cannot authenticate, so `/portal` and the exam-taking page will be measured as the login redirect. Both pages are behind the auth guard; the token is held in memory/refresh cookie (`frontend/src/api/auth.ts`; localStorage is only an E2E seed), and a fresh headless Chrome profile has neither. Lighthouse follows the redirect and scores the login page, which can pass at >= 90 while proving nothing about the portal or exam page. The script has no `--extra-headers`, cookie or user-data-dir hook, and does not check the final URL. -> Add an auth path (e.g. `LH_EXTRA_HEADERS` JSON file passed as `--extra-headers`, or a pre-seeded `--user-data-dir` through `LH_CHROME_FLAGS`), AND assert in the JSON that `finalDisplayedUrl`/`finalUrl` path equals the requested path (exit 3 otherwise). Document the exam-taking in-progress-session recipe in the README.

- [Medium] `scripts/perf/bundle-check.sh` - AC-7/E4 "no admin page module in the portal entry chunk" is not checked at all; only a count of chunks. -> Add a heuristic (assert separate chunk files exist for known admin pages by Vite chunk name, or grep the entry for admin route component names) or state in the README that this half of E4 is manual.
- [Medium] `bundle-check.sh` L53 - `lazy_chunks` counts `preload` chunks (vendor/query/ui from `manualChunks`) as "lazy", so the >= 10 minimum is inflated by up to the number of preloaded vendor chunks. -> Count only `role=lazy`, or print both.
- [Medium] `k6-load-test.js` L142-146/L178-179 - In `TEST_TOKEN` fallback mode five groups send no requests; their `http_req_duration{ep:*}` thresholds have no samples and pass vacuously, so k6 exits 0 looking like a full AC-2 pass (and the comment at L151 acknowledges exactly this risk). -> `console.warn` loudly in `setup()` in fallback mode, and/or tag the run so the summary cannot be mistaken (e.g. add `ep_count` check or a `fallback_only` threshold that fails unless `ALLOW_PARTIAL=1`).
- [Medium] Docker path (README L22, FR E2): `--network host` does not expose the Windows/macOS host's localhost on Docker Desktop, and the alternative `host.docker.internal` is correctly refused as non-local (verified), so on this Windows machine the documented k6 command may be unusable without `ALLOW_REMOTE=1`. -> Document the supported way (native k6 binary, or treat `host.docker.internal` as local explicitly in both guards), rather than leaving users to reach for `ALLOW_REMOTE`.
- [Low] `lighthouse.sh` L77 - label derivation uses `${rest#"$auth"}` after `auth` was stripped of userinfo, so for `http://user:pw@localhost/portal` the label becomes `user-pw-localhost-portal` (credentials in file names). Verified path logic by reading; use the pre-strip authority.
- [Low] `lighthouse.sh` - gate applies to desktop preset too (exit 1 below 90) though FR says mobile gates and desktop is only recorded. -> Exit 0 with a note for desktop, or document.
- [Low] `lighthouse.sh` - guard checks only the initial URL; Chrome follows redirects and loads third-party/remote assets and API calls. A local nginx redirect to the demo host would not be caught. -> After the run, compare `finalUrl` host with the guard (same fix as the High finding) and fail if it differs.
- [Low] `lighthouse.sh` - `lighthouse@11` pins the major only; `--yes` pulls the latest 11.x. FR says "pinned v11" so acceptable, but exact pin (11.7.1) gives reproducible scores.
- [Low] `bundle-check.sh` - budget uses KiB (170*1024) with gzip -9; nginx default gzip level is lower, so real transfer sizes are slightly larger. Also, a non-root Vite `base` would break the `s|^/||` entry matching (exits 2 with a clear message, fail-safe).
- [Low] Guards: `*.localhost` is treated as local (RFC 6761, fine for browsers, but the Go resolver used by k6 does not special-case it; a DNS-resolvable `x.localhost` could point off-box). `HTTP://` uppercase scheme is refused by the shell guard but accepted by the JS guard (harmless inconsistency). `127.300.1.1` passes the loose dotted-quad check (harmless). A newline inside the authority is not caught by the shell `[^!-~]` grep (line-wise), but the result stays consistent with URL parsers and the host logic still decides correctly (tested: `http://bilimbaga-test.ai-dala.com<LF>@localhost/` is treated as host localhost, same as WHATWG parsing).
- [Info] FR D1 says `K6_BASE_URL` default `http://localhost:8080`; the script deliberately has no default (safer; per task intent). FR text should be amended to match. The extended README states this correctly.

## Focus-area results

### 1. Remote-host safety (verified by tests)
Shell guard (stubbed npx, rc 2 = refused, rc 3 = passed guard and reached the stubbed npx) and JS guard (node) agree on every case:
- Accepted: `localhost`, `localhost:5173`, `127.0.0.1`, `127.0.0.1.` , `LOCALHOST.`, `[::1]`, `[::1]:80`, `evil.com@localhost` (host is localhost, consistent with parsers), `user:pw@localhost`.
- Refused: `127.0.0.1.evil.com`, `127.x.evil.com`, `localhost.evil.com`, `evil.com`, `localhost@evil.com`, `%`-encoded and backslash tricks, `[::ffff:127.0.0.1]`, `127.1`, `0.0.0.0` (fail-closed), non-http schemes.
- `bilimbaga-test.ai-dala.com`, uppercase, trailing dot, subdomain, and userinfo-prefixed (`localhost@bilimbaga-test...`) are all refused; `ALLOW_REMOTE=1` does not override it (tested for the shell script). No `BASE_URL` default in k6 (missing -> throws). Userinfo, trailing dots, 127.x.evil.com and IPv6 bypasses: none found. Residual gap is redirect-following (Low above).
- Not executed: k6 and Lighthouse themselves; no host contacted.

### 2. Git Bash / Windows / `set -eu`
- `bash -n` clean on both. `cygpath -m` handling for npx/node paths is correct; `BASH_SOURCE` fine under bash; `${preset_arg[@]+...}` and `scores[@]` are `set -u` safe (runs >= 1); `[ ... ] && preset_arg=` does not trip `set -e`; `|| s=` handles node failure; `grep ... || true` protects the entry/preload pipelines; `tr -d ' \r'` handles CRLF from `wc`. Lower-median for even run counts is conservative and correct. Multi-output `--output-path` yields `<base>.report.json/.html` as the script expects.
- Remaining Windows risk: Chrome must be discoverable (no `CHROME_PATH` hint in README); minor.

### 3. Thresholds vs FR
- k6: six groups present with the exact FR paths and roles (employee for `portal_exams`, admin otherwise); per-group `p(95)<200`; `http_req_failed rate<0.01`; `checks rate>=0.99`; 100 VUs / 2m default; `setup()` login via `/api/v1/auth/login` reading `data.access_token` (matches `LoginResponse`); all six routes exist in `backend/internal/router/router.go`. Matches AC-2/D1 (apart from the intentional no-default BASE_URL).
- Lighthouse: min 90, median of N, mobile default, desktop option, v11 via npx, output dir `docs/test-reports/lighthouse/` - matches AC-3/D3 except the auth gap (High) and desktop gating (Low).
- bundle-check: 170 kB gzip initial JS, >= 10 chunks - match D2/E4 numerically; admin-not-in-entry not enforced (Medium).

## AC coverage (tooling only; evidence ACs are D4/UAT)
- AC-2: script covered; fallback-mode vacuous pass (Medium).
- AC-3: tooling NOT adequate until auth/final-URL handling is added (High).
- AC-7: partially covered (Medium).

## Required changes to reach PASS
1. lighthouse.sh: support authentication (extra headers or user-data-dir) and fail when the report's final URL path/host differs from the requested one; document the recipe.
Recommended before UAT: the three Medium items (E4 admin check or documented manual step, preload miscount, k6 fallback warning, Docker Desktop guidance).

## Cycle 2

Scope: `git diff origin/main...HEAD` plus uncommitted working tree (`lighthouse.sh`, FR-BB65 doc), `k6-load-test.js`, `bundle-check.sh`, `docs/test-reports/perf/README.md`. Checks: `bash -n` (both scripts) and `node --check` (k6 script) clean; guard tested by execution of refused URLs.

**Verdict: PASS** (no Critical/High findings; no code edited by the reviewer).

### Cycle 1 findings status
- High (Lighthouse auth / final URL): fixed. `LH_USER_DATA_DIR`, `LH_EXTRA_HEADERS_FILE`, final path assertion (exit 3), and now also a final-host assertion against the guarded host (closes the redirect-off-host Low).
- Medium k6 fallback vacuous pass: fixed (`TEST_TOKEN` requires `ALLOW_PARTIAL=1`; employee creds required for a full run).
- Medium preload counted as lazy: fixed (only `role=lazy` counted).
- Medium admin-not-in-entry: documented as manual in README (acceptable).
- Medium Docker Desktop: documented in README.
- Low credentials in label: fixed (`auth_raw`). Low desktop gating: fixed (exit 0 with note). FR D1 text amended to the no-default BASE_URL.

### Focus results
1. Remote safety: executed `lighthouse.sh` with `https://bilimbaga-test.ai-dala.com/portal`, `http://BILIMBAGA-TEST.ai-dala.com./x`, `http://localhost@bilimbaga-test.ai-dala.com/`, both with and without `ALLOW_REMOTE=1`: all rc=2 (refused). Remote non-demo hosts without `ALLOW_REMOTE` rc=2. k6 guard reviewed by reading: `BASE_URL` has no default (throws), demo host and subdomains refused before the `ALLOW_REMOTE` check, so it cannot be overridden; logic is equivalent to the shell guard.
2. Git Bash/Windows: `cygpath -m`, `set -u`-safe arrays, CRLF-safe `wc`, fine.
3. Thresholds: six groups with per-group `p(95)<200`, `http_req_failed rate<0.01`, `checks rate>=0.99`, 100 VUs/2m: equal to FR AC-2/D1. Lighthouse min 90, mobile gates, median of N: equal to AC-3.

### Findings
- [Medium] `docs/requirements/FR-BB65.Performance.md` (last line, new): states a measured "174.2 kB" initial-JS figure and "issue #200" while admitting `bundle-check.sh` was not re-run. The number has no evidence in the repo; an unverifiable measurement in a spec invites a false AC-7 baseline. -> Remove the number or replace it with evidence from an actual `bundle-check.sh` run in D4; verify the issue number.
- [Low] `docs/test-reports/perf/README.md` Lighthouse section: "desktop is recorded (use `LH_MIN_SCORE=0` to record only)" is stale; desktop now never fails the script.
- [Low] `lighthouse.sh`: uppercase `HTTP://` scheme refused while the k6 guard accepts it (harmless, fail-closed). `*.localhost` treated as local although it may resolve off-box via some resolvers (carried over).
- [Low] `lighthouse@11` is a major pin only (carried over).
- [Info] FR doc and `lighthouse.sh` changes are uncommitted in the working tree and must be committed for the branch to carry them.

### Reviewer process note (disclosure)
During guard testing the reviewer ran `lighthouse.sh` with `ALLOW_REMOTE=1` against `https://evil.com/portal` and `http://127.0.0.1.evil.com/` (intended to confirm they pass the guard). Those URLs legitimately pass the guard under `ALLOW_REMOTE=1`, so the script went on to invoke `npx lighthouse@11` (Chrome launched, remote hosts possibly contacted; rc=3). This violated the memory/no-network hold. The generated untracked `docs/test-reports/lighthouse/` output was deleted; no tracked file was affected. Some `node.exe` processes were still visible afterwards (not verified to be from this run).
