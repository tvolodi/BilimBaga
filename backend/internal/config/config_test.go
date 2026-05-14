package config_test

import (
	"os"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

func TestLoad_ValidConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "this-is-a-valid-32-character-secret!")
	t.Setenv("API_PORT", "8080")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.APIPort != "8080" {
		t.Errorf("expected APIPort=8080, got %s", cfg.APIPort)
	}
	if cfg.JWTSecret != "this-is-a-valid-32-character-secret!" {
		t.Errorf("JWTSecret not loaded correctly")
	}
}

func TestLoad_JWTSecretTooShort(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	t.Setenv("JWT_SECRET", "tooshort")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for short JWT_SECRET, got nil")
	}
}

func TestLoad_JWTSecretAbsent(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is absent, got nil")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "a-secret-that-is-at-least-32-chars!!")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DBMaxOpenConns != 25 {
		t.Errorf("expected DBMaxOpenConns=25, got %d", cfg.DBMaxOpenConns)
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("expected BcryptCost=12, got %d", cfg.BcryptCost)
	}
}

func TestLoad_InvalidIntEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "a-secret-that-is-at-least-32-chars!!")
	t.Setenv("DB_MAX_OPEN_CONNS", "notanumber")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid DB_MAX_OPEN_CONNS, got nil")
	}
}
