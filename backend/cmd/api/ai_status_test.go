package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

// #510: the startup log names the AI provider and model once, and never the key. With AI off it is one warning.

func TestLogAIStatus_ProviderLineNamesProviderAndModelNotTheKey(t *testing.T) {
	const key = "xai-startup-test-key-do-not-print-0005"
	var buf bytes.Buffer
	logAIStatus(slog.New(slog.NewTextHandler(&buf, nil)), config.AIConfig{
		Provider: config.AIProviderOpenAICompatible,
		BaseURL:  "https://api.x.ai/v1",
		APIKey:   key,
		Model:    "grok-4.7",
	})
	out := buf.String()
	if !strings.Contains(out, "level=INFO") || !strings.Contains(out, "provider=openai_compatible") ||
		!strings.Contains(out, "model=grok-4.7") {
		t.Fatalf("want one INFO line with the provider and model, got %q", out)
	}
	if strings.Contains(out, key) {
		t.Fatalf("the startup line must not carry the key")
	}
}

func TestLogAIStatus_OffIsOneWarningWithTheReason(t *testing.T) {
	var buf bytes.Buffer
	logAIStatus(slog.New(slog.NewTextHandler(&buf, nil)), config.AIConfig{
		OffReason: "AI is off: set ANTHROPIC_API_KEY",
	})
	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, "level=WARN") || !strings.Contains(out, "ANTHROPIC_API_KEY") {
		t.Fatalf("want exactly one WARN line with the reason, got %q", out)
	}
}
