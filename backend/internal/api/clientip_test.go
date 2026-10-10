package api

import (
	"net/http/httptest"
	"testing"
)

// #478: ClientIP is the one address for audit rows, request logs and rate limits. It strips a port and keeps a
// bare address (the router's clientFromXRealIP writes a bare IP).
func TestClientIP(t *testing.T) {
	cases := []struct{ name, remote, want string }{
		{"peer with port", "198.51.100.7:1234", "198.51.100.7"},
		{"bare address from X-Real-IP", "198.51.100.7", "198.51.100.7"},
		{"IPv6 with port", "[2001:db8::1]:443", "2001:db8::1"},
		{"bare IPv6", "2001:db8::1", "2001:db8::1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = c.remote
			if got := ClientIP(r); got != c.want {
				t.Fatalf("ClientIP(%q) = %q; want %q", c.remote, got, c.want)
			}
		})
	}
}
