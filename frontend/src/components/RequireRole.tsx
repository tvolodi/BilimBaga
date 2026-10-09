import { type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'

/** Decode the `role` claim from a JWT without signature verification. */
function jwtRole(token: string): string | undefined {
  try {
    const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
    return payload?.role as string | undefined
  } catch {
    return undefined
  }
}

interface RequireRoleProps {
  children: ReactNode
  roles: string[]
  /** When set, a signed-in user lacking a required role is sent here instead of their home (FR-BB59 AC-2). */
  unauthorizedRedirect?: string
}

export function RequireRole({ children, roles, unauthorizedRedirect }: RequireRoleProps) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  if (!token) return <Navigate to="/login" replace />
  const role = jwtRole(token)
  if (!role || !roles.includes(role)) {
    if (unauthorizedRedirect) return <Navigate to={unauthorizedRedirect} replace />
    return <Navigate to={role === 'employee' ? '/portal' : '/admin'} replace />
  }
  return <>{children}</>
}
