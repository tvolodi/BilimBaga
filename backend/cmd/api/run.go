package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/bilimbaga/bilimbaga/internal/config"
	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
)

const usage = "usage: api [migrate]\n  (no args)  start the API server (applies pending migrations at startup)\n  migrate    apply pending database migrations and exit\n"

// migrator applies pending database migrations.
type migrator interface {
	Migrate() error
}

// deps are the side-effecting collaborators of run, injectable for tests.
type deps struct {
	serve    func()
	migrator migrator
	stdout   io.Writer
	stderr   io.Writer
}

// run dispatches on args (os.Args[1:]) and returns the process exit code.
func run(args []string, d deps) int {
	switch {
	case len(args) == 0:
		d.serve()
		return 0
	case args[0] == "migrate" && len(args) == 1:
		if err := d.migrator.Migrate(); err != nil {
			fmt.Fprintf(d.stderr, "migrate error: %v\n", err)
			return 1
		}
		fmt.Fprintln(d.stdout, "migrations applied")
		return 0
	default:
		fmt.Fprint(d.stderr, usage)
		return 2
	}
}

// openDB opens the database from the typed Config.
func openDB(cfg *config.Config) (*sqlx.DB, error) {
	db, err := dbpkg.New(dbpkg.Config{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		Name:            cfg.DBName,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		SSLMode:         cfg.DBSSLMode,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxIdleTime: time.Duration(cfg.DBConnMaxIdleSeconds) * time.Second,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// dbMigrator is the production migrator: loads Config, opens the DB, runs migrations.
// It never starts the HTTP server, schedulers, email jobs or the admin bootstrap.
type dbMigrator struct{}

func (dbMigrator) Migrate() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	db, err := openDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := dbpkg.RunMigrations(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], deps{serve: serve, migrator: dbMigrator{}, stdout: os.Stdout, stderr: os.Stderr}))
}
