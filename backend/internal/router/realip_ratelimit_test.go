package router_test

import (
	"fmt"
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

// TestRealIP_ClientSuppliedForwardingHeadersDoNotChooseTheBucket: the API takes the client from X-Real-IP
// only. chi RealIP also trusts True-Client-IP and X-Forwarded-For, which a client sends freely, so a new
// value per request would pick a new bucket and escape the sign-in limiter (#475 review). The TCP peer is the
// same for every request here, so the limiter must still count them as one client.
func TestRealIP_ClientSuppliedForwardingHeadersDoNotChooseTheBucket(t *testing.T) {
	for _, header := range []string{"True-Client-IP", "X-Forwarded-For"} {
		t.Run(header, func(t *testing.T) {
			h := newRouter()
			last := 0
			for i := 0; i < 11; i++ {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
				req.Header.Set(header, fmt.Sprintf("198.51.100.%d", 30+i))
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)
				last = rec.Code
			}
			if last != http.StatusTooManyRequests {
				t.Fatalf("11th login with a new %s each time = %d; want 429 (one client, one bucket)", header, last)
			}
		})
	}
}
