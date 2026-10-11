package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// openAICompatibleClient calls any provider that accepts POST {base}/chat/completions with a Bearer key (#232).
// x.ai documents this path (architect-decision 2026-10-11), and most other providers accept the same shape.
type openAICompatibleClient struct {
	baseURL string
	apiKey  string
	httpCli *http.Client
	logger  *slog.Logger
}

// NewOpenAICompatibleClient returns an AnthropicClient for {baseURL}/chat/completions, for example
// https://api.x.ai/v1. The key is sent only in the Authorization header and never logged.
func NewOpenAICompatibleClient(baseURL, apiKey string, logger *slog.Logger) AnthropicClient {
	return &openAICompatibleClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpCli: &http.Client{Timeout: 60 * time.Second},
		logger:  logger,
	}
}

type openAIRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse is the JSON body returned by POST /chat/completions.
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateText sends one user message and returns the first choice's text and the total tokens used. Transport,
// status, parse and empty-answer failures are all ErrAIUnavailable, as for the Anthropic client.
func (c *openAICompatibleClient) GenerateText(ctx context.Context, prompt, model string) (string, int, error) {
	bodyBytes, err := json.Marshal(openAIRequest{
		Model:     model,
		MaxTokens: 4096,
		Messages:  []openAIMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", 0, fmt.Errorf("ai: client: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, fmt.Errorf("ai: client: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %v", ErrAIUnavailable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("ai: client: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logUpstreamFailure(c.logger, AIProviderOpenAICompatible, resp.StatusCode, respBytes, c.apiKey)
		return "", 0, fmt.Errorf("%w: HTTP %d", ErrAIUnavailable, resp.StatusCode)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", 0, fmt.Errorf("%w: unmarshal response: %v", ErrAIUnavailable, err)
	}
	if apiResp.Error != nil {
		return "", 0, fmt.Errorf("%w: %s", ErrAIUnavailable, apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return "", 0, fmt.Errorf("%w: no choices in response", ErrAIUnavailable)
	}

	return apiResp.Choices[0].Message.Content, apiResp.Usage.PromptTokens + apiResp.Usage.CompletionTokens, nil
}
