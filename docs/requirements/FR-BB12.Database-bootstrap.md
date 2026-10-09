# FR-BB12 — Database Bootstrap

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB12 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | implemented |
| Depends On | FR-BB11 |

## Description
Configures PostgreSQL connectivity and the migration runner so that the database schema is always in a known, versioned state before the API begins serving traffic. golang-migrate is embedded into the API binary and executed automatically on startup, eliminating manual migration steps in all environments. The connection pool is tuned via environment variables to prevent connection exhaustion under load.

## Scope

| Layer | Affected files/packages |
|-------|------------------------|
| Backend | `internal/db/db.go`, `internal/db/migrate.go`, `cmd/api/main.go` |
| Database | `migrations/001_init.up.sql`, `migrations/001_init.down.sql`, `schema_migrations` table |

## Acceptance Criteria
- [ ] AC-1: On API startup, golang-migrate automatically applies all pending SQL files in `migrations/` before the HTTP server begins listening.
- [ ] AC-2: Migration files follow the naming convention `{NNN}_{description}.up.sql` / `{NNN}_{description}.down.sql`; the initial files are `001_init.up.sql` and `001_init.down.sql`.
- [ ] AC-3: The `schema_migrations` table (managed by golang-migrate) is present in the database after first startup and correctly records the applied version.
- [ ] AC-4: If a migration fails, the API logs the error with the migration file name and exits with a non-zero status code rather than starting with a broken schema.
- [ ] AC-5: PostgreSQL connection pool is configured from environment variables: `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_IDLE_SECONDS`; defaults are 25, 5, and 300 respectively.
- [ ] AC-6: `db.Ping()` is called after pool creation; if it fails, the API exits with a descriptive error rather than silently degrading.
- [ ] AC-7: Running `make migrate` against an already up-to-date database is a no-op (returns exit 0, logs "no new migrations").
  - Status 2026-10-09 (PR #198, ISS-146, merged at 96bf413): `make migrate` now works (`Makefile:14-15` runs `docker compose run --rm api migrate`; `backend/cmd/api/run.go:31-43` handles the `migrate` subcommand: apply pending migrations, exit 0, exit 1 on error, exit 2 on unknown args). Exit 0 on an up-to-date database is met (`db/migrate.go:32` treats `ErrNoChange` as success). The "no new migrations" log text is **not** met: the subcommand always prints `migrations applied`, and `ErrNoChange` logs nothing. Migrations still also run automatically at API startup (`main.go:92`), so AC-1 is unchanged and `make migrate` is optional (for example to migrate ahead of a deploy).
- [ ] AC-8: Down migrations (`*.down.sql`) exist for every up migration and successfully reverse the schema change when applied.
- [ ] AC-9: If the `dirty` flag in `schema_migrations` is `true` at startup, the API logs a descriptive error (including the version number) and exits with a non-zero status code rather than serving traffic.
- [ ] AC-10: `DB_SSLMODE` environment variable is read and applied to the PostgreSQL DSN; it defaults to `disable` for local development.
- [ ] AC-11: `backend/.env.example` includes `DB_SSLMODE=disable` in the Database section.

## Technical Specification

### Database Schema

```sql
-- Managed automatically by golang-migrate.
-- The table below is created by the migrate library itself; shown here for reference only.
CREATE TABLE IF NOT EXISTS schema_migrations (
    version  BIGINT  NOT NULL PRIMARY KEY,
    dirty    BOOLEAN NOT NULL
);
```

#### `migrations/001_init.up.sql`

```sql
-- Placeholder migration confirming the runner works.
-- Subsequent migrations (FR-BB13 onward) add real tables.
SELECT 1;
```

#### `migrations/001_init.down.sql`

```sql
SELECT 1;
```

### Configuration / Infrastructure

#### DB package (`internal/db/db.go`)

```go
package db

import (
    "fmt"
    "time"

    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

type Config struct {
    Host            string
    Port            int
    Name            string
    User            string
    Password        string
    SSLMode         string        // defaults to "disable"; set via DB_SSLMODE
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxIdleTime time.Duration
}

func New(cfg Config) (*sqlx.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, cfg.SSLMode,
    )
    db, err := sqlx.Connect("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("db.New connect: %w", err)
    }
    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
    return db, nil
}
```

#### Migration runner (`internal/db/migrate.go`)

```go
package db

import (
    "errors"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/jmoiron/sqlx"
)

// RunMigrations applies all pending up migrations from the migrations/ directory.
// It returns an error on failure; ErrNoChange is treated as success.
func RunMigrations(db *sqlx.DB, migrationsPath string) error {
    driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("migrate: create driver: %w", err)
    }
    m, err := migrate.NewWithDatabaseInstance(
        "file://"+migrationsPath,
        "postgres",
        driver,
    )
    if err != nil {
        return fmt.Errorf("migrate: init: %w", err)
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("migrate: up: %w", err)
    }
    return nil
}
```

#### Startup sequence (`cmd/api/main.go` sketch)

```go
import (
    dbpkg "github.com/your-org/bilimbaga/internal/db"
)

func main() {
    cfg := config.Load()                            // panics if env vars missing

    database, err := dbpkg.New(cfg.DB)
    must(err, "open database")

    must(dbpkg.RunMigrations(database, "migrations"), "run migrations")

    r := router.New(cfg, database)
    must(http.ListenAndServe(":"+cfg.APIPort, r), "start server")
}
```

## Notes
- The `migrations/` directory is read from the **OS filesystem** at runtime via the `file://` source driver. The `migrations/` directory must be present at the path passed to `RunMigrations` (typically relative to the working directory where the binary is executed). The binary is **not** self-contained with respect to migrations.
- Never edit an existing migration file after it has been applied to any environment (development, staging, or production). Always add a new numbered file.
- golang-migrate uses advisory locks to prevent concurrent migrations in multi-instance deployments.
- Environment variables for the DB package: `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE` (default: `disable`), `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_IDLE_SECONDS`.

## Out of Scope
- Schema tables beyond the placeholder `001_init` migration (covered by FR-BB13 onward).
- Seeding initial data (roles, tenants, admin users).
- Multi-tenant schema isolation (separate schemas or databases per tenant).
- Database backup or point-in-time recovery configuration.

## Test Strategy
- **Integration test** (`internal/db/db_test.go`): spin up a real PostgreSQL container (or use `DATABASE_URL` from the test environment), call `db.New` + `RunMigrations`, assert `schema_migrations` contains version 1 and `dirty = false`.
- **Integration test — dirty-flag abort**: manually set `dirty = true` in `schema_migrations`, call the startup sequence, assert the process exits with a non-zero code.
- **Unit test** (`internal/db/config_test.go`): verify that `config.Load()` correctly reads `DB_SSLMODE` from environment and populates `db.Config.SSLMode`, defaulting to `"disable"` when unset.
- **Make target smoke test**: run `make migrate` twice against the test DB; assert the second run exits 0 and logs a no-change message.
