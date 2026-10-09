import { useQueryClient } from '@tanstack/react-query'
import { useMe } from '@/api/users'
import { isCustomRole, jwtRole } from '@/lib/routeRoles'

export interface MyPermissions {
  role: string | undefined
  /** True when the role is not built-in, i.e. gated by permissions (FR-BB117 AC-16). */
  isCustom: boolean
  /** Known permissions; undefined while loading, on error, or if /users/me is for another role. */
  permissions: string[] | undefined
  isLoading: boolean
}

/**
 * Permissions of the signed-in user for UI gating. Only fetched for custom roles: built-in
 * roles are decided by the static role lists, so their behaviour and request count are unchanged.
 * The cached /users/me entry is trusted only if its role matches the token's role, so a stale
 * entry from a previous session can never grant access.
 */
export function useMyPermissions(): MyPermissions {
  const qc = useQueryClient()
  const role = jwtRole(qc.getQueryData<string | null>(['auth', 'accessToken']))
  const isCustom = isCustomRole(role)
  const { data, isLoading, isFetching } = useMe({ enabled: isCustom })
  const matches = isCustom && !!data && data.role_name === role
  return {
    role,
    isCustom,
    permissions: matches ? (data?.permissions ?? []) : undefined,
    isLoading: isCustom && (isLoading || (isFetching && !!data && !matches)),
  }
}
