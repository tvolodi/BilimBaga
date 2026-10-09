import { describe, it, expect } from 'vitest'
import { adminPasswordCandidates, loginClearingForceChange } from './e2e-auth'

type Call = { url: string; body: Record<string, string>; auth?: string }

function fakeApi(opts: { good: string; force: boolean; changeOk?: boolean }) {
  const calls: Call[] = []
  const impl = (async (url: string, init: RequestInit) => {
    const body = JSON.parse(String(init.body)) as Record<string, string>
    const auth = (init.headers as Record<string, string>).Authorization
    calls.push({ url, body, auth })
    if (url.endsWith('/auth/login')) {
      if (body.password !== opts.good) return new Response('{}', { status: 401 })
      return new Response(
        JSON.stringify({ data: { access_token: 'tok', user: { force_password_change: opts.force } }, error: null }),
        { status: 200 },
      )
    }
    return new Response('{}', { status: opts.changeOk === false ? 400 : 200 })
  }) as unknown as typeof fetch
  return { impl, calls }
}

describe('loginClearingForceChange (ISS-160)', () => {
  it('returns the token untouched when no change is forced', async () => {
    const { impl, calls } = fakeApi({ good: 'B', force: false })
    const r = await loginClearingForceChange('http://x', 'a@b', ['A', 'B'], 'New1234!', impl)
    expect(r).toEqual({ token: 'tok', password: 'B', changed: false })
    expect(calls.some((c) => c.url.endsWith('/change-password'))).toBe(false)
  })

  it('changes the password with the same token when the login is flagged', async () => {
    const { impl, calls } = fakeApi({ good: 'A', force: true })
    const r = await loginClearingForceChange('http://x', 'a@b', ['A'], 'New1234!', impl)
    expect(r).toEqual({ token: 'tok', password: 'New1234!', changed: true })
    const change = calls.find((c) => c.url.endsWith('/change-password'))
    expect(change?.auth).toBe('Bearer tok')
    expect(change?.body).toEqual({ current_password: 'A', new_password: 'New1234!' })
  })

  it('returns null when nothing logs in or the forced change fails', async () => {
    expect(await loginClearingForceChange('http://x', 'a@b', ['Z'], 'N', fakeApi({ good: 'A', force: false }).impl)).toBeNull()
    expect(await loginClearingForceChange('http://x', 'a@b', ['A'], 'N', fakeApi({ good: 'A', force: true, changeOk: false }).impl)).toBeNull()
  })

  it('orders admin candidates and de-duplicates', () => {
    expect(adminPasswordCandidates({})).toEqual(['E2eAdmin2024!', 'Admin2024!', 'Admin1234!'])
    expect(adminPasswordCandidates({ E2E_ADMIN_PASS: 'P1' })).toHaveLength(3)
    expect(adminPasswordCandidates({ E2E_ADMIN_PASS: 'Admin1234!', E2E_ADMIN_NEW_PASS: 'X' })).toEqual([
      'Admin1234!', 'X', 'Admin2024!',
    ])
  })
})
