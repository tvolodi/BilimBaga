/**
 * Role lists for RBAC-aligned admin route guards.
 * Mirrors backend permission seed (migrations/005_rbac.up.sql).
 */

/** audit:read — super_admin only (FR-BB114 AC-1). */
export const AUDIT_READ_ROLES = ['super_admin']

/** reports:read — super_admin, department_admin, examiner (FR-BB59 AC-2). */
export const REPORTS_READ_ROLES = ['super_admin', 'department_admin', 'examiner']
