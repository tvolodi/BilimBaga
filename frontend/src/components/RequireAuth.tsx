import { type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'

export function RequireAuth({ children }: { children: ReactNode }) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}
