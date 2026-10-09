import { type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { FullPageSpinner } from '@/components/FullPageSpinner'

export function RequireAuth({ children }: { children: ReactNode }) {
  // Observer of the token that useRefreshToken / useLogin own; never fetched here. Subscribing
  // (not just reading the cache) means a token cleared after mount, e.g. endRevokedSession on
  // TOKEN_REVOKED (ISS-249), re-renders this guard and redirects to /login.
  const { data: token, status } = useQuery<string | null>({
    queryKey: ['auth', 'accessToken'],
    queryFn: () => null,
    enabled: false,
  })

  // Query is still initialising (e.g. useRefreshToken hasn't settled yet after
  // a fresh login). Show a spinner instead of bouncing the user to /login.
  if (status === 'pending') {
    return <FullPageSpinner />
  }

  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}
