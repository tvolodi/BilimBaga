package config_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

// #232 (architect-decision 2026-10-11): AI provider resolution. The keys below are fixed test strings. No assertion
// prints a key value on failure, and the tests check that errors and String output never contain one.

const (
	testXAIKey       = "xai-test-key-do-not-print-0001"
	testAnthropicKey = "sk-ant-test-key-do-not-print-0002"
	testExplicitKey  = "explicit-test-key-do-not-print-0003"
)

// aiEnv names every variable the AI resolution reads, so a developer's shell cannot change a test.
var aiEnv = []string{
	"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "AI_PROVIDER", "AI_BASE_URL", "AI_API_KEY", "AI_MODEL",
	"XAI_API_KEY", "xai_api_key",
}

// loadAI loads the config with only the given AI variables set, and returns the resolved AI settings.
func loadAI(t *testing.T, env map[string]string) (config.AIConfig, error) {
	t.Helper()
	t.Setenv("JWT_SECRET", "this-is-a-valid-32-character-secret!")
	for _, name := range aiEnv {
		t.Setenv(name, "")
	}
	for name, value := range env {
		t.Setenv(name, value)
	}
	cfg, err := config.Load()
	if err != nil {
		return config.AIConfig{}, err
	}
	return cfg.AI, nil
}

func TestAI_AnthropicWhenAnthropicKeyIsSet(t *testing.T) {
	ai, err := loadAI(t, map[string]string{"ANTHROPIC_API_KEY": testAnthropicKey})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if ai.Provider != config.AIProviderAnthropic || ai.APIKey != testAnthropicKey || ai.Model != "claude-3-5-haiku-20241022" {
		t.Fatalf("want anthropic with the default model, got provider=%q model=%q", ai.Provider, ai.Model)
	}

	ai, err = loadAI(t, map[string]string{"ANTHROPIC_API_KEY": testAnthropicKey, "ANTHROPIC_MODEL": "claude-custom"})
	if err != nil || ai.Model != "claude-custom" {
		t.Fatalf("ANTHROPIC_MODEL must set the model, got %q err=%v", ai.Model, err)
	}
}

func TestAI_XAIKeyInfersOpenAICompatibleAtXAI(t *testing.T) {
	ai, err := loadAI(t, map[string]string{"XAI_API_KEY": testXAIKey})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if ai.Provider != config.AIProviderOpenAICompatible || ai.BaseURL != "https://api.x.ai/v1" ||
		ai.Model != "grok-4.7" || ai.APIKey != testXAIKey {
		t.Fatalf("want openai_compatible at api.x.ai with grok-4.7, got provider=%q base=%q model=%q", ai.Provider, ai.BaseURL, ai.Model)
	}
}

func TestAI_LowercaseXAIKeyNameIsRead(t *testing.T) {
	ai, err := loadAI(t, map[string]string{"xai_api_key": testXAIKey})
	if err != nil || ai.Provider != config.AIProviderOpenAICompatible || ai.APIKey != testXAIKey {
		t.Fatalf("xai_api_key must infer openai_compatible, got provider=%q err=%v", ai.Provider, err)
	}
}

func TestAI_ExplicitAPIKeyWinsOverXAIAliases(t *testing.T) {
	ai, err := loadAI(t, map[string]string{
		"AI_PROVIDER": "openai_compatible",
		"AI_BASE_URL": "https://api.x.ai/v1",
		"AI_API_KEY":  testExplicitKey,
		"XAI_API_KEY": testXAIKey,
	})
	if err != nil || ai.APIKey != testExplicitKey {
		t.Fatalf("AI_API_KEY must win, got key set=%t err=%v", ai.APIKey != "", err)
	}
}

// Refusal 1 (architect 2026-10-11): an x.ai-named key goes only to api.x.ai.
func TestAI_XAIKeyIsRefusedForAnotherHost(t *testing.T) {
	ai, err := loadAI(t, map[string]string{
		"AI_PROVIDER": "openai_compatible",
		"AI_BASE_URL": "https://example.com/v1",
		"AI_MODEL":    "some-model",
		"XAI_API_KEY": testXAIKey,
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if ai.Provider != "" || ai.APIKey != "" {
		t.Fatalf("an x.ai key must not reach another host: provider=%q", ai.Provider)
	}
	if !strings.Contains(ai.OffReason, "AI_API_KEY") || strings.Contains(ai.OffReason, testXAIKey) {
		t.Fatalf("the off reason must name AI_API_KEY and no value, got %q", ai.OffReason)
	}
}

// Refusal 2 (architect 2026-10-11): an x.ai-named key is never the Anthropic key.
func TestAI_XAIKeyIsNeverUsedAsAnthropicKey(t *testing.T) {
	ai, err := loadAI(t, map[string]string{"AI_PROVIDER": "anthropic", "XAI_API_KEY": testXAIKey})
	if err != nil || ai.Provider != "" || ai.APIKey != "" {
		t.Fatalf("explicit anthropic with only an x.ai key must be off: provider=%q err=%v", ai.Provider, err)
	}

	ai, err = loadAI(t, map[string]string{"ANTHROPIC_API_KEY": testAnthropicKey, "XAI_API_KEY": testXAIKey})
	if err != nil || ai.Provider != config.AIProviderAnthropic || ai.APIKey != testAnthropicKey {
		t.Fatalf("an Anthropic key must beat an x.ai key when the provider is unset, got provider=%q err=%v", ai.Provider, err)
	}
}

func TestAI_UnknownProviderFailsNamingTheVariableNotTheValue(t *testing.T) {
	_, err := loadAI(t, map[string]string{"AI_PROVIDER": "bogus-provider-value"})
	if err == nil || !strings.Contains(err.Error(), "AI_PROVIDER") || strings.Contains(err.Error(), "bogus-provider-value") {
		t.Fatalf("an unknown AI_PROVIDER must fail naming AI_PROVIDER and not its value, got %v", err)
	}
}

func TestAI_BaseURLMustBeHTTPSExceptLocalhost(t *testing.T) {
	for _, bad := range []string{
		"http://example.com/v1",
		"ftp://example.com/v1",
		"https://user:pw-do-not-print@api.x.ai/v1",
	} {
		_, err := loadAI(t, map[string]string{
			"AI_PROVIDER": "openai_compatible", "AI_BASE_URL": bad, "AI_API_KEY": testExplicitKey, "AI_MODEL": "m",
		})
		if err == nil || !strings.Contains(err.Error(), "AI_BASE_URL") || strings.Contains(err.Error(), bad) ||
			strings.Contains(err.Error(), "pw-do-not-print") {
			t.Errorf("base %q must fail naming AI_BASE_URL and not the value, got %v", bad, err)
		}
	}

	ai, err := loadAI(t, map[string]string{
		"AI_PROVIDER": "openai_compatible", "AI_BASE_URL": "http://localhost:8080/v1", "AI_API_KEY": testExplicitKey, "AI_MODEL": "m",
	})
	if err != nil || ai.Provider != config.AIProviderOpenAICompatible {
		t.Fatalf("plain http is allowed for localhost, got err=%v", err)
	}
}

func TestAI_OpenAICompatibleAtAnotherHostNeedsAModel(t *testing.T) {
	_, err := loadAI(t, map[string]string{
		"AI_PROVIDER": "openai_compatible", "AI_BASE_URL": "https://example.com/v1", "AI_API_KEY": testExplicitKey,
	})
	if err == nil || !strings.Contains(err.Error(), "AI_MODEL") || strings.Contains(err.Error(), testExplicitKey) {
		t.Fatalf("an explicit base URL without AI_MODEL must fail naming AI_MODEL, got %v", err)
	}
}

func TestAI_ExplicitModelOverridesTheXAIDefault(t *testing.T) {
	ai, err := loadAI(t, map[string]string{"XAI_API_KEY": testXAIKey, "AI_MODEL": "grok-custom"})
	if err != nil || ai.Model != "grok-custom" {
		t.Fatalf("AI_MODEL must override DefaultXAIModel, got %q err=%v", ai.Model, err)
	}
}

func TestAI_NoKeyStartsOffWithAReason(t *testing.T) {
	ai, err := loadAI(t, nil)
	if err != nil {
		t.Fatalf("no key is not a startup failure: %v", err)
	}
	if ai.Provider != "" || ai.APIKey != "" || !strings.Contains(ai.OffReason, "ANTHROPIC_API_KEY") {
		t.Fatalf("want off with a reason that names the variables, got provider=%q reason=%q", ai.Provider, ai.OffReason)
	}
}

func TestAI_StringAndFormattingNeverPrintTheKey(t *testing.T) {
	ai, err := loadAI(t, map[string]string{"XAI_API_KEY": testXAIKey})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	out := fmt.Sprintf("%v %+v %s", ai, ai, ai)
	if strings.Contains(out, testXAIKey) {
		t.Fatalf("formatting the AI settings printed the key")
	}
}
