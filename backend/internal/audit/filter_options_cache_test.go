package audit

// #563: the filter options are cached per tenant for filterOptionsTTL, so a tenant's log is scanned once per TTL and not on
// every request. The cache belongs to one service; errors are not cached. The tests drive the clock and count the loads.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cacheTestStart = time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC)

// newCacheTestService returns a service whose loader counts its calls per tenant and answers with the tenant's own action.
func newCacheTestService(now *time.Time, loads map[string]int) *service {
	return &service{
		loadFilterOptions: func(_ context.Context, tenantID string) (AuditFilterOptions, error) {
			loads[tenantID]++
			return AuditFilterOptions{Actions: []string{"action." + tenantID}, EntityTypes: []string{"entity"}}, nil
		},
		now:           func() time.Time { return *now },
		filterOptions: map[string]cachedFilterOptions{},
	}
}

func TestFilterOptionsCache_ServesTheTenantWithinTTL(t *testing.T) {
	now := cacheTestStart
	loads := map[string]int{}
	svc := newCacheTestService(&now, loads)
	ctx := context.Background()

	first, err := svc.FilterOptions(ctx, "tenant-a")
	require.NoError(t, err)
	now = now.Add(filterOptionsTTL - time.Second)
	second, err := svc.FilterOptions(ctx, "tenant-a")
	require.NoError(t, err)

	assert.Equal(t, 1, loads["tenant-a"], "the second call is served from the cache")
	assert.Equal(t, first, second)
}

func TestFilterOptionsCache_ReadsAgainAtTheTTL(t *testing.T) {
	now := cacheTestStart
	loads := map[string]int{}
	svc := newCacheTestService(&now, loads)
	ctx := context.Background()

	_, err := svc.FilterOptions(ctx, "tenant-a")
	require.NoError(t, err)
	now = now.Add(filterOptionsTTL)
	_, err = svc.FilterOptions(ctx, "tenant-a")
	require.NoError(t, err)

	assert.Equal(t, 2, loads["tenant-a"], "once the TTL has passed, the log is read again and a new action shows up")
}

func TestFilterOptionsCache_IsPerTenant(t *testing.T) {
	now := cacheTestStart
	loads := map[string]int{}
	svc := newCacheTestService(&now, loads)
	ctx := context.Background()

	a, err := svc.FilterOptions(ctx, "tenant-a")
	require.NoError(t, err)
	b, err := svc.FilterOptions(ctx, "tenant-b")
	require.NoError(t, err)

	assert.Equal(t, []string{"action.tenant-a"}, a.Actions)
	assert.Equal(t, []string{"action.tenant-b"}, b.Actions, "one tenant's options are never served to another")
	assert.Equal(t, 1, loads["tenant-a"])
	assert.Equal(t, 1, loads["tenant-b"])
}

func TestFilterOptionsCache_ErrorIsNotCached(t *testing.T) {
	now := cacheTestStart
	loads := map[string]int{}
	svc := newCacheTestService(&now, loads)
	fail := true
	svc.loadFilterOptions = func(_ context.Context, _ string) (AuditFilterOptions, error) {
		if fail {
			return AuditFilterOptions{}, errors.New("connection reset")
		}
		return AuditFilterOptions{Actions: []string{"action.recovered"}}, nil
	}
	ctx := context.Background()

	_, err := svc.FilterOptions(ctx, "tenant-a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection reset")

	fail = false
	opts, err := svc.FilterOptions(ctx, "tenant-a")
	require.NoError(t, err)
	assert.Equal(t, []string{"action.recovered"}, opts.Actions, "the failed read was not cached, so the next call reads the log")
}
