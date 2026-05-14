# Function: DESIGN_SOLUTION

> Plan the implementation before writing any code. Produces a concrete, ordered implementation plan.

## Metadata

| Property | Value |
|----------|-------|
| **Category** | Design |
| **Used By** | Requirement Implementation, Issue Resolution, Code Fixer |
| **Depends On** | ANALYZE_CONTEXT output |

---

## Purpose

Produce a concrete implementation plan before touching any file. This prevents mid-implementation pivots and ensures all layers are considered.

---

## Steps

1. **Database layer** (if migrations needed):
   - List every new table with its columns, types, constraints, and indexes.
   - List every new column on existing tables.
   - Confirm migration number is available.

2. **Go repository layer**:
   - List every new query function: name, SQL shape, parameters, return type.
   - Note which package the function lives in.

3. **Go service layer**:
   - List every new service function: name, inputs, outputs, business rules it enforces.
   - Note error cases that must be handled.

4. **Go handler layer**:
   - List every new HTTP handler: method, route, request body shape, response shape.
   - Note which middleware is required (auth, RBAC permission check, rate limiter).

5. **Frontend API client**:
   - List every new function in `frontend/src/api/{domain}.ts`.
   - Note the React Query key for each.

6. **Frontend components/pages**:
   - List every new page and component.
   - Note props, state, and which React Query hooks are used.

7. **i18n**:
   - List every new key added to `src/locales/*.json`.

8. **Tests**:
   - List each test file and key test cases.

---

## Output

```markdown
## Implementation Plan

### Database
- Migration: `{NNN}_{slug}.sql`
  - New table: `{name}` — {columns summary}
  - New index: `{name}` on `{table}({column})`

### Go Backend — Package `{name}`
- `repository.go`: add `GetUserByEmail(ctx, email) (*User, error)`
- `service.go`: add `Login(ctx, req) (*TokenPair, error)` — validates password, issues JWT
- `handler.go`: add `POST /api/v1/auth/login` handler
- Middleware: none (public endpoint)

### Frontend
- `src/api/auth.ts`: add `login(email, password) → LoginResponse`
- `src/pages/LoginPage.tsx`: new page component
- New i18n keys: `auth.login_title`, `auth.email_label`, `auth.password_label`, `auth.submit`

### Tests
- `internal/auth/service_test.go`: TestLogin_Success, TestLogin_WrongPassword, TestLogin_LockedAccount
- `internal/auth/handler_test.go`: TestLoginHandler_200, TestLoginHandler_401, TestLoginHandler_423
```
