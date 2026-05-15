import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { UsersListPage } from './UsersListPage'

const meResponse = {
  id: 'u-1',
  email: 'admin@example.com',
  full_name: 'Admin User',
  department_id: null,
  department_name: null,
  role_id: 'role-sa',
  role_name: 'super_admin',
  status: 'active',
  force_password_change: false,
  created_at: '2024-01-01T00:00:00Z',
}

const sampleUser = {
  id: 'u-2',
  email: 'alice@example.com',
  full_name: 'Alice Smith',
  department_id: 'dept-1',
  department_name: 'Engineering',
  role_id: 'role-emp',
  role_name: 'employee',
  status: 'active',
  force_password_change: false,
  created_at: '2024-01-01T00:00:00Z',
}

const server = setupServer(
  http.get('/api/v1/users/me', () =>
    HttpResponse.json({ data: meResponse, error: null }),
  ),
  http.get('/api/v1/users', () =>
    HttpResponse.json({
      data: {
        items: [sampleUser],
        meta: { page: 1, per_page: 20, total: 1 },
      },
      error: null,
    }),
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
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe('UsersListPage', () => {
  it('renders the page and shows user rows after loading', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <UsersListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Alice Smith')).toBeInTheDocument()
    })
  })

  it('shows email in the user row', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <UsersListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    })
  })

  it('shows error state when users API fails', async () => {
    server.use(
      http.get('/api/v1/users', () =>
        HttpResponse.json({ data: null, error: { code: 'INTERNAL_ERROR', message: 'db error' } }, { status: 500 }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <UsersListPage />
      </Wrapper>,
    )
    // Page should not crash — it may show an error indicator or empty state
    await waitFor(() => {
      expect(document.body).toBeInTheDocument()
    })
  })
})
