import { describe, it, expect, vi, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import { RequireRole } from '@/components/RequireRole'
import { Sidebar } from '@/components/admin/Sidebar'
import { AdminHome } from '@/components/admin/AdminHome'
import { AUDIT_READ_ROLES, REPORTS_READ_ROLES, ROLES_MANAGE_ROLES, ADMIN_SHELL_ROLES, ROUTE_PERMISSIONS, can, isCustomRole } from './routeRoles'

vi.mock('@/api/useTenantConfig', () => ({
  useTenantConfig: () => ({ data: { app_name: 'BilimBaga' } }),
}))

function tokenFor(role: string): string {
  const b64 = (o: object) => btoa(JSON.stringify(o)).replace(/=/g, '')
  return `${b64({ alg: 'HS256', typ: 'JWT' })}.${b64({ sub: 'u', role, exp: 9999999999 })}.sig`
}

let meRequests = 0
let mePayload: { role_name: string; permissions?: string[] } | 'error' = { role_name: 'qa', permissions: [] }

const server = setupServer(
  http.get('/api/v1/users/me', () => {
    meRequests++
    if (mePayload === 'error') return HttpResponse.json({ data: null, error: { code: 'INTERNAL', message: 'x' } }, { status: 500 })
    return HttpResponse.json({ data: { id: 'u', ...mePayload }, error: null })
  }),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  meRequests = 0
})
afterAll(() => server.close())

function setMe(role: string, permissions: string[]) {
  mePayload = { role_name: role, permissions }
}

function newClient(role: string | null) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], role ? tokenFor(role) : null)
  return qc
}

function renderGuard(role: string, props: Partial<React.ComponentProps<typeof RequireRole>> & { roles: string[] }) {
  const qc = newClient(role)
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/p']}>
        <Routes>
          <Route path="/p" element={<RequireRole {...props}>{<div>Protected</div>}</RequireRole>} />
          <Route path="/admin" element={<div>Admin Home</div>} />
          <Route path="/portal" element={<div>Portal Home</div>} />
          <Route path="/login" element={<div>Login</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('isCustomRole / can', () => {
  it('only non-built-in role names are custom', () => {
    for (const r of ['super_admin', 'department_admin', 'examiner', 'employee', 'hr_admin']) {
      expect(isCustomRole(r)).toBe(false)
    }
    expect(isCustomRole('qa_reviewer')).toBe(true)
    expect(isCustomRole(undefined)).toBe(false)
  })
  it('can() denies when permissions are unknown', () => {
    expect(can(undefined, 'users:read')).toBe(false)
    expect(can([], 'users:read')).toBe(false)
    expect(can(['users:read'], 'users:read')).toBe(true)
  })
})

describe('RequireRole permission fallback (FR-BB117 AC-16)', () => {
  it('admits a custom role holding the permission (users:read)', async () => {
    setMe('qa_reviewer', ['users:read'])
    renderGuard('qa_reviewer', { roles: ADMIN_SHELL_ROLES, permission: ROUTE_PERMISSIONS.users })
    expect(await screen.findByText('Protected')).toBeInTheDocument()
  })

  it('redirects a custom role lacking the permission to /admin', async () => {
    setMe('qa_reviewer', ['exams:read'])
    renderGuard('qa_reviewer', { roles: ADMIN_SHELL_ROLES, permission: ROUTE_PERMISSIONS.users })
    expect(await screen.findByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Protected')).not.toBeInTheDocument()
  })

  it('never sends a custom role to /login even when unauthorizedRedirect is set', async () => {
    setMe('qa_reviewer', [])
    renderGuard('qa_reviewer', { roles: AUDIT_READ_ROLES, unauthorizedRedirect: '/login', permission: ROUTE_PERMISSIONS.audit })
    expect(await screen.findByText('Admin Home')).toBeInTheDocument()
  })

  it('denies (no over-grant) when /users/me fails', async () => {
    mePayload = 'error'
    renderGuard('qa_reviewer', { roles: ADMIN_SHELL_ROLES, permission: ROUTE_PERMISSIONS.users })
    expect(await screen.findByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Protected')).not.toBeInTheDocument()
  })

  it('ignores a /users/me payload that belongs to another role (stale cache)', async () => {
    setMe('super_admin', ['users:read'])
    renderGuard('qa_reviewer', { roles: ADMIN_SHELL_ROLES, permission: ROUTE_PERMISSIONS.users })
    expect(await screen.findByText('Admin Home')).toBeInTheDocument()
    expect(screen.queryByText('Protected')).not.toBeInTheDocument()
  })

  it('without a permission prop a custom role is denied and /users/me is not needed', async () => {
    setMe('qa_reviewer', ['users:read'])
    renderGuard('qa_reviewer', { roles: ROLES_MANAGE_ROLES })
    expect(await screen.findByText('Admin Home')).toBeInTheDocument()
  })

  it('allowCustomRole admits any custom role (admin shell) without fetching permissions', async () => {
    setMe('qa_reviewer', [])
    renderGuard('qa_reviewer', { roles: ADMIN_SHELL_ROLES, allowCustomRole: true })
    expect(await screen.findByText('Protected')).toBeInTheDocument()
  })

  it('allowCustomRole does not admit employees', () => {
    renderGuard('employee', { roles: ADMIN_SHELL_ROLES, allowCustomRole: true })
    expect(screen.getByText('Portal Home')).toBeInTheDocument()
  })

  it('built-in roles are unchanged and never fetch /users/me', () => {
    renderGuard('super_admin', { roles: ROLES_MANAGE_ROLES, permission: ROUTE_PERMISSIONS.roles })
    expect(screen.getByText('Protected')).toBeInTheDocument()
    renderGuard('examiner', { roles: ROLES_MANAGE_ROLES, permission: ROUTE_PERMISSIONS.roles })
    expect(screen.getAllByText('Admin Home').length).toBe(1)
    renderGuard('employee', { roles: ROLES_MANAGE_ROLES, permission: ROUTE_PERMISSIONS.roles })
    expect(screen.getByText('Portal Home')).toBeInTheDocument()
    expect(meRequests).toBe(0)
  })

  it('built-in role holding no match is not helped by permissions: department_admin on /admin/roles', () => {
    renderGuard('department_admin', { roles: ROLES_MANAGE_ROLES, permission: ROUTE_PERMISSIONS.roles })
    expect(screen.getByText('Admin Home')).toBeInTheDocument()
    expect(meRequests).toBe(0)
  })

  it('unauthenticated goes to /login', () => {
    const qc = newClient(null)
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/p']}>
          <Routes>
            <Route path="/p" element={<RequireRole roles={ADMIN_SHELL_ROLES} permission="users:read"><div>Protected</div></RequireRole>} />
            <Route path="/login" element={<div>Login</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
    expect(screen.getByText('Login')).toBeInTheDocument()
  })
})

describe('built-in role lists are unchanged', () => {
  it.each([
    ['audit', AUDIT_READ_ROLES, ['super_admin']],
    ['reports', REPORTS_READ_ROLES, ['super_admin', 'department_admin', 'examiner']],
    ['roles', ROLES_MANAGE_ROLES, ['super_admin']],
  ])('%s', (_n, list, expected) => {
    expect(list).toEqual(expected)
  })
})

function renderSidebar(role: string) {
  const qc = newClient(role)
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <Sidebar collapsed={false} onToggle={() => {}} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const hrefs = () => screen.queryAllByRole('link').map((a) => a.getAttribute('href'))

describe('Sidebar permission fallback and Roles link', () => {
  it('Roles link is shown to super_admin only, after Users', () => {
    renderSidebar('super_admin')
    const h = hrefs()
    expect(h).toContain('/admin/roles')
    expect(h.indexOf('/admin/roles')).toBe(h.indexOf('/admin/users') + 1)
  })

  it.each(['department_admin', 'examiner', 'employee'])('Roles link hidden for %s', (role) => {
    renderSidebar(role)
    expect(hrefs()).not.toContain('/admin/roles')
  })

  it('built-in department_admin keeps all links except audit/roles/(no change)', () => {
    renderSidebar('department_admin')
    const h = hrefs()
    expect(h).toEqual(expect.arrayContaining(['/admin', '/admin/users', '/admin/departments', '/admin/questions', '/admin/categories', '/admin/tags', '/admin/exams', '/admin/grading', '/admin/reports', '/admin/settings/branding']))
    expect(h).not.toContain('/admin/audit')
    expect(meRequests).toBe(0)
  })

  it('custom role with users:read sees Users and nothing it does not hold', async () => {
    setMe('qa_reviewer', ['users:read'])
    renderSidebar('qa_reviewer')
    expect(await screen.findByRole('link', { name: 'Users' })).toHaveAttribute('href', '/admin/users')
    expect(hrefs()).toEqual(['/admin/users'])
  })

  it('custom role with reports:read sees Dashboard and Reports', async () => {
    setMe('qa_reviewer', ['reports:read', 'exams:read'])
    renderSidebar('qa_reviewer')
    await screen.findByRole('link', { name: 'Reports' })
    expect(hrefs().sort()).toEqual(['/admin', '/admin/exams', '/admin/reports'].sort())
  })

  it('custom role never sees Roles, Audit or Settings unless it holds those permissions (and cannot hold roles/tenant ones)', async () => {
    setMe('qa_reviewer', ['users:read'])
    renderSidebar('qa_reviewer')
    await screen.findByRole('link', { name: 'Users' })
    expect(hrefs()).not.toContain('/admin/roles')
    expect(hrefs()).not.toContain('/admin/audit')
    expect(hrefs()).not.toContain('/admin/settings/branding')
  })

  it('shows no links for a custom role while permissions are unknown (error)', async () => {
    mePayload = 'error'
    renderSidebar('qa_reviewer')
    await new Promise((r) => setTimeout(r, 50))
    expect(hrefs()).toEqual([])
  })
})

describe('AdminHome landing', () => {
  function renderHome(role: string) {
    const qc = newClient(role)
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/admin']}>
          <Routes>
            <Route path="/admin" element={<AdminHome />} />
            <Route path="/admin/dashboard" element={<div>Dashboard Page</div>} />
            <Route path="/admin/users" element={<div>Users Page</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
  }

  it('built-in roles land on the dashboard', () => {
    renderHome('examiner')
    expect(screen.getByText('Dashboard Page')).toBeInTheDocument()
  })

  it('custom role lands on its first permitted page', async () => {
    setMe('qa_reviewer', ['users:read'])
    renderHome('qa_reviewer')
    expect(await screen.findByText('Users Page')).toBeInTheDocument()
  })

  it('custom role with no admin permissions sees a notice instead of a redirect loop', async () => {
    setMe('qa_reviewer', ['portal:read'])
    renderHome('qa_reviewer')
    expect(await screen.findByText('No pages available')).toBeInTheDocument()
  })
})
