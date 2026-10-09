package ratelimit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func doReq(h http.Handler, ip, sessionID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = ip + ":1234"
	if sessionID != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", sessionID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestDisabled(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"", false},
		{"true", true},
		{"1", true},
		{"false", false},
	}
	for _, tc := range tests {
		t.Run("value="+tc.val, func(t *testing.T) {
			t.Setenv("DISABLE_RATE_LIMIT", tc.val)
			if got := disabled(); got != tc.want {
				t.Fatalf("disabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLimiters_BlockAfterLimit(t *testing.T) {
	tests := []struct {
		name  string
		build func() func(http.Handler) http.Handler
		limit int
	}{
		{"auth", AuthLimiter, 10},
		{"global", GlobalLimiter, 300},
		{"answer", AnswerSaveLimiter, 60},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DISABLE_RATE_LIMIT", "")
			h := tc.build()(okHandler())
			for i := 0; i < tc.limit; i++ {
				if rec := doReq(h, "10.0.0.1", ""); rec.Code != http.StatusOK {
					t.Fatalf("request %d: got %d, want 200", i+1, rec.Code)
				}
			}
			rec := doReq(h, "10.0.0.1", "")
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("got %d, want 429", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q", ct)
			}
			if ra := rec.Header().Get("Retry-After"); ra != "60" {
				t.Errorf("Retry-After = %q", ra)
			}
			var body struct {
				Data  any `json:"data"`
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON body: %v", err)
			}
			if body.Data != nil || body.Error.Code != "RATE_LIMITED" || body.Error.Message == "" {
				t.Errorf("unexpected envelope: %+v", body)
			}
			// A different IP is unaffected.
			if rec := doReq(h, "10.0.0.2", ""); rec.Code != http.StatusOK {
				t.Errorf("other IP got %d, want 200", rec.Code)
			}
		})
	}
}

func TestLimiters_DisabledPassThrough(t *testing.T) {
	t.Setenv("DISABLE_RATE_LIMIT", "1")
	for name, build := range map[string]func() func(http.Handler) http.Handler{
		"auth":   AuthLimiter,
		"global": GlobalLimiter,
		"answer": AnswerSaveLimiter,
	} {
		t.Run(name, func(t *testing.T) {
			h := build()(okHandler())
			for i := 0; i < 400; i++ {
				if rec := doReq(h, "10.0.0.1", ""); rec.Code != http.StatusOK {
					t.Fatalf("request %d: got %d, want 200", i+1, rec.Code)
				}
			}
		})
	}
}

func TestAnswerSaveLimiter(t *testing.T) {
	h := AnswerSaveLimiter()(okHandler())

	for i := 0; i < 60; i++ {
		if rec := doReq(h, "10.0.0.1", "sess-a"); rec.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i+1, rec.Code)
		}
	}
	if rec := doReq(h, "10.0.0.1", "sess-a"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("61st request got %d, want 429", rec.Code)
	}
	// Other session from the same IP is keyed separately.
	if rec := doReq(h, "10.0.0.1", "sess-b"); rec.Code != http.StatusOK {
		t.Errorf("other session got %d, want 200", rec.Code)
	}
}

func TestAnswerSaveLimiter_FallsBackToIP(t *testing.T) {
	h := AnswerSaveLimiter()(okHandler())

	for i := 0; i < 60; i++ {
		if rec := doReq(h, "10.0.0.9", ""); rec.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i+1, rec.Code)
		}
	}
	if rec := doReq(h, "10.0.0.9", ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("61st request got %d, want 429", rec.Code)
	}
	if rec := doReq(h, "10.0.0.10", ""); rec.Code != http.StatusOK {
		t.Errorf("other IP got %d, want 200", rec.Code)
	}
}
