package db

import (
	"context"
	"regexp"
	"time"

	appmw "github.com/bilimbaga/bilimbaga/internal/middleware"
	"github.com/rs/zerolog"
)

// slowQueryThreshold is the minimum query execution time that triggers a warn log.
const slowQueryThreshold = 500 * time.Millisecond

// positionalParam matches PostgreSQL positional bind parameters ($1, $2, …).
var positionalParam = regexp.MustCompile(`\$\d+`)

// LogSlowQuery emits a warn-level log when the elapsed time since start exceeds
// 500 ms.  Call it via defer at the beginning of every repository method:
//
//	defer db.LogSlowQuery(ctx, logger, query, time.Now())
//
// The query text is sanitised: all positional bind parameters ($1, $2, …) are
// replaced with "?" so that actual data values are never written to the log.
func LogSlowQuery(ctx context.Context, logger zerolog.Logger, query string, start time.Time) {
	elapsed := time.Since(start)
	if elapsed <= slowQueryThreshold {
		return
	}
	sanitized := positionalParam.ReplaceAllString(query, "?")
	logger.Warn().
		Str("request_id", appmw.GetRequestID(ctx)).
		Int64("latency_ms", elapsed.Milliseconds()).
		Str("query", sanitized).
		Msg("slow query")
}
