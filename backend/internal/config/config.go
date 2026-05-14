package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded once at startup.
type Config struct {
	// Database
	DBHost               string
	DBPort               string
	DBName               string
	DBUser               string
	DBPassword           string
	DBSSLMode            string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxIdleSeconds int

	// API
	APIPort    string
	APIBaseURL string

	// JWT
	JWTSecret           string
	JWTAccessTTLMinutes int
	JWTRefreshTTLDays   int

	// Security
	BcryptCost    int
	CookieDomain  string
	CookieSecure  bool
}

// Load reads all required environment variables and returns a validated Config.
// It returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	cfg := &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "bilimbaga"),
		DBUser:     getEnv("DB_USER", "bilimbaga"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		APIPort:    getEnv("API_PORT", "8080"),
		APIBaseURL: getEnv("API_BASE_URL", "http://localhost:8080"),
		JWTSecret:  getEnv("JWT_SECRET", ""),
		CookieDomain: getEnv("COOKIE_DOMAIN", "localhost"),
	}

	var err error

	cfg.DBMaxOpenConns, err = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return nil, fmt.Errorf("config: DB_MAX_OPEN_CONNS: %w", err)
	}

	cfg.DBMaxIdleConns, err = getEnvInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return nil, fmt.Errorf("config: DB_MAX_IDLE_CONNS: %w", err)
	}

	cfg.DBConnMaxIdleSeconds, err = getEnvInt("DB_CONN_MAX_IDLE_SECONDS", 300)
	if err != nil {
		return nil, fmt.Errorf("config: DB_CONN_MAX_IDLE_SECONDS: %w", err)
	}

	cfg.JWTAccessTTLMinutes, err = getEnvInt("JWT_ACCESS_TTL_MINUTES", 15)
	if err != nil {
		return nil, fmt.Errorf("config: JWT_ACCESS_TTL_MINUTES: %w", err)
	}

	cfg.JWTRefreshTTLDays, err = getEnvInt("JWT_REFRESH_TTL_DAYS", 7)
	if err != nil {
		return nil, fmt.Errorf("config: JWT_REFRESH_TTL_DAYS: %w", err)
	}

	cfg.BcryptCost, err = getEnvInt("BCRYPT_COST", 12)
	if err != nil {
		return nil, fmt.Errorf("config: BCRYPT_COST: %w", err)
	}

	cookieSecureStr := getEnv("COOKIE_SECURE", "false")
	cfg.CookieSecure, err = strconv.ParseBool(cookieSecureStr)
	if err != nil {
		return nil, fmt.Errorf("config: COOKIE_SECURE must be true or false: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required fields meet their constraints.
func (c *Config) validate() error {
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("config: JWT_SECRET must be at least 32 characters (got %d); set a strong secret before starting the server", len(c.JWTSecret))
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", v)
	}
	return n, nil
}
