import { type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'

interface CurrentUser {
  role: string
}

interface RequireRoleProps {
  children: ReactNode
  roles: string[]
}

export function RequireRole({ children, roles }: RequireRoleProps) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  const user = qc.getQueryData<CurrentUser>(['auth', 'currentUser'])

  if (!token) return <Navigate to="/login" replace />
  if (!user || !roles.includes(user.role)) return <Navigate to="/login" replace />
  return <>{children}</>
}
