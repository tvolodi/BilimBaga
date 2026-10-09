import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import '@/i18n'
import { AdminLayout } from './AdminLayout'

const adminUser = {
  id: 'u-1',
  email: 'admin@example.com',
  full_name: 'Admin',
  department_id: null,
  department_name: null,
  role_id: 'role-sa',
  role_name: 'super_admin',
  status: 'active',
  force_password_change: false,
  created_at: '2024-01-01T00:00:00Z',
}

const employeeUser = { ...adminUser, role_name: 'employee', role_id: 'role-emp' }

// Mock useMe to avoid network requests in tests
vi.mock('@/api/users', () => ({
  useMe: vi.fn(),
  // TopBar -> LocaleSwitcher persists the choice through this mutation (FR-BB116 AC-7).
  useUpdateMyLocale: () => ({ mutate: vi.fn(), isPending: false }),
}))

// Mock Breadcrumb to keep the test isolated from router-specific hooks
vi.mock('@/components/admin/Breadcrumb', () => ({
  Breadcrumb: () => null,
}))

import { useMe } from '@/api/users'

function renderLayout(initialPath: string, extraRoutes: { path: string; element: React.ReactNode }[] = []) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/admin" element={<AdminLayout />} />
          {extraRoutes.map(({ path, element }) => (
            <Route key={path} path={path} element={element} />
          ))}
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AdminLayout', () => {
  beforeEach(() => {
    vi.mocked(useMe).mockReturnValue({
      data: adminUser,
      isLoading: false,
    } as ReturnType<typeof useMe>)
  })

  it('renders the sidebar and topbar for admin users', async () => {
    renderLayout('/admin')
    await waitFor(() => {
      expect(screen.getAllByRole('navigation').length).toBeGreaterThan(0)
    })
  })

  it('redirects employee users to /portal', async () => {
    vi.mocked(useMe).mockReturnValue({
      data: employeeUser,
      isLoading: false,
    } as ReturnType<typeof useMe>)

    renderLayout('/admin', [{ path: '/portal', element: <div>Employee Portal</div> }])
    await waitFor(() => {
      expect(screen.getByText('Employee Portal')).toBeInTheDocument()
    })
  })

  it('redirects unauthenticated users to /login', async () => {
    vi.mocked(useMe).mockReturnValue({
      data: undefined,
      isLoading: false,
    } as ReturnType<typeof useMe>)

    renderLayout('/admin', [{ path: '/login', element: <div>Login Page</div> }])
    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument()
    })
  })
})
