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
	"text/template"
	"time"
)

// AnthropicClient defines the interface for communicating with the Anthropic API.
// Keeping it as an interface enables easy mocking in tests.
type AnthropicClient interface {
	// GenerateText sends a prompt to the model and returns the text content and total tokens used.
	GenerateText(ctx context.Context, prompt, model string) (text string, tokensUsed int, err error)
}

// httpAnthropicClient is the production implementation backed by the Anthropic Messages API.
type httpAnthropicClient struct {
	apiKey  string
	httpCli *http.Client
	logger  *slog.Logger
}

// NewAnthropicClient returns a production AnthropicClient using the given API key.
func NewAnthropicClient(apiKey string, logger *slog.Logger) AnthropicClient {
	return &httpAnthropicClient{
		apiKey:  apiKey,
		httpCli: &http.Client{Timeout: 60 * time.Second},
		logger:  logger,
	}
}

// anthropicRequest is the JSON body sent to POST /v1/messages.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the JSON body returned by POST /v1/messages.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *httpAnthropicClient) GenerateText(ctx context.Context, prompt, model string) (string, int, error) {
	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: 4096,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("ai: client: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, fmt.Errorf("ai: client: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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
		c.logger.Warn("anthropic API returned non-200", "status", resp.StatusCode, "body", string(respBytes))
		return "", 0, fmt.Errorf("%w: HTTP %d", ErrAIUnavailable, resp.StatusCode)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", 0, fmt.Errorf("%w: unmarshal response: %v", ErrAIUnavailable, err)
	}

	if apiResp.Error != nil {
		return "", 0, fmt.Errorf("%w: %s", ErrAIUnavailable, apiResp.Error.Message)
	}

	text := ""
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	totalTokens := apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens
	return text, totalTokens, nil
}

// BuildQuestionsPrompt renders the prompt template with the given request data.
func BuildQuestionsPrompt(req GenerateQuestionsRequest) (string, error) {
	tmpl, err := template.New("questions").Parse(generateQuestionsPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("ai: build prompt: parse template: %w", err)
	}

	data := struct {
		Count        int
		CategoryName string
		Difficulty   string
		ContextText  string
	}{
		Count:        req.Count,
		CategoryName: req.CategoryName,
		Difficulty:   req.Difficulty,
		ContextText:  req.ContextText,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("ai: build prompt: execute template: %w", err)
	}
	return buf.String(), nil
}
