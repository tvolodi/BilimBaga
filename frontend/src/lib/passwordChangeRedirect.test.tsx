import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import { MyResultsPage } from '@/pages/portal/MyResultsPage'
import { PasswordChangeGuard } from '@/components/PasswordChangeGuard'
import { useRefreshToken } from '@/api/auth'
import { createAppQueryClient, PASSWORD_CHANGE_FLAG_KEY } from './passwordChangeRequired'

const blocked = () =>
  HttpResponse.json(
    { data: null, error: { code: 'PASSWORD_CHANGE_REQUIRED', message: 'change it' } },
    { status: 403 },
  )

let resultsCalls = 0
const server = setupServer(
  http.get('/api/v1/portal/results', () => {
    resultsCalls++
    return blocked()
  }),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => {
  server.resetHandlers()
  resultsCalls = 0
})
afterAll(() => server.close())

describe('PASSWORD_CHANGE_REQUIRED redirect (ISS-160, post-UAT)', () => {
  it('is not retried by the app QueryClient (queries and mutations)', () => {
    const d = createAppQueryClient().getDefaultOptions()
    const qRetry = d.queries!.retry as (n: number, e: unknown) => boolean
    const mRetry = d.mutations!.retry as (n: number, e: unknown) => boolean
    const pcr = Object.assign(new Error('x'), { code: 'PASSWORD_CHANGE_REQUIRED' })
    const other = new Error('boom')
    expect(qRetry(0, pcr)).toBe(false)
    expect(mRetry(0, pcr)).toBe(false)
    expect(qRetry(0, other)).toBe(true) // default behaviour kept: 3 retries
    expect(qRetry(3, other)).toBe(false)
    expect(mRetry(0, other)).toBe(false) // mutations never retry by default
  })

  it('/portal/results: first 403 redirects immediately, one request, no retry delay', async () => {
    const qc = createAppQueryClient()
    qc.setQueryData(['auth', 'accessToken'], 'tok')
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/portal/results']}>
          <PasswordChangeGuard />
          <Routes>
            <Route path="/portal/results" element={<MyResultsPage />} />
            <Route path="/change-password" element={<div>change password page</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
    // waitFor's default timeout is 1 s: React Query's default retries (1 s, 2 s, 4 s) would miss it.
    await waitFor(() => expect(screen.getByText('change password page')).toBeInTheDocument())
    expect(resultsCalls).toBe(1)
  })

  it('hard reload: bootstrap checks /users/me and flags before any data query fails', async () => {
    server.use(
      http.post('/api/v1/auth/refresh', () => {
        // unsigned JWT with exp far in the future
        const payload = btoa(JSON.stringify({ sub: 'u1', role: 'employee', exp: 4102444800 }))
        return HttpResponse.json({ data: { access_token: `h.${payload}.s` }, error: null })
      }),
      http.get('/api/v1/users/me', () =>
        HttpResponse.json({ data: { id: 'u1', force_password_change: true }, error: null }),
      ),
    )
    const qc = createAppQueryClient()
    function Boot() {
      useRefreshToken()
      return null
    }
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/portal']}>
          <Boot />
          <PasswordChangeGuard />
          <Routes>
            <Route path="/portal" element={<div>portal page</div>} />
            <Route path="/change-password" element={<div>change password page</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
    await waitFor(() => expect(screen.getByText('change password page')).toBeInTheDocument())
    expect(qc.getQueryData(PASSWORD_CHANGE_FLAG_KEY)).toBe(true)
    expect(resultsCalls).toBe(0)
  })
})
