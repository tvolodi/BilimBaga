# FR-BB64 — Security Hardening

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB64 |
| Phase | 6 — Polish & Hardening |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB14, FR-BB15, FR-BB16, FR-BB25, FR-BB37 |

## Description
Closes security gaps across the full stack. Rate limiting is applied to all API endpoints with tighter limits on sensitive auth and answer-save routes. HTTP security headers are configured at the Nginx layer. File uploads are validated by magic bytes server-side. JWT secrets meet minimum entropy requirements and refresh tokens are stored only as SHA-256 hashes. Password complexity is enforced on all password-setting code paths.

## Acceptance Criteria
- [ ] AC-1: Rate limiting middleware is applied globally; auth endpoints (`/api/v1/auth/*`) are limited to 10 req/min per IP, answer-save (`PUT /portal/sessions/*/answers/*`) to 60 req/min per session ID, and all other endpoints to 300 req/min per IP; requests exceeding the limit receive `429 Too Many Requests` with `Retry-After` header.
- [ ] AC-2: Nginx serves all five required security headers (`Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Strict-Transport-Security`) on every response; `Strict-Transport-Security` is only set when the request is HTTPS.
- [ ] AC-3: File upload endpoints (logo, CSV import) validate the uploaded file server-side by reading its magic bytes and reject any file whose detected MIME type does not match the expected type, regardless of the `Content-Type` header; logo files exceeding 2 MB and CSV files exceeding 10 MB are rejected with `413 Payload Too Large`.
- [ ] AC-4: Exam answer submission validates that each supplied option ID belongs to one of the session's active questions; unknown option IDs or questions not present in the session return `400 Bad Request` with error code `INVALID_ANSWER_OPTION`.
- [ ] AC-5: JWT tokens use HS256 with a secret of at least 256 bits (32 bytes); refresh tokens are stored in the database only as a SHA-256 hash (never plaintext); the plaintext token is returned to the client once and never persisted.
- [ ] AC-6: All database queries across the codebase use parameterized statements (sqlx named queries or positional `$N` placeholders); no SQL query is built via string concatenation of user input.
- [ ] AC-7: `go mod verify` completes with no errors and `npm audit --audit-level=high` reports zero high or critical vulnerabilities; both commands are added as required CI steps.
- [ ] AC-8: Password-setting code paths (create user, change password, reset to temp password) enforce the complexity rule: minimum 8 characters, at least one uppercase letter, one lowercase letter, and one digit; violations return `400` with error code `WEAK_PASSWORD`.

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

# HTTPS only
if ($scheme = "https") {
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
}
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
- HSTS header must only be sent over HTTPS to avoid HSTS pinning over HTTP in development. The Nginx `if ($scheme = "https")` block handles this correctly.
