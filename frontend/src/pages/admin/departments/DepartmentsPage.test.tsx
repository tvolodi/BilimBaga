import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { DepartmentsPage } from './DepartmentsPage'

const meAdmin = {
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

const rootDept = {
  id: 'dept-1',
  name: 'Engineering',
  parent_id: null,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [
    {
      id: 'dept-2',
      name: 'Frontend',
      parent_id: 'dept-1',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      children: [],
    },
  ],
}

const server = setupServer(
  http.get('/api/v1/users/me', () => HttpResponse.json({ data: meAdmin, error: null })),
  http.get('/api/v1/departments', () => HttpResponse.json({ data: [rootDept], error: null })),
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

describe('DepartmentsPage', () => {
  it('renders root department node from the API', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
    })
  })

  it('renders child department node (tree is expanded by default)', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Frontend')).toBeInTheDocument()
    })
  })

  it('shows child count badge on parent node', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
    })
    expect(screen.getByText(/1.*children|children.*1/i)).toBeInTheDocument()
  })

  it('shows New Department button for super_admin', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /new department/i })).toBeInTheDocument()
  })

  it('hides New Department button for non-super_admin', async () => {
    server.use(
      http.get('/api/v1/users/me', () =>
        HttpResponse.json({ data: { ...meAdmin, role_name: 'employee' }, error: null }),
      ),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
    })
    expect(screen.queryByRole('button', { name: /new department/i })).not.toBeInTheDocument()
  })

  it('opens the create modal when New Department is clicked', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => screen.getByText('Engineering'))
    await userEvent.click(screen.getByRole('button', { name: /new department/i }))
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
  })

  it('shows empty state when no departments exist', async () => {
    server.use(
      http.get('/api/v1/departments', () => HttpResponse.json({ data: [], error: null })),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText(/no departments yet/i)).toBeInTheDocument()
    })
  })
})
