package auth

import (
	"net/http/httptest"
	"testing"
)

// #478: the audit IP of a sign-in is the address the router resolved into RemoteAddr, never the first
// X-Forwarded-For entry, which a client sends freely and could use to forge the audited address.
func TestClientIP_ForgedForwardedForDoesNotChooseTheAuditIP(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	r.RemoteAddr = "198.51.100.7:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	if got := clientIP(r); got != "198.51.100.7" {
		t.Fatalf("audit ip = %q; want the RemoteAddr 198.51.100.7 (a forged X-Forwarded-For must not choose it)", got)
	}
}
