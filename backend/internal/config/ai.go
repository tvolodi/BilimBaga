package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// AI provider names (FR-BB71, #232).
const (
	AIProviderAnthropic        = "anthropic"
	AIProviderOpenAICompatible = "openai_compatible"

	// DefaultAnthropicModel is used when neither AI_MODEL nor ANTHROPIC_MODEL is set.
	DefaultAnthropicModel = "claude-3-5-haiku-20241022"

	// XAIBaseURL is the default base URL for an x.ai key (architect-decision 2026-10-11). The model list is at
	// https://docs.x.ai/docs/models, which recommends DefaultXAIModel.
	XAIBaseURL      = "https://api.x.ai/v1"
	DefaultXAIModel = "grok-4.7"
)

// AI request timeout (AI_TIMEOUT_SECONDS, #519). An AI route waits for the provider for up to this long. The container
// nginx and both edge vhosts wait MaxAITimeoutSeconds + 30 s on the /api/v1/admin/ai/ location (router guard test). The
// default is 90 s: behind Cloudflare a response is cut at 100 s with a 524.
const (
	DefaultAITimeoutSeconds = 90
	MaxAITimeoutSeconds     = 150
	minAITimeoutSeconds     = 10
)

// AIConfig is the resolved AI provider. Provider is "" when no usable key is configured: the AI routes answer
// 503 AI_UNAVAILABLE, AI grading stays off, and OffReason names the variables to set (never their values).
type AIConfig struct {
	Provider  string // AIProviderAnthropic, AIProviderOpenAICompatible, or "" (off)
	BaseURL   string // openai_compatible only
	APIKey    string // a secret: String never prints it
	Model     string
	Timeout   time.Duration // one provider call, from AI_TIMEOUT_SECONDS (default DefaultAITimeoutSeconds)
	OffReason string
}

// String keeps the key out of %v, %+v and %s output.
func (a AIConfig) String() string {
	key := "unset"
	if a.APIKey != "" {
		key = "set"
	}
	return fmt.Sprintf("AIConfig{provider=%q baseURL=%q model=%q timeout=%s apiKey=%s}",
		a.Provider, a.BaseURL, a.Model, a.Timeout, key)
}

// resolveAI reads the AI settings from the environment. get reads one variable ("" when unset). Errors name a variable
// and never print a value (#232, #519).
func resolveAI(get func(string) string) (AIConfig, error) {
	timeout, err := parseAITimeout(strings.TrimSpace(get("AI_TIMEOUT_SECONDS")))
	if err != nil {
		return AIConfig{}, err
	}
	cfg, err := resolveAIProvider(get)
	if err != nil {
		return AIConfig{}, err
	}
	cfg.Timeout = timeout
	return cfg, nil
}

// parseAITimeout reads AI_TIMEOUT_SECONDS: a whole number of seconds from minAITimeoutSeconds to MaxAITimeoutSeconds.
// Unset means DefaultAITimeoutSeconds.
func parseAITimeout(raw string) (time.Duration, error) {
	if raw == "" {
		return DefaultAITimeoutSeconds * time.Second, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < minAITimeoutSeconds || n > MaxAITimeoutSeconds {
		return 0, fmt.Errorf("config: AI_TIMEOUT_SECONDS must be a whole number from %d to %d",
			minAITimeoutSeconds, MaxAITimeoutSeconds)
	}
	return time.Duration(n) * time.Second, nil
}

// resolveAIProvider picks the provider from the environment. get reads one variable ("" when unset). Errors name a
// variable and never print a value (#232).
//
// Rules (architect-decision 2026-10-10 and 2026-10-11):
//   - AI_PROVIDER, when set, must be anthropic or openai_compatible.
//   - With AI_PROVIDER unset: ANTHROPIC_API_KEY selects anthropic; otherwise an x.ai key (XAI_API_KEY or
//     xai_api_key) selects openai_compatible at api.x.ai; otherwise AI is off.
//   - anthropic uses AI_API_KEY, then ANTHROPIC_API_KEY. An x.ai-named key is never an Anthropic key.
//   - openai_compatible uses AI_API_KEY, then an x.ai-named key only when the base URL host is api.x.ai.
//     AI_MODEL is required unless the host is api.x.ai, where DefaultXAIModel applies.
//   - AI_BASE_URL must be https, or http for localhost, and must carry no credentials.
func resolveAIProvider(get func(string) string) (AIConfig, error) {
	val := func(key string) string { return strings.TrimSpace(get(key)) }

	raw := strings.ToLower(val("AI_PROVIDER"))
	if raw != "" && raw != AIProviderAnthropic && raw != AIProviderOpenAICompatible {
		return AIConfig{}, errors.New("config: AI_PROVIDER must be anthropic or openai_compatible")
	}
	anthropicKey := val("ANTHROPIC_API_KEY")
	explicitKey := val("AI_API_KEY")
	xaiKey := firstNonEmpty(val("XAI_API_KEY"), val("xai_api_key"))
	baseURL := val("AI_BASE_URL")
	model := val("AI_MODEL")

	provider := raw
	if provider == "" {
		switch {
		case anthropicKey != "":
			provider = AIProviderAnthropic
		case xaiKey != "":
			provider = AIProviderOpenAICompatible
		default:
			return aiOff("AI is off: set ANTHROPIC_API_KEY, or AI_PROVIDER=openai_compatible with AI_API_KEY (or XAI_API_KEY for api.x.ai)"), nil
		}
	}

	if provider == AIProviderAnthropic {
		key := firstNonEmpty(explicitKey, anthropicKey)
		if key == "" {
			return aiOff("AI is off: AI_PROVIDER=anthropic needs ANTHROPIC_API_KEY or AI_API_KEY"), nil
		}
		return AIConfig{
			Provider: AIProviderAnthropic,
			APIKey:   key,
			Model:    firstNonEmpty(model, val("ANTHROPIC_MODEL"), DefaultAnthropicModel),
		}, nil
	}

	base := firstNonEmpty(baseURL, XAIBaseURL)
	host, err := checkBaseURL(base)
	if err != nil {
		return AIConfig{}, err
	}
	if model == "" {
		if host != "api.x.ai" {
			return AIConfig{}, errors.New("config: AI_PROVIDER=openai_compatible at a base URL other than api.x.ai needs AI_MODEL")
		}
		model = DefaultXAIModel
	}
	key := explicitKey
	if key == "" && host == "api.x.ai" {
		key = xaiKey
	}
	if key == "" {
		return aiOff("AI is off: openai_compatible needs AI_API_KEY (or XAI_API_KEY when AI_BASE_URL is api.x.ai)"), nil
	}
	return AIConfig{Provider: AIProviderOpenAICompatible, BaseURL: base, APIKey: key, Model: model}, nil
}

// checkBaseURL returns the lower-case host of an AI base URL, or an error that names AI_BASE_URL and not its value.
func checkBaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return "", errors.New("config: AI_BASE_URL must be an absolute URL without credentials")
	}
	switch {
	case u.Scheme == "https":
	case u.Scheme == "http" && strings.EqualFold(u.Hostname(), "localhost"):
	default:
		return "", errors.New("config: AI_BASE_URL must use https (plain http is allowed for localhost only)")
	}
	return strings.ToLower(u.Hostname()), nil
}

func aiOff(reason string) AIConfig {
	return AIConfig{OffReason: reason}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
