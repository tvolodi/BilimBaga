# Code Review: ISS-141 (malformed UUID params -> 422/404)

Run ID: iss-141

Result: PASS

## Findings
- [Medium] backend/internal/sessions/handler.go:299 (HandleListGradingQueue) - `exam_id` query param is passed to Postgres unvalidated; a malformed value still yields a 500. Same defect class, outside the listed scope. -> use `api.UUIDQuery(w, r, "exam_id")` (+ test).
  RESOLVED: fixed with api.UUIDQuery + test TestHandleListGradingQueue_ExamIDFilter.
- [Medium] backend/internal/router - no router-level test that the middleware is mounted in the authenticated group (only the unit test in internal/api/uuid_test.go uses a chi Group). -> optional router test (e.g. GET /api/v1/users/not-a-uuid -> 404).
- [Low] Request-body UUID fields (exams rules category_id/section_id/question_id, assignee_id, etc.) are not covered; out of scope for this issue.
- [Low] Middleware runs before RBAC (With), so an unprivileged caller gets 404 instead of 403 for a malformed id. Acceptable.

## Verification of requested points
1. chi ordering: `r.Use` inside `r.Group` creates an inline mux whose middlewares wrap the matched endpoint handler, so they run after route matching; `chi.URLParam` is populated. Covered by TestRequireUUIDPathParams_InGroup. OK.
2. Route audit: every `{id}`, `{userId}`, `{sessionId}`, `{examId}`, `{questionId}`, `{tagId}`, `{sectionId}`, `{ruleId}`, `{assignmentId}` in the authenticated group is a UUID resource. `/users/me`, `/users/roles`, `/users/import` are static routes registered ahead of `/users/{id}` (chi prefers static), so `me` never hits the middleware as an id. The only non-UUID params are `{locale}` (translations) and `{code}` (public verify); neither is in the list. No offending route found.
3. Public `/verify/{code}` lives in the separate public group without the middleware and `code` is not in the name list. No behaviour change.
4. Remaining unvalidated id-like params: only sessions `exam_id` (see Medium). Other Query().Get usages are ints/enums/dates/flags. Body UUIDs out of scope.

## Other checks
- Handlers remain thin; responses use the standard error envelope; 422 VALIDATION_ERROR for query, 404 NOT_FOUND for path. IsUUID pins length 36, rejecting urn/braced/unhyphenated forms.
- users handler no longer uses google/uuid directly (import removed correctly). Behaviour of role_id preserved.
- No secrets, SQL, os.Getenv, or debug output added.
- `go test -p 2 ./internal/api ./internal/users ./internal/audit ./internal/questions ./internal/router`: all ok. `go vet` on the changed packages: clean.

## AC Coverage
- users department_id/role_id -> 422: covered
- audit actor_id -> 422 (List and Export via parseFilters): covered
- questions list category_id/tag_ids/tag_id -> 422: covered
- questions export ids/category_id/tag_ids -> 422: covered
- malformed path UUIDs -> 404 via middleware in authenticated group: covered

Summary: Zero Critical/High findings; fix is correct and safe, with one same-class gap (sessions grading exam_id) recommended as a follow-up.

## Cycle 2 - added tests (supervisor merge condition)

- `TestRouter_MalformedUUIDPathParamIs404`: real router, valid JWT, malformed `{id}`/`{sessionId}` on users, exams, questions, portal/sessions, admin/grading returns 404 NOT_FOUND from the middleware (mutation-checked: removing the `r.Use` fails 7 cases).
- `TestRouter_NonUUIDParamRoutesUnaffected`: authenticated `{locale}` route and public `/verify/{code}` are not rejected by the middleware.
- `TestRouter_AllPathParamsAreUUIDOrDocumentedExceptions`: `chi.Walk` over every registered route; each param must be in `api.UUIDPathParamNames` (single shared list, now used by the router) or in the documented exceptions (`locale`, `code`). Failure message tells the dev what to do.
- Finding: only non-UUID params are `{locale}` (authenticated translations route) and `{code}` (public verify). No other drift.
- go build/vet/test ./... green.
