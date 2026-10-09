/**
 * Role lists for RBAC-aligned admin route guards.
 * Mirrors backend permission seed (migrations/005_rbac.up.sql).
 */

/** audit:read — super_admin only (FR-BB114 AC-1). */
export const AUDIT_READ_ROLES = ['super_admin']

/** reports:read — super_admin, department_admin, examiner (FR-BB59 AC-2). */
export const REPORTS_READ_ROLES = ['super_admin', 'department_admin', 'examiner']

/** roles:read — super_admin only; roles:* can never be granted to a custom role (FR-BB117 AC-6/AC-11). */
export const ROLES_MANAGE_ROLES = ['super_admin']

/** Roles allowed into the admin shell by name (built-in staff roles; hr_admin is legacy). */
export const ADMIN_SHELL_ROLES = ['super_admin', 'department_admin', 'examiner', 'hr_admin']

/**
 * Role names whose UI access is decided by the role lists above. Any other role name is a
 * custom role (FR-BB117) and is gated by permissions from GET /users/me instead (AC-16).
 */
export const BUILTIN_ROLES = [...ADMIN_SHELL_ROLES, 'employee']

export function isCustomRole(role: string | undefined): boolean {
  return !!role && !BUILTIN_ROLES.includes(role)
}

/** Permission required per admin area for custom roles (FR-BB117 AC-16 mapping). */
export const ROUTE_PERMISSIONS = {
  dashboard: 'reports:read',
  users: 'users:read',
  departments: 'departments:read',
  questions: 'questions:read',
  questionsWrite: 'questions:write',
  categories: 'categories:manage',
  tags: 'tags:read',
  exams: 'exams:read',
  examsWrite: 'exams:write',
  grading: 'grading:read',
  reports: 'reports:read',
  audit: 'audit:read',
  settings: 'tenant:manage',
  roles: 'roles:read',
} as const

/**
 * UI-only permission check. The backend stays authoritative: this only decides what a
 * custom role sees. Deny when permissions are unknown.
 */
export function can(permissions: readonly string[] | undefined, permission: string): boolean {
  return !!permissions && permissions.includes(permission)
}

/** Decode the `role` claim from a JWT without signature verification (UI gating only). */
export function jwtRole(token: string | null | undefined): string | undefined {
  if (!token) return undefined
  try {
    const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
    return payload?.role as string | undefined
  } catch {
    return undefined
  }
}

/** Admin pages in landing-preference order, used to route a custom role to its first allowed page. */
export const ADMIN_LANDING_ORDER: ReadonlyArray<{ path: string; permission: string }> = [
  { path: '/admin/dashboard', permission: ROUTE_PERMISSIONS.dashboard },
  { path: '/admin/users', permission: ROUTE_PERMISSIONS.users },
  { path: '/admin/departments', permission: ROUTE_PERMISSIONS.departments },
  { path: '/admin/questions', permission: ROUTE_PERMISSIONS.questions },
  { path: '/admin/categories', permission: ROUTE_PERMISSIONS.categories },
  { path: '/admin/tags', permission: ROUTE_PERMISSIONS.tags },
  { path: '/admin/exams', permission: ROUTE_PERMISSIONS.exams },
  { path: '/admin/grading', permission: ROUTE_PERMISSIONS.grading },
  { path: '/admin/reports', permission: ROUTE_PERMISSIONS.reports },
  { path: '/admin/audit', permission: ROUTE_PERMISSIONS.audit },
]
