import { describe, it, expect } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { RequireRole } from './RequireRole'
import { endRevokedSession } from '@/lib/sessionRevoked'

// A minimal valid JWT with role=super_admin and exp far in the future.
// Header: {"alg":"HS256","typ":"JWT"}
// Payload: {"sub":"uid-1","role":"super_admin","exp":9999999999}
// (Signature is fake — RequireRole only decodes, never verifies)
const SUPER_ADMIN_TOKEN =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9' +
  '.eyJzdWIiOiJ1aWQtMSIsInJvbGUiOiJzdXBlcl9hZG1pbiIsImV4cCI6OTk5OTk5OTk5OX0' +
  '.fake-signature'

const EMPLOYEE_TOKEN =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9' +
  '.eyJzdWIiOiJ1aWQtMiIsInJvbGUiOiJlbXBsb3llZSIsImV4cCI6OTk5OTk5OTk5OX0' +
  '.fake-signature'

function renderWithToken(
  token: string | null,
  roles: string[],
  seedCurrentUser = false,
) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  qc.setQueryData(['auth', 'accessToken'], token)

  // Simulate the GC scenario: currentUser is deliberately NOT set.
  // In production this happens after ~5 min when TanStack Query GCs the entry.
  if (seedCurrentUser) {
    qc.setQueryData(['auth', 'currentUser'], { role: 'super_admin' })
  }

  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route
            path="/protected"
            element={
              <RequireRole roles={roles}>
                <div>Protected Content</div>
              </RequireRole>
            }
          />
          <Route path="/login" element={<div>Login Page</div>} />
          <Route path="/admin" element={<div>Admin Home</div>} />
          <Route path="/portal" element={<div>Portal Home</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('RequireRole', () => {
  it('renders children when token has the required role', () => {
    renderWithToken(SUPER_ADMIN_TOKEN, ['super_admin', 'hr_admin'])
    expect(screen.getByText('Protected Content')).toBeInTheDocument()
  })

  it('renders children with the required role even when currentUser cache entry is absent (ISS-019)', () => {
    // This is the regression test for the GC bug:
    // token is valid, but ['auth', 'currentUser'] was never seeded (simulates GC).
    renderWithToken(SUPER_ADMIN_TOKEN, ['super_admin'], false)
    expect(screen.getByText('Protected Content')).toBeInTheDocument()
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
  })

  it('redirects to /login when token is null', () => {
    renderWithToken(null, ['super_admin'])
    expect(screen.getByText('Login Page')).toBeInTheDocument()
  })

  it('redirects employee to /portal when token role does not match (FR-BB47 AC-1: prevents blank-page loop)', () => {
    renderWithToken(EMPLOYEE_TOKEN, ['super_admin', 'hr_admin'])
    expect(screen.getByText('Portal Home')).toBeInTheDocument()
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
    expect(screen.queryByText('Admin Home')).not.toBeInTheDocument()
  })

  it('redirects to /admin when token is malformed (prevents login-loop on access denied)', () => {
    renderWithToken('not.a.jwt', ['super_admin'])
    expect(screen.getByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
  })

  it('redirects to /login when the token is cleared after mount (ISS-249)', async () => {
    // The guard subscribes to the token, so endRevokedSession on TOKEN_REVOKED re-renders it.
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], SUPER_ADMIN_TOKEN)
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/protected']}>
          <Routes>
            <Route
              path="/protected"
              element={
                <RequireRole roles={['super_admin']}>
                  <div>Protected Content</div>
                </RequireRole>
              }
            />
            <Route path="/login" element={<div>Login Page</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
    expect(screen.getByText('Protected Content')).toBeInTheDocument()

    act(() => endRevokedSession(qc))

    expect(await screen.findByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
  })
})
