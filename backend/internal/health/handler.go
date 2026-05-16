// Package health provides the GET /api/v1/health liveness and readiness endpoint.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
)

// Pinger is any type that can verify database connectivity.
// *sqlx.DB satisfies this interface.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// healthData is the payload returned by the health endpoint.
type healthData struct {
	Status  string `json:"status"`
	DBOk    bool   `json:"db_ok"`
	Version string `json:"version"`
}

// Handler returns a http.HandlerFunc that performs a DB liveness check and
// reports the current build version.
//
//   - 200: DB reachable, status "ok", db_ok true
//   - 503: DB unreachable, status "degraded", db_ok false
//
// The version string is injected at build time via -ldflags in the Makefile.
func Handler(db Pinger, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbOk := db.PingContext(ctx) == nil
		status := "ok"
		httpStatus := http.StatusOK
		if !dbOk {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}

		api.WriteJSON(w, httpStatus, map[string]interface{}{
			"data": healthData{
				Status:  status,
				DBOk:    dbOk,
				Version: version,
			},
			"error": nil,
		})
	}
}
