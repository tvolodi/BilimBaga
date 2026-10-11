package router_test

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/config"
)

// #519: an insights answer waits for the provider for up to AI_TIMEOUT_SECONDS. Every hop that proxies the insights
// path must wait longer than that: the container nginx and both edge vhosts. The location names the path with ^~, so it
// wins over the /api/ location. deploy/nginx/bilimbaga-test.conf (customer demo, owner-only) is not covered here.

var (
	insightsLocation = regexp.MustCompile(`location \^~ /api/v1/admin/ai/ \{([^}]*)\}`)
	insightsTimeout  = regexp.MustCompile(`proxy_read_timeout\s+(\d+)s;`)
	insightsProxy    = regexp.MustCompile(`proxy_pass\s+http://`)
	insightsClearTCI = regexp.MustCompile(`proxy_set_header\s+True-Client-IP\s+"";`)
)

func TestNginx_InsightsLocationWaitsLongerThanTheLargestAITimeout(t *testing.T) {
	need := config.MaxAITimeoutSeconds + 30
	for _, path := range []string{
		"../../../deploy/nginx.conf",
		"../../../deploy/nginx/bilimbaga.conf",
		"../../../deploy/nginx/bilimbaga-qa.conf",
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		m := insightsLocation.FindSubmatch(raw)
		if m == nil {
			t.Errorf("%s: no `location ^~ /api/v1/admin/ai/` block", path)
			continue
		}
		body := string(m[1])
		pt := insightsTimeout.FindStringSubmatch(body)
		if pt == nil {
			t.Errorf("%s: the insights location sets no proxy_read_timeout", path)
			continue
		}
		if secs, _ := strconv.Atoi(pt[1]); secs < need {
			t.Errorf("%s: insights proxy_read_timeout %ds must be at least %ds (AI_TIMEOUT_SECONDS max + 30)", path, secs, need)
		}
		if !insightsProxy.MatchString(body) {
			t.Errorf("%s: the insights location must proxy to the API", path)
		}
		if !insightsClearTCI.MatchString(body) {
			t.Errorf("%s: the insights location must clear True-Client-IP (#475), as /api/ does", path)
		}
	}
}
