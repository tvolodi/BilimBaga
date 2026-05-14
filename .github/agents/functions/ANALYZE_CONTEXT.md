# Function: ANALYZE_CONTEXT

> Analyze gathered context to determine scope, affected components, and implementation approach.

## Metadata

| Property | Value |
|----------|-------|
| **Category** | Analysis |
| **Used By** | Issue Resolution, Requirement Implementation |
| **Depends On** | Reading source files + requirement or bug description |

---

## Purpose

Analyze gathered context to:
- Determine the full scope of work
- Identify all affected components (Go packages, routes, React components)
- Plan the implementation order
- Surface risks before writing code

---

## Output Structure

```typescript
{
  scope: {
    type: 'backend-only' | 'frontend-only' | 'full-stack' | 'db-only' | 'config-only',
    go_packages: string[],         // e.g. ["auth", "users"]
    api_endpoints: string[],       // e.g. ["POST /api/v1/auth/login"]
    db_tables: string[],           // e.g. ["users", "audit_log"]
    react_pages: string[],         // e.g. ["LoginPage", "UserListPage"]
    react_components: string[],    // e.g. ["UserDrawer"]
    migrations_needed: boolean,
    estimated_complexity: 'low' | 'medium' | 'high'
  },
  affected_files: {
    path: string,
    action: 'create' | 'modify',
    reason: string
  }[],
  approach: {
    strategy: string,
    implementation_order: string[]
  },
  risks: {
    description: string,
    mitigation: string
  }[]
}
```

---

## Steps

1. Read the requirement doc or bug description in full.
2. Trace the affected code path: route → handler → service → repository → SQL → frontend.
3. List every file that must change or be created.
4. Identify required migrations (new tables, new columns, new indexes).
5. State risks: auth middleware missing, RBAC not applied, i18n keys not added, missing error paths.
