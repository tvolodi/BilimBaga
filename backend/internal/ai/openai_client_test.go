package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// #232: the second AI client (POST {base}/chat/completions) and the upstream log rules. The key is a fixed test
// string; no assertion prints it, and the log tests check it never appears in captured output.

const testOpenAIKey = "xai-test-key-do-not-log-0004"

// captureLogger writes every log line into buf, so a test can check what was logged.
func captureLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, nil))
}

// mustURL serves h from an httptest server and returns its URL, for rewriteTransport.
func mustURL(t *testing.T, h http.HandlerFunc) *url.URL {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func newOpenAITestClient(t *testing.T, h http.HandlerFunc, buf *bytes.Buffer) AnthropicClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewOpenAICompatibleClient(srv.URL+"/v1", testOpenAIKey, captureLogger(buf))
}

func TestOpenAIClient_RequestShapeAndParsing(t *testing.T) {
	var (
		method, path, auth string
		body               map[string]any
	)
	var buf bytes.Buffer
	c := newOpenAITestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path, auth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"hello"}}],`+
			`"usage":{"prompt_tokens":11,"completion_tokens":4}}`)
	}, &buf)

	text, tokens, err := c.GenerateText(context.Background(), "the prompt", "grok-4.7")
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if method != http.MethodPost || path != "/v1/chat/completions" {
		t.Fatalf("want POST /v1/chat/completions, got %s %s", method, path)
	}
	if auth != "Bearer "+testOpenAIKey {
		t.Fatalf("the key must go in a Bearer header")
	}
	if body["model"] != "grok-4.7" || body["max_tokens"] == nil {
		t.Fatalf("request body: model and max_tokens are required, got %v", body)
	}
	messages, _ := body["messages"].([]any)
	first, _ := messages[0].(map[string]any)
	if len(messages) != 1 || first["role"] != "user" || first["content"] != "the prompt" {
		t.Fatalf("request body: one user message with the prompt, got %v", messages)
	}
	if text != "hello" || tokens != 15 {
		t.Fatalf("want text hello and 15 tokens (prompt + completion), got %q %d", text, tokens)
	}
}

func TestOpenAIClient_TrailingSlashInBaseIsNotDoubled(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	t.Cleanup(srv.Close)
	var buf bytes.Buffer
	c := NewOpenAICompatibleClient(srv.URL+"/v1/", testOpenAIKey, captureLogger(&buf))
	if _, _, err := c.GenerateText(context.Background(), "p", "m"); err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if path != "/v1/chat/completions" {
		t.Fatalf("a trailing slash must not double, got %q", path)
	}
}

func TestOpenAIClient_NoChoiceOrErrorBodyIsUnavailable(t *testing.T) {
	for _, body := range []string{`{"choices":[]}`, `{"error":{"message":"nope"}}`, `not json`} {
		var buf bytes.Buffer
		c := newOpenAITestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}, &buf)
		_, _, err := c.GenerateText(context.Background(), "p", "m")
		if !errors.Is(err, ErrAIUnavailable) {
			t.Fatalf("body %q: want ErrAIUnavailable, got %v", body, err)
		}
	}
}

func TestOpenAIClient_Non200LogsStatusAndNeverTheKey(t *testing.T) {
	var buf bytes.Buffer
	c := newOpenAITestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		// The upstream echoes the key back in its error text: the log must not carry it.
		_, _ = io.WriteString(w, `{"error":{"message":"Incorrect API key provided: `+testOpenAIKey+`"}}`)
	}, &buf)

	_, _, err := c.GenerateText(context.Background(), "p", "m")
	if !errors.Is(err, ErrAIUnavailable) || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want ErrAIUnavailable naming the status, got %v", err)
	}
	logged := buf.String()
	if !strings.Contains(logged, "401") {
		t.Fatalf("the log must carry the status, got %q", logged)
	}
	if strings.Contains(logged, testOpenAIKey) || strings.Contains(err.Error(), testOpenAIKey) {
		t.Fatalf("the key must not reach the log or the error")
	}
	if !strings.Contains(logged, "[redacted]") {
		t.Fatalf("an echoed key should be replaced by [redacted], got %q", logged)
	}
}

func TestOpenAIClient_Non200LogBodyIsCutAt500Characters(t *testing.T) {
	var buf bytes.Buffer
	c := newOpenAITestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, strings.Repeat("x", 1200))
	}, &buf)
	_, _, _ = c.GenerateText(context.Background(), "p", "m")
	logged := buf.String()
	if !strings.Contains(logged, strings.Repeat("x", 500)) || strings.Contains(logged, strings.Repeat("x", 501)) {
		t.Fatalf("the log body must be exactly the first 500 characters")
	}
}

func TestOpenAIClient_LogCutCountsCharactersNotBytes(t *testing.T) {
	var buf bytes.Buffer
	c := newOpenAITestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, strings.Repeat("é", 600)) // two bytes per character
	}, &buf)
	_, _, _ = c.GenerateText(context.Background(), "p", "m")
	logged := buf.String()
	if !strings.Contains(logged, strings.Repeat("é", 500)) || strings.Contains(logged, strings.Repeat("é", 501)) {
		t.Fatalf("the cut must keep 500 characters and never split one")
	}
}

func TestOpenAIClient_TransportFailureIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL + "/v1"
	srv.Close()
	var buf bytes.Buffer
	c := NewOpenAICompatibleClient(url, testOpenAIKey, captureLogger(&buf))
	_, _, err := c.GenerateText(context.Background(), "p", "m")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("a refused connection must be ErrAIUnavailable, got %v", err)
	}
}

func TestAnthropicClient_Non200LogIsRedactedAndCut(t *testing.T) {
	var buf bytes.Buffer
	c := &httpAnthropicClient{
		apiKey: testOpenAIKey,
		httpCli: &http.Client{Transport: rewriteTransport{mustURL(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":{"message":"bad key `+testOpenAIKey+`"}}`+strings.Repeat("y", 900))
		})}},
		logger: captureLogger(&buf),
	}
	_, _, err := c.GenerateText(context.Background(), "p", "m")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("want ErrAIUnavailable, got %v", err)
	}
	logged := buf.String()
	if strings.Contains(logged, testOpenAIKey) || !strings.Contains(logged, "401") || !strings.Contains(logged, "[redacted]") {
		t.Fatalf("the Anthropic log must carry the status, redact the key, and omit it")
	}
	if strings.Contains(logged, strings.Repeat("y", 501)) {
		t.Fatalf("the Anthropic log body must be cut at 500 characters")
	}
}

func TestNewProviderClient_SelectsTheClientOrOffWithoutAKey(t *testing.T) {
	var buf bytes.Buffer
	logger := captureLogger(&buf)

	if _, ok := NewProviderClient("anthropic", "", testOpenAIKey, logger).(*httpAnthropicClient); !ok {
		t.Fatalf("anthropic must select the Anthropic client")
	}
	if _, ok := NewProviderClient("openai_compatible", "https://api.x.ai/v1", testOpenAIKey, logger).(*openAICompatibleClient); !ok {
		t.Fatalf("openai_compatible must select the OpenAI-compatible client")
	}

	// No key, or no provider: unavailable without any request. The server counts hits and must see none.
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"x"}}]}`)
	}))
	t.Cleanup(srv.Close)
	for _, c := range []AnthropicClient{
		NewProviderClient("", "", "", logger),
		NewProviderClient("openai_compatible", srv.URL+"/v1", "", logger),
		NewProviderClient("anthropic", "", "", logger),
	} {
		_, _, err := c.GenerateText(context.Background(), "p", "m")
		if !errors.Is(err, ErrAIUnavailable) {
			t.Fatalf("no provider or no key must be ErrAIUnavailable, got %v", err)
		}
	}
	if hits != 0 {
		t.Fatalf("with no key the client must not call upstream, got %d requests", hits)
	}
}
