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
