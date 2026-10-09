import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { RequireRole } from '@/components/RequireRole'
import { AUDIT_READ_ROLES, REPORTS_READ_ROLES } from './routeRoles'

function tokenFor(role: string): string {
  const b64 = (o: object) => btoa(JSON.stringify(o)).replace(/=/g, '')
  return `${b64({ alg: 'HS256', typ: 'JWT' })}.${b64({ sub: 'u', role, exp: 9999999999 })}.sig`
}

function renderGuard(roles: string[], token: string | null) {
  const qc = new QueryClient()
  qc.setQueryData(['auth', 'accessToken'], token)
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/p']}>
        <Routes>
          <Route
            path="/p"
            element={
              <RequireRole roles={roles} unauthorizedRedirect="/login">
                <div>Protected</div>
              </RequireRole>
            }
          />
          <Route path="/login" element={<div>Login</div>} />
          <Route path="/admin" element={<div>Admin</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const ALL_ROLES = ['super_admin', 'department_admin', 'examiner', 'hr_admin', 'employee']

describe.each([
  ['/admin/audit (audit:read)', AUDIT_READ_ROLES, ['super_admin']],
  ['/admin/reports (reports:read)', REPORTS_READ_ROLES, ['super_admin', 'department_admin', 'examiner']],
])('%s guard', (_name, roles, allowed) => {
  it.each(ALL_ROLES)('role %s', (role) => {
    renderGuard(roles, tokenFor(role))
    if (allowed.includes(role)) {
      expect(screen.getByText('Protected')).toBeInTheDocument()
    } else {
      expect(screen.getByText('Login')).toBeInTheDocument()
      expect(screen.queryByText('Protected')).not.toBeInTheDocument()
    }
  })

  it('no token redirects to /login', () => {
    renderGuard(roles, null)
    expect(screen.getByText('Login')).toBeInTheDocument()
  })
})
