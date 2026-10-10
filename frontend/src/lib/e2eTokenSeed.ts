/**
 * E2E-only access-token seed (#415). Production builds never read or write the access token in
 * localStorage (FR-BB110 AC-3: the token lives in the React Query cache only, and a reload goes
 * through POST /auth/refresh with the httpOnly cookie).
 *
 * The local E2E stack builds with VITE_E2E_TOKEN_SEED=true (mode "e2e", see .env.e2e). Specs
 * share one admin session and refresh tokens rotate, so the seed keeps working there. Every
 * other build has the seed off: nothing is read or written.
 *
 * clearE2eToken() removes any stored token in every build, so a token written by an older build
 * is never left behind after logout or a revoked session.
 */

export const E2E_TOKEN_KEY = '__e2e_access_token__'

export const E2E_TOKEN_SEED_ENABLED = import.meta.env.VITE_E2E_TOKEN_SEED === 'true'

/** The seeded token, or null when the seed is off or nothing is stored. */
export function readE2eToken(): string | null {
  if (!E2E_TOKEN_SEED_ENABLED) return null
  try {
    return localStorage.getItem(E2E_TOKEN_KEY)
  } catch {
    return null
  }
}

/** Store the token for the E2E seed. A no-op in every build without the seed. */
export function writeE2eToken(token: string): void {
  if (!E2E_TOKEN_SEED_ENABLED) return
  try {
    localStorage.setItem(E2E_TOKEN_KEY, token)
  } catch {
    /* storage unavailable: the seed is best-effort */
  }
}

/**
 * Clear a token left by an older build, once at startup, when the seed is off (#415).
 * With the seed on, the e2e build keeps its token for the spec run.
 */
export function clearStaleE2eToken(): void {
  if (!E2E_TOKEN_SEED_ENABLED) clearE2eToken()
}

/** Remove any stored token, in every build (logout and revoked sessions). */
export function clearE2eToken(): void {
  try {
    localStorage.removeItem(E2E_TOKEN_KEY)
  } catch {
    /* storage unavailable */
  }
}
