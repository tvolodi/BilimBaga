import { describe, expect, it } from 'vitest'

// #249 attempt 4: every API call goes through the shared apiFetch, so a 401 TOKEN_REVOKED on any page
// refreshes the token or ends the session. A module that defines its own apiFetch, or calls fetch
// directly, skips that. The allow-list below is the only place a raw fetch may stay, with the reason.

/** Files that may call fetch directly. Any other module in src/api fails the guard. */
const RAW_FETCH_ALLOWED: Record<string, string> = {
  'apiFetch.ts': 'the shared authenticated fetch itself',
  'auth.ts': 'sign-in, refresh, logout, change-password and the boot user check (auth flow)',
  'download.ts': 'blob downloads (not JSON); uses the shared refresh and session-end functions',
  'tenant.ts': 'public tenant configuration (no token)',
  'useTenantConfig.ts': 'public tenant configuration (no token)',
  'verify.ts': 'public certificate verification (no token)',
  'recovery.ts': 'public account recovery (no token)',
}

/** True when module text calls fetch directly (a bare call, not apiFetch, refetch or another object's fetch). */
function rawFetchIn(text: string): boolean {
  return /(?<![\w$.])fetch\(/.test(text)
}

async function apiModules(): Promise<Array<{ name: string; text: string }>> {
  const fs = await import('node:fs')
  const path = await import('node:path')
  const dir = path.resolve(process.cwd(), 'src', 'api')
  return fs
    .readdirSync(dir)
    .filter((f) => f.endsWith('.ts') && !f.endsWith('.test.ts'))
    .map((name) => ({ name, text: fs.readFileSync(path.join(dir, name), 'utf8') }))
}

describe('API modules use the shared apiFetch (#249)', () => {
  it('no module other than apiFetch.ts defines its own apiFetch', async () => {
    const offenders = (await apiModules())
      .filter((m) => m.name !== 'apiFetch.ts' && /\b(function|const|let)\s+apiFetch\b/.test(m.text))
      .map((m) => m.name)
    expect(offenders).toEqual([])
  })

  it('no module calls fetch directly outside the allow-list', async () => {
    const offenders = (await apiModules())
      .filter((m) => !(m.name in RAW_FETCH_ALLOWED) && rawFetchIn(m.text))
      .map((m) => m.name)
    expect(offenders).toEqual([])
  })

  // #448: fetch reached through window, globalThis or self is a raw fetch too.
  it('flags window.fetch and globalThis.fetch in an API module (#448)', () => {
    expect(rawFetchIn("const r = await window.fetch('/api/v1/exams')")).toBe(true)
    expect(rawFetchIn("const r = await globalThis.fetch('/api/v1/exams')")).toBe(true)
    expect(rawFetchIn('const send = globalThis.fetch')).toBe(true)
    expect(rawFetchIn("self.fetch('/api/v1/exams')")).toBe(true)
  })

  it('does not flag the shared apiFetch, a method named fetch on another object, or refetch (#448)', () => {
    expect(rawFetchIn('return apiFetch<T>(qc, url)')).toBe(false)
    expect(rawFetchIn('await client.fetch(url)')).toBe(false)
    expect(rawFetchIn('void refetch()')).toBe(false)
  })

  it('every allow-listed file exists, so the list cannot go stale', async () => {
    const names = (await apiModules()).map((m) => m.name)
    for (const file of Object.keys(RAW_FETCH_ALLOWED)) expect(names, file).toContain(file)
  })

  it('download.ts shares the refresh and session-end rules with apiFetch', async () => {
    const download = (await apiModules()).find((m) => m.name === 'download.ts')
    expect(download?.text).toContain('refreshAccessTokenOnce')
    expect(download?.text).toContain('endRevokedSession')
  })
})
