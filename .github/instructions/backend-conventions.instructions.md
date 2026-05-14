---
description: "Use when writing, reviewing, or debugging Go backend code. Covers Chi router patterns, service/repository layer split, error handling, JWT/RBAC middleware, audit logging, and API response conventions. Auto-loaded for all files under backend/."
applyTo: "backend/**"
---

# Backend Conventions — Go 1.22

## Package Structure

One package per domain. No circular imports.

```
backend/
  cmd/api/          main.go — wires everything together
  internal/
    auth/           authentication: handler, service, repository
    users/          user management: handler, service, repository
    departments/    department management
    questions/      question bank
    exams/          exam configuration
    sessions/       exam session delivery and grading
    certificates/   certificate generation
    reports/        analytics and reporting
    audit/          audit log helper
    config/         typed Config struct
    middleware/     JWT, RBAC, rate limiting, recovery
    db/             DB connection pool setup
```

## Architecture Rules

- **Handlers** — thin. Extract request body, validate, call service, write response. No business logic.
- **Services** — business logic only. No SQL. No HTTP types.
- **Repository/queries** — all SQL lives here. Returns domain types. No business logic.
- This separation is non-negotiable. Violations → Code Reviewer FAIL.

## Config

```go
// WRONG — never in handlers or services
secret := os.Getenv("JWT_SECRET")

// RIGHT — inject Config at startup
type Config struct {
    DatabaseURL   string `env:"DATABASE_URL,required"`
    JWTSecret     string `env:"JWT_SECRET,required"`
    BCryptCost    int    `env:"BCRYPT_COST" envDefault:"12"`
}
```

Load once in `main.go`. Pass as dependency to all handlers and services.

## Error Handling

```go
// Always wrap with context
return nil, fmt.Errorf("getUserByEmail: %w", err)

// Never swallow
if err != nil {
    // handle it — do not ignore
}
```

Standard HTTP status codes:
| Code | Use |
|------|-----|
| 200 | OK (reads) |
| 201 | Created |
| 400 | Invalid request body |
| 401 | Missing or invalid token |
| 403 | Valid token, insufficient permission |
| 404 | Resource not found |
| 409 | Conflict (duplicate) |
| 422 | Validation error |
| 423 | Account locked |
| 500 | Unexpected internal error — never leak details |

## API Response Format

```go
// Success
render.JSON(w, r, map[string]any{
    "data":  result,
    "error": nil,
})

// Error
render.Status(r, statusCode)
render.JSON(w, r, map[string]any{
    "data": nil,
    "error": map[string]any{
        "code":    "ERROR_CODE",
        "message": "Human-readable message",
    },
})

// List
render.JSON(w, r, map[string]any{
    "data": items,
    "meta": map[string]any{
        "page":     page,
        "per_page": perPage,
        "total":    total,
    },
    "error": nil,
})
```

## Database

- Use `sqlx` with named parameters: `db.NamedQueryContext(ctx, query, params)`.
- Never concatenate user input into SQL strings.
- All IDs: UUID v4 (`github.com/google/uuid`).
- All timestamps: `time.Time` stored as UTC; serialized as ISO 8601.
- Indexes required for: `(session_id)` on session_answers, `(user_id, exam_id)` on exam_sessions, `(created_at, actor_id)` on audit_log, `(question_id, locale)` on question_translations.

## Migrations

- File naming: `backend/migrations/{NNN}_{slug}.sql` (three-digit zero-padded number).
- **Never edit an existing migration file.** Always add a new numbered file.
- Applied at API container startup via golang-migrate.

## Authentication & RBAC

```go
// Route groups — apply middleware once per group
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(cfg))
    r.Use(middleware.RequirePermission("users", "manage"))
    r.Get("/api/v1/users", usersHandler.List)
})
```

- JWT middleware injects `user_id`, `role`, `department_id` into context.
- `RequirePermission(resource, action)` used as route-level guard.
- Public endpoints (login, tenant config, logo, cert verification) have no auth middleware.

## Audit Log

Write to audit log for every state-changing operation:

```go
audit.Write(ctx, "user.created", "user", newUser.ID, map[string]any{
    "email": newUser.Email,
    "role":  newUser.RoleID,
})
```

## Dev Commands

```bash
cd backend && go run ./cmd/api     # start API
cd backend && go build ./...       # compile check
cd backend && go test ./...        # run all tests
cd backend && go vet ./...         # lint check
make migrate                       # apply pending migrations
```
