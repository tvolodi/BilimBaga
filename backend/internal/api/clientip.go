package api

import (
	"net"
	"net/http"
)

// ClientIP is the one address the API uses for audit rows, request logs and rate limits (#478). It is the
// RemoteAddr that the router's clientFromXRealIP resolved from X-Real-IP (#475), without a port. Never read
// X-Forwarded-For here: a client sends it freely, so it could forge the address that is audited and logged.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
