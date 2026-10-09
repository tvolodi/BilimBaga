/**
 * Roles the signed-in user may assign on user create/edit/import (FR-BB117 D-1, FR-BB18 AC-4/6/10).
 * Mirrors backend users.canReachRole/checkRoleAssignment (PR #222); the backend stays
 * authoritative and answers 403 FORBIDDEN for anything this helper hides or misjudges.
 */

/** Strict rank hierarchy of built-in roles (backend builtinRank). */
export const ROLE_RANK: Readonly<Record<string, number>> = {
  employee: 1,
  examiner: 2,
  department_admin: 3,
  super_admin: 4,
}

/** Permissions that a non-super_admin may never hand out through a custom role. */
const SENSITIVE_PERMISSIONS = ['roles:read', 'roles:manage', 'tenant:manage']

export interface AssignableRoleCandidate {
  id: string
  name: string
  /** "resource:action" list; only present when the API exposes it. Unknown = undefined. */
  permissions?: string[]
}

function permissionSubset(role: AssignableRoleCandidate, callerPermissions: readonly string[] | undefined): boolean {
  // Fail closed like the backend: unknown/empty permission sets are not provably a subset.
  if (!callerPermissions || !role.permissions || role.permissions.length === 0) return false
  return role.permissions.every((p) => callerPermissions.includes(p))
}

/** True when `callerRole` may assign `role`. */
export function canAssignRole(
  callerRole: string | undefined,
  callerPermissions: readonly string[] | undefined,
  role: AssignableRoleCandidate,
): boolean {
  if (!callerRole) return false
  if (callerRole === 'super_admin') return true
  if (role.name === 'super_admin') return false

  const callerRank = ROLE_RANK[callerRole]
  const roleRank = ROLE_RANK[role.name]
  if (callerRank !== undefined) {
    // Built-in caller.
    if (roleRank !== undefined) return roleRank < callerRank
    return !hasSensitive(role) && permissionSubset(role, callerPermissions)
  }
  // Custom caller: ranks below department_admin.
  if (roleRank !== undefined && roleRank >= ROLE_RANK.department_admin) return false
  return !hasSensitive(role) && permissionSubset(role, callerPermissions)
}

function hasSensitive(role: AssignableRoleCandidate): boolean {
  return (role.permissions ?? []).some((p) => SENSITIVE_PERMISSIONS.includes(p))
}

export function assignableRoles<T extends AssignableRoleCandidate>(
  callerRole: string | undefined,
  callerPermissions: readonly string[] | undefined,
  roles: readonly T[],
): T[] {
  return roles.filter((r) => canAssignRole(callerRole, callerPermissions, r))
}

/** Maps a server error from a user mutation to an i18n key (users.messages.*). */
export function userErrorKey(err: unknown): string | null {
  const code = (err as { code?: string } | null)?.code
  return code === 'FORBIDDEN' ? 'users.messages.forbidden' : null
}
