import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { clearPasswordChangeRequired, markPasswordChangeRequired } from '@/lib/passwordChangeRequired'
import { clearE2eToken, readE2eToken, writeE2eToken } from '@/lib/e2eTokenSeed'
import { clearNonPublicQueries } from '@/lib/queryCache'

export interface ApiError {
  code: string
  message: string
  details?: Record<string, string>
}

export interface CurrentUser {
  id: string
  full_name: string
  email: string
  role: string
  force_password_change: boolean
}


interface LoginPayload {
  email: string
  password: string
}

interface LoginResponse {
  access_token: string
  token_type: string
  expires_in: number
  user: CurrentUser
}

interface ChangePasswordPayload {
  current_password: string
  new_password: string
}

/** ISS-171: change-password revokes every earlier access token and returns the caller's replacement. */
interface ChangePasswordResponse {
  message: string
  access_token: string
  token_type: string
  expires_in: number
}

/**
 * ISS-160: the token claims carry no force_password_change, so ask GET /users/me (allowlisted
 * while the flag is set) once per bootstrap. A flagged user is redirected before any data query
 * fails. Best effort: any failure leaves the 403 handler as the safety net.
 */
async function checkForcePasswordChange(qc: QueryClient, token: string): Promise<void> {
  try {
    const res = await fetch('/api/v1/users/me', { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) return
    const json = (await res.json()) as { data?: { force_password_change?: boolean } | null }
    if (json.data?.force_password_change === true) {
      const user = qc.getQueryData<CurrentUser>(['auth', 'currentUser'])
      if (user) qc.setQueryData(['auth', 'currentUser'], { ...user, force_password_change: true })
      markPasswordChangeRequired(qc)
    }
  } catch {
    /* ignore */
  }
}

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const payload = token.split('.')[1]
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation<LoginResponse, ApiError, LoginPayload>({
    mutationFn: async (payload) => {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
        credentials: 'include',
      })
      const json = await res.json()
      if (!res.ok) throw json.error as ApiError
      return json.data as LoginResponse
    },
    onSuccess: (data) => {
      // #436: no data from the previous session survives a login (FR-BB116: its profile and locale included).
      clearNonPublicQueries(qc)
      qc.setQueryData(['auth', 'currentUser'], data.user)
      qc.setQueryData(['auth', 'accessToken'], data.access_token)
      // A seeded e2e build boots from the stored token, so a fresh sign-in must replace the seed (ISS-249).
      if (readE2eToken()) writeE2eToken(data.access_token)
      clearPasswordChangeRequired(qc) // the login response carries the authoritative flag
    },
  })
}

export function useChangePassword() {
  const qc = useQueryClient()
  return useMutation<ChangePasswordResponse, ApiError, ChangePasswordPayload>({
    mutationFn: async (payload) => {
      const token = qc.getQueryData<string>(['auth', 'accessToken'])
      const res = await fetch('/api/v1/auth/change-password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(payload),
        credentials: 'include',
      })
      const json = await res.json()
      if (!res.ok) throw json.error as ApiError
      return json.data as ChangePasswordResponse
    },
    onSuccess: (data) => {
      // The pre-change token is now rejected with TOKEN_REVOKED; switch to the new one at once.
      if (data?.access_token) {
        qc.setQueryData(['auth', 'accessToken'], data.access_token)
        // E2E-seeded token (global-setup) would otherwise be re-served by useRefreshToken.
        try {
          if (readE2eToken()) writeE2eToken(data.access_token)
        } catch { /* ignore */ }
      }
    },
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation<void, ApiError, void>({
    mutationFn: async () => {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
      })
    },
    onSettled: () => {
      clearNonPublicQueries(qc) // #436, FR-BB116: no profile (or locale) outlives the session
      clearE2eToken() // #415: no stored token outlives the session, in any build
      qc.setQueryData(['auth', 'accessToken'], null)
      qc.setQueryData(['auth', 'currentUser'], null)
      clearPasswordChangeRequired(qc)
    },
  })
}

export function useRefreshToken() {
  const qc = useQueryClient()
  return useQuery<string | null, Error>({
    queryKey: ['auth', 'accessToken'],
    queryFn: async () => {
      // E2E: if a token was seeded into localStorage by global-setup, use it directly.
      // The token stays in localStorage so subsequent page navigations within the same
      // test run can reuse it without hitting the auth rate limit on /auth/refresh.
      const seeded = readE2eToken()
      if (seeded) {
        const claims = decodeJwtPayload(seeded)
        // Use the cached token only when it has more than 65 seconds of lifetime left.
        // The 65-second buffer ensures that when refetchInterval fires 60 s before expiry,
        // queryFn falls through to /auth/refresh instead of serving the soon-to-expire token.
        const exp = claims?.exp as number | undefined
        if (exp && exp * 1000 > Date.now() + 65_000) {
          if (claims) {
            qc.setQueryData(['auth', 'currentUser'], {
              id: (claims.sub as string) ?? '',
              email: (claims.email as string) ?? '',
              role: (claims.role as string) ?? '',
              full_name: '',
              force_password_change: false,
            })
          }
          await checkForcePasswordChange(qc, seeded)
          return seeded
        }
        // Token expired — remove and fall through to refresh.
        clearE2eToken()
      }

      try {
        const res = await fetch('/api/v1/auth/refresh', {
          method: 'POST',
          credentials: 'include',
        })
        if (!res.ok) {
          // If a login mutation already placed a valid token in the cache (e.g. fresh
          // login with no pre-existing refresh cookie), preserve it rather than
          // overwriting with null — which would immediately log the user out.
          const cached = qc.getQueryData<string | null>(['auth', 'accessToken'])
          if (cached) return cached
          return null
        }
        const json = await res.json()
        if (json.error || !json.data) return null
        const accessToken = json.data.access_token as string
        // Persist the new token so subsequent page navigations (fresh JS contexts)
        // can use it directly without triggering another refresh / token rotation.
        writeE2eToken(accessToken)
        // Reconstruct minimal user info from JWT claims (refresh response has no user object)
        const claims = decodeJwtPayload(accessToken)
        if (claims) {
          qc.setQueryData(['auth', 'currentUser'], {
            id: (claims.sub as string) ?? '',
            email: (claims.email as string) ?? '',
            role: (claims.role as string) ?? '',
            full_name: '',
            force_password_change: false,
          })
        }
        await checkForcePasswordChange(qc, accessToken)
        return accessToken
      } catch {
        return null
      }
    },
    staleTime: Infinity,
    retry: false,
    // Proactively refresh the access token 60 seconds before it expires so that
    // mid-session API calls never hit a TOKEN_EXPIRED 401.  The wider localStorage
    // buffer (65 s) above guarantees that queryFn actually calls /auth/refresh when
    // this interval fires, rather than returning the still-cached stale token.
    refetchInterval: (query) => {
      const token = query.state.data
      if (!token) return false
      const claims = decodeJwtPayload(token)
      if (!claims || typeof claims.exp !== 'number') return false
      const exp = (claims.exp as number) * 1000   // seconds → ms
      const refreshAt = exp - 60_000              // 60 s before expiry
      const now = Date.now()
      return refreshAt > now ? refreshAt - now : 0
    },
    refetchIntervalInBackground: true,
  })
}
