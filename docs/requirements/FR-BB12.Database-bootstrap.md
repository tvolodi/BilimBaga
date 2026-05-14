# FR-BB12 — Database Bootstrap

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB12 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB11 |

## Description
Configures PostgreSQL connectivity and the migration runner so that the database schema is always in a known, versioned state before the API begins serving traffic. golang-migrate is embedded into the API binary and executed automatically on startup, eliminating manual migration steps in all environments. The connection pool is tuned via environment variables to prevent connection exhaustion under load.

## Acceptance Criteria
- [ ] AC-1: On API startup, golang-migrate automatically applies all pending SQL files in `migrations/` before the HTTP server begins listening.
- [ ] AC-2: Migration files follow the naming convention `{NNN}_{description}.up.sql` / `{NNN}_{description}.down.sql`; the initial file is `001_init.sql`.
- [ ] AC-3: The `schema_migrations` table (managed by golang-migrate) is present in the database after first startup and correctly records the applied version.
- [ ] AC-4: If a migration fails, the API logs the error with the migration file name and exits with a non-zero status code rather than starting with a broken schema.
- [ ] AC-5: PostgreSQL connection pool is configured from environment variables: `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_IDLE_SECONDS`; defaults are 25, 5, and 300 respectively.
- [ ] AC-6: `db.Ping()` is called after pool creation; if it fails, the API exits with a descriptive error rather than silently degrading.
- [ ] AC-7: Running `make migrate` against an already up-to-date database is a no-op (returns exit 0, logs "no new migrations").
- [ ] AC-8: Down migrations (`*.down.sql`) exist for every up migration and successfully reverse the schema change when applied.

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

#### `migrations/001_init.sql` (up)

```sql
-- Placeholder migration confirming the runner works.
-- Subsequent migrations (FR-BB13 onward) add real tables.
SELECT 1;
```

#### `migrations/001_init.sql` (down)

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
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxIdleTime time.Duration
}

func New(cfg Config) (*sqlx.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password,
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
func main() {
    cfg := config.Load()                            // panics if env vars missing

    db, err := db.New(cfg.DB)
    must(err, "open database")

    must(db.RunMigrations(db, "migrations"), "run migrations")

    r := router.New(cfg, db)
    must(http.ListenAndServe(":"+cfg.APIPort, r), "start server")
}
```

## Notes
- The `migrations/` directory is embedded into the binary with `//go:embed migrations` so the single binary is self-contained.
- Never edit an existing migration file after it has been applied to any environment (development, staging, or production). Always add a new numbered file.
- golang-migrate uses advisory locks to prevent concurrent migrations in multi-instance deployments.
- The `dirty` flag in `schema_migrations` indicates a failed migration; the API should detect this on startup and refuse to proceed, prompting manual intervention.
