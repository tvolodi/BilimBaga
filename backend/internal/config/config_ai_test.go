package config_test

import (
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

func TestLoad_AIInsightsDailyLimit(t *testing.T) {
	t.Setenv("JWT_SECRET", "this-is-a-valid-32-character-secret!")

	cfg, err := config.Load()
	if err != nil || cfg.AIInsightsDailyLimit != 50 {
		t.Fatalf("default want 50, got %+v err=%v", cfg, err)
	}

	t.Setenv("AI_INSIGHTS_DAILY_LIMIT", "7")
	if cfg, err = config.Load(); err != nil || cfg.AIInsightsDailyLimit != 7 {
		t.Fatalf("want 7, got %+v err=%v", cfg, err)
	}

	t.Setenv("AI_INSIGHTS_DAILY_LIMIT", "0")
	if cfg, err = config.Load(); err != nil || cfg.AIInsightsDailyLimit != 0 {
		t.Fatalf("0 (disabled) must load, got %+v err=%v", cfg, err)
	}

	for _, bad := range []string{"-1", "abc"} {
		t.Setenv("AI_INSIGHTS_DAILY_LIMIT", bad)
		if _, err := config.Load(); err == nil {
			t.Errorf("AI_INSIGHTS_DAILY_LIMIT=%q must be rejected", bad)
		}
	}
}

func TestLoad_ExportMaxRows(t *testing.T) {
	t.Setenv("JWT_SECRET", "this-is-a-valid-32-character-secret!")

	cfg, err := config.Load()
	if err != nil || cfg.ExportMaxRows != 200000 {
		t.Fatalf("default want 200000, got %+v err=%v", cfg, err)
	}
	t.Setenv("EXPORT_MAX_ROWS", "500")
	if cfg, err = config.Load(); err != nil || cfg.ExportMaxRows != 500 {
		t.Fatalf("want 500, got %+v err=%v", cfg, err)
	}
	for _, bad := range []string{"0", "-1", "abc"} {
		t.Setenv("EXPORT_MAX_ROWS", bad)
		if _, err := config.Load(); err == nil {
			t.Errorf("EXPORT_MAX_ROWS=%q must be rejected", bad)
		}
	}
}
