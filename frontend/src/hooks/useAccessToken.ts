import { useQuery } from '@tanstack/react-query'

/**
 * Subscribed read of the access token held under ['auth', 'accessToken'] (ISS-249).
 *
 * Guards and layouts that gate rendering on the token must use this hook, not
 * getQueryData: a subscription re-renders them when the token is cleared after mount
 * (endRevokedSession on TOKEN_REVOKED), so they redirect to /login instead of staying on a blank page.
 *
 * Observer only: useRefreshToken and useLogin own the value, so this never fetches
 * (enabled: false), the same pattern as PreferredLocaleSync.
 */
export function useAccessToken(): { token: string | null | undefined; status: 'pending' | 'error' | 'success' } {
  const { data, status } = useQuery<string | null>({
    queryKey: ['auth', 'accessToken'],
    queryFn: () => null,
    enabled: false,
  })
  return { token: data, status }
}
