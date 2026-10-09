package rbac

import (
	"database/sql/driver"
	"errors"
	"testing"
)

func TestCacheLoad_BuildsPermissionSets(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"name", "resource", "action"}, [][]driver.Value{
		{"super_admin", "users", "read"},
		{"super_admin", "users", "manage"},
		{"learner", "portal", "read"},
	})
	c := NewCache()
	if err := c.Load(db); err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, tc := range []struct {
		role, res, act string
		want           bool
	}{
		{"super_admin", "users", "read", true},
		{"super_admin", "users", "manage", true},
		{"learner", "portal", "read", true},
		{"learner", "users", "read", false},
		{"ghost", "users", "read", false},
	} {
		if got := c.Has(tc.role, tc.res, tc.act); got != tc.want {
			t.Errorf("Has(%s,%s,%s)=%v want %v", tc.role, tc.res, tc.act, got, tc.want)
		}
	}
}

func TestCacheLoad_ReplacesPreviousData(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"name", "resource", "action"}, [][]driver.Value{{"learner", "portal", "read"}})
	c := NewCache()
	c.LoadFromMap(map[string]PermissionSet{"old": {"x:y": true}})
	if err := c.Load(db); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Has("old", "x", "y") {
		t.Error("stale permission survived reload")
	}
	if !c.Has("learner", "portal", "read") {
		t.Error("new permission missing")
	}
}

func TestCacheLoad_QueryError_KeepsExistingCache(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("db down")
	c := NewCache()
	c.LoadFromMap(map[string]PermissionSet{"learner": {"portal:read": true}})
	if err := c.Load(db); err == nil {
		t.Fatal("expected error")
	}
	if !c.Has("learner", "portal", "read") {
		t.Error("cache should be untouched after failed load")
	}
}

func TestCacheLoad_ScanError(t *testing.T) {
	db, f := newFakeDB(t)
	// Two columns for three scan targets -> scan error.
	f.queue([]string{"name", "resource"}, [][]driver.Value{{"a", "b"}})
	if err := NewCache().Load(db); err == nil {
		t.Fatal("expected scan error")
	}
}

func TestCacheLoadFromMap(t *testing.T) {
	c := NewCache()
	if c.Has("r", "a", "b") {
		t.Fatal("empty cache must deny")
	}
	c.LoadFromMap(map[string]PermissionSet{"r": {"a:b": true}})
	if !c.Has("r", "a", "b") || c.Has("r", "a", "c") {
		t.Error("LoadFromMap lookup wrong")
	}
}
