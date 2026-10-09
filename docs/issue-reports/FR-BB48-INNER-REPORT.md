# FR-BB48 Inner Report - Public certificate verification page (issue #25)

## Changes
- Backend: `Config.PublicAppURL` (`PUBLIC_APP_URL`, default `http://localhost:5173`, documented in `.env.example`); `cmd/api/main.go` passes it to `certificates.NewHandler`; new `certificates.BuildVerifyURL` (trims trailing slashes) used by `GeneratePDF` for QR and printed URL.
- Frontend: public lazy route `/verify/:code` (`VerifyCertificatePage`, `api/verify.ts`), rendered before the auth bootstrap in `App.tsx` so no token refresh or RequireAuth applies; noindex meta; tenant branding and LocaleSwitcher; `verify.*` keys in en/ru/kk.
- Docs: requirement status and README set to Implemented.

## Tests
- Go: `go vet ./...`, `go test ./...` pass (new: BuildVerifyURL, config default/override).
- Frontend: `npx tsc --noEmit` clean, `npm test` (vitest + check:i18n) pass; new VerifyCertificatePage tests (loading, valid, invalid, 500 + retry, network failure, noindex/no Authorization, locale switch).

## Not verified
- Nginx fallback for `/verify/*` and a live scan of a generated QR (no docker/live stack per instructions).
- Dedicated code-review subagent not spawned; self-reviewed.
