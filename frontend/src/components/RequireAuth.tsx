import { type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { FullPageSpinner } from '@/components/FullPageSpinner'

export function RequireAuth({ children }: { children: ReactNode }) {
  const qc = useQueryClient()
  const queryState = qc.getQueryState<string | null>(['auth', 'accessToken'])

  // Query is still initialising (e.g. useRefreshToken hasn't settled yet after
  // a fresh login). Show a spinner instead of bouncing the user to /login.
  if (!queryState || queryState.status === 'pending') {
    return <FullPageSpinner />
  }

  const token = queryState.data
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}
