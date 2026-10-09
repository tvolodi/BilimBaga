import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query'

/**
 * ISS-160: the backend answers 403 PASSWORD_CHANGE_REQUIRED on every authenticated route
 * except change-password / users/me while users.force_password_change is set. The SPA turns
 * that into a client-side flag that PasswordChangeGuard (App.tsx) redirects on.
 */
export const PASSWORD_CHANGE_REQUIRED = 'PASSWORD_CHANGE_REQUIRED'

/** Query key holding the "must change password" flag (set by a 403, cleared on change/login/logout). */
export const PASSWORD_CHANGE_FLAG_KEY = ['auth', 'passwordChangeRequired'] as const

export function isPasswordChangeRequired(err: unknown): boolean {
  return (err as { code?: unknown } | null)?.code === PASSWORD_CHANGE_REQUIRED
}

/** Raise the flag; the guard navigates to /change-password. Idempotent. */
export function markPasswordChangeRequired(qc: QueryClient): void {
  qc.setQueryData(PASSWORD_CHANGE_FLAG_KEY, true)
}

/** Lower the flag (after a successful change, login or logout). */
export function clearPasswordChangeRequired(qc: QueryClient): void {
  qc.setQueryData(PASSWORD_CHANGE_FLAG_KEY, false)
}

/** React Query's default: queries retry 3 times, mutations never. A forced password change is permanent, never retried. */
const retryExceptPasswordChange = (max: number) => (failureCount: number, err: unknown) =>
  !isPasswordChangeRequired(err) && failureCount < max

/**
 * Application QueryClient: any query or mutation failing with PASSWORD_CHANGE_REQUIRED raises
 * the flag, so every API module is covered without per-module handling.
 */
export function createAppQueryClient(): QueryClient {
  const qc: QueryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: retryExceptPasswordChange(3) },
      mutations: { retry: retryExceptPasswordChange(0) },
    },
    queryCache: new QueryCache({
      onError: (err) => {
        if (isPasswordChangeRequired(err)) markPasswordChangeRequired(qc)
      },
    }),
    mutationCache: new MutationCache({
      onError: (err) => {
        if (isPasswordChangeRequired(err)) markPasswordChangeRequired(qc)
      },
    }),
  })
  return qc
}
