import type { QueryClient } from '@tanstack/react-query'

/**
 * Query families that may outlive a session: tenant branding (read before sign-in) and the public
 * certificate check. The 'auth' family is kept out of the sweep because the session functions set its
 * values (token, current user, the session-ended notice) explicitly. Every other family is per-user.
 */
const KEPT_FAMILIES: readonly unknown[] = ['tenant', 'verify', 'auth']

/**
 * Drop every cached query that is not public or auth state, so no data from the previous user survives
 * a session end, a logout or a login (#436). Runs before the new session's values are written.
 */
export function clearNonPublicQueries(qc: QueryClient): void {
  qc.removeQueries({ predicate: (query) => !KEPT_FAMILIES.includes(query.queryKey[0]) })
}
