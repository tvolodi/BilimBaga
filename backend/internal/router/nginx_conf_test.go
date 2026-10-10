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

// #402: above client_max_body_size nginx rejects the request itself, so it must answer with
// the API error envelope (same code as the app's own 413) instead of its HTML page. The edge
// vhost enforces the same limit first, so the QA vhost carries the same handler.
func TestNginxConf_Oversize413ReturnsJSONEnvelope(t *testing.T) {
	for _, path := range []string{"../../../deploy/nginx.conf", "../../../deploy/nginx/bilimbaga-qa.conf"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !regexp.MustCompile(`error_page\s+413\s+@payload_too_large;`).Match(raw) {
			t.Errorf("%s: missing `error_page 413 @payload_too_large;`", path)
		}
		block := regexp.MustCompile(`(?s)location\s+@payload_too_large\s*\{(.*?)\n\s*\}`).FindSubmatch(raw)
		if block == nil {
			t.Errorf("%s: no `location @payload_too_large` block", path)
			continue
		}
		if !regexp.MustCompile(`default_type\s+application/json;`).Match(block[1]) {
			t.Errorf("%s: 413 handler does not serve application/json", path)
		}
		if !regexp.MustCompile(`return\s+413\s+'\{"data":null,"error":\{"code":"ERR_FILE_TOO_LARGE","message":"[^"]+"\}\}';`).Match(block[1]) {
			t.Errorf("%s: 413 handler does not return the ERR_FILE_TOO_LARGE envelope:\n%s", path, block[1])
		}
	}
}
