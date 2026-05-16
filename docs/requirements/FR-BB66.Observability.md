# FR-BB66 — Observability

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB66 |
| Phase | 6 — Polish & Hardening |
| Priority | 1 |
| Status | Implemented |
| Depends On | FR-BB11, FR-BB14, FR-BB65 |

## Description
Equips the Go API with structured JSON logging, per-request correlation IDs, a health check endpoint, and panic recovery middleware. Docker Compose is configured with log rotation to prevent unbounded disk usage. The API binary embeds its Git commit SHA at build time for traceability. Slow database queries are surfaced at warn level to assist performance investigation.

## Acceptance Criteria
- [x] AC-1: Every HTTP request produces a structured JSON log line containing at minimum: `method`, `path`, `status_code`, `latency_ms`, `user_id` (authenticated) or `"-"` (anonymous), `ip`, and `request_id` fields.
- [x] AC-2: A UUID request ID is generated per incoming request by middleware, stored in the request context, and included in every log line emitted within that request's lifetime (handler, service, repository layers).
- [x] AC-3: Log verbosity is controlled by the `LOG_LEVEL` environment variable accepting `debug`, `info`, `warn`, `error`; the default level is `info`; invalid values fall back to `info` with a startup warning.
- [x] AC-4: `GET /api/v1/health` responds with no authentication required; returns HTTP 200 and `{ "data": { "status": "ok", "db_ok": true, "version": "<git-sha>" }, "error": null }` when the database is reachable; returns HTTP 503 and `db_ok: false` when `db.Ping()` fails; the `version` field contains the Git commit SHA injected at build time.
- [x] AC-5: Panic recovery middleware wraps all route handlers; any panic is recovered, its full stack trace is logged at error level with the associated `request_id`, and the client receives a `500` response with `{ "data": null, "error": { "code": "INTERNAL_ERROR", "message": "An unexpected error occurred" } }` — no stack trace or internal detail is ever included in the response body.
- [x] AC-6: Docker Compose log driver for all services is configured with `max-size: 100m` and `max-file: 3` so log files do not grow unboundedly.
- [x] AC-7: The API binary version string is injected at build time via linker flags `-ldflags "-X main.Version=$(git rev-parse --short HEAD)"`; the `Makefile` `build` target includes these flags.
- [x] AC-8: Any database query whose execution time exceeds 500ms is logged at warn level with the query text sanitized (parameter bind values are replaced with `?` placeholders, not logged); the log line includes `latency_ms` and `query` fields.

## Technical Specification

### Configuration / Infrastructure

**New environment variable**:
```
LOG_LEVEL=info   # debug | info | warn | error
```

**Docker Compose log configuration** (`deploy/docker-compose.yml`):
```yaml
services:
  api:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "3"
  frontend:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "3"
  db:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "3"
```

**Makefile build target**:
```makefile
build:
	cd backend && go build \
	  -ldflags "-X main.Version=$$(git rev-parse --short HEAD)" \
	  -o bin/api ./cmd/api
```

### API Endpoints

#### GET /api/v1/health
- **Auth**: none
- **Success response** `200`:
  ```json
  {
    "data": { "status": "ok", "db_ok": true, "version": "a1b2c3d" },
    "error": null
  }
  ```
- **DB failure response** `503`:
  ```json
  {
    "data": { "status": "degraded", "db_ok": false, "version": "a1b2c3d" },
    "error": null
  }
  ```

### Implementation Details

**Library**: `github.com/rs/zerolog` for structured JSON logging.

**Logger initialization** (`cmd/api/main.go`):
```go
var Version = "dev" // overridden by -ldflags

func initLogger(level string) zerolog.Logger {
    lvl, err := zerolog.ParseLevel(level)
    if err != nil {
        lvl = zerolog.InfoLevel
        // emit startup warning after logger is created
    }
    zerolog.SetGlobalLevel(lvl)
    return zerolog.New(os.Stdout).With().Timestamp().Str("version", Version).Logger()
}
```

**Request ID middleware** (`internal/middleware/request_id.go`):
```go
type contextKey string
const RequestIDKey contextKey = "request_id"

func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := uuid.NewString()
        ctx := context.WithValue(r.Context(), RequestIDKey, id)
        w.Header().Set("X-Request-ID", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(RequestIDKey).(string); ok {
        return id
    }
    return "-"
}
```

**Request logger middleware** (`internal/middleware/logger.go`):
```go
func RequestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
            next.ServeHTTP(ww, r)

            userID := "-"
            if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
                userID = claims.UserID.String()
            }

            logger.Info().
                Str("request_id", GetRequestID(r.Context())).
                Str("method", r.Method).
                Str("path", r.URL.Path).
                Int("status_code", ww.Status()).
                Int64("latency_ms", time.Since(start).Milliseconds()).
                Str("user_id", userID).
                Str("ip", realIP(r)).
                Msg("request")
        })
    }
}
```

**Panic recovery middleware** (`internal/middleware/recovery.go`):
```go
func Recovery(logger zerolog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if rv := recover(); rv != nil {
                    logger.Error().
                        Str("request_id", GetRequestID(r.Context())).
                        Interface("panic", rv).
                        Bytes("stack", debug.Stack()).
                        Msg("panic recovered")
                    render.Status(r, http.StatusInternalServerError)
                    render.JSON(w, r, api.ErrorResponse("INTERNAL_ERROR", "An unexpected error occurred"))
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}
```

**Slow query wrapper** (`internal/database/instrumented.go`):
```go
func LogSlowQuery(ctx context.Context, logger zerolog.Logger, query string, start time.Time) {
    elapsed := time.Since(start)
    if elapsed > 500*time.Millisecond {
        sanitized := sanitizeQuery(query) // replace $1, $2 with ?
        logger.Warn().
            Str("request_id", middleware.GetRequestID(ctx)).
            Int64("latency_ms", elapsed.Milliseconds()).
            Str("query", sanitized).
            Msg("slow query")
    }
}
```

Called by wrapping repository methods with `defer LogSlowQuery(ctx, logger, query, time.Now())`.

**Logger propagation**: the initialized `zerolog.Logger` is stored in the request context by `RequestLogger` middleware using `logger.With().Str("request_id", ...).Logger()` and retrieved in service/repository layers via a helper `LoggerFromContext(ctx)`. This ensures every log line within a request carries the `request_id` without manual threading.

**Middleware chain order** in `cmd/api/main.go`:
```go
r.Use(middleware.RequestID)       // 1. assign request ID
r.Use(middleware.Recovery(log))   // 2. recover panics (wraps everything)
r.Use(middleware.RequestLogger(log)) // 3. log after response written
r.Use(middleware.RealIP)
r.Use(middleware.Heartbeat("/ping"))
```

## Notes
- `zerolog` outputs NDJSON (newline-delimited JSON); Docker's `json-file` log driver will wrap each line in an additional JSON envelope. Log aggregators (Grafana Loki, Datadog, etc.) can parse both layers.
- The health endpoint deliberately returns 200-level response shape even for 503 status to allow load balancers that only check response body to detect degraded state via `db_ok`.
- Slow query sanitization must not use regex on untrusted SQL; a simple positional-parameter replacement (`$\d+` → `?`) is safe and sufficient.
- `debug.Stack()` in the panic recovery is stored as bytes in the log, not sent to the client. Ensure the zerolog JSON output does not encode the bytes as base64 — use `.Str("stack", string(debug.Stack()))` for readability.
