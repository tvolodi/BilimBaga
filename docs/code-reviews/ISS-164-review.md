# Code Review ISS-164 — email case normalisation

Result: PASS (0 Critical, 0 High)

Scope: backend/internal/api/email.go (+test), auth/service.go (Login), auth/recovery_service.go (ForgotPassword),
users/service.go (CreateUser, ImportUsers/commitImportRow), tests in auth/email_case_test.go and users/service_test.go.
Verified: `go test -p 1 ./internal/auth ./internal/users ./internal/api` ok; `go vet` clean; api package has no
internal imports, so no import cycle.

## Security analysis
- Account takeover: none introduced. Both write and read sides now apply the identical trim+lowercase, and
  `users.email` is UNIQUE (case-sensitive) so two accounts cannot collide post-normalisation on new writes. Reset
  tokens are bound to the user ID resolved from the normalised address; the mail goes to the stored address.
- User enumeration / timing: ForgotPassword normalises before validation and lookup, then follows the unchanged
  decoy path (neutral response, handler padding, decoy IssueResetToken). Normalisation is O(len), no observable delta.
  Login keeps the dummy bcrypt on ErrNotFound. Unknown mixed-case email stays neutral (tested).
- Lowercase-only users: unchanged behaviour (idempotent function).

## Coverage of every email entry point
- Login, ForgotPassword, CreateUser, ImportUsers (validate + commit): normalised.
- Bootstrap (`auth/bootstrap.go`): uses constant lowercase `admin@bilimbaga.local`; fine.
- Users update/invite: no email-changing path exists; CSV handler only TrimSpaces then service normalises.
- Email service (`internal/email`): sends to emails read from DB; no user-typed input.
- SSO: none in code. No other repository lookups by email (only auth.GetUserByEmail and bootstrap).
- Seed migration 029: lowercase.

## Findings
- [Medium] auth/handler.go:89,94 — audit log records raw `req.Email` (mixed case, untrimmed, attacker-controlled)
  for auth.login.failure/success, inconsistent with stored form and harder to correlate. -> log the normalised
  address (or user ID on success). Not blocking.
- [Medium] No data migration lowercases pre-existing rows. Any legacy row with uppercase letters (created by direct
  SQL or before create-time lowercasing) becomes unreachable at login, a regression for that user. Create paths
  already lowercased, and only the seed exists in migrations, so likely none; confirm with
  `SELECT count(*) FROM users WHERE email <> lower(btrim(email))` and, if nonzero, add a new numbered migration
  (checking for case-insensitive duplicates first). Not blocking.
- [Low] strings.ToLower is Unicode-aware (e.g. "İ"); ForgotPassword's ParseAddress and users emailRegex limit
  practical impact. Optional: ASCII-only fold.
- [Low] Mixed errors: ForgotPassword returns validation 400 for malformed emails (pre-existing, unchanged).

## AC coverage
- Login with mixed-case/whitespace email succeeds: covered (service + handler tests)
- Forgot-password with mixed-case email issues token and mails: covered
- Unknown mixed-case stays neutral / wrong email still 401: covered
- Create/import store normalised email: covered
- Lowercase users unaffected: covered by existing suites passing
