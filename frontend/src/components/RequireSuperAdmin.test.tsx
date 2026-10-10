import { describe, it, expect } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { RequireSuperAdmin } from './RequireSuperAdmin'

/**
 * Build a JWT-shaped string with a base64url payload. The guard only decodes the payload
 * (no signature check), so the header and signature are fixed placeholders.
 */
function makeToken(claims: object | string): string {
  const encode = (value: object | string) =>
    btoa(typeof value === 'string' ? value : JSON.stringify(value))
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '')
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(claims)}.fake-signature`
}

/**
 * Seed the access-token query. `undefined` leaves the query entry absent (never settled),
 * `null` seeds an explicit "no session" value.
 */
function renderGuard(token: string | null | undefined) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  if (token !== undefined) {
    qc.setQueryData(['auth', 'accessToken'], token)
  }

  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/super']}>
        <Routes>
          <Route
            path="/super"
            element={
              <RequireSuperAdmin>
                <div>Super Admin Content</div>
              </RequireSuperAdmin>
            }
          />
          <Route path="/login" element={<div>Login Page</div>} />
          <Route path="/admin" element={<div>Admin Home</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('RequireSuperAdmin', () => {
  it('renders children when the token role is super_admin', () => {
    renderGuard(makeToken({ sub: 'uid-1', role: 'super_admin', exp: 9999999999 }))
    expect(screen.getByText('Super Admin Content')).toBeInTheDocument()
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
    expect(screen.queryByText('Admin Home')).not.toBeInTheDocument()
  })

  it('decodes a base64url payload that contains - and _ characters', () => {
    // The `note` value makes the standard base64 of the payload contain '+' or '/',
    // so the token only decodes correctly if the guard maps '-' and '_' back.
    const token = makeToken({ sub: 'uid-9', role: 'super_admin', note: '~~', exp: 9999999999 })
    expect(token.split('.')[1]).toMatch(/[-_]/)
    renderGuard(token)
    expect(screen.getByText('Super Admin Content')).toBeInTheDocument()
  })

  it('redirects an employee to the admin fallback route', () => {
    renderGuard(makeToken({ sub: 'uid-2', role: 'employee', exp: 9999999999 }))
    expect(screen.getByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
  })

  it('redirects a non-super-admin role such as hr_admin to the admin fallback route', () => {
    renderGuard(makeToken({ sub: 'uid-3', role: 'hr_admin', exp: 9999999999 }))
    expect(screen.getByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
  })

  it('redirects to /login when the access token query is null', () => {
    renderGuard(null)
    expect(screen.getByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
  })

  it('redirects to /login when the access token query has never been seeded', () => {
    // The guard has no spinner state: an unsettled query reads as "no token".
    renderGuard(undefined)
    expect(screen.getByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
  })

  it('treats a token without a dot-separated payload as not super_admin', () => {
    renderGuard('not-a-jwt')
    expect(screen.getByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
  })

  it('treats a token with an undecodable payload as not super_admin', () => {
    // '%%%' is not valid base64, so atob throws and the decoder falls back to undefined.
    renderGuard('eyJhbGciOiJIUzI1NiJ9.%%%.fake-signature')
    expect(screen.getByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
  })

  it('treats a token whose payload is JSON null as not super_admin', () => {
    renderGuard(makeToken('null'))
    expect(screen.getByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
  })

  it('treats a token whose payload has no role claim as not super_admin', () => {
    renderGuard(makeToken({ sub: 'uid-4', exp: 9999999999 }))
    expect(screen.getByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
  })

  it('redirects to /login when the token is cleared after mount (ISS-249)', async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], makeToken({ sub: 'uid-1', role: 'super_admin', exp: 9999999999 }))
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/super']}>
          <Routes>
            <Route
              path="/super"
              element={
                <RequireSuperAdmin>
                  <div>Super Admin Content</div>
                </RequireSuperAdmin>
              }
            />
            <Route path="/login" element={<div>Login Page</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
    expect(screen.getByText('Super Admin Content')).toBeInTheDocument()

    act(() => qc.setQueryData(['auth', 'accessToken'], null))

    expect(await screen.findByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Super Admin Content')).not.toBeInTheDocument()
  })
})
