package db

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Config holds all parameters needed to open and tune the PostgreSQL connection pool.
type Config struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	SSLMode         string // defaults to "disable"; set via DB_SSLMODE
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration // AC-8: FR-BB65 — total connection lifetime
}

// New opens a sqlx connection pool using the provided Config, applies pool
// settings, and verifies the connection with a Ping.
func New(cfg Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db.New connect: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	return db, nil
}
