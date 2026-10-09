# ISS-209 - ROLE_IN_USE returns details.count

Root cause: the roles handler emitted only `{code,message}`; the frontend regex-parsed the user count from the English message.

Fix:
- `api.WriteErrorWithDetails` (backend/internal/api/response.go) writes the standard envelope plus `error.details`.
- roles handler: `ROLE_IN_USE` -> `details: {count: N}` (InUseError.Count already set by the service).
- Frontend `roleErrorKey` reads `details.count` (default 0), no message parsing; existing kk/ru/en `{{count}}` interpolation unchanged.
- docs/requirements/api-conventions.md notes the new details user.

Tests: api response test, roles handler_test (details.count asserted, absent for other errors), service_test already covers InUseError.Count; vitest roles.test.ts and RolesPage.test.tsx updated.
Results: go test -p 2 ./... and go vet green; vitest 653 pass; tsc and eslint clean.
