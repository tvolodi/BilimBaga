import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// #415: production builds never read or write the access token in localStorage.
// The seed is on only when VITE_E2E_TOKEN_SEED=true (the e2e build mode).

const KEY = '__e2e_access_token__'

describe('e2e token seed, default build (#415)', () => {
  beforeEach(() => localStorage.clear())

  it('is off when the flag is unset', async () => {
    const seed = await import('./e2eTokenSeed')
    expect(seed.E2E_TOKEN_SEED_ENABLED).toBe(false)
  })

  it('never reads a stored token', async () => {
    const seed = await import('./e2eTokenSeed')
    localStorage.setItem(KEY, 'stored-token')
    expect(seed.readE2eToken()).toBeNull()
  })

  it('never writes a token', async () => {
    const seed = await import('./e2eTokenSeed')
    seed.writeE2eToken('new-token')
    expect(localStorage.getItem(KEY)).toBeNull()
  })

  it('always removes a stored token (logout, revoked session)', async () => {
    const seed = await import('./e2eTokenSeed')
    localStorage.setItem(KEY, 'old-token')
    seed.clearE2eToken()
    expect(localStorage.getItem(KEY)).toBeNull()
  })
})

describe('e2e token seed, e2e build (#415)', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.resetModules()
  })
  afterEach(() => vi.unstubAllEnvs())

  it('reads and writes the token when the flag is true', async () => {
    vi.stubEnv('VITE_E2E_TOKEN_SEED', 'true')
    const seed = await import('./e2eTokenSeed')
    expect(seed.E2E_TOKEN_SEED_ENABLED).toBe(true)
    seed.writeE2eToken('seeded-token')
    expect(seed.readE2eToken()).toBe('seeded-token')
  })
})

describe('startup clears a stale token in the default build (#415)', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it('removes a token left by an older build', async () => {
    const seed = await import('./e2eTokenSeed')
    localStorage.setItem(KEY, 'stale-token')
    seed.clearStaleE2eToken()
    expect(localStorage.getItem(KEY)).toBeNull()
  })
})

describe('the E2E flag stays out of deploy files and workflows (#415)', () => {
  it('appears in no deploy file or workflow', async () => {
    const fs = await import('node:fs')
    const path = await import('node:path')
    const repo = path.resolve(process.cwd(), '..')
    const walk = (dir: string): string[] => {
      if (!fs.existsSync(dir)) return []
      return fs.readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
        const p = path.join(dir, e.name)
        return e.isDirectory() ? walk(p) : [p]
      })
    }
    const scanned = [...walk(path.join(repo, 'deploy')), ...walk(path.join(repo, '.github', 'workflows'))]
    expect(scanned.length).toBeGreaterThan(0)
    const hits = scanned.filter((f) => fs.readFileSync(f, 'utf8').includes('VITE_E2E_TOKEN_SEED'))
    expect(hits).toEqual([])
  })
})
