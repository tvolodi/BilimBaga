package db_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
	appmw "github.com/bilimbaga/bilimbaga/internal/middleware"
	"github.com/rs/zerolog"
)

func TestLogSlowQuery_FastQueryIsSilent(t *testing.T) {
	var buf bytes.Buffer
	dbpkg.LogSlowQuery(context.Background(), zerolog.New(&buf), "SELECT 1", time.Now())
	if buf.Len() != 0 {
		t.Fatalf("expected no log output for a fast query, got %q", buf.String())
	}
}

func TestLogSlowQuery_JustUnderThresholdIsSilent(t *testing.T) {
	var buf bytes.Buffer
	dbpkg.LogSlowQuery(context.Background(), zerolog.New(&buf), "SELECT 1", time.Now().Add(-100*time.Millisecond))
	if buf.Len() != 0 {
		t.Fatalf("100ms must not be logged, got %q", buf.String())
	}
}

func TestLogSlowQuery_SlowQueryLogsSanitisedWarn(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.WithValue(context.Background(), appmw.RequestIDKey, "req-123")

	dbpkg.LogSlowQuery(ctx, zerolog.New(&buf), "SELECT * FROM users WHERE id = $1 AND email = $12", time.Now().Add(-2*time.Second))

	out := buf.String()
	for _, want := range []string{`"level":"warn"`, `"request_id":"req-123"`, `"message":"slow query"`, `"latency_ms"`, "id = ? AND email = ?"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q: %s", want, out)
		}
	}
	if strings.Contains(out, "$1") {
		t.Errorf("positional params must be sanitised: %s", out)
	}
}

func TestLogSlowQuery_MissingRequestIDFallsBack(t *testing.T) {
	var buf bytes.Buffer
	dbpkg.LogSlowQuery(context.Background(), zerolog.New(&buf), "SELECT 1", time.Now().Add(-time.Second))
	if !strings.Contains(buf.String(), `"request_id":"-"`) {
		t.Errorf("expected request_id fallback '-', got %s", buf.String())
	}
}
