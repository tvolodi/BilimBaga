package main

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// #522: the api image is FROM scratch, so it has no system zone database and time.LoadLocation cannot find
// TENANT_TIMEZONE=Asia/Almaty there. The zones are embedded through time/tzdata, which the api binary must link.
// A host with system zoneinfo would resolve the zone either way, so the guard checks the binary's dependency:
// with it missing, the scratch image falls back to UTC again.
func TestAPIBinaryEmbedsTheZoneDatabase(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "time/tzdata") {
		t.Fatal("the api binary does not link time/tzdata: a scratch image cannot load TENANT_TIMEZONE")
	}
}

// The configured zone must load in the process that runs the api, as the email scheduler loads it.
func TestTenantTimezoneAsiaAlmatyLoads(t *testing.T) {
	if _, err := time.LoadLocation("Asia/Almaty"); err != nil {
		t.Fatalf("load Asia/Almaty: %v", err)
	}
}
