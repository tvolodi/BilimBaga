import { type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAccessToken } from '@/hooks/useAccessToken'

/** Decode the `role` claim from a JWT without signature verification. */
function jwtRole(token: string): string | undefined {
  try {
    const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
    return payload?.role as string | undefined
  } catch {
    return undefined
  }
}

interface RequireSuperAdminProps {
  children: ReactNode
}

export function RequireSuperAdmin({ children }: RequireSuperAdminProps) {
  // Subscribed (ISS-249): a token cleared after mount redirects to /login instead of a blank page.
  const { token } = useAccessToken()
  const location = useLocation()
  // `from` lets LoginPage send the user back here after signing in again (AC-11).
  if (!token) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  if (jwtRole(token) !== 'super_admin') return <Navigate to="/admin" replace />
  return <>{children}</>
}
