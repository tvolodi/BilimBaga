import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
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
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

describe('UserCreateDrawer — ISS-018 regression', () => {
  it('renders department options from GET /api/v1/departments', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <UserCreateDrawer open onClose={() => {}} />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'Engineering' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'HR' })).toBeInTheDocument()
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

  it('shows placeholder options when API returns empty lists', async () => {
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
    // Placeholder options should still render (the empty <option value="">)
    await waitFor(() => {
      const deptOptions = screen.getAllByRole('option')
      // At least the placeholder options exist (one per select)
      expect(deptOptions.length).toBeGreaterThanOrEqual(2)
    })
  })
})
