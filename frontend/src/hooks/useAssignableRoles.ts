import { useMemo } from 'react'
import { useRoles, type RoleRow } from '@/api/users'
import { assignableRoles } from '@/lib/assignableRoles'
import { useMyPermissions } from './useMyPermissions'

/** Roles the signed-in user may assign (FR-BB117 D-1), plus the full list for display of current roles. */
export function useAssignableRoles(): { roles: RoleRow[]; allRoles: RoleRow[] } {
  const { data: allRoles = [] } = useRoles()
  const { role, permissions } = useMyPermissions()
  const roles = useMemo(() => assignableRoles(role, permissions, allRoles), [role, permissions, allRoles])
  return { roles, allRoles }
}
