import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import i18n from '@/i18n'
import { LocaleSwitcher } from './LocaleSwitcher'

const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
  if (init?.method === 'PATCH') {
    return new Response(JSON.stringify({ data: null, error: { code: 'INTERNAL_ERROR', message: 'boom' } }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    })
  }
  return new Response(JSON.stringify({ data: null, error: null }), { status: 200 })
})

function renderSwitcher(token: string | null) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  if (token) qc.setQueryData(['auth', 'accessToken'], token)
  render(
    <QueryClientProvider client={qc}>
      <LocaleSwitcher />
    </QueryClientProvider>,
  )
}

function patchCalls() {
  return fetchMock.mock.calls.filter(([, init]) => init?.method === 'PATCH')
}

beforeEach(async () => {
  localStorage.clear()
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  fetchMock.mockClear()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// FR-BB116 AC-7: the header switcher persists the choice for signed-in users only, best-effort.
describe('LocaleSwitcher persistence (FR-BB116 AC-7)', () => {
  it('switches without any request when nobody is signed in', async () => {
    const user = userEvent.setup()
    renderSwitcher(null)

    await user.selectOptions(screen.getByRole('combobox'), 'kk')

    await waitFor(() => expect(i18n.language).toBe('kk'))
    expect(localStorage.getItem('i18n-lang')).toBe('kk')
    expect(patchCalls()).toHaveLength(0)
  })

  it('persists the choice through PATCH /users/me when signed in', async () => {
    fetchMock.mockImplementationOnce(async () => new Response(JSON.stringify({ data: {}, error: null }), { status: 200 }))
    const user = userEvent.setup()
    renderSwitcher('tok-1')

    await user.selectOptions(screen.getByRole('combobox'), 'ru')

    await waitFor(() => expect(patchCalls()).toHaveLength(1))
    const [url, init] = patchCalls()[0]
    expect(url).toBe('/api/v1/users/me')
    expect(JSON.parse(init?.body as string)).toEqual({ preferred_locale: 'ru' })
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer tok-1')
    expect(i18n.language).toBe('ru')
  })

  it('keeps the UI switch when the persistence request fails', async () => {
    const user = userEvent.setup()
    renderSwitcher('tok-1')

    await user.selectOptions(screen.getByRole('combobox'), 'kk')

    await waitFor(() => expect(patchCalls()).toHaveLength(1))
    expect(i18n.language).toBe('kk')
    expect(localStorage.getItem('i18n-lang')).toBe('kk')
    expect(document.documentElement.lang).toBe('kk')
  })
})
