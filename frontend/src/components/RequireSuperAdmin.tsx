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

interface RequireSuperAdminProps {
  children: ReactNode
}

export function RequireSuperAdmin({ children }: RequireSuperAdminProps) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  if (!token) return <Navigate to="/login" replace />
  if (jwtRole(token) !== 'super_admin') return <Navigate to="/admin" replace />
  return <>{children}</>
}
