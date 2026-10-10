package router

import (
	"net"
	"net/http"
	"strings"
)

// clientFromXRealIP sets RemoteAddr to the client address the container nginx resolved and sent as
// X-Real-IP (#475). Only X-Real-IP is read. chi's RealIP also trusts True-Client-IP and X-Forwarded-For,
// which any client can send, so each request could pick a new rate-limit bucket and forge the audit IP. A
// request without a valid X-Real-IP keeps its TCP peer address.
func clientFromXRealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(ip) != nil {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}
