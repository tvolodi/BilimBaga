package rbac

import (
	"testing"
)

func TestCacheHas(t *testing.T) {
	// Build a cache with test data matching the seeded permission matrix.
	c := NewCache()
	c.data = map[string]PermissionSet{
		"super_admin": {
			"users:read": true, "users:manage": true,
			"questions:read": true, "questions:write": true,
			"exams:read": true, "exams:write": true, "exams:assign": true,
			"reports:read":  true,
			"portal:read":   true, "portal:submit": true,
			"tenant:manage": true, "audit:read": true,
		},
		"department_admin": {
			"users:read": true, "users:manage": true,
			"questions:read": true,
			"exams:read":     true, "exams:assign": true,
			"reports:read":   true,
		},
		"examiner": {
			"questions:read": true, "questions:write": true,
			"exams:read": true, "exams:write": true, "exams:assign": true,
			"reports:read": true,
		},
		"employee": {
			"portal:read":   true,
			"portal:submit": true,
		},
	}

	tests := []struct {
		role     string
		resource string
		action   string
		want     bool
	}{
		// super_admin holds every permission
		{"super_admin", "questions", "write", true},
		{"super_admin", "tenant", "manage", true},
		{"super_admin", "audit", "read", true},
		{"super_admin", "portal", "read", true},
		{"super_admin", "portal", "submit", true},
		// employee — only portal:read and portal:submit
		{"employee", "questions", "write", false},
		{"employee", "portal", "read", true},
		{"employee", "portal", "submit", true},
		{"employee", "exams", "read", false},
		// examiner
		{"examiner", "questions", "write", true},
		{"examiner", "questions", "read", true},
		{"examiner", "tenant", "manage", false},
		{"examiner", "users", "manage", false},
		// department_admin
		{"department_admin", "questions", "write", false},
		{"department_admin", "questions", "read", true},
		{"department_admin", "tenant", "manage", false},
		// unknown role
		{"unknown_role", "questions", "read", false},
		{"unknown_role", "portal", "read", false},
	}

	for _, tt := range tests {
		name := tt.role + " " + tt.resource + ":" + tt.action
		t.Run(name, func(t *testing.T) {
			got := c.Has(tt.role, tt.resource, tt.action)
			if got != tt.want {
				t.Errorf("Has(%q, %q, %q) = %v, want %v", tt.role, tt.resource, tt.action, got, tt.want)
			}
		})
	}
}
