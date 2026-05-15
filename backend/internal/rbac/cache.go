package rbac

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
)

// PermissionSet maps "resource:action" → true for fast O(1) lookup.
type PermissionSet map[string]bool

// Cache holds the in-memory role → permission mapping loaded from the database at startup.
type Cache struct {
	mu   sync.RWMutex
	data map[string]PermissionSet // key: role name
}

// NewCache returns an empty Cache ready to be populated via Load.
func NewCache() *Cache {
	return &Cache{
		data: make(map[string]PermissionSet),
	}
}

// Load queries the database and builds the in-memory permission cache.
// It should be called once at startup after the DB connection is established.
func (c *Cache) Load(db *sqlx.DB) error {
	rows, err := db.Query(`
		SELECT r.name, p.resource, p.action
		FROM role_permissions rp
		JOIN roles r       ON r.id = rp.role_id
		JOIN permissions p ON p.id = rp.permission_id
	`)
	if err != nil {
		return fmt.Errorf("rbac: load cache: %w", err)
	}
	defer rows.Close()

	data := make(map[string]PermissionSet)
	for rows.Next() {
		var roleName, resource, action string
		if err := rows.Scan(&roleName, &resource, &action); err != nil {
			return fmt.Errorf("rbac: load cache: scan row: %w", err)
		}
		if data[roleName] == nil {
			data[roleName] = make(PermissionSet)
		}
		data[roleName][resource+":"+action] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rbac: load cache: iterate rows: %w", err)
	}

	c.mu.Lock()
	c.data = data
	c.mu.Unlock()

	return nil
}

// Has returns true if the given role holds the specified resource:action permission.
// It performs an O(1) lookup against the in-memory cache — no database query per request.
func (c *Cache) Has(role, resource, action string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[role][resource+":"+action]
}

// LoadFromMap populates the cache directly from the provided map. It is intended
// for use in tests and integration setups where a live database is not available.
func (c *Cache) LoadFromMap(data map[string]PermissionSet) {
	c.mu.Lock()
	c.data = data
	c.mu.Unlock()
}
