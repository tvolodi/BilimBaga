package ai

import (
	"sync"
	"time"
)

// maxScopedInsightEntries bounds the in-process scoped insight cache.
const maxScopedInsightEntries = 512

// scopedInsightKey identifies one cached insight: the exam plus the hash of
// the department-id set the caller may see (deptscope.ScopeKey). A struct key
// (not a joined string) rules out delimiter collisions in examID.
type scopedInsightKey struct {
	examID string
	scope  string
}

type scopedInsightEntry struct {
	insights    []string
	generatedAt time.Time
}

// scopedInsightCache is a small bounded TTL cache for insights generated for
// department-scoped callers (ISS-218). The persistent ai_insight_cache table
// has exam_id as its PRIMARY KEY and no scope column, so scoped results cannot
// be stored there without a migration; this in-process cache is the fallback.
// It is per API process (not shared across replicas, lost on restart): the
// worst case is an extra paid call, never a cross-scope read, because the key
// always carries the scope hash.
type scopedInsightCache struct {
	mu      sync.Mutex
	max     int
	ttl     time.Duration
	entries map[scopedInsightKey]scopedInsightEntry
	now     func() time.Time
}

func newScopedInsightCache(max int, ttl time.Duration) *scopedInsightCache {
	return &scopedInsightCache{max: max, ttl: ttl, entries: map[scopedInsightKey]scopedInsightEntry{}, now: time.Now}
}

// get returns a copy of a fresh entry (Cached=true) or nil.
func (c *scopedInsightCache) get(k scopedInsightKey) *InsightResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[k]
	if !ok {
		return nil
	}
	if c.now().Sub(e.generatedAt) >= c.ttl {
		delete(c.entries, k)
		return nil
	}
	return &InsightResult{Insights: append([]string(nil), e.insights...), GeneratedAt: e.generatedAt, Cached: true}
}

// put stores (or overwrites) an entry, evicting expired entries and then the
// oldest one when the cache is full.
func (c *scopedInsightCache) put(k scopedInsightKey, insights []string, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[k]; !exists && len(c.entries) >= c.max {
		now := c.now()
		for ek, e := range c.entries {
			if now.Sub(e.generatedAt) >= c.ttl {
				delete(c.entries, ek)
			}
		}
		if len(c.entries) >= c.max {
			var oldestKey scopedInsightKey
			var oldest time.Time
			first := true
			for ek, e := range c.entries {
				if first || e.generatedAt.Before(oldest) {
					oldestKey, oldest, first = ek, e.generatedAt, false
				}
			}
			delete(c.entries, oldestKey)
		}
	}
	c.entries[k] = scopedInsightEntry{insights: append([]string(nil), insights...), generatedAt: at}
}
