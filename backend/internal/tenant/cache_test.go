package tenant

// cache_test.go — FR-BB65: concurrent-safety tests for the in-memory tenant
// config cache embedded in service.  The service uses a sync.RWMutex to protect
// the cache map; these tests verify correct behaviour under concurrent access.

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCache_GetPublicConfig_NilOnEmpty verifies that GetPublicConfig returns an
// empty map (not nil, not panicking) before LoadCache is called.
func TestCache_GetPublicConfig_NilOnEmpty(t *testing.T) {
	svc := NewService(newMockRepository(nil))
	cfg := svc.GetPublicConfig()
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg)
}

// TestCache_SetAndGet verifies that after LoadCache the stored config is
// returned correctly by GetPublicConfig and GetAllConfig.
func TestCache_SetAndGet(t *testing.T) {
	seed := defaultSeedData()
	svc := NewService(newMockRepository(seed))
	require.NoError(t, svc.LoadCache(context.Background()))

	pub := svc.GetPublicConfig()
	assert.Equal(t, json.RawMessage(`"BilimBaga"`), pub["app_name"])
	assert.NotContains(t, pub, "logo", "logo must be excluded from public config")

	all := svc.GetAllConfig()
	assert.Contains(t, all, "logo")
	assert.Contains(t, all, "app_name")
}

// TestCache_GetLogo_NilWhenNoLogo verifies that GetLogoData returns (nil,"",nil)
// when the logo key holds a JSON null.
func TestCache_GetLogo_NilWhenNoLogo(t *testing.T) {
	svc := NewService(newMockRepository(defaultSeedData()))
	require.NoError(t, svc.LoadCache(context.Background()))

	imgBytes, _, err := svc.GetLogoData()
	require.NoError(t, err)
	assert.Nil(t, imgBytes)
}

// TestCache_InvalidateAndRefresh_ClearsAndReloads verifies that
// InvalidateAndRefresh picks up new data from the repository.
func TestCache_InvalidateAndRefresh_ClearsAndReloads(t *testing.T) {
	seed := defaultSeedData()
	repo := newMockRepository(seed)
	svc := NewService(repo)
	require.NoError(t, svc.LoadCache(context.Background()))

	// Mutate the underlying repo to simulate an out-of-band write.
	repo.(*mockRepository).data["app_name"] = json.RawMessage(`"NewName"`)
	require.NoError(t, svc.InvalidateAndRefresh(context.Background()))

	pub := svc.GetPublicConfig()
	assert.Equal(t, json.RawMessage(`"NewName"`), pub["app_name"])
}

// TestCache_ConcurrentAccess stress-tests the RWMutex by running concurrent
// readers (GetPublicConfig) alongside a writer (InvalidateAndRefresh).
// The test must not deadlock, race, or panic.
func TestCache_ConcurrentAccess(t *testing.T) {
	repo := newMockRepository(defaultSeedData())
	svc := NewService(repo)
	require.NoError(t, svc.LoadCache(context.Background()))

	const (
		readers  = 20
		writers  = 5
		iters    = 50
	)

	var wg sync.WaitGroup

	// Spawn concurrent readers.
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				cfg := svc.GetPublicConfig()
				_ = cfg
				all := svc.GetAllConfig()
				_ = all
				_ = svc.GetDefaultLocale()
				_ = svc.GetAvailableLocales()
			}
		}()
	}

	// Spawn concurrent writers (InvalidateAndRefresh).
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_ = svc.InvalidateAndRefresh(context.Background())
			}
		}()
	}

	wg.Wait()
	// If we reach here without a data race or deadlock the test passes.
}

// TestCache_UpdateConfig_InvalidatesAndReloads verifies that UpdateConfig
// triggers a reload so subsequent reads reflect the new values.
func TestCache_UpdateConfig_InvalidatesAndReloads(t *testing.T) {
	repo := newMockRepository(defaultSeedData())
	svc := NewService(repo)
	require.NoError(t, svc.LoadCache(context.Background()))

	updated, err := svc.UpdateConfig(context.Background(), map[string]json.RawMessage{
		"app_name": json.RawMessage(`"Acme Corp"`),
	})
	require.NoError(t, err)
	assert.Contains(t, updated, "app_name")

	pub := svc.GetPublicConfig()
	assert.Equal(t, json.RawMessage(`"Acme Corp"`), pub["app_name"])
}
