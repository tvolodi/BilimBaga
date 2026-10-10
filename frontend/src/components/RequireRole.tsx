import { type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { useAccessToken } from '@/hooks/useAccessToken'
import { useMyPermissions } from '@/hooks/useMyPermissions'
import { can, jwtRole } from '@/lib/routeRoles'

interface RequireRoleProps {
  children: ReactNode
  roles: string[]
  /** When set, a signed-in user lacking a required role is sent here instead of their home (FR-BB59 AC-2). */
  unauthorizedRedirect?: string
  /**
   * FR-BB117 AC-16: a custom (non-built-in) role is admitted when it holds this permission.
   * Built-in roles are always decided by `roles` only. UI gating only; the API enforces access.
   */
  permission?: string
  /** Admit any custom role (used for the admin shell; each page then gates by permission). */
  allowCustomRole?: boolean
}

export function RequireRole({
  children,
  roles,
  unauthorizedRedirect,
  permission,
  allowCustomRole,
}: RequireRoleProps) {
  // Subscribed (ISS-249): a token cleared after mount redirects to /login instead of a blank page.
  const { token } = useAccessToken()
  const location = useLocation()
  const { isCustom, permissions, isLoading } = useMyPermissions()
  // `from` lets LoginPage send the user back here after signing in again (AC-11).
  if (!token) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  const role = jwtRole(token)
  if (role && roles.includes(role)) return <>{children}</>

  if (isCustom) {
    if (allowCustomRole) return <>{children}</>
    if (permission) {
      if (isLoading) return <FullPageSpinner />
      if (can(permissions, permission)) return <>{children}</>
    }
    // Custom roles land on /admin, which routes them to a page they may open (no /login bounce).
    return <Navigate to="/admin" replace />
  }

  if (unauthorizedRedirect) return <Navigate to={unauthorizedRedirect} replace />
  return <Navigate to={role === 'employee' ? '/portal' : '/admin'} replace />
}
