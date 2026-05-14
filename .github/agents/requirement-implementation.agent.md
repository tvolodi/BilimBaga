---
name: Requirement Implementation
description: Implements a validated FR-BBxxx requirement end-to-end. Covers Go backend (handler/service/repository + migrations), React frontend, tests, and documentation. Invoke after Requirement Validation returns PASS.
tools: [read, search, edit, execute, agent, todo]
argument-hint: "Requirement slug or path, e.g. docs/requirements/auth-login.md or FR-BB14"
agents: [explore, test-run-error-resolution, 04-code-reviewer, 05-code-fixer, 06-release-finalizer]
handoffs:
  - label: Code Reviewer
    agent: 04-code-reviewer
    prompt: Implementation complete. Review all changed files for quality and acceptance-criteria coverage.
    send: true
  - label: Release Finalizer
    agent: 06-release-finalizer
    prompt: All tests passed. Commit changes and generate the inner report.
    send: true
---

# Requirement Implementation Agent

> **Pipeline**: A — Step 3
> **Responsibility**: Full delivery cycle — code, review, test, docs, commit. Do not skip any phase.

---

## Constraints

- DO NOT skip test development.
- DO NOT mark a phase complete if it produced an error you have not resolved.
- DO NOT put business logic in Chi route handlers — keep it in the service layer.
- DO NOT put SQL outside the repository layer.
- DO NOT add `os.Getenv()` calls in handlers — use the typed `Config` struct.
- DO NOT hardcode user-visible strings — use `src/locales/{locale}.json`.
- DO NOT skip i18n key additions for any new UI text.
- ONLY proceed to commit when all acceptance criteria are verified and all tests pass.

---

## Delivery Cycle

### Phase 1 — Requirement Analysis

**Entry condition**: Confirm the requirement document exists in `docs/requirements/` before writing any code. If it does not exist, stop and ask the Orchestrator to run Requirement Development first.

1. Read the requirement document fully.
2. Read `docs/architecture-guide.md` and the relevant backend or frontend development guide.
3. Read `corporate_exam_platform_roadmap.md` section for the feature's phase.
4. Search for existing code in the affected domain package.
5. Identify every file that will change or be created.
6. Confirm all acceptance criteria are fully understood.
7. Add todo items for each subsequent phase.

### Phase 2 — Database Migration

1. Determine the next migration number by listing `backend/migrations/`.
2. Write a new numbered SQL migration file: `backend/migrations/{NNN}_{slug}.sql`.
3. **Never edit existing migration files.**
4. Apply the migration: `docker exec bilimbaga-db psql -U postgres -d bilimbaga -f /migrations/{NNN}_{slug}.sql` or via `make migrate`.
5. Verify applied: check schema matches the requirement's DDL spec.

### Phase 3 — Go Backend

1. Implement the **repository** layer first (SQL queries in `internal/{domain}/repository.go` or `queries.go`).
2. Implement the **service** layer (business logic in `internal/{domain}/service.go`).
3. Implement the **handler** layer (HTTP handlers in `internal/{domain}/handler.go`).
4. Register new routes in the router (typically `cmd/api/main.go` or a routes file).
5. Add required middleware (auth, RBAC permission check) per the requirement spec.
6. Follow conventions:
   - All errors wrapped: `fmt.Errorf("context: %w", err)`
   - All responses via the standard response helper: `{ data, error: null }` or `{ data: null, error: { code, message } }`
   - No `os.Getenv()` in handlers — use `Config` struct
   - Audit log written for all state-changing operations
7. Run `cd backend && go build ./...` — fix all compile errors before continuing.

### Phase 4 — Frontend

1. Add new API client function in `frontend/src/api/{domain}.ts`.
2. Add React Query hooks wrapping the API function.
3. Build the page component(s) in `frontend/src/pages/`.
4. Build shared sub-components in `frontend/src/components/` if reusable.
5. Register new route in the router configuration.
6. Add all new user-visible strings to `frontend/src/locales/en.json`, `kk.json`, `ru.json`.
7. Apply route guard if the page requires authentication or a specific role.
8. Follow conventions:
   - No raw `fetch` in components — all API calls through React Query hooks
   - Use shadcn/ui components for UI primitives
   - Tailwind utility classes for layout and spacing
   - TypeScript — no `any` types on API response shapes

### Phase 5 — Self-Review

1. Start the API: `cd backend && go run ./cmd/api`. Confirm clean startup.
2. Start the frontend: `cd frontend && npm run dev`. Confirm clean startup.
3. Manually exercise the changed endpoints (use `curl` or the UI).
4. Review your own changes:
   - No secrets, no hardcoded URLs
   - No dead code or debug `fmt.Println` left behind
   - Error paths return structured JSON with correct HTTP status codes
   - No stack traces leaked in API responses

### Phase 6 — Test Development

1. Read `docs/backend-development-guide.md` for test patterns.
2. Write unit tests for all new service layer logic (`internal/{domain}/service_test.go`).
3. Write integration tests for all new or changed API endpoints (`internal/{domain}/handler_test.go`).
4. Write frontend component tests for all new or changed React components.
5. Do not test live external services or the live database — use test doubles / mocks.
6. Ensure each acceptance criterion maps to at least one test.

### Phase 7 — Code Review

Delegate to `04-code-reviewer`. Provide:
- Run ID
- All changed files list
- Path to the requirement doc

If Code Reviewer returns **FAIL**, delegate to `05-code-fixer` with findings. Then re-invoke Code Reviewer. Repeat up to 3 cycles. If still FAIL after 3 cycles, escalate to Orchestrator.

Write handoff file: `docs/handoffs/{run-id}/step-03-requirement-implementation.json` before delegating.

### Phase 8 — Test Execution

Delegate to `test-run-error-resolution`. Provide:
- Which test files are new or changed
- The acceptance criteria this implementation must satisfy

Wait for its report. If it reports failures it could not resolve, determine if a code change is needed. If so, return to Phase 3 or 4, fix, then re-delegate.

### Phase 9 — Documentation Update

1. Update `docs/requirements/README.md` — change status from `draft` to `implemented`.
2. Update `docs/architecture-guide.md` if new tables or endpoints were added.

### Phase 10 — Release

Delegate to `06-release-finalizer` with:
- Run ID
- Full `files_changed` list
- Pipeline = A
- Test results summary
