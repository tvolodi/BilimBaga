import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { UserCreateDrawer } from './UserCreateDrawer'

const sampleDepartments = [
  {
    id: 'dept-1',
    name: 'Engineering',
    parent_id: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    children: [],
  },
  {
    id: 'dept-2',
    name: 'HR',
    parent_id: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    children: [],
  },
]

const sampleRoles = [
  { id: 'role-emp', name: 'employee' },
  { id: 'role-mgr', name: 'manager' },
  { id: 'role-hr', name: 'hr' },
  { id: 'role-adm', name: 'admin' },
]

const server = setupServer(
  http.get('/api/v1/departments', () =>
    HttpResponse.json({ data: sampleDepartments, error: null }),
  ),
  http.get('/api/v1/users/roles', () =>
    HttpResponse.json({ data: sampleRoles, error: null }),
  ),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  // super_admin may assign every role (FR-BB117 D-1), keeping the role-list assertions meaningful.
  queryClient.setQueryData(['auth', 'accessToken'], `h.${btoa(JSON.stringify({ role: 'super_admin' }))}.s`)
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

describe('UserCreateDrawer — ISS-018 regression', () => {
  it('renders department options from GET /api/v1/departments', async () => {
    const user = userEvent.setup()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <UserCreateDrawer open onClose={() => {}} />
      </Wrapper>,
    )
    // Wait for the department selector to show the placeholder (loading resolved)
    const deptTrigger = await screen.findByText('Select department')
    await user.click(deptTrigger)
    await waitFor(() => {
      expect(screen.getByRole('treeitem', { name: 'Engineering' })).toBeInTheDocument()
      expect(screen.getByRole('treeitem', { name: 'HR' })).toBeInTheDocument()
    })
  })

  it('renders role options from GET /api/v1/users/roles', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <UserCreateDrawer open onClose={() => {}} />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'employee' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'manager' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'admin' })).toBeInTheDocument()
    })
  })

  it('shows placeholder when department API returns empty list', async () => {
    server.use(
      http.get('/api/v1/departments', () =>
        HttpResponse.json({ data: [], error: null }),
      ),
      http.get('/api/v1/users/roles', () =>
        HttpResponse.json({ data: [], error: null }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <UserCreateDrawer open onClose={() => {}} />
      </Wrapper>,
    )
    // Role select placeholder should still render
    await waitFor(() => {
      const roleOptions = screen.getAllByRole('option')
      expect(roleOptions.length).toBeGreaterThanOrEqual(1)
    })
  })
})
