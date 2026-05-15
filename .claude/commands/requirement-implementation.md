Implement the FR-BBxxx requirement given as argument (or the currently open requirement doc if no argument).

Follow this exact sequence. Do NOT skip any phase. Do NOT ask the user to run anything.

---

## Phase 1 — Analysis

1. Read the requirement doc fully (`docs/requirements/`).
2. Read `docs/architecture-guide.md` and the relevant backend/frontend development guide.
3. Search for existing code in the affected domain packages.
4. Identify every file that will change or be created.
5. Confirm all acceptance criteria are fully understood.
6. Create a todo list covering every phase below.

---

## Phase 2 — Database Migration

1. List `backend/migrations/` to determine the next sequential number.
2. Write `backend/migrations/{NNN}_{slug}.up.sql` and `{NNN}_{slug}.down.sql`.
3. **Never edit existing migration files.**
4. Apply the migration immediately:
   - Try `make migrate` first.
   - If unavailable, use: `docker exec bilimbaga-db-1 psql -U bilimbaga -d bilimbaga < migrations/{NNN}_{slug}.up.sql`
   - Then record it: `docker exec bilimbaga-db-1 psql -U bilimbaga -d bilimbaga -c "INSERT INTO schema_migrations (version, dirty) VALUES ({NNN}, false) ON CONFLICT DO NOTHING;"`
5. Verify tables exist (`\dt`, `\d {table}`) before proceeding.

---

## Phase 3 — Go Backend

1. Create `backend/internal/{domain}/` with:
   - `model.go` — structs, sentinel errors, filter/input types
   - `repository.go` — Repository interface + postgresRepository implementation
   - `service.go` — Service interface + service implementation
   - `handler.go` — HTTP handlers
2. Register routes in `internal/router/router.go` (with RBAC middleware).
3. Wire repo → service → handler in `cmd/api/main.go`.
4. Run `cd backend && go build ./...` — fix all compile errors before continuing.

---

## Phase 4 — Frontend (if the requirement has UI)

1. Add API client in `frontend/src/api/{domain}.ts`.
2. Add React Query hooks.
3. Build page components in `frontend/src/pages/`.
4. Register route in the router.
5. Add all new strings to `frontend/src/locales/en.json`, `kk.json`, `ru.json`.

---

## Phase 5 — Test Development

**Do NOT skip this phase.**

1. Write `backend/internal/{domain}/service_test.go`:
   - Manual mock implementing the Repository interface (no generated mocks)
   - Tests for every Service method: happy path + all error branches
   - Each acceptance criterion must map to at least one test case
2. Write `backend/internal/{domain}/handler_test.go`:
   - Manual mock implementing the Service interface
   - Tests for every HTTP handler: 200/201/204 success + 400/404/409/422/500 error cases
   - Use `httptest.NewRecorder` + `httptest.NewRequest`; set chi URL params via `withChiParam`
3. Write frontend component tests for any new React components (if Phase 4 ran).
4. Run the backend tests immediately:
   ```
   cd backend && go test ./internal/{domain}/... -v
   ```
5. Fix every failing test before proceeding. Do NOT use `t.Skip()`, do NOT weaken assertions.
6. Run the full backend suite to confirm no regressions:
   ```
   cd backend && go test ./...
   ```

---

## Phase 6 — Documentation Update

1. Update `docs/requirements/README.md` — change status to `implemented`.
2. Update the `Status` field in the requirement doc to `implemented`.
3. Update `docs/architecture-guide.md` if new tables or endpoints were added.

---

## Phase 7 — Final Verification

Before finishing:
- [ ] Migration applied and tables verified in DB
- [ ] `go build ./...` succeeds
- [ ] All new tests pass
- [ ] Full `go test ./...` passes with zero failures
- [ ] No hardcoded strings, no `os.Getenv()` in handlers, no secrets
- [ ] All acceptance criteria covered by at least one test

Summarize what was implemented. Do NOT include any commands for the user to run.
