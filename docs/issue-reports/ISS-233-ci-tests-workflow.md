# ISS-233: CI tests workflow

Added `.github/workflows/tests.yml` (jobs `backend-tests`, `frontend-tests`) and a README "Continuous integration" section.

- Backend: go vet, staticcheck 2025.1.1 (pin may need a bump if the run complains), `go test -p 2 ./...` with postgres:16 service; `TEST_DATABASE_URL` is read by `backend/internal/db/unique_email_index_test.go`.
- Frontend: tsc, lint, check:i18n, vitest (CI=true). Playwright e2e excluded.
- No tests were run locally (host memory); the first CI run decides.

## First run results

(pending)
