package rbac

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache_PermissionsFor(t *testing.T) {
	c := NewCache()
	c.LoadFromMap(map[string]PermissionSet{"qa_lead": {"users:read": true, "exams:read": true, "x:y": false}})
	assert.Equal(t, []string{"exams:read", "users:read"}, c.PermissionsFor("qa_lead"))
	assert.Equal(t, []string{}, c.PermissionsFor("unknown"))
}

// FR-BB117 AC-18: Load reflects a newly created custom role (and PermissionsFor sees it).
func TestCacheLoad_ReflectsNewlyCreatedRole(t *testing.T) {
	db, f := newFakeDB(t)
	c := NewCache()
	c.LoadFromMap(map[string]PermissionSet{"super_admin": {"users:read": true}})
	assert.False(t, c.Has("qa_lead", "exams", "read"))

	f.queue([]string{"name", "resource", "action"}, [][]driver.Value{
		{"super_admin", "users", "read"},
		{"qa_lead", "exams", "read"},
	})
	assert.NoError(t, c.Load(db))
	assert.True(t, c.Has("qa_lead", "exams", "read"))
	assert.Equal(t, []string{"exams:read"}, c.PermissionsFor("qa_lead"))
}
