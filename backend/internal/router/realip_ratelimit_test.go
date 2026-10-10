package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// #475: behind the proxies every user reached the API from the same gateway address, so the per-IP limits
// keyed on one client for the whole instance. The API keys on the address chi's RealIP takes from
// X-Real-IP, which the container nginx now sets to the real client. /health is not an auth endpoint and
// must not spend the auth bucket (10 requests per minute per IP).

// limitedCall sends one request from client realIP and returns the status code.
func limitedCall(h http.Handler, method, path, realIP string) int {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Real-IP", realIP)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// TestRealIP_HealthDoesNotSpendTheAuthBucket fails while /health sits inside the AuthLimiter group.
func TestRealIP_HealthDoesNotSpendTheAuthBucket(t *testing.T) {
	h := newRouter()
	const client = "198.51.100.10"
	for i := 0; i < 10; i++ {
		limitedCall(h, http.MethodPost, "/api/v1/auth/login", client)
	}
	if got := limitedCall(h, http.MethodGet, "/api/v1/health", client); got == http.StatusTooManyRequests {
		t.Fatalf("/api/v1/health answered 429 from the auth bucket; it must not be in AuthLimiter")
	}
	if got := limitedCall(h, http.MethodPost, "/api/v1/auth/login", client); got != http.StatusTooManyRequests {
		t.Fatalf("11th login from one client = %d; want 429 (its auth bucket is full)", got)
	}
}

// TestRealIP_DifferentClientsGetSeparateAuthBuckets passes already on origin/main, because RealIP keys on
// X-Real-IP. It guards the behaviour the nginx change relies on: a second client is not limited by the first.
func TestRealIP_DifferentClientsGetSeparateAuthBuckets(t *testing.T) {
	h := newRouter()
	const first, second = "198.51.100.20", "198.51.100.21"
	for i := 0; i < 10; i++ {
		limitedCall(h, http.MethodPost, "/api/v1/auth/login", first)
	}
	if got := limitedCall(h, http.MethodPost, "/api/v1/auth/login", first); got != http.StatusTooManyRequests {
		t.Fatalf("11th login from %s = %d; want 429", first, got)
	}
	if got := limitedCall(h, http.MethodPost, "/api/v1/auth/login", second); got == http.StatusTooManyRequests {
		t.Fatalf("client %s was limited by client %s's bucket", second, first)
	}
}
