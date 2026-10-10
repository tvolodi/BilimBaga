package router_test

import (
	"os"
	"regexp"
	"testing"
)

// #475: the rate limits need the real client address. The container nginx trusts only the private ranges
// (the edge's address on the Docker network) and takes the client from the forwarded chain. Each edge vhost
// trusts only Cloudflare and takes the client from CF-Connecting-IP. Ranges are Cloudflare's published lists
// (https://www.cloudflare.com/ips-v4/ and /ips-v6/); re-check them when they change.
//
// deploy/nginx/bilimbaga-test.conf is NOT covered here: its header makes it an owner-only protected target
// (customer demo), so it changes only with the owner's decision (#475).

var (
	privateRanges = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}

	cloudflareRanges = []string{
		"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22", "141.101.64.0/18",
		"108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20", "197.234.240.0/22", "198.41.128.0/17",
		"162.158.0.0/15", "104.16.0.0/13", "104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",
		"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32", "2405:8100::/32",
		"2a06:98c0::/29", "2c0f:f248::/32",
	}
)

// setRealIPFromRanges returns the set of ranges named by set_real_ip_from directives in an nginx config.
func setRealIPFromRanges(t *testing.T, path string) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	ranges := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^\s*set_real_ip_from\s+([^;\s]+);`).FindAllSubmatch(raw, -1) {
		ranges[string(m[1])] = true
	}
	return ranges
}

func TestNginxConf_ContainerTrustsPrivateRangesAndReadsTheForwardedChain(t *testing.T) {
	const path = "../../../deploy/nginx.conf"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	ranges := setRealIPFromRanges(t, path)
	for _, want := range privateRanges {
		if !ranges[want] {
			t.Errorf("%s: no `set_real_ip_from %s;`", path, want)
		}
	}
	for r := range ranges {
		known := false
		for _, p := range privateRanges {
			known = known || r == p
		}
		if !known {
			t.Errorf("%s: trusts %s, which is not a private range (a public peer could set the client address)", path, r)
		}
	}
	if !regexp.MustCompile(`(?m)^\s*real_ip_header\s+X-Forwarded-For;`).Match(raw) {
		t.Errorf("%s: missing `real_ip_header X-Forwarded-For;`", path)
	}
	if !regexp.MustCompile(`(?m)^\s*real_ip_recursive\s+on;`).Match(raw) {
		t.Errorf("%s: missing `real_ip_recursive on;` (the client is the rightmost untrusted address)", path)
	}
	if !regexp.MustCompile(`proxy_set_header\s+X-Real-IP\s+\$remote_addr;`).Match(raw) {
		t.Errorf("%s: the API must still receive X-Real-IP $remote_addr (the resolved client)", path)
	}
}

func TestNginxEdgeVhosts_TrustCloudflareAndReadTheConnectingIP(t *testing.T) {
	for _, path := range []string{
		"../../../deploy/nginx/bilimbaga.conf",
		"../../../deploy/nginx/bilimbaga-qa.conf",
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !regexp.MustCompile(`(?m)^\s*real_ip_header\s+CF-Connecting-IP;`).Match(raw) {
			t.Errorf("%s: missing `real_ip_header CF-Connecting-IP;`", path)
		}
		ranges := setRealIPFromRanges(t, path)
		for _, want := range cloudflareRanges {
			if !ranges[want] {
				t.Errorf("%s: no `set_real_ip_from %s;`", path, want)
			}
		}
		if len(ranges) != len(cloudflareRanges) {
			t.Errorf("%s: %d set_real_ip_from ranges, want exactly the %d Cloudflare ranges", path, len(ranges), len(cloudflareRanges))
		}
	}
}

// #475 (architect review): nginx passes a client-sent True-Client-IP through by default, and chi RealIP reads
// it before X-Real-IP, so every proxy in front of the API must clear it. Each location that proxies to the API
// clears it with an empty proxy_set_header.
func TestNginxConf_ClearsClientSentTrueClientIP(t *testing.T) {
	cases := []struct{ path, location string }{
		{"../../../deploy/nginx.conf", `location\s+/api/\s*\{`},
		{"../../../deploy/nginx/bilimbaga.conf", `location\s+/\s*\{`},
		{"../../../deploy/nginx/bilimbaga-qa.conf", `location\s+/\s*\{`},
	}
	for _, c := range cases {
		raw, err := os.ReadFile(c.path)
		if err != nil {
			t.Fatalf("read %s: %v", c.path, err)
		}
		block := regexp.MustCompile(`(?s)` + c.location + `(.*?)\n\s*\}`).FindSubmatch(raw)
		if block == nil {
			t.Errorf("%s: no proxying location block matching %s", c.path, c.location)
			continue
		}
		if !regexp.MustCompile(`(?m)^\s*proxy_set_header\s+True-Client-IP\s+"";`).Match(block[1]) {
			t.Errorf("%s: the proxying location does not clear True-Client-IP (`proxy_set_header True-Client-IP \"\";`)", c.path)
		}
	}
}

// #475 (architect review): the Cloudflare range list needs a periodic comparison with the published lists; the
// comment above it must say so, so the list is not treated as permanent.
func TestNginxEdgeVhosts_CloudflareListNoteAsksForPeriodicComparison(t *testing.T) {
	for _, path := range []string{
		"../../../deploy/nginx/bilimbaga.conf",
		"../../../deploy/nginx/bilimbaga-qa.conf",
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !regexp.MustCompile(`(?i)#[^\n]*periodic comparison`).Match(raw) {
			t.Errorf("%s: no comment above the Cloudflare range list asking for a periodic comparison", path)
		}
	}
}
