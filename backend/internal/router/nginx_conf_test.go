package router_test

import (
	"os"
	"regexp"
	"testing"
)

// ISS-105: the password-reset page carries a one-time token in its URL, so nginx must
// answer /reset-password with Referrer-Policy: no-referrer (add_header is not inherited
// into a location that declares its own, hence the dedicated block).
func TestNginxConf_ResetPasswordLocationSendsNoReferrer(t *testing.T) {
	raw, err := os.ReadFile("../../../deploy/nginx.conf")
	if err != nil {
		t.Fatalf("read nginx.conf: %v", err)
	}
	block := regexp.MustCompile(`(?s)location\s*=\s*/reset-password\s*\{(.*?)\n\s*\}`).FindSubmatch(raw)
	if block == nil {
		t.Fatal("no `location = /reset-password` block in deploy/nginx.conf")
	}
	if !regexp.MustCompile(`add_header\s+Referrer-Policy\s+"no-referrer"\s+always;`).Match(block[1]) {
		t.Errorf("reset-password location lacks Referrer-Policy no-referrer:\n%s", block[1])
	}
}
