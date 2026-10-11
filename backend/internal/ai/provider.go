package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// AI provider names as the config package resolves them (#232).
const (
	AIProviderAnthropic        = "anthropic"
	AIProviderOpenAICompatible = "openai_compatible"
)

// upstreamLogLimit is how many characters of a non-200 body the log keeps (#232).
const upstreamLogLimit = 500

// logUpstreamFailure logs a non-200 answer: the status and at most 500 characters of the body. The key is replaced
// by [redacted] first, so an upstream that echoes it back cannot leak it into the log. The cut counts characters,
// so it never splits one.
func logUpstreamFailure(logger *slog.Logger, provider string, status int, body []byte, apiKey string) {
	text := string(body)
	if apiKey != "" {
		text = strings.ReplaceAll(text, apiKey, "[redacted]")
	}
	if runes := []rune(text); len(runes) > upstreamLogLimit {
		text = string(runes[:upstreamLogLimit])
	}
	logger.Warn("AI provider returned non-200", "provider", provider, "status", status, "body", text)
}

// NewProviderClient returns the AnthropicClient for the resolved provider (#232). Without a provider or a key it
// returns a client that answers ErrAIUnavailable and makes no request, so the AI routes answer 503 AI_UNAVAILABLE.
func NewProviderClient(provider, baseURL, apiKey string, logger *slog.Logger, opts ...ClientOption) AnthropicClient {
	if apiKey == "" {
		return unavailableClient{}
	}
	switch provider {
	case AIProviderAnthropic:
		return NewAnthropicClient(apiKey, logger, opts...)
	case AIProviderOpenAICompatible:
		return NewOpenAICompatibleClient(baseURL, apiKey, logger, opts...)
	default:
		return unavailableClient{}
	}
}

// unavailableClient answers every call with ErrAIUnavailable and makes no request.
type unavailableClient struct{}

func (unavailableClient) GenerateText(context.Context, string, string) (string, int, error) {
	return "", 0, fmt.Errorf("%w: no AI provider is configured", ErrAIUnavailable)
}
