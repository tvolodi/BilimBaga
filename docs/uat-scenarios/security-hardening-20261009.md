---
slug: security-hardening
title: "Security Hardening Remaining ACs (Import Validation, HSTS, CI, Temp Password) — UAT Scenario"
feature: security-hardening (FR-BB64 AC-2, AC-3, AC-7, AC-8; GitHub issue #19, parent #5)
version: 1
created: 2026-10-09
author: Business Analyst
---

## Code that must be merged before running

- **Issue #19** (FR-BB64 remaining): (a) `Strict-Transport-Security` in `deploy/nginx.conf` via `map $http_x_forwarded_proto $hsts`; (b) `backend/internal/questions/import_export_handler.go` Import uses `internal/upload` validation (magic bytes, 10 MB limit, 413); (c) `.github/workflows/security.yml` running `make security-check`; (d) tests for ratelimit and temp-password complexity. Status ready, no PR at authoring time.
- Rebuild/restart nginx (config change) and the API.
- Expected pre-fix baseline (FR-BB64 Implementation Delta):
  - S2: Import has only `ParseMultipartForm(32<<20)` and picks CSV vs JSON by filename suffix; no content check and no 10 MB limit. Steps 3-5, 10 FAIL.
  - S1: no HSTS in any case (step 2 FAILS).
  - S3: `.github/workflows/` does not exist (FAIL).
  - S4: PASS at baseline (regression guard).

## Priority

S2 (question import) first, then S1, S3, S4.

## Preconditions and accounts

| Account | Role | Password | Source |
|---------|------|----------|--------|
| `admin@bilimbaga.local` | super_admin | `Admin1234!` (or current) | seeded |
| `uat.employee@test.com` | employee | `NewPass123!` | earlier UAT |

- Route: `POST /api/v1/questions/import[?dry_run=true]`, multipart field `file` (router.go ~line 143). Employee must be denied.
- Platform at `http://localhost` (nginx :80); API also at `http://localhost:8080` if the port is exposed.
- Generate test files with shell into the scratchpad directory:

| File | How to build |
|------|--------------|
| `valid.csv` | Output of `GET /api/v1/questions/export` (CSV) trimmed to 1-2 rows, or format from `question-authoring-20260609.md` |
| `valid.json` | Valid JSON import in the shape accepted by `parseJSONImport` (derive from the JSON export) |
| `fake.csv` | PNG signature `\x89PNG\r\n\x1a\n` plus padding, named `questions.csv` |
| `binary.csv` | 2 KB random bytes including NULs, named `questions.csv` |
| `big.csv` | Valid header+row padded to 10,485,761 bytes (10 MB + 1) |
| `exact.csv` | Valid CSV of exactly 10,485,760 bytes (pad a long text cell); if it cannot be made valid, SKIP step 6 |
| `huge.csv` | 40 MB (over the 32 MB form limit) |
| `evil.exe` | `MZ` header bytes, named `questions.exe` |
| `empty.csv` | 0 bytes |
| `rows501.csv` | Valid CSV with 501 rows |

- Use `dry_run=true` for all steps except where noted so nothing is written.
- Tools: `curl -i -F "file=@x;type=..."`.

## Scenario S2: Question import upload validation (priority)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | `POST /questions/import?dry_run=true` with `valid.csv` (`text/csv`) | 200; dry-run report | |
| 2 | Admin | Same with `valid.json` (`application/json`) | 200 | |
| 3 | Admin | `fake.csv` (PNG bytes named .csv, declared `text/csv`) | 4xx rejection in the standard error envelope; never 200 or 500; nothing written | |
| 4 | Admin | `binary.csv` | Rejected as in step 3 | |
| 5 | Admin | `big.csv` (10 MB + 1) | HTTP 413 with a `FILE_TOO_LARGE`-style code (as in users CSV import); server healthy afterwards | |
| 6 | Admin | `exact.csv` (10 MB) | Not rejected for size (200, or 400 only for row content) | |
| 7 | Admin | `huge.csv` (40 MB) | 413 (no 500 or connection reset); note which layer answered (nginx or API) | |
| 8 | Admin | `valid.csv` declared as `image/png`, then as `application/octet-stream` | Spec says validation is by magic bytes regardless of Content-Type: expected 200. Record actual; header-only rejection is a REQ question | |
| 9 | Admin | `valid.csv` uploaded as `questions.png` | Record behaviour; no 500 | |
| 10 | Admin | `evil.exe` | Rejected 4xx | |
| 11 | Admin | `empty.csv` | 400, no 500 | |
| 12 | Admin | Multipart without `file`; non-multipart JSON body | 400 `ERR_INVALID_BODY` | |
| 13 | Employee | `valid.csv` with employee token | 403 | |
| 14 | Anonymous | `valid.csv` without token | 401 | |
| 15 | Admin | UI `/admin/questions` import dialog: choose `fake.csv`, then `big.csv` | Localized error each time (no raw JSON, dialog not frozen); a valid file still imports afterwards | |
| 16 | Tester | After all rejected uploads | Question count unchanged; `GET /api/v1/health` 200 | |
| 17 | Admin | `rows501.csv` dry-run | 413 `ERR_BATCH_TOO_LARGE` (existing behaviour kept, distinct code from size limit) | |

## Scenario S1: HSTS via X-Forwarded-Proto

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | `curl -sI http://localhost/` | `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin` present; NO `Strict-Transport-Security` header (not even empty) | |
| 2 | Tester | `curl -sI -H "X-Forwarded-Proto: https" http://localhost/` | `Strict-Transport-Security: max-age=31536000; includeSubDomains` plus the other four | |
| 3 | Tester | `-H "X-Forwarded-Proto: http"` | No HSTS | |
| 4 | Tester | Step 2 against `/api/v1/health`, `/login`, `/nope` (404 path) and a static asset | HSTS and the four headers on every response including errors | |
| 5 | Tester | `X-Forwarded-Proto: HTTPS` and `https, http` | Informational: record behaviour (map is case-sensitive) | |
| 6 | Tester | Load `/login` and `/admin/dashboard` in the browser | CSP unchanged from baseline; fonts and app work; no CSP violations in console | |
| 7 | Tester | `nginx -t` inside the nginx container | Syntax OK | |

## Scenario S3: CI workflow existence (file check)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | List `.github/workflows/` on `main` | Directory exists with a workflow file (e.g. `security.yml`) | |
| 2 | Tester | Read it | Triggers `push` and `pull_request`; runs `make security-check` (or `go mod verify` and `npm audit --audit-level=high` as separate steps); Go and Node set-up present; correct working directories | |
| 3 | Tester | Parse YAML (`python -I -c "import yaml;yaml.safe_load(open(F))"`, or `actionlint`) | Valid | |
| 4 | Tester | Run `make security-check` locally | `go mod verify` OK; zero high/critical npm findings (else DEFECT on AC-7 first clause) | |
| 5 | Tester | Optional: `gh run list --workflow security.yml --limit 3` | A passing run exists (skip if gh rate limited) | |

## Scenario S4: Temp-password complexity (regression guard)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Create 5 users without supplying a password; capture each temp password from Mailhog | Each: >= 8 chars, has upper, lower, digit | |
| 2 | Admin | `POST /users/{id}/reset-password` for 5 users; capture | Same property | |
| 3 | Employee | Change password to `short1A`, `alllowercase1`, `ALLUPPERCASE1`, `NoDigitsHere` | Each 400 `WEAK_PASSWORD` | |
| 4 | Employee | Change password to `NewPass123!` (then back if needed) | 200 | |
| 5 | Tester | File check | A test in `backend/internal/users/` calls `generateTempPassword()` over many iterations against the complexity rule | |

Steps 1-2 need Mailhog wiring. **PRECONDITION GAP:** SMTP delivery in the compose `api` service is unverified (see `account-recovery-20261009.md`, Email observability). If unobservable mark steps 1-2 ENV ISSUE and rely on step 5.

## Pass / Fail criteria

- PASS: S1-S4 as expected; no 500 on any malformed upload; rejected uploads write nothing.
- FAIL (defect): invalid-content file accepted or causing 500; > 10 MB not 413; HSTS on plain HTTP or absent when forwarded as https; HSTS missing on error responses; no workflow or no `security-check` step; generated temp password violates complexity.
- ENV ISSUE: #19 not merged; nginx not rebuilt; Mailhog not wired (S4 1-2).
- REQ GAP candidates: S2 step 8 (Content-Type vs magic bytes semantics); 400 vs 415 for wrong content; case sensitivity of `X-Forwarded-Proto`.

## Acceptance Criteria Coverage

| FR | AC# | Covered by |
|----|-----|------------|
| FR-BB64 | AC-2 (five headers; HSTS only when HTTPS) | S1 |
| FR-BB64 | AC-3 (magic bytes; CSV 10 MB -> 413; question import) | S2 (logo 2 MB part covered in `tenant-configuration-20260609.md`) |
| FR-BB64 | AC-7 (CI steps) | S3 |
| FR-BB64 | AC-8 (password complexity) | S4 |
| FR-BB64 | AC-1, AC-4, AC-5, AC-6 | Already satisfied; not retested (ratelimit tests are unit-level) |

## Out of Scope

Tightening CSP; ratelimit middleware unit tests; SQL parameterization audit; JWT secret entropy.
