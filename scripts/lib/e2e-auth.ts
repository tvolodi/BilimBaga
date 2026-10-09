/**
 * Shared e2e/seed admin login that survives backend force_password_change enforcement (ISS-160).
 *
 * While users.force_password_change is true the API answers 403 PASSWORD_CHANGE_REQUIRED on every
 * route except POST /auth/change-password and GET /users/me, so a freshly created user, or the
 * seeded admin still on the default password, must change the password before its token is usable.
 */

export interface ClearedLogin {
  token: string
  /** The password that is valid now (the new one when a forced change was performed). */
  password: string
  /** True when a forced password change was performed during this login. */
  changed: boolean
}

interface LoginBody {
  data?: { access_token?: string; user?: { force_password_change?: boolean } } | null
  error?: { code?: string } | null
}

interface ChangeBody {
  data?: { access_token?: string } | null
}

/**
 * Try each candidate password in order. On the first successful login, if the response says
 * force_password_change, change the password to `newPassword`. ISS-171: change-password revokes every
 * earlier access token, so the returned token is the one from the change-password response (the login
 * token is dead after the change).
 * Returns null when no candidate logs in or when the forced change itself fails.
 */
export async function loginClearingForceChange(
  base: string,
  email: string,
  candidates: readonly string[],
  newPassword: string,
  fetchImpl: typeof fetch = fetch,
): Promise<ClearedLogin | null> {
  for (const candidate of candidates) {
    const res = await fetchImpl(`${base}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password: candidate }),
    })
    if (!res.ok) continue
    const body = (await res.json().catch(() => null)) as LoginBody | null
    const token = body?.data?.access_token
    if (!token) continue
    if (!body?.data?.user?.force_password_change) return { token, password: candidate, changed: false }

    const changed = await fetchImpl(`${base}/api/v1/auth/change-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ current_password: candidate, new_password: newPassword }),
    })
    if (!changed.ok) return null
    const next = ((await changed.json().catch(() => null)) as ChangeBody | null)?.data?.access_token
    if (!next) return null
    return { token: next, password: newPassword, changed: true }
  }
  return null
}

/** Admin password candidates in priority order: operator-provided, E2E-walkthrough, default. */
export function adminPasswordCandidates(env: Record<string, string | undefined>): string[] {
  const list = [env.E2E_ADMIN_PASS, env.E2E_ADMIN_NEW_PASS ?? 'E2eAdmin2024!', 'Admin2024!', 'Admin1234!']
  // At most 3 attempts per run: the account locks after 5 failures.
  return [...new Set(list.filter((p): p is string => typeof p === 'string' && p.length > 0))].slice(0, 3)
}
