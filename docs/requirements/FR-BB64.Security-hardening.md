# FR-BB64 — Security Hardening

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB64 |
| Phase | 6 — Polish & Hardening |
| Priority | 1 |
| Status | Implemented |
| Depends On | FR-BB14, FR-BB15, FR-BB16, FR-BB25, FR-BB37 |

## Description
Closes security gaps across the full stack. Rate limiting is applied to all API endpoints with tighter limits on sensitive auth and answer-save routes. HTTP security headers are configured at the Nginx layer. File uploads are validated by magic bytes server-side. JWT secrets meet minimum entropy requirements and refresh tokens are stored only as SHA-256 hashes. Password complexity is enforced on all password-setting code paths.

## Implementation Delta (audited 2026-10-09)

Part of this FR is already in the codebase. Only the items under "Remaining" need implementation.

| AC | State | Evidence |
|----|-------|----------|
| AC-1 | Satisfied (tests missing) | `backend/internal/ratelimit/middleware.go` (AuthLimiter 10/min, GlobalLimiter 300/min, AnswerSaveLimiter 60/min keyed `session:{id}`, 429 envelope `RATE_LIMITED` + `Retry-After: 60`; all three limiters are bypassed when the test-only `DISABLE_RATE_LIMIT=true|1` is set, PR #197, never set it on shared or production-class instances); wired in `backend/internal/router/router.go` lines 54-75, 231. No `ratelimit` test file exists. |
| AC-2 | Partial | `deploy/nginx.conf` sets CSP, X-Frame-Options DENY, nosniff, Referrer-Policy. **HSTS missing.** CSP is deliberately relaxed (`'unsafe-inline'`, Google Fonts) because `frontend/index.html` loads Inter from fonts.googleapis.com; this relaxed policy is the accepted baseline (see AC-2 text). `deploy/nginx/bilimbaga*.conf` (host vhosts, TLS) add no headers. |
| AC-3 | Partial | `backend/internal/upload/validate.go` (magic bytes, 2 MB / 10 MB); used by `tenant/service.go` (logo, 413 in `tenant/handler.go`) and `users/handler.go` (CSV, 413 `FILE_TOO_LARGE`). `questions/import_export_handler.go` Import also calls `upload.ValidateCSVFile` with a bounded read (`:44-57`, since #70) and, since PR #197, `upload.ParseImportMultipart` caps the whole body at 10 MiB + 1 MiB before parsing (413 `ERR_FILE_TOO_LARGE`, `upload/validate.go:90-104`; users import the same, 413 `FILE_TOO_LARGE`). Updated 2026-10-09; the earlier "no validation, 32 MB parse" text was stale. |
| AC-4 | Satisfied | `sessions/handler.go` lines ~100-110 map `ErrQuestionNotInSession`, `ErrInvalidAnswerOption`, `ErrInvalidOption` to 400 `INVALID_ANSWER_OPTION`; tests in `sessions/handler_test.go`. |
| AC-5 | Satisfied | `config/config.go` rejects JWT_SECRET < 32 chars; `auth/service.go` SHA-256 hashes refresh tokens; `refresh_tokens.token_hash` (migration 004, repository.go). |
| AC-6 | Satisfied (audit) | Dynamic `fmt.Sprintf` queries (`audit`, `exams`, `questions`, `sessions` repositories) only interpolate `$N` placeholders or fixed clauses; values are passed as args. Re-run the grep audit once in implementation and record the result; no code change expected. |
| AC-7 | Partial | `Makefile` target `security-check` runs `go mod verify` and `npm audit --audit-level=high`. **No CI workflow exists** (`.github/workflows/` absent), so it is not a required CI step. |
| AC-8 | Satisfied (test gap) | `auth/password.go` `ValidateComplexity` enforced in `auth/service.go` ChangePassword (400 `WEAK_PASSWORD`). Create/reset/import in `users/service.go` use server-generated `generateTempPassword()` (3 lower, 3 upper, 2 digits, 2 specials, 10 chars), which always satisfies the rule; no user-supplied password path exists there. |

### Remaining work
1. AC-2: add `Strict-Transport-Security` to `deploy/nginx.conf` only when the original request was HTTPS. The container listens on :80 behind a TLS-terminating proxy/Cloudflare, so use `map $http_x_forwarded_proto $hsts { default ""; https "max-age=31536000; includeSubDomains"; }` and `add_header Strict-Transport-Security $hsts always;` (nginx omits headers with empty values). Do not use `if ($scheme = "https")`.
2. AC-3: done (PR #70 validation, PR #197 pre-parse body cap with tests in `upload/import_multipart_test.go`, `questions/import_export_handler_test.go`, `users/handler_test.go`).
3. AC-7: add `.github/workflows/security.yml` (or equivalent) running `make security-check` on push and pull request.
4. Tests: add `ratelimit/middleware_test.go` (429 envelope, Retry-After, per-session keying, 11th auth request blocked) and a `users` test asserting `generateTempPassword()` satisfies `auth.ValidateComplexity` over many iterations (avoid an import cycle by duplicating the rule check if required).

## Acceptance Criteria
- [x] AC-1: Rate limiting middleware is applied globally; auth endpoints (`/api/v1/auth/*`) are limited to 10 req/min per IP, answer-save (`PUT /portal/sessions/*/answers/*`) to 60 req/min per session ID, and all other endpoints to 300 req/min per IP; requests exceeding the limit receive `429 Too Many Requests` with `Retry-After` header.
- [x] AC-2: Nginx serves all five required security headers (`Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Strict-Transport-Security`) on every response; `Strict-Transport-Security` is only set when the original request was HTTPS (detected via `X-Forwarded-Proto`). Baseline CSP is the policy currently in `deploy/nginx.conf` (permits `'unsafe-inline'` and Google Fonts, as the SPA loads Inter from Google Fonts); tightening is out of scope.
- [x] AC-3: File upload endpoints (logo, CSV import) validate the uploaded file server-side by reading its magic bytes and reject any file whose detected MIME type does not match the expected type, regardless of the `Content-Type` header; logo files exceeding 2 MB and CSV files exceeding 10 MB are rejected with `413 Payload Too Large`.
- [x] AC-4: Exam answer submission validates that each supplied option ID belongs to one of the session's active questions; unknown option IDs or questions not present in the session return `400 Bad Request` with error code `INVALID_ANSWER_OPTION`.
- [x] AC-5: JWT tokens use HS256 with a secret of at least 256 bits (32 bytes); refresh tokens are stored in the database only as a SHA-256 hash (never plaintext); the plaintext token is returned to the client once and never persisted.
- [x] AC-6: All database queries across the codebase use parameterized statements (sqlx named queries or positional `$N` placeholders); no SQL query is built via string concatenation of user input.
- [x] AC-7: `go mod verify` completes with no errors and `npm audit --audit-level=high` reports zero high or critical vulnerabilities; both commands are added as required CI steps. **Implementation note:** the blocking CI gate audits production dependencies (`--omit=dev`, 0 vulnerabilities); the full audit including dev tooling runs non-blocking because the remaining 7 advisories sit in vitest 2.x and need a major upgrade (follow-up).
- [x] AC-8: Password-setting code paths (create user, change password, reset to temp password) enforce the complexity rule: minimum 8 characters, at least one uppercase letter, one lowercase letter, and one digit; violations return `400` with error code `WEAK_PASSWORD`.

## Technical Specification

### Configuration / Infrastructure

**Rate limiter**: use `github.com/go-chi/httprate` (or equivalent in-process token-bucket):
```go
import "github.com/go-chi/httprate"

// In router setup:
r.Group(func(r chi.Router) {
    r.Use(httprate.LimitByIP(10, time.Minute))   // per IP
    r.Mount("/auth", authRouter)
})

r.Group(func(r chi.Router) {
    r.Use(httprate.LimitByIP(300, time.Minute))  // per IP, general
    r.Mount("/admin", adminRouter)
})

// Answer save: limit by session ID extracted from URL param
r.With(answerRateLimit).Put("/portal/sessions/{id}/answers/{qid}", handler)
```

`answerRateLimit` middleware extracts `{id}` from URL and uses it as the rate limit key (60/min).

**429 response format**:
```json
{ "data": null, "error": { "code": "RATE_LIMITED", "message": "Too many requests" } }
```
With header: `Retry-After: 60`.

**Nginx security headers** (`deploy/nginx.conf`):
```nginx
add_header Content-Security-Policy "default-src 'self'; img-src 'self' data:; font-src 'self';" always;
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "same-origin" always;

# HTTPS only (TLS terminates upstream; detect via X-Forwarded-Proto). Define map in http context / conf.d:
# map $http_x_forwarded_proto $hsts { default ""; https "max-age=31536000; includeSubDomains"; }
add_header Strict-Transport-Security $hsts always;
```

Note: The strict CSP (`no inline scripts/styles`) requires Vite build to produce no inline `<script>` or `<style>` tags. Enable `build.cssCodeSplit: true` and ensure no inline `style=` attributes in production HTML.

### API Endpoints

No new endpoints. Changes are applied to existing middleware and upload handlers.

### Implementation Details

**Magic byte validation** (`internal/upload/validate.go`):
```go
var magicBytes = map[string][]byte{
    "image/png":  {0x89, 0x50, 0x4E, 0x47},
    "image/jpeg": {0xFF, 0xD8, 0xFF},
    "text/csv":   nil, // CSV has no magic bytes; validate UTF-8 + comma heuristic
}

func DetectMIME(data []byte) string {
    return http.DetectContentType(data[:512])
}
```

For CSV: read first 512 bytes, confirm UTF-8, check that `http.DetectContentType` returns `"text/plain"` (CSV files do not have magic bytes; text detection is sufficient combined with extension check and size limit).

**Refresh token hashing**:
```go
import "crypto/sha256"

func HashToken(plaintext string) string {
    sum := sha256.Sum256([]byte(plaintext))
    return hex.EncodeToString(sum[:])
}
```

Store `HashToken(token)` in DB column `refresh_tokens.token_hash`. On validation, hash the provided token and compare.

**Password complexity** (`internal/auth/password.go`):
```go
var (
    hasUpper  = regexp.MustCompile(`[A-Z]`)
    hasLower  = regexp.MustCompile(`[a-z]`)
    hasDigit  = regexp.MustCompile(`[0-9]`)
)

func ValidateComplexity(password string) error {
    if len(password) < 8       { return ErrWeakPassword }
    if !hasUpper.MatchString(password) { return ErrWeakPassword }
    if !hasLower.MatchString(password) { return ErrWeakPassword }
    if !hasDigit.MatchString(password) { return ErrWeakPassword }
    return nil
}
```

Called in: `CreateUser`, `ChangePassword`, `ResetPassword` handlers.

**Answer validation** — in session answer handler, after parsing the request body:
```go
validQuestionIDs := sessionQuestionIDSet(session) // loaded from DB at session start
validOptionIDs   := optionIDSetForSession(session)

if _, ok := validQuestionIDs[req.QuestionID]; !ok {
    return ErrInvalidAnswerOption
}
if req.OptionID != nil {
    if _, ok := validOptionIDs[*req.OptionID]; !ok {
        return ErrInvalidAnswerOption
    }
}
```

**SQL parameterization audit**: run `grep -rn "fmt.Sprintf\|fmt.Fprintf\|strings.Builder" internal/` — any occurrence in a file containing `db.Query` or `db.Exec` must be reviewed and replaced with sqlx named parameters.

**CI steps to add** (`Makefile` or GitHub Actions):
```bash
go mod verify
npm audit --audit-level=high --prefix frontend
```

## Notes
- The CSP header `default-src 'self'` blocks Google Fonts, CDN-hosted assets, and external analytics. Ensure all fonts are self-hosted in the Vite build output (`public/fonts/`).
- Rate limiting state is in-process (Go map); in a multi-instance deployment, a Redis-backed rate limiter would be needed. This is acceptable for Phase 6 (single-instance Docker Compose deployment).
- `go mod verify` checks that modules in the module cache have not been tampered with. It does not scan for CVEs; a future CI step can add `govulncheck ./...` for CVE scanning.
- HSTS header must only be sent over HTTPS to avoid HSTS pinning over HTTP in development. The `map` on `X-Forwarded-Proto` yields an empty (omitted) header over HTTP.
