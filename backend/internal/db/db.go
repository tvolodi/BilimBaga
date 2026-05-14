package db

import (
	"fmt"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Connect opens a sqlx connection pool using the provided config and verifies
// the connection with a ping.
func Connect(cfg *config.Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword,
	)

	pool, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}

	pool.SetMaxOpenConns(cfg.DBMaxOpenConns)
	pool.SetMaxIdleConns(cfg.DBMaxIdleConns)
	pool.SetConnMaxIdleTime(time.Duration(cfg.DBConnMaxIdleSeconds) * time.Second)

	return pool, nil
}
