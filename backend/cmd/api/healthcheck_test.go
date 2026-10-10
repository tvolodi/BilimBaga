package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// #473: the api image is FROM scratch (no shell, no curl), so the binary answers the container's healthcheck itself.

func runHealthcheck(probeErr error) (code, served int, errOut string) {
	var o, e bytes.Buffer
	m := &fakeMigrator{}
	code = run([]string{"healthcheck"}, deps{
		serve:    func() { served++ },
		migrator: m,
		stdout:   &o,
		stderr:   &e,
		probe:    func() error { return probeErr },
	})
	return code, served, e.String()
}

func TestRun_HealthcheckHealthyExitsZero(t *testing.T) {
	code, served, errOut := runHealthcheck(nil)
	if code != 0 || served != 0 || errOut != "" {
		t.Fatalf("code=%d served=%d stderr=%q", code, served, errOut)
	}
}

func TestRun_HealthcheckUnhealthyExitsOneWithReason(t *testing.T) {
	code, served, errOut := runHealthcheck(errors.New("health answered 503"))
	if code != 1 || served != 0 || !strings.Contains(errOut, "health answered 503") {
		t.Fatalf("code=%d served=%d stderr=%q", code, served, errOut)
	}
}

func TestProbeHealth_StatusDecides(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	if err := probeHealth(ok.URL + "/api/v1/health"); err != nil {
		t.Fatalf("200 must pass: %v", err)
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer down.Close()
	if err := probeHealth(down.URL + "/api/v1/health"); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("503 must fail and name the status: %v", err)
	}
}

func TestProbeHealth_UnreachableFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL + "/api/v1/health"
	srv.Close()
	if err := probeHealth(url); err == nil {
		t.Fatal("a server that is not listening must fail")
	}
}

func TestHealthURL_PortFromEnvWithConfigDefaults(t *testing.T) {
	t.Setenv("API_PORT", "")
	t.Setenv("BB_API_PORT", "")
	if got := healthURL(); got != "http://127.0.0.1:8080/api/v1/health" {
		t.Fatalf("default = %q", got)
	}
	t.Setenv("BB_API_PORT", "7001")
	if got := healthURL(); got != "http://127.0.0.1:7001/api/v1/health" {
		t.Fatalf("BB_API_PORT = %q", got)
	}
	t.Setenv("API_PORT", "9090")
	if got := healthURL(); got != "http://127.0.0.1:9090/api/v1/health" {
		t.Fatalf("API_PORT must win over BB_API_PORT: %q", got)
	}
}
