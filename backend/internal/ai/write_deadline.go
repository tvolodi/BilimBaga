package ai

import (
	"net/http"
	"time"
)

// AIWriteBudget is how long a response on an AI route may take to be written (#519): the largest AI_TIMEOUT_SECONDS
// plus the 30 s nginx margin. The nginx ^~ /api/v1/admin/ai/ location waits the same 180 s.
const AIWriteBudget = 180 * time.Second

// WriteDeadline gives the response of every AI route a write deadline of budget (#519). The server's WriteTimeout
// (15 s) would otherwise cut an answer that waits for the provider. http.ResponseController reaches the connection
// through the wrappers that implement Unwrap, such as the request logger's.
func WriteDeadline(budget time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(budget))
			next.ServeHTTP(w, r)
		})
	}
}
