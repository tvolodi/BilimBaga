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

// #402 / api-conventions section 8: above client_max_body_size nginx rejects the request itself, so
// it must answer with the API error envelope instead of its HTML page, using the size code the API
// itself uses for that endpoint. The edge vhost enforces the same limit first, so the QA vhost carries
// the same handler (with its own variable prefix: map variables are global to the host's http context).
func TestNginxConf_Oversize413ReturnsJSONEnvelopeWithEndpointCode(t *testing.T) {
	cases := []struct{ path, prefix string }{
		{"../../../deploy/nginx.conf", "too_large"},
		{"../../../deploy/nginx/bilimbaga-qa.conf", "bb_qa_413"},
		{"../../../deploy/nginx/bilimbaga.conf", "bb_prod_413"},
		{"../../../deploy/nginx/bilimbaga-test.conf", "bb_test_413"},
	}
	// $uri has no query string, so the users import (?commit=true) still matches its entry.
	wantCodes := map[string]string{
		"default":                  "ERR_FILE_TOO_LARGE",
		"/api/v1/questions/import": "ERR_FILE_TOO_LARGE",
		"/api/v1/users/import":     "FILE_TOO_LARGE",
		"/api/v1/tenant/config":    "LOGO_TOO_LARGE",
	}
	entry := regexp.MustCompile(`(?m)^\s*(\S+)\s+("[^"]*"|\S+);`)
	for _, c := range cases {
		raw, err := os.ReadFile(c.path)
		if err != nil {
			t.Fatalf("read %s: %v", c.path, err)
		}
		if !regexp.MustCompile(`error_page\s+413\s+@payload_too_large;`).Match(raw) {
			t.Errorf("%s: missing `error_page 413 @payload_too_large;`", c.path)
		}
		m := regexp.MustCompile(`(?s)map\s+\$uri\s+\$` + c.prefix + `_code\s*\{(.*?)\n\}`).FindSubmatch(raw)
		if m == nil {
			t.Errorf("%s: no `map $uri $%s_code` block", c.path, c.prefix)
			continue
		}
		got := map[string]string{}
		for _, e := range entry.FindAllSubmatch(m[1], -1) {
			got[string(e[1])] = string(e[2])
		}
		for k, v := range wantCodes {
			if got[k] != v {
				t.Errorf("%s: map %s_code[%q] = %q, want %q", c.path, c.prefix, k, got[k], v)
			}
		}
		if len(got) != len(wantCodes) {
			t.Errorf("%s: map %s_code has %d entries, want %d: %v", c.path, c.prefix, len(got), len(wantCodes), got)
		}
		if !regexp.MustCompile(`map\s+\$uri\s+\$` + c.prefix + `_msg\s*\{`).Match(raw) {
			t.Errorf("%s: no `map $uri $%s_msg` block", c.path, c.prefix)
		}
		block := regexp.MustCompile(`(?s)location\s+@payload_too_large\s*\{(.*?)\n\s*\}`).FindSubmatch(raw)
		if block == nil {
			t.Errorf("%s: no `location @payload_too_large` block", c.path)
			continue
		}
		if !regexp.MustCompile(`default_type\s+application/json;`).Match(block[1]) {
			t.Errorf("%s: 413 handler does not serve application/json", c.path)
		}
		ret := `return\s+413\s+'\{"data":null,"error":\{"code":"\$` + c.prefix + `_code","message":"\$` + c.prefix + `_msg"\}\}';`
		if !regexp.MustCompile(ret).Match(block[1]) {
			t.Errorf("%s: 413 handler does not return the envelope built from the maps: %s", c.path, block[1])
		}
	}
}

// FR-BB321 AC-4: theme-init.js sets the theme before first paint. The browser must revalidate it on
// every load (Cache-Control no-cache), and, like /reset-password, its own location must repeat the
// security headers because add_header is not inherited.
func TestNginxConf_ThemeInitLocationRevalidatesAndKeepsSecurityHeaders(t *testing.T) {
	raw, err := os.ReadFile("../../../deploy/nginx.conf")
	if err != nil {
		t.Fatalf("read nginx.conf: %v", err)
	}
	block := regexp.MustCompile(`(?s)location\s*=\s*/theme-init\.js\s*\{(.*?)\n\s*\}`).FindSubmatch(raw)
	if block == nil {
		t.Fatal("no `location = /theme-init.js` block in deploy/nginx.conf")
	}
	if !regexp.MustCompile(`add_header\s+Cache-Control\s+"no-cache"\s+always;`).Match(block[1]) {
		t.Errorf("theme-init.js location lacks Cache-Control no-cache:\n%s", block[1])
	}
	for _, header := range []string{"Content-Security-Policy", "X-Frame-Options", "X-Content-Type-Options", "Referrer-Policy", "Strict-Transport-Security"} {
		if !regexp.MustCompile(`add_header\s+` + header + `\s+`).Match(block[1]) {
			t.Errorf("theme-init.js location does not repeat %s", header)
		}
	}
}

// #511 / #461 (FR-BB65 AC-3): text compression, a long immutable cache for the content-hashed bundles and a
// revalidating HTML shell. add_header is not inherited into a location that declares its own, so each of
// the two locations must repeat the security headers; /api/ must stay uncompressed.
func TestNginxConf_CompressionAndCachePolicy(t *testing.T) {
	raw, err := os.ReadFile("../../../deploy/nginx.conf")
	if err != nil {
		t.Fatalf("read nginx.conf: %v", err)
	}
	conf := string(raw)
	if !regexp.MustCompile(`(?m)^\s*gzip\s+on;`).MatchString(conf) {
		t.Error("server does not enable gzip")
	}
	types := regexp.MustCompile(`gzip_types\s+([^;]+);`).FindStringSubmatch(conf)
	if types == nil {
		t.Fatal("no gzip_types in deploy/nginx.conf")
	}
	for _, mt := range []string{"text/css", "application/javascript", "text/javascript", "application/json", "image/svg+xml"} {
		if !regexp.MustCompile(`(^|\s)` + regexp.QuoteMeta(mt) + `(\s|$)`).MatchString(types[1]) {
			t.Errorf("gzip_types lacks %s", mt)
		}
	}
	api := regexp.MustCompile(`(?s)location\s+/api/\s*\{(.*?)\n\s*\}`).FindStringSubmatch(conf)
	if api == nil || !regexp.MustCompile(`gzip\s+off;`).MatchString(api[1]) {
		t.Error("/api/ must switch gzip off")
	}
	security := []string{"Content-Security-Policy", "X-Frame-Options", "X-Content-Type-Options", "Referrer-Policy", "Strict-Transport-Security"}
	cases := []struct {
		name, location, cache, always string
		extra                         *regexp.Regexp
	}{
		// no "always" on the long TTL: a 404 for a missing hashed file must not be cached for a year
		{"hashed assets", `location\s+/assets/`, `public, max-age=31536000, immutable`, ``, regexp.MustCompile(`try_files\s+\$uri\s+=404;`)},
		{"HTML shell", `location\s+/`, `no-cache`, `\s+always`, regexp.MustCompile(`try_files\s+\$uri\s+\$uri/\s+/index\.html;`)},
	}
	for _, c := range cases {
		block := regexp.MustCompile(`(?s)` + c.location + `\s*\{(.*?)\n\s*\}`).FindStringSubmatch(conf)
		if block == nil {
			t.Errorf("%s: no %s block", c.name, c.location)
			continue
		}
		if !regexp.MustCompile(`add_header\s+Cache-Control\s+"` + regexp.QuoteMeta(c.cache) + `"` + c.always + `;`).MatchString(block[1]) {
			t.Errorf("%s: Cache-Control %q missing:\n%s", c.name, c.cache, block[1])
		}
		if !c.extra.MatchString(block[1]) {
			t.Errorf("%s: wrong try_files (a missing asset must 404, the shell keeps the SPA fallback):\n%s", c.name, block[1])
		}
		for _, h := range security {
			if !regexp.MustCompile(`add_header\s+` + h + `\s+`).MatchString(block[1]) {
				t.Errorf("%s does not repeat %s", c.name, h)
			}
		}
	}
}
