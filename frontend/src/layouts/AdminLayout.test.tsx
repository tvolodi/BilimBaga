import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

describe('AdminLayout sidebar below sm (FR-BB321 AC-6, #472)', () => {
  function stubWidth(sm: boolean) {
    window.matchMedia = vi.fn(
      (query: string) =>
        ({ matches: sm, media: query, addEventListener: vi.fn(), removeEventListener: vi.fn() }) as unknown as MediaQueryList,
    )
  }

  beforeEach(() => {
    vi.mocked(useMe).mockReturnValue({ data: adminUser, isLoading: false } as ReturnType<typeof useMe>)
    window.localStorage.removeItem('sidebar-collapsed')
  })

  afterEach(() => {
    delete (window as { matchMedia?: unknown }).matchMedia
    window.localStorage.removeItem('sidebar-collapsed')
  })

  it('starts as a rail below sm and opens over the page when the toggle is pressed', async () => {
    stubWidth(false)
    renderLayout('/admin')
    const aside = document.querySelector('aside') as HTMLElement

    expect(aside).toHaveClass('w-16')
    expect(aside).not.toHaveClass('fixed')

    await userEvent.click(screen.getByRole('button', { name: 'Expand sidebar' }))
    expect(aside).toHaveClass('fixed', 'w-56')
  })

  // jsdom has no layout, so the page position is read from the in-flow widths before the content: a Tailwind
  // width class counts as its rem size times 4 px (w-16 is 64 px), and a fixed element is out of the flow.
  function contentOffset(): number {
    const content = document.querySelector('main')!.parentElement!
    let offset = 0
    for (let el = content.previousElementSibling; el; el = el.previousElementSibling) {
      if (el.classList.contains('fixed')) continue
      const width = /(?:^|\s)w-(\d+)(?=\s|$)/.exec(el.getAttribute('class') ?? '')
      offset += width ? Number(width[1]) * 4 : 0
    }
    return offset
  }

  it('opening the overlay does not move the page content (#472)', async () => {
    stubWidth(false)
    renderLayout('/admin')
    const before = contentOffset()

    await userEvent.click(screen.getByRole('button', { name: 'Expand sidebar' }))

    expect(document.querySelector('aside')).toHaveClass('fixed', 'w-56')
    expect(before).toBe(64)
    expect(contentOffset()).toBe(before)
  })

  it('follows the stored preference at sm and up', () => {
    stubWidth(true)
    window.localStorage.setItem('sidebar-collapsed', 'true')
    renderLayout('/admin')
    const aside = document.querySelector('aside') as HTMLElement

    expect(aside).toHaveClass('w-16')
    expect(aside).not.toHaveClass('fixed')
  })
})

describe('AdminLayout sidebar overlay keyboard (FR-BB321 AC-6, #472)', () => {
  function stubWidth(sm: boolean) {
    window.matchMedia = vi.fn(
      (query: string) =>
        ({ matches: sm, media: query, addEventListener: vi.fn(), removeEventListener: vi.fn() }) as unknown as MediaQueryList,
    )
  }

  beforeEach(() => {
    vi.mocked(useMe).mockReturnValue({ data: adminUser, isLoading: false } as ReturnType<typeof useMe>)
    window.localStorage.removeItem('sidebar-collapsed')
  })

  afterEach(() => {
    delete (window as { matchMedia?: unknown }).matchMedia
    window.localStorage.removeItem('sidebar-collapsed')
  })

  it('Escape closes the open sidebar and returns focus to its toggle', async () => {
    stubWidth(false)
    renderLayout('/admin')
    await userEvent.click(screen.getByRole('button', { name: 'Expand sidebar' }))
    const aside = document.querySelector('aside') as HTMLElement
    expect(aside).toHaveClass('fixed')

    await userEvent.keyboard('{Escape}')

    expect(aside).not.toHaveClass('fixed')
    expect(screen.getByRole('button', { name: 'Expand sidebar' })).toHaveFocus()
  })

  it('closes the open sidebar when focus moves to the top bar', async () => {
    stubWidth(false)
    renderLayout('/admin')
    await userEvent.click(screen.getByRole('button', { name: 'Expand sidebar' }))
    const aside = document.querySelector('aside') as HTMLElement
    expect(aside).toHaveClass('fixed')

    await userEvent.click(screen.getByRole('combobox'))

    expect(aside).not.toHaveClass('fixed')
  })
})
