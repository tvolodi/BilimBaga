import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import type { ReactNode } from 'react'

// #249 attempt 4: in a seeded e2e build, a fresh sign-in replaces the stored token, so a reload boots
// as the user who signed in, not as the seeded admin. The seed flag is read when the module loads,
// so each test imports the modules after stubbing the environment.

const KEY = '__e2e_access_token__'

const LOGIN_OK = {
  data: {
    access_token: 'tok-new',
    token_type: 'Bearer',
    expires_in: 900,
    user: { id: 'u1', full_name: 'User', email: 'u@example.com', role: 'employee', force_password_change: false },
  },
  error: null,
}

async function signIn() {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(LOGIN_OK), { status: 200 })))
  const { useLogin } = await import('./auth')
  const { QueryClient, QueryClientProvider } = await import('@tanstack/react-query')
  const { renderHook, act } = await import('@testing-library/react')
  const { createElement } = await import('react')
  const qc = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => createElement(QueryClientProvider, { client: qc }, children)
  const { result } = renderHook(() => useLogin(), { wrapper })
  await act(async () => {
    await result.current.mutateAsync({ email: 'u@example.com', password: 'pw' })
  })
}

describe('useLogin and the e2e token seed (#249)', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
  })

  afterEach(() => {
    vi.unstubAllEnvs()
    vi.unstubAllGlobals()
  })

  it('replaces an existing seed with the token from a fresh sign-in', async () => {
    vi.stubEnv('VITE_E2E_TOKEN_SEED', 'true')
    localStorage.setItem(KEY, 'tok-old')
    await signIn()
    expect(localStorage.getItem(KEY)).toBe('tok-new')
  })

  it('leaves storage untouched in a build without the seed', async () => {
    vi.stubEnv('VITE_E2E_TOKEN_SEED', 'false')
    await signIn()
    expect(localStorage.getItem(KEY)).toBeNull()
  })
})
