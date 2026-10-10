package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/router"
	"github.com/rs/zerolog"
)

// #478: the request log records the client the router resolved from X-Real-IP, never a forged X-Forwarded-For.
// RequestLogger must run after clientFromXRealIP for that to hold.
func TestRequestLog_IPIsTheResolvedClientNotAForgedForwardedFor(t *testing.T) {
	var buf bytes.Buffer
	h := router.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"test-secret", nil, nil, "test", zerolog.New(&buf))
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Real-IP", "198.51.100.40")
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got := requestLogIP(t, buf.Bytes()); got != "198.51.100.40" {
		t.Fatalf("request log ip = %q; want the resolved client 198.51.100.40 (not the forged X-Forwarded-For)", got)
	}
}

// requestLogIP returns the ip field of the first "request" line in the JSON log output.
func requestLogIP(t *testing.T, logs []byte) string {
	t.Helper()
	for _, line := range bytes.Split(bytes.TrimSpace(logs), []byte("\n")) {
		var entry map[string]any
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		if entry["message"] == "request" {
			ip, _ := entry["ip"].(string)
			return ip
		}
	}
	t.Fatalf("no request log line in %q", logs)
	return ""
}
