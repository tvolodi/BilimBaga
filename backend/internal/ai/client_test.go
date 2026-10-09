package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// rewriteTransport redirects every outgoing request to the httptest server so
// the production client (which hard-codes api.anthropic.com) never touches the
// network.
type rewriteTransport struct{ target *url.URL }

func (rt rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.URL.Scheme = rt.target.Scheme
	r2.URL.Host = rt.target.Host
	return http.DefaultTransport.RoundTrip(r2)
}

func newTestClient(t *testing.T, h http.HandlerFunc) AnthropicClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return &httpAnthropicClient{
		apiKey:  "test-key",
		httpCli: &http.Client{Transport: rewriteTransport{u}},
		logger:  newLogger(),
	}
}

func TestNewAnthropicClient_Defaults(t *testing.T) {
	c, ok := NewAnthropicClient("k", newLogger()).(*httpAnthropicClient)
	if !ok || c.apiKey != "k" || c.httpCli == nil || c.httpCli.Timeout == 0 {
		t.Fatalf("unexpected client: %+v", c)
	}
}

func TestClient_GenerateText_Success(t *testing.T) {
	var gotReq anthropicRequest
	var gotHeaders http.Header
	var gotPath, gotMethod string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeaders, gotPath, gotMethod = r.Header.Clone(), r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotReq)
		_, _ = w.Write([]byte(`{"content":[{"type":"thinking","text":"x"},{"type":"text","text":"hello"},{"type":"text","text":"second"}],"usage":{"input_tokens":10,"output_tokens":5}}`))
	})

	text, tokens, err := c.GenerateText(context.Background(), "the prompt", "model-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello" || tokens != 15 {
		t.Fatalf("got (%q,%d), want (hello,15)", text, tokens)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/messages" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if gotHeaders.Get("x-api-key") != "test-key" || gotHeaders.Get("anthropic-version") != "2023-06-01" ||
		gotHeaders.Get("Content-Type") != "application/json" {
		t.Fatalf("headers = %v", gotHeaders)
	}
	if gotReq.Model != "model-x" || gotReq.MaxTokens != 4096 ||
		len(gotReq.Messages) != 1 || gotReq.Messages[0].Role != "user" || gotReq.Messages[0].Content != "the prompt" {
		t.Fatalf("request body = %+v", gotReq)
	}
}

func TestClient_GenerateText_NoTextBlock(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"content":[],"usage":{"input_tokens":1,"output_tokens":2}}`))
	})
	text, tokens, err := c.GenerateText(context.Background(), "p", "m")
	if err != nil || text != "" || tokens != 3 {
		t.Fatalf("got (%q,%d,%v)", text, tokens, err)
	}
}

func TestClient_GenerateText_Non200(t *testing.T) {
	for _, code := range []int{http.StatusBadRequest, http.StatusTooManyRequests, http.StatusInternalServerError} {
		c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
		})
		_, _, err := c.GenerateText(context.Background(), "p", "m")
		if !errors.Is(err, ErrAIUnavailable) {
			t.Fatalf("status %d: want ErrAIUnavailable, got %v", code, err)
		}
		if !strings.Contains(err.Error(), "HTTP") {
			t.Fatalf("status %d: error should mention status: %v", code, err)
		}
	}
}

func TestClient_GenerateText_BadJSON(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})
	_, _, err := c.GenerateText(context.Background(), "p", "m")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("want ErrAIUnavailable, got %v", err)
	}
}

func TestClient_GenerateText_APIErrorBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"type":"overloaded_error","message":"overloaded"}}`))
	})
	_, _, err := c.GenerateText(context.Background(), "p", "m")
	if !errors.Is(err, ErrAIUnavailable) || !strings.Contains(err.Error(), "overloaded") {
		t.Fatalf("got %v", err)
	}
}

func TestClient_GenerateText_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	u, _ := url.Parse(srv.URL)
	srv.Close() // connection refused
	c := &httpAnthropicClient{
		apiKey:  "k",
		httpCli: &http.Client{Transport: rewriteTransport{u}},
		logger:  newLogger(),
	}
	_, _, err := c.GenerateText(context.Background(), "p", "m")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("want ErrAIUnavailable, got %v", err)
	}
}

func TestClient_GenerateText_ContextCancelled(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := c.GenerateText(ctx, "p", "m")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("want ErrAIUnavailable, got %v", err)
	}
}

func TestClient_GenerateText_BadRequestURL(t *testing.T) {
	// A nil context makes http.NewRequestWithContext fail.
	c := newTestClient(t, func(http.ResponseWriter, *http.Request) {})
	//nolint:staticcheck // deliberately passing a nil context
	_, _, err := c.GenerateText(nil, "p", "m") //lint:ignore SA1012 test
	if err == nil || errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("want request-creation error, got %v", err)
	}
}

func TestBuildQuestionsPrompt(t *testing.T) {
	p, err := BuildQuestionsPrompt(GenerateQuestionsRequest{
		Count: 3, CategoryName: "Fire Safety", Difficulty: "hard", ContextText: "CTX-MARKER",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"Fire Safety", "hard", "CTX-MARKER", "3"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildInsightAndLoyaltyPrompts(t *testing.T) {
	p, err := BuildInsightPrompt(ExamInsightData{
		ExamTitle: "Safety 101", TotalAttempts: 7, PassRate: 0.5, AvgScorePct: 0.8,
		QuestionStats: []InsightQuestionStat{{OrderNum: 1, Stem: "STEM-X", CorrectRate: 0.25}},
	})
	if err != nil || !strings.Contains(p, "Safety 101") || !strings.Contains(p, "STEM-X") {
		t.Fatalf("insight prompt: %v %q", err, p)
	}
	lp, err := BuildLoyaltyPrompt(LoyaltyPromptData{Responses: []LikertResponseData{{DimensionLabel: "DIM-A", NormalizedWeight: 4}}})
	if err != nil || !strings.Contains(lp, "DIM-A") {
		t.Fatalf("loyalty prompt: %v %q", err, lp)
	}
}
