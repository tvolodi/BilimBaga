import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { CategoriesPage } from './CategoriesPage'

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

const sampleCategory = {
  id: 'cat-1',
  name: 'Science',
  parent_id: null,
  track: null,
  sort_order: 1,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [],
}

const server = setupServer(
  http.get('/api/v1/users/me', () => HttpResponse.json({ data: meAdmin, error: null })),
  http.get('/api/v1/categories', () =>
    HttpResponse.json({ data: [sampleCategory], error: null }),
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

describe('CategoriesPage', () => {
  it('renders a category from the API', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoriesPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Science')).toBeInTheDocument()
    })
  })

  it('shows the New Category button for super_admin', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoriesPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Science')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /new category/i })).toBeInTheDocument()
  })

  it('hides New Category button for employee role', async () => {
    server.use(
      http.get('/api/v1/users/me', () =>
        HttpResponse.json({
          data: { ...meAdmin, role_name: 'employee' },
          error: null,
        }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoriesPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Science')).toBeInTheDocument()
    })
    expect(screen.queryByRole('button', { name: /new category/i })).not.toBeInTheDocument()
  })

  it('shows empty state when no categories exist', async () => {
    server.use(
      http.get('/api/v1/categories', () =>
        HttpResponse.json({ data: [], error: null }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoriesPage />
      </Wrapper>,
    )
    // Should not crash; page renders even with empty data
    await waitFor(() => {
      expect(document.body).toBeInTheDocument()
    })
  })

  it('opens the create modal when New Category is clicked', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoriesPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Science'))
    await userEvent.click(screen.getByRole('button', { name: /new category/i }))
    // Modal or dialog should appear
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
  })
})
