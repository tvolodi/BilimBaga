package config_test

import (
	"strings"
	"testing"
	"time"
)

// #519: AI_TIMEOUT_SECONDS bounds one provider call. Its largest accepted value plus 30 s must fit under the nginx
// insights location, which the router guard test checks.

func TestAI_TimeoutDefaultsTo120Seconds(t *testing.T) {
	ai, err := loadAI(t, map[string]string{"ANTHROPIC_API_KEY": testAnthropicKey})
	if err != nil || ai.Timeout != 120*time.Second {
		t.Fatalf("default timeout want 120s, got %v err=%v", ai.Timeout, err)
	}
}

func TestAI_TimeoutIsReadFromEnv(t *testing.T) {
	ai, err := loadAI(t, map[string]string{"ANTHROPIC_API_KEY": testAnthropicKey, "AI_TIMEOUT_SECONDS": "90"})
	if err != nil || ai.Timeout != 90*time.Second {
		t.Fatalf("want 90s, got %v err=%v", ai.Timeout, err)
	}
}

// The bad values avoid the digits of the error text, so a value echoed into the message would be caught.
func TestAI_TimeoutOutOfRangeFailsNamingTheVariableNotTheValue(t *testing.T) {
	for _, bad := range []string{"8", "151", "abc"} {
		_, err := loadAI(t, map[string]string{"ANTHROPIC_API_KEY": testAnthropicKey, "AI_TIMEOUT_SECONDS": bad})
		if err == nil || !strings.Contains(err.Error(), "AI_TIMEOUT_SECONDS") || strings.Contains(err.Error(), bad) {
			t.Errorf("AI_TIMEOUT_SECONDS=%q must fail naming the variable and not its value, got %v", bad, err)
		}
	}
}
