package middleware

import (
	"net/http/httptest"
	"testing"
)

// #478: the request log's IP is the address the router resolved into RemoteAddr, never the first
// X-Forwarded-For entry, which a client sends freely and could use to forge its IP in the log.
func TestRealIP_ForgedForwardedForDoesNotChooseTheLoggedIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/ping", nil)
	r.RemoteAddr = "198.51.100.7:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	if got := realIP(r); got != "198.51.100.7" {
		t.Fatalf("logged ip = %q; want the RemoteAddr 198.51.100.7 (a forged X-Forwarded-For must not choose it)", got)
	}
}
