package db_test

import (
	"testing"
	"time"

	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
)

// TestConfigDefaults verifies the db.Config struct holds the expected field values
// when constructed with explicit values (no live connection required).
func TestConfigDefaults(t *testing.T) {
	cfg := dbpkg.Config{
		Host:            "localhost",
		Port:            "5432",
		Name:            "testdb",
		User:            "testuser",
		Password:        "secret",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxIdleTime: 300 * time.Second,
	}

	if cfg.SSLMode != "disable" {
		t.Errorf("expected SSLMode=disable, got %q", cfg.SSLMode)
	}
	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns=25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns=5, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxIdleTime != 300*time.Second {
		t.Errorf("expected ConnMaxIdleTime=300s, got %v", cfg.ConnMaxIdleTime)
	}
}

// TestNewFailsOnBadDSN verifies that db.New returns an error when the DSN
// points to an unreachable host, rather than panicking or returning a nil DB.
func TestNewFailsOnBadDSN(t *testing.T) {
	cfg := dbpkg.Config{
		Host:            "127.0.0.1",
		Port:            "1", // port 1 is never open
		Name:            "nonexistent",
		User:            "nobody",
		Password:        "",
		SSLMode:         "disable",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxIdleTime: time.Second,
	}

	_, err := dbpkg.New(cfg)
	if err == nil {
		t.Fatal("expected error when connecting to unreachable host, got nil")
	}
}
