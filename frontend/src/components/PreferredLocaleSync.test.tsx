import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import i18n from '@/i18n'
import { PreferredLocaleSync } from './PreferredLocaleSync'

function meBody(preferred: string | null) {
  return {
    id: 'u1',
    email: 'ana@example.com',
    full_name: 'Ana Test',
    department_id: null,
    department_name: null,
    role_id: 'r1',
    role_name: 'employee',
    status: 'active',
    force_password_change: false,
    is_locked: false,
    preferred_locale: preferred,
    created_at: '2026-01-01T00:00:00Z',
  }
}

let preferred: string | null = null
const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
  if (String(input) === '/api/v1/users/me') {
    return new Response(JSON.stringify({ data: meBody(preferred), error: null }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }
  return new Response(JSON.stringify({ data: null, error: { code: 'NOT_FOUND', message: 'x' } }), { status: 404 })
})

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function renderSync(qc: QueryClient) {
  return render(
    <QueryClientProvider client={qc}>
      <PreferredLocaleSync />
    </QueryClientProvider>,
  )
}

beforeEach(async () => {
  localStorage.clear()
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  preferred = null
  fetchMock.mockClear()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// FR-BB116 AC-7: the persisted preference becomes the UI language once a session exists.
describe('PreferredLocaleSync (FR-BB116 AC-7)', () => {
  it('applies the persisted preference after login, once the session token appears', async () => {
    preferred = 'ru'
    const qc = newClient()
    renderSync(qc)

    // No session yet: nothing is fetched and the language is untouched.
    expect(fetchMock).not.toHaveBeenCalled()
    expect(i18n.language).toBe('en')

    // useLogin stores the access token; the sync then reads /users/me and switches.
    act(() => {
      qc.setQueryData(['auth', 'accessToken'], 'tok-1')
    })

    await waitFor(() => expect(i18n.language).toBe('ru'))
    expect(localStorage.getItem('i18n-lang')).toBe('ru')
    expect(document.documentElement.lang).toBe('ru')
  })

  it('applies the persisted preference on bootstrap when a session already exists', async () => {
    preferred = 'kk'
    const qc = newClient()
    qc.setQueryData(['auth', 'accessToken'], 'tok-1')
    renderSync(qc)

    await waitFor(() => expect(i18n.language).toBe('kk'))
    expect(localStorage.getItem('i18n-lang')).toBe('kk')
  })

  it('leaves the language alone when the preference is null', async () => {
    preferred = null
    localStorage.setItem('i18n-lang', 'kk')
    const qc = newClient()
    qc.setQueryData(['auth', 'accessToken'], 'tok-1')
    renderSync(qc)

    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    await act(async () => {})
    expect(i18n.language).toBe('en')
    expect(localStorage.getItem('i18n-lang')).toBe('kk')
  })

  it('ignores a stored code that is not a supported locale', async () => {
    preferred = 'de'
    const qc = newClient()
    qc.setQueryData(['auth', 'accessToken'], 'tok-1')
    renderSync(qc)

    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    await act(async () => {})
    expect(i18n.language).toBe('en')
  })
})
