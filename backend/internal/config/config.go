package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded once at startup.
type Config struct {
	// Database
	DBHost              string
	DBPort              string
	DBName              string
	DBUser              string
	DBPassword          string
	DBSSLMode           string
	DBMaxOpenConns      int
	DBMaxIdleConns      int
	DBConnMaxIdleSeconds int
	DBConnMaxLifetime   time.Duration

	// API
	APIPort    string
	APIBaseURL string
	// PublicAppURL is the public base URL of the SPA (used for certificate QR verify links).
	PublicAppURL string

	// JWT
	JWTSecret           string
	JWTAccessTTLMinutes int
	JWTRefreshTTLDays   int

	// Security
	BcryptCost   int
	CookieDomain string
	CookieSecure bool

	// Email / SMTP
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	SMTPTLS        bool
	SMTPFrom       string
	TenantTimezone string

	// Logging
	LogLevel string // debug | info | warn | error; default "info"

	// Anthropic AI (FR-BB71)
	AnthropicAPIKey string
	AnthropicModel  string
}

// Load reads all required environment variables and returns a validated Config.
// It returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	// API_PORT wins; BB_API_PORT is the project-wide port override; default 8080.
	apiPort := getEnv("API_PORT", getEnv("BB_API_PORT", "8080"))

	cfg := &Config{
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBName:       getEnv("DB_NAME", "bilimbaga"),
		DBUser:       getEnv("DB_USER", "bilimbaga"),
		DBPassword:   getEnv("DB_PASSWORD", ""),
		DBSSLMode:    getEnv("DB_SSLMODE", "disable"),
		APIPort:      apiPort,
		APIBaseURL:   getEnv("API_BASE_URL", "http://localhost:"+apiPort),
		PublicAppURL: getEnv("PUBLIC_APP_URL", "http://localhost:5173"),
		JWTSecret:    getEnv("JWT_SECRET", ""),
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

	connMaxLifetimeStr := getEnv("DB_CONN_MAX_LIFETIME", "5m")
	cfg.DBConnMaxLifetime, err = time.ParseDuration(connMaxLifetimeStr)
	if err != nil {
		return nil, fmt.Errorf("config: DB_CONN_MAX_LIFETIME must be a valid duration (e.g. 5m): %w", err)
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

	cfg.SMTPHost = getEnv("SMTP_HOST", "")
	cfg.SMTPPort, err = getEnvInt("SMTP_PORT", 587)
	if err != nil {
		return nil, fmt.Errorf("config: SMTP_PORT: %w", err)
	}
	cfg.SMTPUser = getEnv("SMTP_USER", "")
	cfg.SMTPPass = getEnv("SMTP_PASS", "")
	cfg.SMTPFrom = getEnv("SMTP_FROM", "")
	cfg.TenantTimezone = getEnv("TENANT_TIMEZONE", "UTC")

	smtpTLSStr := getEnv("SMTP_TLS", "false")
	cfg.SMTPTLS, err = strconv.ParseBool(smtpTLSStr)
	if err != nil {
		return nil, fmt.Errorf("config: SMTP_TLS must be true or false: %w", err)
	}

	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	// Anthropic AI (FR-BB71) — optional; handlers return 503 if key is blank.
	cfg.AnthropicAPIKey = getEnv("ANTHROPIC_API_KEY", "")
	cfg.AnthropicModel = getEnv("ANTHROPIC_MODEL", "claude-3-5-haiku-20241022")

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
