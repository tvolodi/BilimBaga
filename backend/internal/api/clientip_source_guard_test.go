package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #478: a client sends X-Forwarded-For (and True-Client-IP) freely, so production code under backend/internal
// must not read them. Every audit, log and rate-limit address comes from api.ClientIP, which reads the
// RemoteAddr that the router resolved from X-Real-IP (#475). Only api/clientip.go may read RemoteAddr, and the
// router's clientFromXRealIP writes it (an assignment, which is allowed everywhere).

// remoteAddrWrite matches an assignment to RemoteAddr (the router's resolution), not a read.
var remoteAddrWrite = regexp.MustCompile(`\.RemoteAddr\s*=[^=]`)

// clientIPReader is the one file allowed to read RemoteAddr directly: the helper itself.
const clientIPReader = "clientip.go"

// clientAddressViolations returns the reasons src reads the client address outside the helper.
func clientAddressViolations(name, src string) []string {
	var out []string
	if strings.Contains(src, `"X-Forwarded-For"`) {
		out = append(out, name+`: reads the X-Forwarded-For header, which a client controls; use api.ClientIP`)
	}
	for _, line := range strings.Split(src, "\n") {
		if !strings.Contains(line, ".RemoteAddr") || remoteAddrWrite.MatchString(line) {
			continue
		}
		out = append(out, name+": reads .RemoteAddr directly; use api.ClientIP: "+strings.TrimSpace(line))
	}
	return out
}

// TestClientAddressGuard_FixtureForwardedForIsFlagged shows the guard sees a forged-header read.
func TestClientAddressGuard_FixtureForwardedForIsFlagged(t *testing.T) {
	fixture := `xff := r.Header.Get("X-Forwarded-For")`
	assert.NotEmpty(t, clientAddressViolations("fixture.go", fixture))
}

// TestClientAddressGuard_FixtureRemoteAddrReadIsFlagged shows the guard sees a direct RemoteAddr read.
func TestClientAddressGuard_FixtureRemoteAddrReadIsFlagged(t *testing.T) {
	fixture := "ipAddress := r.RemoteAddr"
	assert.NotEmpty(t, clientAddressViolations("fixture.go", fixture))
}

// TestClientAddressGuard_FixtureRemoteAddrWriteIsAccepted shows the router's resolution (a write) passes.
func TestClientAddressGuard_FixtureRemoteAddrWriteIsAccepted(t *testing.T) {
	fixture := "r.RemoteAddr = ip"
	assert.Empty(t, clientAddressViolations("fixture.go", fixture))
}

// #480: a client sends True-Client-IP freely, and nothing in production may read it.
func TestClientAddressGuard_FixtureTrueClientIPReadIsFlagged(t *testing.T) {
	fixture := `ip := r.Header.Get("True-Client-IP")`
	assert.NotEmpty(t, clientAddressViolations("fixture.go", fixture))
}

// #480: X-Real-IP is read only by the router's resolution, in router/realip.go.
func TestClientAddressGuard_FixtureXRealIPReadOutsideRealIPIsFlagged(t *testing.T) {
	fixture := `ip := r.Header.Get("X-Real-IP")`
	assert.NotEmpty(t, clientAddressViolations(filepath.Join("audit", "writer.go"), fixture))
}

// #480: the router's own read of X-Real-IP, in router/realip.go, is allowed.
func TestClientAddressGuard_FixtureXRealIPReadInRealIPIsAccepted(t *testing.T) {
	fixture := `ip := r.Header.Get("X-Real-IP")`
	assert.Empty(t, clientAddressViolations(filepath.Join("router", "realip.go"), fixture))
}

// TestClientAddressGuard_NoProductionReadBypassesTheHelper scans the non-test sources under backend/internal.
func TestClientAddressGuard_NoProductionReadBypassesTheHelper(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..")) // backend/internal
	require.NoError(t, err)
	var violations []string
	scanned := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Base(path) == clientIPReader && filepath.Base(filepath.Dir(path)) == "api" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		rel, _ := filepath.Rel(root, path)
		violations = append(violations, clientAddressViolations(rel, string(b))...)
		return nil
	})
	require.NoError(t, err)
	require.NotZero(t, scanned, "the guard must scan real sources")
	assert.Empty(t, violations, "production code must read the client address through api.ClientIP only")
}
