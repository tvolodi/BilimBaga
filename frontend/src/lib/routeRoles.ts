/**
 * Role lists for RBAC-aligned admin route guards.
 * Mirrors backend permission seed (migrations/005_rbac.up.sql).
 */

/** audit:read — super_admin only (FR-BB114 AC-1). */
export const AUDIT_READ_ROLES = ['super_admin']

/** reports:read — super_admin, department_admin, examiner (FR-BB59 AC-2). */
export const REPORTS_READ_ROLES = ['super_admin', 'department_admin', 'examiner']

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
