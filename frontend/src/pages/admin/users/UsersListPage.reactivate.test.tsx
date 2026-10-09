import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { UsersListPage } from './UsersListPage'

// FR-BB18 AC-13: a Reactivate row action is offered for deactivated users only.
const users = [
  {
    id: 'u-active',
    email: 'active@example.com',
    full_name: 'Active Person',
    department_id: 'dept-child',
    department_name: 'Engineering',
    role_id: 'role-uuid-emp',
    role_name: 'employee',
    status: 'active',
    is_locked: false,
    force_password_change: false,
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'u-inactive',
    email: 'inactive@example.com',
    full_name: 'Inactive Person',
    department_id: 'dept-child',
    department_name: 'Engineering',
    role_id: 'role-uuid-emp',
    role_name: 'employee',
    status: 'inactive',
    is_locked: false,
    force_password_change: false,
    created_at: '2026-01-01T00:00:00Z',
  },
]

const tree = [
  {
    id: 'dept-root',
    name: 'Headquarters',
    parent_id: null,
    children: [{ id: 'dept-child', name: 'Engineering', parent_id: 'dept-root', children: [] }],
  },
]

const server = setupServer(
  http.get('/api/v1/departments', () => HttpResponse.json({ data: tree, error: null })),
  http.get('/api/v1/users/roles', () =>
    HttpResponse.json({ data: [{ id: 'role-uuid-emp', name: 'employee' }], error: null }),
  ),
  http.get('/api/v1/users', () =>
    HttpResponse.json({
      data: { items: users, meta: { page: 1, per_page: 20, total: users.length } },
      error: null,
    }),
  ),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  queryClient.setQueryData(
    ['auth', 'accessToken'],
    `h.${btoa(JSON.stringify({ role: 'super_admin', sub: 'me-1' }))}.s`,
  )
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
  return render(<UsersListPage />, { wrapper: Wrapper })
}

function rowOf(name: string): HTMLElement {
  const row = screen.getByText(name).closest('tr')
  if (!row) throw new Error(`no row for ${name}`)
  return row
}

describe('UsersListPage reactivate action', () => {
  it('offers Reactivate on a deactivated user and not on an active one', async () => {
    renderPage()

    const inactiveRow = await screen.findByText('Inactive Person').then(() => rowOf('Inactive Person'))
    expect(within(inactiveRow).getByRole('button', { name: 'Reactivate' })).toBeInTheDocument()

    const activeRow = rowOf('Active Person')
    expect(within(activeRow).queryByRole('button', { name: 'Reactivate' })).not.toBeInTheDocument()
    expect(within(activeRow).getByRole('button', { name: 'Deactivate' })).toBeInTheDocument()
  })
})
