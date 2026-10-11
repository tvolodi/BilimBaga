package ai

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

// #519: a provider client's whole-request timeout comes from AI_TIMEOUT_SECONDS (WithTimeout), not a fixed 60 s.

func TestClientTimeout_DefaultIsTheConfigDefault(t *testing.T) {
	want := time.Duration(config.DefaultAITimeoutSeconds) * time.Second
	logger := captureLogger(&bytes.Buffer{})
	a := NewAnthropicClient("k", logger).(*httpAnthropicClient)
	o := NewOpenAICompatibleClient("https://api.x.ai/v1", "k", logger).(*openAICompatibleClient)
	if DefaultClientTimeout != want || a.httpCli.Timeout != want || o.httpCli.Timeout != want {
		t.Fatalf("default timeouts must equal config.DefaultAITimeoutSeconds (%v): const=%v anthropic=%v openai=%v",
			want, DefaultClientTimeout, a.httpCli.Timeout, o.httpCli.Timeout)
	}
}

func TestClientTimeout_WithTimeoutSetsBothClients(t *testing.T) {
	logger := captureLogger(&bytes.Buffer{})
	a := NewAnthropicClient("k", logger, WithTimeout(7*time.Second)).(*httpAnthropicClient)
	o := NewOpenAICompatibleClient("https://api.x.ai/v1", "k", logger, WithTimeout(7*time.Second)).(*openAICompatibleClient)
	if a.httpCli.Timeout != 7*time.Second || o.httpCli.Timeout != 7*time.Second {
		t.Fatalf("WithTimeout(7s) must set both clients: anthropic=%v openai=%v", a.httpCli.Timeout, o.httpCli.Timeout)
	}
}

func TestClientTimeout_ProviderFactoryPassesTheOption(t *testing.T) {
	logger := captureLogger(&bytes.Buffer{})
	a := NewProviderClient("anthropic", "", "k", logger, WithTimeout(9*time.Second)).(*httpAnthropicClient)
	o := NewProviderClient("openai_compatible", "https://api.x.ai/v1", "k", logger, WithTimeout(9*time.Second)).(*openAICompatibleClient)
	if a.httpCli.Timeout != 9*time.Second || o.httpCli.Timeout != 9*time.Second {
		t.Fatalf("NewProviderClient must pass WithTimeout through: anthropic=%v openai=%v", a.httpCli.Timeout, o.httpCli.Timeout)
	}
}

// The provider answers after the client's limit: the call must end as ErrAIUnavailable at the limit, not at the
// provider's answer.
func TestClientTimeout_IsEnforcedAsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(time.Second)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"late"}}]}`)
	}))
	t.Cleanup(srv.Close)

	c := NewOpenAICompatibleClient(srv.URL+"/v1", "k", captureLogger(&bytes.Buffer{}), WithTimeout(200*time.Millisecond))
	start := time.Now()
	_, _, err := c.GenerateText(context.Background(), "p", "m")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("a call past the timeout must be ErrAIUnavailable, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 900*time.Millisecond {
		t.Fatalf("the call took %v; the 200 ms timeout did not apply", elapsed)
	}
}
