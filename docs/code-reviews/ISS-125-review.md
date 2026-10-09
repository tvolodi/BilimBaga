# Code Review: ISS-125 (empty SMTP_FROM -> 550)

Run ID: iss-125
Result: PASS

Verified: `go vet` clean on config/email/cmd; `go test ./internal/config ./internal/email` pass.

## Findings
- [Medium] backend/internal/config/config.go:DefaultSMTPFrom - `noreply@localhost` removes the empty-sender syntax error but real relays commonly reject a non-resolvable sender domain (still 550, different text). Only a startup warning flags this. Consider making the default opt-in or logging at error level in non-dev environments.
- [Low] backend/internal/config/config.go - gofmt realignment of unrelated Config fields (DB block) adds diff noise; harmless.
- [Low] backend/internal/email/service.go - `extractEmailAddress` error text for empty From is good, but TestSend surfaces it verbatim in the HTTP 503 message (config hint only, no secrets).
- [Low] docs/issue-reports/ISS-125-empty-smtp-from-550.md - `module: tenant` looks inaccurate (should be email/config); table separator row has a malformed Recurrence Log header.
- [Low] Config test does not cover whitespace-only SMTP_FROM with SMTP_HOST set (service layer test does cover it for send).

## Checklist
- No secrets, no os.Getenv in services (config loaded via getEnv in Load only): OK
- Errors returned instead of swallowed (fallback to raw string removed): OK
- Config validated once at startup, typed Config: OK
- Existing test replaced with inverse behavior test; new table test for config: OK
- Docs (.env.example, FR-BB61) updated and consistent with code (503 EMAIL_UNAVAILABLE verified in handler.go): OK
- No migrations, no frontend changes, no new endpoints: N/A

## AC Coverage
No requirement AC list for this issue; the issue report's fix items (config default+validation, explicit service error, startup warning, docs) are all implemented and tested.
