import { type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { useAccessToken } from '@/hooks/useAccessToken'

export function RequireAuth({ children }: { children: ReactNode }) {
  // Subscribed to the token that useRefreshToken / useLogin own (ISS-249): a token cleared after
  // mount, e.g. endRevokedSession on TOKEN_REVOKED, re-renders this guard and redirects to /login.
  const { token, status } = useAccessToken()
  const location = useLocation()

  // Query is still initialising (e.g. useRefreshToken hasn't settled yet after
  // a fresh login). Show a spinner instead of bouncing the user to /login.
  if (status === 'pending') {
    return <FullPageSpinner />
  }

  // `from` lets LoginPage send the user back here after signing in again (AC-11).
  if (!token) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  return <>{children}</>
}
